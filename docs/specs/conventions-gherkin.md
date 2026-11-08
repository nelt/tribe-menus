# Conventions d'écriture des scénarios Gherkin

Les scénarios de `features/` sont exécutés tels quels (ADR 0005). Pour que chaque idée corresponde à une seule définition d'étape godog ou Playwright, ils suivent les règles ci-dessous. Un scénario nouveau ou modifié les respecte ; une formulation nouvelle n'est introduite que si aucune formulation existante ne convient.

## Mots-clés

- `# language: fr` en tête de chaque fichier.
- `Plan du scénario` (et non `Plan du Scénario`) pour les scénarios à exemples, suivi de `Exemples:`.
- `Étant donné`, `Quand`, `Alors`, `Et`, `Mais`. `Étant donné que` / `Étant donné qu'` seulement quand la phrase l'exige (« Étant donné que nous sommes le … »).

## Tags

- Chaque scénario porte l'identifiant de sa story ou de son exigence (`@C2`, `@ENF-01`, `@EF-10`) ; plusieurs si besoin.
- `@ui` : prouvé dans le navigateur par Playwright, à partir du `.feature` (ADR 0028), et ignoré par godog ; posé quand le plan d'une story fixe le niveau de preuve de ses scénarios (ADR 0005, point 2).
- `@manuel` : vérifié à la main en recette avant chaque release, exclu de `make ci` (ADR 0005, point 4).
- `@compact`, `@moyen`, `@etendu` : formats d'écran dans lesquels un scénario `@ui` vaut (ENF-03, plan `grands-ecrans`, D14) ; un scénario en porte un ou plusieurs, et **sans tag de format, il vaut pour tous les formats**. Le scénario n'est lancé que dans les projets Playwright de ses formats. Posés, comme `@ui`, quand le plan d'une story fixe le niveau de preuve de ses scénarios.

## Personnes

- Une personne est désignée par son adresse e-mail entre guillemets : `"alice@exemple.fr"`.
- Les prénoms sont des alias fixes, admis dans la prose des étapes : Alice = `alice@exemple.fr`, Bruno = `bruno@exemple.fr`, Chloé = `chloe@exemple.fr`, David = `david@exemple.fr`.
- « je » désigne le membre connecté, déclaré dans le `Contexte` ou dans la première étape du scénario.

## Tribus

- Une tribu est désignée par son identifiant d'URL entre guillemets : `la tribu "martin"`. Son nom (`"Les Martin"`) n'apparaît qu'à sa création ou quand il est vérifié.

## Connexion

- Forme de référence : `je suis connecté à la tribu "<identifiant>" en tant que "<e-mail>"`, pour un tiers `"<e-mail>" est connecté à la tribu "<identifiant>"`.
- Précisions facultatives, dans cet ordre : l'appareil (`dans mon navigateur`, `dans l'app installée`, `sur mon téléphone`), puis l'écran déjà affiché (`et j'ai déjà affiché le planning`).

## Dates et repas

- Un jour s'écrit avec le jour de la semaine et l'année : `le mardi 6 octobre 2026`. Le premier du mois s'écrit `1er` : `le samedi 1er janvier 2000`.
- Un repas : `le mardi 6 octobre 2026 à midi` ou `le mardi 6 octobre 2026 au soir` ; `le repas du mardi 6 octobre 2026 au soir`.
- La période dont on ajoute les courses s'écrit sans jour de la semaine : `du 6 au 12 octobre 2026`. Les jours affichés par le planning gardent le jour de la semaine (`du lundi 5 au dimanche 11 octobre 2026`), puisque c'est ce que montre l'écran.
- `dans la période` et `de la période` désignent la période dont le même scénario ajoute les courses ; sans date précisée, c'est la période par défaut (C4) relative à la date du `Contexte`.
- Les libellés d'affichage abrégés (`mardi 6 oct., soir`) ne figurent que là où le scénario vérifie ce qui est affiché.

## Listes de courses

- Une liste est désignée par son nom entre guillemets : `la liste "Courses"`. La liste principale du `Contexte` s'appelle `"Courses"`.
- Ajout des courses : `j'ajoute les courses des repas du 6 au 12 octobre 2026 à la liste "Courses"`.
- Article calculé posé directement : `la liste "Courses" contenant 500 g de "bœuf haché" provenant du "Chili con carne" du mardi 6 octobre 2026 au soir`.
- Coche : `"pain" est coché dans la liste "Courses"` (état), `je coche "pain" dans la liste "Courses"` (action), `"pain" apparaît comme coché dans la liste "Courses"` (vérification).

## Formats d'écran

- Le format vient du projet Playwright (tags de format ci-dessus) ; un scénario qui dépend d'une taille précise la fixe : `une fenêtre de 1366 px de large sur 520 px de haut`.
- Rotation : `je tourne la tablette` (largeur et hauteur échangées) ; `une tablette tenue en paysage` dans un scénario `@manuel`.
- Ce qui se vérifie à l'écran se nomme par ce qu'on voit, sans valeur de mise en page : `les onglets sont en bas de l'écran`, `les onglets sont sur le côté gauche de l'écran`, `la confirmation s'affiche au milieu de l'écran`, `le plat "Chili con carne" s'affiche à côté de la bibliothèque`.

## Quantités

- Nombre, unité, puis ingrédient entre guillemets : `500 g de "bœuf haché"`, `2 cuillères à café de "huile d'olive"`, `1 pièce de "oignon"` (pas d'élision devant le guillemet).
- Unités au singulier ou au pluriel selon le nombre ; le pluriel ne change pas l'unité.

## Plats

- `le plat "<nom>" défini pour <n> parts` (bibliothèque) ; `le plat "<nom>" servi pour <n> parts le <repas>` (planning).
