# Exigences non fonctionnelles

Exigences transverses de l'application familiale de menus de la semaine (PWA).

## ENF-01. Authentification simple et session persistante

- **Statut** : retenue (2026-09-27)

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
- **Révocation** : l'utilisateur peut consulter la liste de ses appareils connectés et fermer une session à distance.
- **Hors-ligne** : l'interface et les données en cache restent consultables sans réseau ; la validité de la session est vérifiée au retour de la connexion.

### Hors périmètre (pour l'instant)

- Connexion via un fournisseur tiers (Google, etc.) : explorée puis écartée.
- Passkeys (WebAuthn) : évolution envisageable plus tard, en complément du code par e-mail.
- Jetons stockés côté client en `localStorage` : exclus (exposition au XSS, révocation difficile, purge possible du stockage par Safari).
