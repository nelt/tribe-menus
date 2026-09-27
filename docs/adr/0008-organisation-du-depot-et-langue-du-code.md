# 0008. Organisation du dépôt et langue du code

- **Date** : 2026-09-27
- **Statut** : accepté

## Contexte

Le projet réunit un back-end Go (ADR 0002), un front-end TypeScript (ADR 0004), un site public statique et la configuration Caddy (ADR 0006), des scénarios d'acceptation (ADR 0005) et la documentation. Les spécifications sont en français. Le dépôt s'appelle encore `personal-sandbox`, un nom de bac à sable, alors que le chemin du module Go contient le nom du dépôt et apparaît dans chaque import.

## Décision

1. **Un seul dépôt**, renommé **`tribe-menus`** avant l'initialisation du code. Chemin du module Go : `github.com/nelt/tribe-menus`.
2. **Tout le code est en anglais** : identifiants, noms de paquets, commentaires, messages de log. Les spécifications et les scénarios Gherkin restent en français ; la correspondance des termes métier est fixée par `docs/specs/glossaire.md`, que le code suit à la lettre. Les textes affichés à l'utilisateur sont en français en V1.
3. **Organisation**, selon les conventions de la communauté Go :

   ```
   go.mod               module github.com/nelt/tribe-menus
   Makefile
   cmd/tribe-menus/     point d'entrée : sous-commandes serve et admin
   internal/
     tribe/             membres, sessions, connexion, audit
     dish/              plats et référentiel d'ingrédients
     mealplan/          repas et plats servis
     shopping/          listes de courses, articles, recalcul
     quantity/          unités, conversions, arrondis
     storage/           registre, ouverture des bases, migrations SQL embarquées
     server/            routage HTTP, middlewares, résolution de la tribu
   acceptance/          scénarios godog, lisent docs/specs/features/
   web/                 front-end TypeScript
     package.json, tsconfig.json
     src/               composants, écrans, modules purs, service worker
     e2e/               scénarios Playwright @ui
     dist/              généré, embarqué dans le binaire
     embed.go           paquet Go qui embarque dist/
   site/                pages publiques statiques
   deploy/              Caddyfile, unité systemd
   docs/
   ```

   - **un paquet par domaine métier**, pas par couche technique ; pas de `utils/`, `common/` ni `pkg/` ;
   - **`internal/`** pour tout le code Go hors point d'entrée : le compilateur interdit son import depuis l'extérieur du module ;
   - **noms de paquets** courts, en minuscules et au singulier ;
   - **`quantity/`** est un paquet à part : logique pure utilisée par plusieurs domaines (calcul C2) ;
   - **`acceptance/`** est à la racine : les scénarios traversent tout le système via l'API.
4. **`web/dist/`** n'est pas versionné, à l'exception d'un fichier témoin : `//go:embed` exige que le dossier existe à la compilation, y compris sur un poste où le front n'a jamais été construit.

L'arborescence est un point de départ ; un paquet n'est découpé davantage que lorsque le besoin apparaît.

## Alternatives envisagées

- **Deux dépôts (front et back)** : écarté. Les specs, les ADR et le code évoluent ensemble ; deux dépôts n'apporteraient que de la synchronisation.
- **Vocabulaire métier en français, termes techniques en anglais** : écarté. Fidèle aux specs, mais mélange deux langues dans chaque fichier.
- **Organisation par couches** (`handlers/`, `services/`, `models/`) : écartée. Contraire aux usages Go ; disperse chaque fonctionnalité dans plusieurs dossiers.

## Conséquences

- **Positif** : code conventionnel pour tout développeur Go ou TypeScript ; une fonctionnalité se lit dans un seul paquet ; une PR peut porter une story de bout en bout (spec, code, tests).
- **Négatif** : une traduction permanente entre specs et code, rendue sûre par le glossaire ; les définitions d'étapes godog font le pont entre les phrases françaises et le code anglais.
- **Renommage du dépôt** : GitHub redirige l'ancienne adresse, mais les clones existants doivent mettre à jour leur remote, et les liens dans les documents (maquettes, projet claude.ai) sont à vérifier.
- **Reporté à une version ultérieure** : l'internationalisation de l'interface (choix de la langue, traduction des textes et des unités).
