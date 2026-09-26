# Exigences non fonctionnelles

Exigences transverses de l'application familiale de menus de la semaine (PWA).

## ENF-01. Authentification simple et session persistante

- **Statut** : retenue (2026-09-27)
- **Décision d'architecture** : [ADR 0001](../adr/0001-authentification-code-email-et-session-cookie.md)

### Exigence

L'utilisateur se connecte une seule fois par appareil. Sa session survit à la fermeture et au redémarrage du navigateur, ainsi qu'à la fermeture de la PWA installée.

### Modalités retenues

- **Login initial par code à usage unique envoyé par e-mail** : l'utilisateur saisit son adresse, reçoit un code à 6 chiffres et le tape dans l'application. Pas de mot de passe.
  - Un code, plutôt qu'un lien magique : sur iOS, une PWA installée a un stockage séparé de Safari, et un lien ouvert depuis le mail créerait la session dans Safari et non dans la PWA.
  - Le code a une durée de validité courte et un nombre de tentatives limité.
- **Session portée par un cookie persistant posé par le serveur** :
  - attributs `HttpOnly`, `Secure`, `SameSite=Lax`, durée de 90 jours ;
  - expiration glissante : chaque utilisation prolonge la session ;
  - le cookie contient un jeton opaque aléatoire ; le serveur n'en conserve qu'une empreinte (hachage), associée à l'utilisateur et à l'appareil.
- **Portée** : une session est ouverte pour une famille donnée et ne vaut que pour elle (voir ENF-02).
- **Révocation** : l'utilisateur peut consulter la liste de ses appareils connectés et fermer une session à distance (détail : [gestion des membres et des sessions](gestion-membres-et-sessions.md), EF-04 et EF-05).
- **Hors-ligne** : l'interface et les données en cache restent consultables sans réseau ; la validité de la session est vérifiée au retour de la connexion.

### Hors périmètre (pour l'instant)

- Connexion via un fournisseur tiers (Google, etc.) : explorée puis écartée.
- Passkeys (WebAuthn) : évolution envisageable plus tard, en complément du code par e-mail.
- Jetons stockés côté client en `localStorage` : exclus (exposition au XSS, révocation difficile, purge possible du stockage par Safari).

## ENF-02. Compartimentage strict des données entre familles

- **Statut** : retenue (2026-09-27)

### Exigence

Les données d'une famille ne sont jamais accessibles depuis une autre famille, ni en lecture ni en écriture.

### Modalités retenues

- **Rattachement systématique** : toute donnée métier (membres, sessions, codes de connexion, menus, journal d'audit…) appartient à exactement une famille. Aucune donnée n'est partagée entre familles.
- **Famille déterminée par la session** : la famille de chaque requête authentifiée est celle de la session, jamais un paramètre fourni par le client. Le discriminant de l'URL doit correspondre à la famille de la session ; sinon la requête est traitée comme non authentifiée pour cette famille.
- **Filtrage imposé en un point central** : l'accès aux données passe par une couche qui applique le filtre par famille, plutôt que par un filtre ajouté à la main dans chaque requête.
- **Pas de fuite indirecte** : les messages d'erreur, les réponses de connexion et les identifiants exposés ne permettent pas de déduire l'existence de données ou de membres d'une autre famille.
- **Même adresse, familles distinctes** : une personne membre de plusieurs familles a une identité et des sessions distinctes dans chacune ; rien ne relie ses appartenances côté application.
- **Application installée** : une PWA installée est rattachée à une famille (son URL de démarrage contient le discriminant).
- **Vérification** : des tests automatisés couvrent les tentatives d'accès croisé (lecture, modification, suppression) entre deux familles.

### Hors périmètre

- Consultation ou administration transverse des familles depuis l'interface : l'administration passe par les scripts serveur.
