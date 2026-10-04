# Feuille de route

- **Mise à jour** : 2026-10-04

Ordre des plans de travail. Ce document dit **dans quel ordre** on avance et pourquoi ; le détail de chaque sujet est dans son plan (`plans/`), les décisions dans les ADR et les specs. Il est mis à jour quand un plan est créé, terminé ou déplacé, dans la même PR.

## Plans, dans l'ordre

| # | Sujet | Plan | État |
| --- | --- | --- | --- |
| 1 | Environnement de développement | [`2026-09-29-environnement-de-developpement.md`](plans/2026-09-29-environnement-de-developpement.md) | terminé |
| 2 | Intégration continue | [`2026-09-30-integration-continue.md`](plans/2026-09-30-integration-continue.md) | terminé |
| 3 | Socle de données et connexion | [`2026-10-03-socle-donnees-et-connexion.md`](plans/2026-10-03-socle-donnees-et-connexion.md) | en cours : lots A et B fusionnés, lot C à réaliser |
| 4 | Revue de sécurité avant mise en ligne | [`2026-10-04-revue-de-securite-avant-mise-en-ligne.md`](plans/2026-10-04-revue-de-securite-avant-mise-en-ligne.md) | prêt ; commence après le lot C |
| 5 | Application prête pour la production | à écrire | après le plan 4 |
| 6 | Serveur, recette et première release | à écrire | après le plan 5 |

### 5. Application prête pour la production

Ce qui manque au binaire, sans toucher au serveur : envoi SMTP réel et alerte sur échecs répétés (ADR 0014) ; fichier de configuration et secrets en credentials systemd, écoute sur le socket transmis par systemd, logs en JSON (ADR 0015) ; lecture de `X-Forwarded-For` pour les seules requêtes venant de Caddy, avec le décompte des IPv6 par préfixe /64 (ADR 0006, point 7) ; noms de fichiers du front avec empreinte, `release.yml`, `RELEASING.md` (ADR 0012).

### 6. Serveur, recette et première release

`deploy/provision.sh`, unité systemd confinée, configuration de Caddy, script `tribe-menus-deploy` avec instantané et retour arrière (ADR 0015, 0016) ; DNS et e-mail (CAA, DNSSEC, SPF, DKIM, DMARC, boîtes `no-reply@` et `server@`, ADR 0007 et 0014) ; déploiement en recette et vérification sur téléphone : scénarios `@manuel`, connexion sous Safari (plan 3, D14), réception du code sur les messageries réelles ; relecture de sécurité des scripts de déploiement dans leurs PR, puis contrôle du serveur en place.

## Décisions d'ordonnancement

Le 2026-10-04, avec le développeur.

- **Le déploiement passe avant le métier.** Il rend l'application utilisable ailleurs qu'en local et permet les vérifications renvoyées à la recette.
- **Une revue de sécurité précède le déploiement**, pour ne pas livrer de vulnérabilité : le plan 4 porte sur le code existant, le plan 6 relit ses propres scripts.
- **Le premier déploiement vise la recette seule.** À la fin du plan 3, l'application ne fait que connecter un membre ; la production ouvre quand le planning existe.
- **Le site public attend l'ouverture de la production** : mentions légales et page Confidentialité doivent être en ligne en même temps qu'elle, pas avant.

## Ensuite, sans ordre arrêté

- **Membres et sessions dans l'interface** (EF-01 à EF-07). Débloque le scénario EF-08 « Le premier membre peut se connecter et ajouter des membres ».
- **Métier** : bibliothèque de plats, planning des repas, listes de courses ; avec eux, les scénarios de compartimentage des données métier (ENF-02).
- **Hors-ligne et PWA installable** : service worker, IndexedDB, manifeste par tribu, icônes.
- **Administration** : EF-09, EF-10, EF-11.
- **Ouverture de la production** : site public dans `site/`, boîte `contact@codingmatters.org`, première release déployée en production.
- **Sauvegarde hors site** par Litestream (ADR 0007, point 3).

## Points reportés, à reprendre dans un plan

| Point | Origine | À reprendre |
| --- | --- | --- |
| Force brute sur le code de connexion ; blocage ciblé d'une tribu par ses limites | revue de la PR #29 | plan 4 |
| Forme de l'appareil détecté dans le journal d'audit ; iPad vu comme un ordinateur | revue de la PR #29, D12 | plan 4, puis EF-04 |
| Adresse IP du client derrière Caddy ; IPv6 par préfixe /64 | revue de la PR #29 | plan 5 |
| Fichier de base orphelin si le processus meurt pendant la création d'une tribu | revue de la PR #26 | EF-10 |
| Lien entre Playwright et les `.feature` (tag `@ui`) | plan 3, D5 | première story prouvée seulement dans le navigateur (C5 ou C7) |
| Écriture de la dernière activité à chaque requête | plan 3, questions ouvertes | si la mesure le justifie |
