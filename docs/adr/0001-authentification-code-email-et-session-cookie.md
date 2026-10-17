# 0001. Authentification par code e-mail et session par cookie serveur

- **Date** : 2026-09-27
- **Statut** : accepté ; précisé par 0021 (base de limitation des demandes de code)

## Contexte

L'application est une PWA familiale, utilisée sur téléphone (dont iOS en mode installé) et sur ordinateur. L'exigence ENF-01 (`docs/specs/exigences-non-fonctionnelles.md`) demande une connexion simple, une seule fois par appareil, avec une session qui survit au redémarrage du navigateur, et la possibilité de révoquer une session.

Contraintes relevées :

- sur iOS, une PWA installée a un stockage (cookies compris) séparé de Safari ;
- Safari peut purger le stockage écrit en JavaScript (`localStorage`, IndexedDB) d'un site non installé après quelques jours sans visite ;
- les membres de la tribu n'ont pas tous un compte chez un même fournisseur d'identité, et les comptes d'enfants y sont souvent restreints.

## Décision

1. **Login par code à usage unique envoyé par e-mail.** L'utilisateur saisit son adresse, reçoit un code à 6 chiffres et le tape dans l'application *(porté à 8 chiffres par ENF-01, plan `revue-securite`, D1)*. Le code a une validité courte et un nombre de tentatives limité ; il ne se saisit que depuis le navigateur qui l'a demandé *(cookie temporaire posé à la demande, ENF-01, plan `revue-securite`, D5)*. Seules les adresses de membres enregistrés peuvent obtenir un code ; la réponse est identique que l'adresse soit connue ou non.
2. **Session portée par un cookie posé par le serveur** : `HttpOnly`, `Secure`, `SameSite=Lax`, durée de 90 jours à expiration glissante.
3. **Jeton opaque, stocké haché côté serveur.** Le cookie contient une valeur aléatoire ; la base conserve son empreinte avec le membre, les informations d'appareil, la date de création et la dernière activité. Révoquer une session revient à supprimer (ou marquer) cette ligne.

## Alternatives envisagées

- **Google Sign-In (OIDC)** : écarté. Impose un compte Google à chaque membre (comptes enfants supervisés), un projet Google Cloud, un nom de domaine public, et une dépendance à un tiers.
- **Lien magique par e-mail** : écarté. Sur iOS, le lien s'ouvre dans Safari et la session est créée hors de la PWA installée.
- **Mot de passe** : écarté. Gestion (oubli, réinitialisation, robustesse) disproportionnée pour un usage familial.
- **JWT stocké côté client (`localStorage`)** : écarté. Exposé au XSS, révocation difficile avant expiration, stockage purgeable par Safari.
- **Passkeys (WebAuthn) seules** : reportées. Très fluides, mais plus complexes à mettre en place et à récupérer en cas de perte d'appareil ; pourront compléter le code par e-mail plus tard.

## Conséquences

- **Positif** : pas de mot de passe, pas de fournisseur tiers, révocation immédiate et granulaire, session robuste aux redémarrages et aux purges de stockage.
- **Envoi d'e-mails** : un service d'envoi (SMTP ou service transactionnel) devient une dépendance ; sa délivrabilité conditionne la première connexion sur chaque appareil.
- **Stockage des sessions** : une table de sessions est nécessaire, consultée à chaque requête authentifiée (coût négligeable à cette échelle).
- **Une session par tribu** : une instance héberge plusieurs tribus (ENF-02) et une même adresse peut appartenir à plusieurs d'entre elles. Le cookie de session doit donc être propre à une tribu (par exemple `Path` limité au préfixe d'URL de la tribu, ou nom de cookie par tribu), afin qu'un même navigateur puisse être connecté à plusieurs tribus sans que les sessions se mélangent. Le choix précis relève de l'implémentation. *Tranché par l'ADR 0006 : `Path=/tribes/<identifiant>/` ; tranché autrement le 2026-10-08 par l'ADR 0024 : nom de cookie par tribu, préfixé `__Host-`, avec `Path=/`.*
- **Même origine** : le front et l'API doivent être servis depuis le même site pour que le cookie reste first-party.
- **CSRF** : `SameSite=Lax` couvre l'essentiel ; les requêtes modifiant des données passent par des méthodes non-GET et une vérification de l'en-tête `Origin`.
- **Hors-ligne** : sans réseau, la PWA affiche les données en cache ; la session est revérifiée au retour de la connexion.
