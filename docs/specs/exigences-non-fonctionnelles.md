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
  - **Le code ne se saisit que depuis le navigateur qui l'a demandé** (2026-10-05, plan `revue-securite`, D5) : la demande de code pose un cookie temporaire, propre à la tribu et valable 10 minutes comme le code, dont le serveur ne garde que l'empreinte, avec le code. Un essai présenté sans ce cookie reçoit la réponse d'un code à redemander et ne consomme aucun essai. Sans cela, qui connaît l'adresse d'un membre épuiserait les 3 essais de chacun de ses codes, sans demande ni e-mail, et l'empêcherait de se connecter sans que rien ne le signale. La règle vaut de la même façon pour une adresse inconnue, révoquée ou d'une autre tribu, et pour une tribu qui n'existe pas (ENF-02). Limite assumée : un code demandé sur un appareil ne se saisit pas sur un autre ; il faut le redemander depuis celui-ci.
  - **Limitation des demandes de code** : 3 demandes par quart d'heure pour une même adresse dans une tribu, 10 par heure depuis une même adresse IP, 30 par heure adressées aux membres actifs d'une même tribu. Au-delà des deux premières, l'application affiche « Réessayez dans quelques minutes » et n'envoie rien ; au-delà de la limite par tribu, elle n'envoie rien non plus, mais répond comme pour toute adresse (2026-10-20, ADR 0026). La limite par adresse s'applique de la même façon à une adresse inconnue, révoquée ou d'une autre tribu, pour ne rien révéler (ENF-02). Elle protège aussi le quota d'envoi du compte e-mail (ADR 0014).
  - **L'adresse IP comptée est celle du client** (2026-10-06, plan `production`, D3 et D4). Derrière Caddy, c'est celle qu'il transmet, la dernière de l'en-tête `X-Forwarded-For` ; l'en-tête n'est lu que pour les requêtes venues de Caddy. Une adresse IPv6 compte pour son préfixe /64, taille de ce qu'un fournisseur attribue au moins à un abonné : sans cela, la limite se contournerait en changeant d'adresse dans son propre préfixe. Limite assumée : les abonnés d'un même /64 partagent la limite, comme ceux d'une même IPv4 derrière un NAT.
  - **La limite par tribu ne compte que les demandes adressées à ses membres actifs** (2026-10-05, plan `revue-securite`, D2) : des demandes pour des adresses au hasard ne peuvent pas empêcher toute connexion à la tribu ; il faut connaître des adresses de membres. Une tribu inexistante n'atteint donc jamais cette limite, comme une tribu existante visée par des adresses au hasard. Le refus est silencieux (2026-10-20, plan `recette`, étape 31 ; [ADR 0026](../adr/0026-demande-de-code-traitee-en-arriere-plan.md)) : la réponse est écrite avant que le membre soit cherché, et ne peut donc pas dire qu'une limite propre aux membres est atteinte. Jusque-là, une adresse de membre recevait « Réessayez dans quelques minutes » et une autre adresse non. Limite assumée : pendant une attaque, un membre attend un code qui ne vient pas ; l'administrateur, lui, est alerté.
  - **Les limites atteintes de façon répétée sont signalées à l'administrateur** par e-mail (2026-10-05, plan `revue-securite`, D1), avec l'envoi réel des e-mails (plan `production`) : une force brute ou un blocage ciblé est visible des victimes, cette alerte le rend visible de l'administrateur. « De façon répétée » (2026-10-06, plan `production`, D8 et D12 ; ADR 0023) : sur la dernière heure et pour toute l'instance, **dix demandes refusées** par une limite, **cinq codes invalidés** par leur dernier essai erroné, ou **neuf demandes acceptées pour une même adresse** dans une même tribu (blocage ciblé au rythme exact de la limite). Une alerte par signal et par six heures au plus, tant que la situation dure. L'e-mail ne nomme ni adresse e-mail, ni adresse IP, ni tribu : les journaux d'accès du serveur donnent le reste.
- **Session portée par un cookie persistant posé par le serveur** :
  - attributs `HttpOnly`, `Secure`, `SameSite=Lax`, durée de 90 jours ;
  - préfixe `__Host-`, sans attribut `Domain`, `Path=/` et un nom propre à la tribu, de même que le cookie de la demande de code (2026-10-08, plan `recette`, D2 ; [ADR 0024](../adr/0024-cookies-prefixes-host-et-nommes-par-tribu.md)) : aucun autre hôte du domaine, recette comprise, ne peut poser un cookie que l'application lirait, et un même navigateur garde une session par tribu ;
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
- **Pas de fuite indirecte** : les messages d'erreur, les réponses de connexion et les identifiants exposés ne permettent pas de déduire l'existence de données ou de membres d'une autre tribu. Le temps de réponse non plus : une demande de code est acceptée ou refusée par les seules limites par adresse et par IP, et la réponse est écrite avant que la base de la tribu soit ouverte ; la recherche du membre, l'écriture du code et l'envoi viennent ensuite, en arrière-plan (2026-10-20, ADR 0026).
- **Rien n'est révélé avant connexion** : sans session valide, l'URL d'une tribu affiche un écran de connexion générique, identique que la tribu existe ou non. Le nom de la tribu n'apparaît qu'une fois la connexion réussie.
- **Même adresse, tribus distinctes** : une personne membre de plusieurs tribus a une identité et des sessions distinctes dans chacune ; rien ne relie ses appartenances côté application.
- **Application installée** : une PWA installée est rattachée à une tribu (son URL de démarrage contient le discriminant).
- **Vérification** : des tests automatisés couvrent les tentatives d'accès croisé (lecture, modification, suppression) entre deux tribus.

### Hors périmètre

- Consultation ou administration transverse des tribus depuis l'interface : l'administration passe par les scripts serveur.

## ENF-03. Utilisable sur téléphone, tablette et ordinateur

- **Statut** : retenue (2026-11-08, plan `grands-ecrans`)
- **Décision d'architecture** : [ADR 0030](../adr/0030-mises-en-page-selon-la-taille-de-la-fenetre.md)
- **Critères d'acceptation** : `features/formats-ecran.feature`

### Exigence

L'application, connexion comprise, s'utilise sur un téléphone, une tablette (en portrait comme en paysage) et un ordinateur, au doigt, à la souris et au clavier, sans défilement horizontal de la page ni zone vide démesurée, de 320 px à 2560 px de large.

### Modalités retenues

- **La fenêtre décide, jamais l'appareil** : trois formats, d'après la largeur et la hauteur de la fenêtre. **Compact** : moins de 720 px de large, ou moins de 560 px de haut (téléphone, en portrait et en paysage). **Moyen** : de 720 à 1023 px de large, 560 px de haut au moins (tablette en portrait, fenêtre étroite). **Étendu** : 1024 px de large et 560 px de haut au moins (tablette en paysage, ordinateur). Aucune détection de l'appareil : un iPad se présente comme un Mac.
- **Ce que montre chaque format** (`docs/design/README.md`, « Formats d'écran ») : en compact, la mise en page du téléphone, inchangée ; en moyen et en étendu, un rail de navigation à gauche et des dialogues centrés à la place des feuilles ; en étendu, le planning en semainier avec le détail d'un repas en panneau, la bibliothèque et le plat, le choix des listes et la liste en deux volets.
- **Rien ne se perd en changeant de format** (rotation d'une tablette, fenêtre redimensionnée) : l'écran, son chemin, la saisie en cours et la position du volet de détail restent.
- **Confirmation avant d'abandonner une saisie**, en moyen et en étendu : une navigation dans l'application qui ferait perdre une saisie non enregistrée (plat, article à la main, nom d'une liste) demande « Abandonner la modification ? ». Ni le retour arrière du navigateur ni la fermeture de l'onglet ne la demandent ; en compact, aucune confirmation.
- **Clavier virtuel** : il ne fait pas basculer une tablette en paysage dans le format compact.
- **Interaction** : cibles de 44 px dans tous les formats ; survol et curseur seulement avec une souris (`(hover: hover) and (pointer: fine)`) ; focus visible ; Échap ferme un dialogue ou le panneau d'un repas.
- **Vérification** : les scénarios `@ui` de `formats-ecran.feature`, avec leurs tags de format, sur chaque PR dans les projets Playwright de la tablette (portrait et paysage) et de l'ordinateur ; tous les scénarios `@ui` dans tous les formats sur la PR de release et chaque nuit (plan `grands-ecrans`, D14) ; le clavier virtuel en recette, sur une tablette Android.

### Hors périmètre

- Raccourcis clavier.
- Mise en page propre à l'impression.
- iPad et Safari sur Mac ne sont prouvés que par WebKit sous Playwright, faute d'appareil Apple en recette.
