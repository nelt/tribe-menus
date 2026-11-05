# Feuille de route

- **Mise à jour** : 2026-11-05

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
| `canal-prive` | Traitement des vulnérabilités (canal privé écarté) | [`2026-10-04-canal-prive-pour-les-vulnerabilites.md`](plans/2026-10-04-canal-prive-pour-les-vulnerabilites.md) | terminé |
| `revue-securite` | Revue de sécurité avant mise en ligne | [`2026-10-04-revue-de-securite-avant-mise-en-ligne.md`](plans/2026-10-04-revue-de-securite-avant-mise-en-ligne.md) | terminé |
| `production` | Application prête pour la production | [`2026-10-06-application-prete-pour-la-production.md`](plans/2026-10-06-application-prete-pour-la-production.md) | terminé |
| `recette` | Serveur, recette, ouverture de la production et première release | [`2026-10-07-serveur-recette-et-premiere-release.md`](plans/2026-10-07-serveur-recette-et-premiere-release.md) | terminé |
| `bibliotheque` | Bibliothèque de plats, référentiel d'ingrédients, taille de la tribu | [`2026-10-29-bibliotheque-de-plats.md`](plans/2026-10-29-bibliotheque-de-plats.md) | terminé |
| `planning` | Planning des repas | [`2026-11-05-planning-des-repas.md`](plans/2026-11-05-planning-des-repas.md) | prêt |

### `production` : application prête pour la production

Ce qui manque au binaire, sans toucher au serveur : envoi SMTP réel et alerte sur échecs répétés (ADR 0014) ; alerte à l'administrateur sur les limites de demandes de code atteintes de façon répétée (ENF-01, `revue-securite`, D1) ; fichier de configuration et secrets en credentials systemd, écoute sur le socket transmis par systemd, logs en JSON (ADR 0015) ; lecture de `X-Forwarded-For` pour les seules requêtes venant de Caddy, avec le décompte des IPv6 par préfixe /64 (ADR 0006, point 7) ; refus de `-dev` avec l'écoute sur le socket de systemd ou une configuration SMTP, en-tête `To` et enveloppe construits par `net/mail` avec une règle de `ParseEmail` resserrée (`revue-securite`, constats 7 et 8); noms de fichiers du front avec empreinte, `release.yml`, `RELEASING.md` (ADR 0012), qui décrit aussi une version corrective préparée dans la PR du correctif (ADR 0022).

### `recette` : serveur, recette, ouverture de la production et première release

`deploy/provision.sh`, unité systemd confinée, configuration de Caddy, script `tribe-menus-deploy` avec instantané et retour arrière (ADR 0015, 0016) ; DNS et e-mail (CAA, DNSSEC, SPF, DKIM, DMARC, boîtes `no-reply@` et `server@`, ADR 0007 et 0014) ; déploiement en recette et vérification sur téléphone : scénarios `@manuel`, connexion sous Safari (`socle`, D14), réception du code sur les messageries réelles ; ce que la revue de sécurité renvoie à Caddy et au serveur (`revue-securite`, constats 4, 6, 9 et 11 : HSTS, cookie `__Host-` ou recette hors du nom d'hôte de production, dossier de données en 0700 et `UMask=0077`, adresses de la tribu de recette) ; relecture de sécurité des scripts de déploiement dans leurs PR, puis contrôle du serveur en place ; ce que demande le traitement des vulnérabilités (`canal-prive`) : déploiement en production tenant dans une séance, recette ciblée sur une archive de PR, retour arrière fiable, mesure d'attente documentée, répétition d'un traitement accéléré sur un faux constat ; ce que le plan `production` lui remet (« Ce que ce plan ne prouve pas », « Ce que ce plan demande à `recette` ») : unité lancée avec `serve -config`, adresse du client en dernière position de `X-Forwarded-For`, relais par `msmtp` des alertes écrites dans le journal (enregistrements portant l'attribut `alert`, ADR 0023) et adresse de l'administrateur dans `alerts.to`, `deploy/` au complet dans l'archive, première release par une étiquette (contrôle de l'étiquette, notes de version et job `publish` de `release.yml`, que l'essai à blanc n'exerce pas).

Terminé le 2026-11-03 : cinq lots (cookies `__Host-` et tribu de recette, provisionnement, déploiement et supervision, site public minimal, suites de la recette), cinq séances du développeur, chacune avec sa check-list. **La production est ouverte depuis le 2026-11-02** avec `v0.1.0` et la seule tribu du développeur (`recette`, D1). `v0.1.1` a servi à répéter un traitement accéléré, de la PR à la production en 17 minutes, dont 15 d'exposition publique (S5).

### `bibliotheque` : bibliothèque de plats

Premier plan du métier : barre d'onglets, bibliothèque et recherche (P3) dans le navigateur, création et modification d'un plat avec autocomplétion du référentiel (P1, P2, P4), taille de la tribu sur un écran Tribu réduit (T3), compartimentage des plats et du référentiel (ENF-02) ; premiers scénarios `@ui` exécutés par Playwright à partir des `.feature`. Conception dans `conception/bibliotheque-de-plats.md`. Quatre lots (specs et ADR, domaine et API, bibliothèque, fiche d'un plat et écran Tribu), puis la release `v0.2.0`. Préparé pendant la fin de `recette`, rien n'en est fusionné avant l'étiquette `v0.1.0` (D17) : *posée le 2026-11-02, le plan peut avancer.* En cours depuis le 2026-11-03 : lot A (specs, ADR 0027 à 0029, design, conception), lot B (domaine et API), lot C (navigation, bibliothèque, scénarios `@ui` sous Playwright), lot D (fiche d'un plat, écran Tribu, noms affichés) 

Terminé le 2026-11-05 : recette sur l'archive de la PR de release (#83), un défaut et deux remarques corrigés par #84, puis un second passage ; **`v0.2.0` en production le 2026-11-05**. Safari sur iPhone reste à vérifier, faute d'appareil.

### `planning` : planning des repas

Deuxième plan du métier, pour une application viable à montrer : le Planning de `Doux-Main` remplace l'accueil provisoire, 7 jours à partir d'aujourd'hui, midi et soir, semaines précédentes et suivantes (R1, R5) ; détail d'un repas d'après `Doux-Repas`, ajout de plats de la bibliothèque à la taille de la tribu, parts, retrait (R2 à R4) ; planning partagé et compartimenté (T1, T3, ENF-02) ; « Se déconnecter » passe sur l'écran Tribu. Planning chargé en entier comme collection (ADR 0029), « aujourd'hui » pris sur l'appareil. Conception dans `conception/planning-des-repas.md`. Quatre lots (specs et design, domaine et API, planning de la semaine, détail d'un repas), puis la release `v0.3.0` en production. Cadré le 2026-11-05.

## Décisions d'ordonnancement

Le 2026-10-04, avec le développeur.

- **Le déploiement passe avant le métier.** Il rend l'application utilisable ailleurs qu'en local et permet les vérifications renvoyées à la recette.
- **Une revue de sécurité précède le déploiement**, pour ne pas livrer de vulnérabilité : `revue-securite` porte sur le code existant, `recette` relit ses propres scripts.
- **Le traitement des vulnérabilités est fixé avant la revue de sécurité.** Le dépôt étant public, tout ce que les sessions Claude s'écrivent par les PR l'est aussi, vulnérabilités comprises. `canal-prive` a écarté l'idée d'un canal privé et retient une analyse de risques, qui choisit pour chaque vulnérabilité un traitement normal, accéléré ou urgent ; `revue-securite` rode la grille sur ses constats, avant qu'une version soit en ligne.
- **Le premier déploiement vise la recette seule.** À la fin de `socle`, l'application ne fait que connecter un membre ; la production ouvre quand le planning existe. *Révisé le 2026-10-08 (`recette`, D1) : la production ouvre dans le plan `recette`, avec la seule tribu du développeur, pour se placer au plus tôt dans ses conditions ; c'est l'invitation de la famille qui attend le planning.*
- **Le site public attend l'ouverture de la production** : mentions légales et page Confidentialité doivent être en ligne en même temps qu'elle, pas avant. *Toujours vrai au 2026-10-08 : les deux ouvrent ensemble, dans le plan `recette` (lot D).*

Le 2026-10-29, avec le développeur.

- **Le métier commence pendant la fin de `recette`.** Ce qui reste de `recette` attend le poste du développeur et des séances ; le plan `bibliotheque` avance en parallèle, sur ses propres branches, sans rien fusionner avant l'étiquette `v0.1.0` (`bibliotheque`, D17). *La décision du 2026-10-04, « le déploiement passe avant le métier », tient pour la livraison : aucun plat n'est en production avant `v0.2.0`.*

Le 2026-11-05, avec le développeur, après `v0.2.0`.

- **Le fonctionnel passe avant la sauvegarde hors site.** Il s'agit de pouvoir montrer une application viable (planning des repas, listes de courses) ; la sauvegarde hors site et la reconstruction du serveur avec des données suivent, au plus tard avant d'inviter la famille. D'ici là, les données de production (la tribu du développeur et ses plats) n'ont que les instantanés de PulseHeberg et ceux de `tribe-menus-deploy`, sur le même disque (ADR 0025, point 4).
- **Le planning, puis les listes de courses, en deux plans.** `planning` d'abord, livré seul en `v0.3.0` : il est utilisable sans les listes, qui s'appuient sur lui (plats servis, parts). Les listes de courses suivent dans leur propre plan.

## Ensuite, sans ordre arrêté

Ces sujets n'ont pas encore de plan, donc pas encore de nom.

- **Membres et sessions dans l'interface** (EF-01 à EF-07). Débloque le scénario EF-08 « Le premier membre peut se connecter et ajouter des membres ».
- **Métier, suite** : listes de courses, après `planning` (décision du 2026-11-05), leurs collections servies par le mécanisme générique de versions et d'ETag (`bibliotheque`, D28 ; ADR 0029) ; avec elles, les derniers scénarios de compartimentage des données métier (ENF-02), et P2 « La modification d'un plat s'applique aux repas où il est servi ».
- **Hors-ligne et PWA installable** : service worker, IndexedDB, manifeste par tribu, icônes.
- **Administration** : EF-09, EF-10, EF-11.
- **Invitation de la famille** : révocation d'un membre et fermeture d'une session à distance (`revue-securite`, constat 5), puis ouverture de la production à d'autres membres que le développeur. Le site public, la boîte `contact@` et la première release en production passent au plan `recette` (D1).
- **Sauvegarde hors site** par Litestream (ADR 0007, point 3), à décider avant `v0.2.0` : jusque-là, le niveau de PulseHeberg est accepté (ADR 0025, point 4). *Révisé le 2026-11-05 par le développeur : le fonctionnel passe d'abord ; la sauvegarde hors site vient après une application viable à montrer (planning et listes de courses), au plus tard avant d'inviter la famille.*

## Points reportés, à reprendre dans un plan

| Point | Origine | À reprendre |
| --- | --- | --- |
| Réception du code sur Outlook et iCloud non vérifiée | `recette`, S3 | avant d'inviter un membre qui utilise l'une de ces messageries |
| Connexion sous Safari sur un vrai iPhone non vérifiée (WebKit seul) ; de même la bibliothèque et le `304` sur `GET dishes` au retour sur l'application | `recette`, S3 (décision du 2026-11-01) ; `bibliotheque`, recette (revue #77, point 4) | avant d'inviter la famille |
| Sauvegarde hors du VPS | ADR 0025, point 4 | après une application viable à montrer (planning et listes de courses), au plus tard avant d'inviter la famille (décision du 2026-11-05) |
| Un démarrage en échec envoie un e-mail de panne à chaque relance de systemd, dont le texte dit qu'il a cessé de le relancer | `recette`, S4 | prochaine PR touchant `deploy/` |
| `make probe -timing` : le membre passe toujours en premier, sur une connexion à rétablir après l'heure d'attente | `recette`, mesure du 2026-11-01 | avant la prochaine mesure |
| Retour arrière automatique quand une migration échoue à mi-chemin : non éprouvé | `recette`, « Ce que ce plan ne prouve pas » | première migration risquée (la `0007` de `bibliotheque` a réussi sans l'éprouver) |
| Reconstruction du serveur avec des données à reprendre : procédure écrite (étape 19), jouée seulement à vide | `recette`, « Ce que ce plan ne prouve pas » | avec la sauvegarde hors site |
| Délivrabilité dans la durée : rapports DMARC à lire avant de durcir la politique | `recette` ; ADR 0014, point 3 | après quelques semaines de production |
| Seuils des alertes à revoir | `production`, D8 ; `recette` | après les premières semaines de production |
| Durée de HSTS à allonger | `recette` | après quelques semaines sans incident |
| Avis de sécurité GitHub publié après une version corrective, en plus de l'analyse et de `CHANGELOG.md` | `canal-prive` et `recette`, questions ouvertes | à la première vraie vulnérabilité, ou s'il existe d'autres instances |
| Effacement automatique du journal d'audit après 12 mois, annoncé par la page Confidentialité | PT-07 ; `recette`, lot D | avant octobre 2027, au plus tard avec EF-07 ou EF-11 |
| Mise en page d'ordinateur de l'application (navigation latérale, bibliothèque et plat côte à côte) : les écrans restent une colonne de téléphone | recette de `bibliotheque` (2026-11-04) | à maquetter dans Design, puis un plan |
| Mise à jour automatique de l'application : après un déploiement, un onglet resté ouvert garde l'ancienne version sans prévenir, jusqu'à ce qu'on le recharge | recette de `bibliotheque` (2026-11-05) | avec le service worker (ADR 0004), au plus tard avant d'inviter la famille |
| iPad vu comme un ordinateur par la détection de l'appareil | revue de la PR #29 ; `socle`, D12 | EF-04 |
| Fichier de base orphelin si le processus meurt pendant la création d'une tribu | revue de la PR #26 | EF-10 |
| Écriture de la dernière activité à chaque requête | `socle`, questions ouvertes | si la mesure le justifie |
| Aucun moyen de fermer une session à distance ni de révoquer un membre (téléphone perdu) | `revue-securite`, constat 5 | EF-03 et EF-05, ou à défaut une commande d'administration, avant d'inviter un autre membre que le développeur (`recette`, D1) |
