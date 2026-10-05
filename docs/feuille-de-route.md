# Feuille de route

- **Mise à jour** : 2026-10-05

Ordre des plans de travail. Ce document dit **dans quel ordre** on avance et pourquoi ; le détail de chaque sujet est dans son plan (`plans/`), les décisions dans les ADR et les specs. Il est mis à jour quand un plan est créé, terminé ou déplacé, dans la même PR.

## Vocabulaire et nommage

| Terme | Ce que c'est | Comment on le désigne |
| --- | --- | --- |
| **Feuille de route** | La liste ordonnée des plans. | Ce document. |
| **Plan** | Un objectif et ce qu'il faut pour l'atteindre : contexte, étapes, décisions. Un fichier `plans/AAAA-MM-JJ-sujet.md`. | Par son **nom** : court, en minuscules, sans accent, fixé à la création et jamais changé (`socle`). |
| **Lot** | Une partie d'un plan livrée par une PR fusionnable seule. N'existe que si le plan demande plusieurs PR. | Par le nom de son plan et une lettre : `socle/C`, ou « lot C du plan `socle` ». Dans le plan lui-même, « lot C » suffit. |
| **Étape** | Une case à cocher d'un plan. | Par son numéro, continu d'un lot à l'autre. |

- **Un plan n'a pas de numéro.** Son rang est sa position dans le tableau ci-dessous, qui change quand un plan est inséré ; son nom, lui, ne change pas.
- **Les lettres des lots** suivent l'ordre de réalisation et ne sont jamais réattribuées : la lettre d'un lot abandonné ou scindé n'est pas reprise.
- **Les décisions** d'un plan sont numérotées dans ce plan (D1, D2…) ; hors du plan, on écrit `socle`, D14.
- **Le nom figure dans l'en-tête du plan** (ligne « Nom »), à côté de la date et du statut.
- **Titres de PR et lignes de `CHANGELOG.md`** : le lot est cité à côté des stories, sous la forme `socle/C`.
- **Branches** : décrites par leur sujet (`feature/enf-01-ecrans-de-connexion`), sans le nom du plan.

## Plans, dans l'ordre

| Nom | Sujet | Plan | État |
| --- | --- | --- | --- |
| `environnement` | Environnement de développement | [`2026-09-29-environnement-de-developpement.md`](plans/2026-09-29-environnement-de-developpement.md) | terminé |
| `ci` | Intégration continue | [`2026-09-30-integration-continue.md`](plans/2026-09-30-integration-continue.md) | terminé |
| `socle` | Socle de données et connexion | [`2026-10-03-socle-donnees-et-connexion.md`](plans/2026-10-03-socle-donnees-et-connexion.md) | terminé |
| `canal-prive` | Traitement des vulnérabilités (canal privé écarté) | [`2026-10-04-canal-prive-pour-les-vulnerabilites.md`](plans/2026-10-04-canal-prive-pour-les-vulnerabilites.md) | en cours : document, ADR 0022 et réglages faits (étapes 1 à 6) ; reste la mise à l'épreuve de la grille par `revue-securite` |
| `revue-securite` | Revue de sécurité avant mise en ligne | [`2026-10-04-revue-de-securite-avant-mise-en-ligne.md`](plans/2026-10-04-revue-de-securite-avant-mise-en-ligne.md) | en cours : décisions de specs, revue, arbitrage et corrections faits (étapes 1 à 6) ; restent les protections du dépôt (étape 7) et la clôture |
| `production` | Application prête pour la production | à écrire | après `revue-securite` |
| `recette` | Serveur, recette et première release | à écrire | après `production` |

### `production` : application prête pour la production

Ce qui manque au binaire, sans toucher au serveur : envoi SMTP réel et alerte sur échecs répétés (ADR 0014) ; alerte à l'administrateur sur les limites de demandes de code atteintes de façon répétée (ENF-01, `revue-securite`, D1) ; fichier de configuration et secrets en credentials systemd, écoute sur le socket transmis par systemd, logs en JSON (ADR 0015) ; lecture de `X-Forwarded-For` pour les seules requêtes venant de Caddy, avec le décompte des IPv6 par préfixe /64 (ADR 0006, point 7) ; refus de `-dev` avec l'écoute sur le socket de systemd ou une configuration SMTP, en-tête `To` et enveloppe construits par `net/mail` avec une règle de `ParseEmail` resserrée (`revue-securite`, constats 7 et 8); noms de fichiers du front avec empreinte, `release.yml`, `RELEASING.md` (ADR 0012), qui décrit aussi une version corrective préparée dans la PR du correctif (ADR 0022).

### `recette` : serveur, recette et première release

`deploy/provision.sh`, unité systemd confinée, configuration de Caddy, script `tribe-menus-deploy` avec instantané et retour arrière (ADR 0015, 0016) ; DNS et e-mail (CAA, DNSSEC, SPF, DKIM, DMARC, boîtes `no-reply@` et `server@`, ADR 0007 et 0014) ; déploiement en recette et vérification sur téléphone : scénarios `@manuel`, connexion sous Safari (`socle`, D14), réception du code sur les messageries réelles ; ce que la revue de sécurité renvoie à Caddy et au serveur (`revue-securite`, constats 4, 6, 9 et 11 : HSTS, cookie `__Host-` ou recette hors du nom d'hôte de production, dossier de données en 0700 et `UMask=0077`, adresses de la tribu de recette) ; relecture de sécurité des scripts de déploiement dans leurs PR, puis contrôle du serveur en place ; ce que demande le traitement des vulnérabilités (`canal-prive`) : déploiement en production tenant dans une séance, recette ciblée sur une archive de PR, retour arrière fiable, mesure d'attente documentée, répétition d'un traitement accéléré sur un faux constat.

## Décisions d'ordonnancement

Le 2026-10-04, avec le développeur.

- **Le déploiement passe avant le métier.** Il rend l'application utilisable ailleurs qu'en local et permet les vérifications renvoyées à la recette.
- **Une revue de sécurité précède le déploiement**, pour ne pas livrer de vulnérabilité : `revue-securite` porte sur le code existant, `recette` relit ses propres scripts.
- **Le traitement des vulnérabilités est fixé avant la revue de sécurité.** Le dépôt étant public, tout ce que les sessions Claude s'écrivent par les PR l'est aussi, vulnérabilités comprises. `canal-prive` a écarté l'idée d'un canal privé et retient une analyse de risques, qui choisit pour chaque vulnérabilité un traitement normal, accéléré ou urgent ; `revue-securite` rode la grille sur ses constats, avant qu'une version soit en ligne.
- **Le premier déploiement vise la recette seule.** À la fin de `socle`, l'application ne fait que connecter un membre ; la production ouvre quand le planning existe.
- **Le site public attend l'ouverture de la production** : mentions légales et page Confidentialité doivent être en ligne en même temps qu'elle, pas avant.

## Ensuite, sans ordre arrêté

Ces sujets n'ont pas encore de plan, donc pas encore de nom.

- **Membres et sessions dans l'interface** (EF-01 à EF-07). Débloque le scénario EF-08 « Le premier membre peut se connecter et ajouter des membres ».
- **Métier** : bibliothèque de plats, planning des repas, listes de courses ; avec eux, les scénarios de compartimentage des données métier (ENF-02).
- **Hors-ligne et PWA installable** : service worker, IndexedDB, manifeste par tribu, icônes.
- **Administration** : EF-09, EF-10, EF-11.
- **Ouverture de la production** : site public dans `site/`, boîte `contact@codingmatters.org`, première release déployée en production.
- **Sauvegarde hors site** par Litestream (ADR 0007, point 3).

## Points reportés, à reprendre dans un plan

| Point | Origine | À reprendre |
| --- | --- | --- |
| iPad vu comme un ordinateur par la détection de l'appareil | revue de la PR #29 ; `socle`, D12 | EF-04 |
| Adresse IP du client derrière Caddy ; IPv6 par préfixe /64 | revue de la PR #29 | `production` |
| Fichier de base orphelin si le processus meurt pendant la création d'une tribu | revue de la PR #26 | EF-10 |
| Lien entre Playwright et les `.feature` (tag `@ui`) | `socle`, D5 | première story prouvée seulement dans le navigateur (C5 ou C7) |
| Écriture de la dernière activité à chaque requête | `socle`, questions ouvertes | si la mesure le justifie |
| Aucun moyen de fermer une session à distance ni de révoquer un membre (téléphone perdu) | `revue-securite`, constat 5 | EF-03 et EF-05, ou à défaut une commande d'administration, avant l'ouverture de la production |
| Cookie de session sans préfixe `__Host-`, recette sous-domaine de la production | `revue-securite`, constat 6 | `recette`, avant la première session réelle |
| `ParseEmail` accepte des adresses que l'envoi SMTP devra refuser | `revue-securite`, constat 7 | `production`, avec l'envoi SMTP |
| Rien ne refuse `-dev` hors du développement | `revue-securite`, constat 8 | `production` |
| Tribu de démonstration en `@exemple.fr`, domaine que le projet ne contrôle pas | `revue-securite`, constat 9 | `recette` |
| `webcheck` ne connaît pas toutes les échappatoires au rendu échappé | `revue-securite`, constat 10 | premier écran qui affiche un texte saisi par un membre (EF-01, EF-02 ou bibliothèque de plats) |
| Ce qui revient à Caddy : HSTS, en-têtes du site public, `Host` transmis tel quel, `/healthz` non transmis, délais et taille des requêtes, journaux d'accès | `revue-securite`, constat 11 | `recette` |
| Dossier de données en 0700 et `UMask=0077` dans l'unité systemd | `revue-securite`, constat 4 | `recette` |
| Alerte de D1 : compter aussi les codes invalidés par essais épuisés | `revue-securite`, constat 1 | `production` |
| Temps de réponse : membre ou non, tribu existante ou non, sur `POST sessions` d'abord | `revue-securite`, ce qui n'a pas pu être vérifié | `recette` |
