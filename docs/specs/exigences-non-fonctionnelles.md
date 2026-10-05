# Exigences non fonctionnelles

Exigences transverses de l'application familiale de menus de la semaine (PWA).

## ENF-01. Authentification simple et session persistante

- **Statut** : retenue (2026-09-27)
- **Décision d'architecture** : [ADR 0001](../adr/0001-authentification-code-email-et-session-cookie.md)
- **Critères d'acceptation** : `features/authentification.feature`

### Exigence

L'utilisateur se connecte une seule fois par appareil. Sa session survit à la fermeture et au redémarrage du navigateur, ainsi qu'à la fermeture de la PWA installée.

### Modalités retenues

- **Login initial par code à usage unique envoyé par e-mail** : l'utilisateur saisit son adresse, reçoit un code à 8 chiffres et le tape dans l'application. Pas de mot de passe.
  - 8 chiffres plutôt que 6 (2026-10-05, plan `revue-securite`, D1) : avec les limites ci-dessous, une force brute continue sur une adresse a environ 0,3 % de chances de réussir en un an, contre 1 sur 4 avec 6 chiffres ; sur toute une tribu, 0,8 % contre 1 sur 2. Allonger le code ne crée pas de nouvelle façon de bloquer un membre, ce que ferait un plafond plus strict des demandes.
  - Un code, plutôt qu'un lien magique : sur iOS, une PWA installée a un stockage séparé de Safari, et un lien ouvert depuis le mail créerait la session dans Safari et non dans la PWA.
  - Le code est valable **10 minutes** et accepte **3 essais** ; au troisième essai erroné, il est invalidé.
  - Un seul code valable à la fois par adresse et par tribu : une nouvelle demande invalide le code précédent.
  - **Limitation des demandes de code** : 3 demandes par quart d'heure pour une même adresse dans une tribu, 10 par heure depuis une même adresse IP, 30 par heure adressées aux membres actifs d'une même tribu. Au-delà, l'application affiche « Réessayez dans quelques minutes » et n'envoie rien. La limite par adresse s'applique de la même façon à une adresse inconnue, révoquée ou d'une autre tribu, pour ne rien révéler (ENF-02). Elle protège aussi le quota d'envoi du compte e-mail (ADR 0014).
  - **La limite par tribu ne compte que les demandes adressées à ses membres actifs** (2026-10-05, plan `revue-securite`, D2) : des demandes pour des adresses au hasard ne peuvent pas empêcher toute connexion à la tribu ; il faut connaître des adresses de membres. Une tribu inexistante n'atteint donc jamais cette limite, comme une tribu existante visée par des adresses au hasard. Limite assumée : quand la limite d'une tribu est atteinte, une adresse de membre reçoit « Réessayez dans quelques minutes » et une autre adresse non ; ce cas demande 30 demandes pour des membres dans l'heure.
  - **Les limites atteintes de façon répétée sont signalées à l'administrateur** par e-mail (2026-10-05, plan `revue-securite`, D1), avec l'envoi réel des e-mails (plan `production`) : une force brute ou un blocage ciblé est visible des victimes, cette alerte le rend visible de l'administrateur.
- **Session portée par un cookie persistant posé par le serveur** :
  - attributs `HttpOnly`, `Secure`, `SameSite=Lax`, durée de 90 jours ;
  - expiration glissante : chaque utilisation prolonge la session ;
  - le cookie contient un jeton opaque aléatoire ; le serveur n'en conserve qu'une empreinte (hachage), associée à l'utilisateur et à l'appareil.
- **Portée** : une session est ouverte pour une tribu donnée et ne vaut que pour elle (voir ENF-02).
- **Révocation** : l'utilisateur peut consulter la liste de ses appareils connectés et fermer une session à distance (détail : [gestion des membres et des sessions](gestion-membres-et-sessions.md), EF-04 et EF-05).
- **Hors-ligne** (PT-04, 2026-09-28) :
  - restent consultables sans réseau le planning, la bibliothèque de plats, le référentiel d'ingrédients (pour l'autocomplétion) et les listes de courses déjà affichés ;
  - seules les coches et l'ajout d'articles fonctionnent hors ligne (Q16) ; toute autre modification (planning, plats, membres, sessions) affiche « Réseau nécessaire » et ne change rien ;
  - la validité de la session est vérifiée au retour de la connexion ;
  - les données de la tribu stockées sur l'appareil (caches, IndexedDB, file d'opérations) sont effacées à la déconnexion et dès que le serveur signale une session invalide (session révoquée ou expirée, membre révoqué).

### Hors périmètre (pour l'instant)

- Connexion via un fournisseur tiers (Google, etc.) : explorée puis écartée.
- Passkeys (WebAuthn) : évolution envisageable plus tard, en complément du code par e-mail.
- Jetons stockés côté client en `localStorage` : exclus (exposition au XSS, révocation difficile, purge possible du stockage par Safari).

## ENF-02. Compartimentage strict des données entre tribus

- **Statut** : retenue (2026-09-27)
- **Décision d'architecture** : [ADR 0003](../adr/0003-stockage-sqlite-une-base-par-tribu.md) (une base SQLite par tribu)
- **Critères d'acceptation** : `features/compartimentage-tribus.feature`

### Exigence

Les données d'une tribu ne sont jamais accessibles depuis une autre tribu, ni en lecture ni en écriture.

### Modalités retenues

- **Rattachement systématique** : toute donnée métier (membres, sessions, codes de connexion, bibliothèque de plats, référentiel d'ingrédients, planning, listes de courses, journal d'audit…) appartient à exactement une tribu. Aucune donnée n'est partagée entre tribus.
- **Tribu déterminée par la session** : la tribu de chaque requête authentifiée est celle de la session, jamais un paramètre fourni par le client. Le discriminant de l'URL doit correspondre à la tribu de la session ; sinon la requête est traitée comme non authentifiée pour cette tribu.
- **Filtrage imposé en un point central** : l'accès aux données passe par une couche qui applique le filtre par tribu, plutôt que par un filtre ajouté à la main dans chaque requête.
- **Pas de fuite indirecte** : les messages d'erreur, les réponses de connexion et les identifiants exposés ne permettent pas de déduire l'existence de données ou de membres d'une autre tribu.
- **Rien n'est révélé avant connexion** : sans session valide, l'URL d'une tribu affiche un écran de connexion générique, identique que la tribu existe ou non. Le nom de la tribu n'apparaît qu'une fois la connexion réussie.
- **Même adresse, tribus distinctes** : une personne membre de plusieurs tribus a une identité et des sessions distinctes dans chacune ; rien ne relie ses appartenances côté application.
- **Application installée** : une PWA installée est rattachée à une tribu (son URL de démarrage contient le discriminant).
- **Vérification** : des tests automatisés couvrent les tentatives d'accès croisé (lecture, modification, suppression) entre deux tribus.

### Hors périmètre

- Consultation ou administration transverse des tribus depuis l'interface : l'administration passe par les scripts serveur.
