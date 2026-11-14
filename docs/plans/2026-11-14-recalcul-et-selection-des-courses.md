# Plan : recalcul et sélection des courses

- **Nom** : `courses-suite`
- **Date** : 2026-11-14
- **Statut** : en cours depuis le 2026-11-14. D1 à D4 prises par le développeur le 2026-11-14 au cadrage ; D5 à D22 confirmées le 2026-11-14 par la fusion de la PR du plan (#114) et l'accord du développeur au lancement du lot A. Place dans la feuille de route : après `grands-ecrans` (décisions du 2026-11-06 et du 2026-11-07), avant le plan hors-ligne.

## Objectif

Quatrième plan du métier, qui termine les listes de courses en ligne : une liste signale qu'elle ne correspond plus aux repas et se recalcule, et on en sort des articles vers une autre liste. Le plan `courses` a posé le modèle pour cela (`courses`, D5 : provenances avec ce qu'elles ont apporté, plats servis reçus avec leurs parts, les parts de référence du plat et ses lignes au moment du calcul).

À la fin du plan :

- une liste dont des plats servis ont changé depuis l'ajout de leurs courses porte le **bandeau de liste périmée**, d'après l'état « périmée » de `Doux-Courses`, et la pastille « Repas modifiés » dans le choix des listes ; **« Recalculer »** la remet d'accord avec les repas (C8) ;
- un article coché dont la quantité augmente montre **ce qui reste à prendre** (« 750 g · encore 250 g ») au lieu de redemander toute la quantité (D1) ;
- on **sélectionne** des articles d'une liste pour les **déplacer** vers une autre liste (C9) ou **créer une nouvelle liste** avec eux (C10), d'après `Doux-CoursesSelection`, dans les trois formats (`grands-ecrans`, D13) ;
- les points reportés à ce cadrage sont traités : message d'une liste supprimée entre-temps, période par défaut qui suit la date de l'appareil, provenance absente du dépliage, clavier virtuel sur une tablette en paysage (D4), navigations par événement de `mt-app` (revue #109, point 4) ;
- la version `v0.6.0` est publiée et déployée en production.

La conception (règle du recalcul, péremption, déplacement, API) va dans [`docs/conception/listes-de-courses.md`](../conception/listes-de-courses.md), avec ses diagrammes de classes et de séquence (`bibliotheque`, D13) ; ce plan dit dans quel ordre on la réalise et ce qui reste à décider.

## Contexte et contraintes

- **À lire avant de commencer** : le plan `courses` (D3, D5, D7 à D13, questions ouvertes, notes d'exécution) et sa conception ; ADR 0002, 0003 (point 7), 0004, 0005, 0028, 0029, 0030 ; `docs/specs/fonctionnalites.md` (concepts Article et Plats servis reçus ; « Ajout des courses d'une période », « Recalcul d'une liste », « Sélection d'articles » ; C8 à C10, Q10, Q11, Q22, Q24, Q36 à Q40) ; `features/liste-courses.feature` (scénarios C8 à C10, en attente) ; `formats-ecran.feature` ; `docs/design/README.md` (« Listes de courses », « Messages des listes de courses », « Formats d'écran ») et les maquettes `Doux-Courses` (états périmée et sélection), `Doux-CoursesSelection`, `Doux-ListesCourses` (pastille « Repas modifiés »), `GE-Courses-*`.
- **Après `grands-ecrans`** : `v0.5.0` est en production depuis le 2026-11-09 ; rien n'attend.
- **Où se fait le travail** : le lot A (maquettes dans Design, specs, conception, ADR) depuis une session cloud ; les lots B à E dans le Dev Container.
- **Git** : une branche `feature/…` et une PR par lot ; un commit par étape, `git commit -s` ; une ligne de `CHANGELOG.md` par PR, citant le lot (`courses-suite/B`) et les stories ; `make ci` avant de pousser. Aucune branche dont le nom contient `main` (`CLAUDE.md`).
- **Aucune dépendance nouvelle**, ni en Go ni en npm.
- **Une migration** de la base de la tribu (`0010`, D1) : revenir en arrière sur `v0.5.0` demandera la restauration de l'instantané (ADR 0012, point 6 ; plan `recette`, D7). Le registre et la base de limitation ne changent pas.
- **Hors-ligne hors du plan**, comme `courses`, D2 : recalculer et déplacer demandent le réseau de toute façon (Q16) ; les scénarios C7, dont « La coche d'un article déplacé entre-temps… » (`@C7 @C9`) et « La coche d'un article supprimé par un recalcul… » (`@C7 @C8`), restent au plan hors-ligne.
- **Q10 tient** : les articles calculés ne changent que sur action explicite (ajout de courses, « Recalculer », déplacement). La péremption est seulement signalée.
- **Rien d'une autre tribu** (ENF-02) : la route de péremption, le recalcul et le déplacement lisent la seule base de la tribu de la session ; une liste d'arrivée d'une autre tribu est inconnue (`404`).
- **Recette** : téléphone Android, tablette Android (portrait et paysage), ordinateur Ubuntu, sous Chrome ; deux appareils pour les écritures croisées ; pas d'appareil Apple (WebKit par Playwright).

### Scénarios couverts

| Fichier | Scénario | Preuve | Lot |
| --- | --- | --- | --- |
| `liste-courses.feature` | C8 : « Un bandeau signale que des plats servis ont changé », « Recalculer met la liste à jour et fait disparaître le bandeau » | navigateur, `@ui` | D |
| | C8 : « Les changements qui rendent une liste périmée », « Un plat ajouté au planning ne rend pas la liste périmée », « Un changement sans effet sur les articles ne rend pas la liste périmée » (exemple de la quantité réécrite dans une autre unité ajouté, revue #92, point 6), le bandeau prouvé par la route de péremption (D7) | API | B |
| | C8 : articles ajoutés conservés, coche conservée, coche retirée (accordé à D1 : « encore 250 g »), article sans provenance qui disparaît, articles achetés non recréés | API | B |
| | C9 : « Déplacer des articles sélectionnés vers une autre liste » | navigateur, `@ui` | E |
| | C9 : coche et provenances gardées, fusion, plat servi non compté deux fois, coche de l'article fusionné, article ajouté distinct, plats servis restés reçus par la liste de départ | API | C |
| | C10 : « Créer une liste à partir d'une sélection » | navigateur, `@ui` | E |
| | C10 : « Une liste créée à partir d'une sélection garde les provenances » | API | C |
| | C9, C10 : « La sélection est impossible sans article sélectionné » | navigateur, `@ui` | E |
| | Scénarios nouveaux du lot A : part prise (D1, D9), ligne ajoutée au plat (D2, D5), plat servi dont tous les articles sont partis (D2), recalcul au-delà de 300 articles, déplacement (liste d'arrivée supprimée, articles partis entre-temps, même liste, 300 articles, 20 listes, nom de la nouvelle liste), écritures sur une liste supprimée entre-temps (exemples recalculer et déplacer) | API | B, C |
| | C7 avec C8 ou C9 | **restent en attente** : plan hors-ligne | — |
| `formats-ecran.feature` | Nouveau : la sélection survit à la rotation ; les actions de sélection en moyen et en étendu | navigateur, `@ui` | E |
| `compartimentage-tribus.feature` | ENF-02 : « Une liste d'une autre tribu ne peut être ni lue, ni modifiée, ni supprimée », exemples recalculer et déplacer vers ; péremption d'une autre tribu jamais visible | API | B, C |

## Décisions

Le 2026-11-14, au cadrage, avec le développeur.

- **D1. Un article coché dont la quantité augmente montre ce qui reste à prendre.** Remplace la règle « décoché avec toute sa quantité » (`fonctionnalites.md`, « Ajout des courses d'une période » ; Q11), relevée à la recette de `v0.4.0`. L'article est décoché et garde la **part prise** : la quantité qu'il avait quand il était coché. Il s'affiche « 750 g · encore 250 g ». Une colonne de plus, `taken_amount`, en unités de base, dans `shopping_items` (migration `0010`). Règles détaillées en D9. Écartés : une ligne séparée pour le supplément, qui casse « une ligne par ingrédient et famille » (Q24) et le dépliage ; l'état actuel.
- **D2. Une ligne ajoutée à un plat rend la liste périmée, et « Recalculer » crée son article**, sauf si le plat servi n'a plus aucune provenance dans la liste (tous ses articles sont partis, achetés ou déplacés, ou il n'a jamais rien apporté) : rien n'est alors signalé ni recréé. Confirme `courses`, D5 (dernier point), et décide la question ouverte « Recalcul et dépliage » de `courses`. Écartés : créer l'article même quand le plat servi n'a plus rien dans la liste, qui fait reparaître un repas déjà acheté ; ne jamais signaler une ligne ajoutée, contraire au concept « Liste périmée » de la spec.
- **D3. La péremption est calculée par le serveur, à la demande** : `GET shopping-lists/stale` renvoie les identifiants des listes périmées, à chaque appel, sans ETag (D7). Elle se fonde sur la même fonction pure que le recalcul : une liste est périmée si le recalcul y changerait quelque chose (D6). Écartés : un drapeau par liste dans `GET shopping-lists`, avec un ETag qui combine les versions des listes, du planning et des plats, qui fait recharger toutes les listes à chaque modification du planning et demande de préciser l'ADR 0029 ; le calcul dans le navigateur, qui exposerait les lignes gardées et écrirait la règle en TypeScript, contre `courses`, D6.
- **D4. Clavier virtuel** (écart accepté à la recette de `v0.5.0`) : `interactive-widget=resizes-content` dans la balise `viewport` de l'application. Sous Chrome et Firefox pour Android, la fenêtre de mise en page rétrécit au-dessus du clavier, et « Enregistrer le plat », fixé en bas, reste visible. Le format est figé tant qu'un champ a le focus (`grands-ecrans`, D11 ; déjà en place dans `layout.ts`), donc une tablette en paysage ne bascule pas en compact. Safari ignore l'attribut : vérifié par WebKit seul. Le scénario `@manuel` de `formats-ecran.feature` redevient tenu en entier, et le point reporté de la feuille de route est fermé à la recette. Écartés : le bouton dans le flux en moyen et en étendu, qui s'écarte de la maquette ; alléger le scénario.

Proposées le 2026-11-14, confirmées le même jour par la fusion de la PR du plan (#114).

- **D5. Règle du recalcul**, pure, dans `internal/shopping`. Pour chaque plat servi reçu par la liste qui y a **encore une provenance** (D2), on compare les lignes actuelles du plat aux **lignes gardées** (`courses`, D5), par clé (ingrédient et famille d'unités, ou ingrédient sans quantité) :
  - **la clé a une ligne gardée et un article de la liste a une provenance de ce plat servi pour elle** : la provenance est recalculée sur l'état actuel (parts du plat servi, parts de référence et quantité du plat) ;
  - **la clé a une ligne gardée, mais plus d'article ni de provenance pour elle** (article acheté, ou déplacé vers une autre liste) : rien n'est recréé (spec : « Le recalcul ne recrée pas les articles déjà achetés ») ;
  - **la clé est nouvelle** (ligne ajoutée au plat, ou ligne dont l'unité a changé de famille, « sel » devenu « 5 g ») : une provenance est créée, dans l'article calculé de cette clé, existant ou créé (D2) ;
  - **la clé n'est plus dans le plat**, ou le plat servi a été retiré du planning (R4) : la provenance disparaît.

  Ensuite, chaque article calculé est recalculé à partir de toutes ses provenances et arrondi (Q37). Un article sans provenance disparaît, même coché (spec). La coche suit D9. Les lignes gardées, les parts et les parts de référence des plats servis traités sont remplacées par l'état actuel ; celles d'un plat servi retiré du planning, ou sans provenance restante, ne changent pas. Les articles ajoutés ne sont pas touchés. Si la liste dépasse 300 articles, rien n'est écrit (`409 too_many_items`, Q38). Le recalcul se fait dans une transaction qui lit le planning, les plats et la liste, comme l'ajout des courses (`courses`, D6).
- **D6. Une liste est périmée** quand le recalcul (D5) changerait ses articles : existence, quantité, part prise, coche ou provenances (plat servi, parts, quantité apportée). Les quantités se comparent en unités de base. Un changement qui ne touche que les lignes gardées n'en est donc pas un : quantité réécrite dans une autre unité de sa famille (500 g en 0,5 kg), parts modifiées d'un plat servi sans ingrédient ou sans quantité, renommage du plat (revue #92, point 6 ; scénario « Un changement sans effet sur les articles… »). Un plat servi ajouté ensuite au planning n'y entre pas (spec) : on l'ajoute par les courses des repas (C4).
- **D7. Routes** :
  - `GET shopping-lists/stale` : `200 { lists: [ListID] }`, dans l'ordre des identifiants ; `Cache-Control: no-store`, sans ETag ni `304`. La copie des listes l'appelle à chaque vérification (visite de l'onglet Courses, retour au premier plan), juste après la sienne. Coût : au plus 20 listes, leurs plats servis reçus et les plats concernés, dans une base de tribu.
  - `POST shopping-lists/{id}/recompute` : `200 { list, versions }`, `versions` vides si rien ne change ; `404` ; `409 too_many_items`. La liste renvoyée n'est plus périmée.
  - `POST shopping-lists/{id}/move` : `{ items: [ItemID], to: ListID }` vers une liste existante, ou `{ items, name }` vers une liste nouvelle ; `200 { list, target, moved, missing, versions }` (la liste de départ et la liste d'arrivée à jour, le nombre d'articles déplacés et le nombre d'articles qui n'étaient plus dans la liste) ; `400` : `no_items`, `same_list`, les codes du nom d'une liste ; `404` pour la liste de départ, `404 { error: "target_not_found" }` pour la liste d'arrivée ; `409 too_many_items` (liste d'arrivée), `409 too_many_lists`. Corps limité à 16 Kio, comme `checkout`.
  - Toutes derrière la session ; `404` identique pour une liste inconnue, supprimée entre-temps ou d'une autre tribu (ENF-02).
- **D8. Une écriture de liste reste une collection** (ADR 0029 tel quel) : le recalcul et le déplacement changent `shopping_lists` et renvoient sa version. La péremption n'est pas une collection : un état dérivé, calculé à chaque lecture. **ADR 0031** « État dérivé calculé par le serveur, hors des collections » (lot A) : la route, son absence d'ETag, quand la copie l'appelle, et pourquoi pas un ETag combiné (D3). C'est le premier état de ce genre ; d'autres pourront suivre (membres, sessions).
- **D9. Part prise** (D1). Q41.
  - **Quand une quantité augmente** (ajout des courses des repas, recalcul) sur un article coché : il est décoché, et sa part prise devient la quantité qu'il avait. Sur un article déjà décoché avec une part prise, la part prise ne change pas.
  - **Quand une quantité baisse** (recalcul) jusqu'à la part prise ou en dessous : l'article est de nouveau coché, sans part prise (tout ce qu'il faut est pris).
  - **Cocher** un article efface sa part prise (tout est pris) ; **décocher** un article coché ne lui en donne pas (on a remis l'article en rayon).
  - **Courses faites** : seuls les articles cochés partent ; un article avec une part prise reste, avec elle.
  - **Fusion** au déplacement (D10) : l'article fusionné est coché si les deux l'étaient ; sinon, sa part prise est la somme des parts prises (la quantité entière pour un article coché), s'il y en a une.
  - **Article sans quantité** et **article ajouté** : jamais de part prise (leur quantité n'augmente pas).
  - **Affichage** : « 750 g · encore 250 g », le reste dans la plus grande unité de sa famille qui donne au moins 1 (Q37) ; lu « bœuf haché, 750 g, encore 250 g à prendre ». API : `taken` dans l'article, au format des quantités d'un article calculé.
- **D10. Déplacement** (C9, C10), dans une seule transaction, selon « Sélection d'articles » de la spec :
  - seuls les articles sélectionnés **encore dans la liste de départ** sont déplacés ; les autres (cochés puis partis, déplacés ou recalculés entre-temps) sont comptés dans `missing`, comme les courses faites ne retirent que ce que la feuille a annoncé (`courses`, D9) ;
  - un article calculé rejoint l'article calculé de même clé de la liste d'arrivée (fusion) : provenances réunies par plat servi, sans compter deux fois une provenance déjà présente, quantité recalculée et arrondie, coche et part prise selon D9 ; sinon il est déplacé tel quel, son identifiant gardé ;
  - un article ajouté reste une ligne distincte ;
  - les plats servis des provenances déplacées deviennent des plats servis reçus de la liste d'arrivée, avec leurs lignes gardées, copiés de la liste de départ, sauf ceux que la liste d'arrivée a déjà reçus (elle garde les siens) ; la liste de départ les garde aussi (Q22) ;
  - **nouvelle liste** : nom selon les règles des listes (`courses`, D12), créée en dernier rang, non principale (scénario « …la liste principale de la tribu est toujours la liste "Courses" »), dans la même transaction que le déplacement ;
  - refus : aucun article, liste d'arrivée identique à celle de départ, 300 articles dépassés dans la liste d'arrivée après fusion, 20 listes : rien n'est déplacé.
- **D11. Écritures croisées.** Q42, qui complète Q39.
  - **recalcul et ajout des courses, ou deux recalculs simultanés** : sérialisés par la transaction ; le second voit l'état du premier ;
  - **déplacement d'articles partis entre-temps** : D10, `missing` ; l'écran dit « 2 articles déplacés vers « Marché ». 1 article n'était plus dans la liste. » ;
  - **liste d'arrivée supprimée entre-temps** : rien n'est déplacé ; l'écran reste sur la liste de départ, sélection gardée, et dit « La liste « Marché » a été supprimée entre-temps. Rien n'a été déplacé. » ;
  - **coche d'un article déplacé entre-temps** : `404` sur l'ancienne liste, sans message (`courses`, D8) ; la copie se vérifie et l'article disparaît de la liste. Au plan hors-ligne, Q18 la reportera sur la nouvelle liste.
- **D12. Message d'une liste supprimée entre-temps** (point reporté à la recette de `v0.4.0`) : il nomme la liste, d'après la copie d'avant la vérification, et dit ce qui n'a pas été fait. Il s'affiche sur la liste principale, où l'écran se rend (Q39) : « La liste « Jetable » a été supprimée entre-temps. L'article n'a pas été ajouté. » ; selon l'écriture : « Les courses des repas n'ont pas été ajoutées. », « Elle n'a pas été renommée. », « Rien n'a été recalculé. », « Rien n'a été déplacé. » ; sans seconde phrase pour une coche ou un retrait, qui restent sans message (`courses`, D8 et D11). Dans `docs/design/README.md`, « Messages des listes de courses ».
- **D13. Bandeau et pastille** (C8), d'après l'état « périmée » de `Doux-Courses` : sous l'en-tête d'une liste, bandeau ambre « Des repas ont changé depuis l'ajout de leurs courses à cette liste. » et « Recalculer » en aplat, absent en mode sélection ; pastille « Repas modifiés » sur la carte de la liste dans le choix des listes (`Doux-ListesCourses`). Pendant le recalcul, « Recalcul en cours… », bouton inactif. Après, le bandeau disparaît et un bandeau vert dit « Liste recalculée. », comme les autres retours (un recalcul peut faire disparaître un article coché : le membre doit le savoir). À valider dans Design (lot A).
- **D14. Mode sélection dans les trois formats** (`grands-ecrans`, D13), d'après `Doux-CoursesSelection`, maquetté au lot A en compact, moyen et étendu :
  - « Sélectionner » dans l'en-tête d'une liste non vide ; en sélection, l'en-tête devient « Annuler », « N sélectionnés », « Tout » (« Aucun » quand tout est sélectionné) ; tous les articles se sélectionnent, cochés compris ; la ligne entière sélectionne, sans dépliage ;
  - **compact** : la barre d'onglets laisse place à « Déplacer vers… » (aplat) et « Nouvelle liste » (contour foncé), comme la maquette ;
  - **moyen et étendu** : le rail reste ; les deux actions forment une barre fixée en bas du volet de la liste ; en étendu, le choix des listes reste à gauche, et toucher une autre liste quitte la sélection ;
  - « Déplacer vers… » ouvre la feuille (dialogue centré hors compact, `grands-ecrans`, D7) des autres listes, dans l'ordre du choix des listes, avec leur nombre d'articles ; « Nouvelle liste » ouvre la feuille « Nom de la liste », « Créer la liste » ;
  - après le déplacement, l'écran reste sur la liste de départ, hors sélection, avec un bandeau vert « 2 articles déplacés vers « Marché ». », comme la maquette ; la liste d'arrivée s'ouvre en touchant son nom dans le bandeau ;
  - **une sélection n'est pas une saisie** (`grands-ecrans`, D9) : la quitter par une navigation la perd sans confirmation ; la rotation la garde (`grands-ecrans`, D8 : l'écran de la liste n'est pas recréé) ; un article qui disparaît de la copie quitte la sélection ;
  - **les navigations nouvelles passent par `#go`** (revue #109, point 4) : ouvrir la liste d'arrivée depuis le bandeau, quitter la sélection par le choix des listes.
- **D15. Points reportés de `courses`** :
  - **période par défaut** (revue #101, point 4) : tant que le membre n'a pas touché à la période, elle suit la date de l'appareil au retour sur l'application, comme le planning (Q34) ;
  - **provenance absente du dépliage** (revue #101, point 3) : un plat absent de la copie de la bibliothèque fait vérifier celle-ci une fois, puis la provenance s'affiche ;
  - **croissance de `received`** (`courses`, questions ouvertes) : la taille de `GET shopping-lists` en production est mesurée à la recette ; si elle gêne, un plan suivant sort `received` de l'élément de liste. D3 ne l'aggrave pas : les lignes gardées restent dans le serveur.
- **D16. Preuves** : règles, refus et écritures croisées contre l'API ; ce que montre l'écran (bandeau, « Recalculer », geste de sélection, actions inactives sans sélection, déplacement et nouvelle liste depuis l'écran) dans le navigateur, `@ui`, sans tag de format : `make e2e-formats` les passe dans les sept projets. Le bandeau des scénarios « API » est prouvé par `GET shopping-lists/stale`, comme « aucun message » l'a été par un code de réponse (`courses`, notes du lot B). « Un bandeau signale… » et « Recalculer met la liste à jour… » deviennent `@ui` ; « Une coche est retirée si la quantité augmente » gagne « Et il reste 250 g de "bœuf haché" à prendre » (D1). Deux scénarios de `formats-ecran.feature` : la sélection survit à la rotation ; les actions de sélection en bas du volet de la liste en étendu.
- **D17. Maquettes du plan** dans Design d'abord (canevas de travail), validées, puis reportées dans le dépôt : état « périmée » de la liste et « Liste recalculée. », pastille « Repas modifiés », part prise (« encore 250 g »), mode sélection et ses deux feuilles, bandeaux après un déplacement (articles partis entre-temps, liste d'arrivée supprimée), message nommé d'une liste supprimée entre-temps (D12) ; chacune en téléphone (390 × 844), tablette portrait et paysage, ordinateur, comme les `GE-*`. `Doux-Courses` et `Doux-CoursesSelection` en sont le point de départ.
- **D18. Code** : dans `internal/shopping`, `Recompute` (D5) et `Stale` (D6), une seule fonction pure dont `Stale` compare le résultat ; `Move` (D10) ; la part prise dans le calcul des courses des repas (D9) ; adaptateur `ShoppingStore` ; côté navigateur, `shopping.ts` (part prise, libellés, messages, sélection : sélectionnés, « Tout », actions possibles), `shopping-lists.ts` (péremption après chaque vérification, recalcul, déplacement), `shopping-list-screen.ts` et `list-picker-screen.ts`. Tables de cas communes à Go et TypeScript pour la part prise et son affichage (`internal/shopping/testdata/`).
- **D19. Vocabulaire** (`glossaire.md`, lot A) : part prise (*taken quantity*), recalculer (*recompute*, déjà là), liste périmée (*stale list*), liste de départ (*source list*), liste d'arrivée (*target list*, déjà la destination des courses), mode sélection (*selection mode*), articles partis entre-temps (*missing items*).
- **D20. `make seed`** : à la tribu `demo`, une liste « Courses » périmée (les parts d'un plat servi changées après l'ajout de ses courses) et un article avec une part prise, pour le développement et les états des écrans. Rien en mode serveur.
- **D21. `fonctionnalites.md`** : « Ajout des courses d'une période » et Q11 accordés à D1 ; « Recalcul d'une liste » précisé par D2, D5 et D6 ; « Sélection d'articles » précisé par D10 (articles partis entre-temps, nouvelle liste non principale, limites) ; Q41 (part prise, D9), Q42 (écritures croisées, D11), Q43 (péremption et recalcul, D2, D5, D6).
- **D22. Release `v0.6.0`** avec la migration `0010` : recette du retour arrière avec restauration, comme `courses`.

## Écrans concernés

| Écran | Compact | Moyen | Étendu |
| --- | --- | --- | --- |
| Liste (`Doux-Liste`) | bandeau périmée, « Recalculer », part prise, « Sélectionner » | idem, colonne élargie | idem, volet de détail |
| Mode sélection (`Doux-CoursesSelection`) | actions à la place de la barre d'onglets | rail, actions en bas de la colonne | rail, choix des listes à gauche, actions en bas du volet de la liste |
| Choix des listes (`Doux-ChoixListes`) | pastille « Repas modifiés » | idem | idem, volet de liste |
| Feuilles « Déplacer vers… », « Nouvelle liste » | en bas | dialogue centré | dialogue centré |
| Plat (`Doux-PlatModif`) | inchangé | inchangé | « Enregistrer le plat » visible au-dessus du clavier (D4) |

## Étapes

### Lot A : design, specs, conception et ADR (`feature/specs-courses-suite`)

Depuis une session cloud, une fois ce plan revu et ses décisions confirmées.

- [x] **1. Maquettes dans Design** (D17) : états et formats listés en D17 ; validation par le développeur. *Fait le 2026-11-14* : cinq planches sur le canevas « Menus de la semaine », rangées K et L : `Doux-ListeSuite`, `Doux-ChoixListesSuite`, `GE-CoursesSuite-Paysage`, `-Ordinateur`, `-Portrait` ; validées par le développeur le 2026-11-14, sans retouche, interprétations comprises.
- [x] **2. Report dans le dépôt** : maquettes dans `docs/design/maquettes/` ; `docs/design/README.md` : « Listes de courses » (périmée, recalcul, part prise, sélection dans les trois formats ; « Laissé de côté » mis à jour), « Messages des listes de courses » (D11, D12, refus du déplacement et du recalcul), « Formats d'écran » (actions de sélection). *Fait le 2026-11-14* : planches dans `docs/design/maquettes/` ; README : tableau des écrans, « Recalcul, part prise et sélection », messages, composants, « Formats d'écran ».
- [x] **3. `fonctionnalites.md`** (D21) et **`glossaire.md`** (D19). *Fait le 2026-11-14* : concept Article, « Ajout des courses d'une période », « Recalcul d'une liste », « Sélection d'articles », Q11, Q41 à Q43 ; glossaire.
- [x] **4. Scénarios**, selon `conventions-gherkin.md` : tags `@ui` (D16) ; « Une coche est retirée si la quantité augmente » accordé à D1 ; exemple de la quantité réécrite dans « Un changement sans effet… » ; scénarios nouveaux du tableau « Scénarios couverts » ; deux scénarios de `formats-ecran.feature` ; exemples recalculer et déplacer dans « Une écriture sur une liste supprimée entre-temps est refusée » et dans l'accès croisé de `compartimentage-tribus.feature`. Titres nouveaux dans `acceptance/pending.txt`. *Fait le 2026-11-14* : voir les notes d'exécution du lot A ; 22 titres dans `pending.txt`.
- [x] **5. Conception** : `listes-de-courses.md` : modèle (part prise), règle du recalcul et de la péremption, déplacement, API (D7), données dans le navigateur, échanges (recalcul ; déplacement avec articles partis entre-temps), avec leurs diagrammes ; « Ce que ce document ne couvre pas encore » réduit au hors-ligne. *Fait le 2026-11-14* : sections « Recalcul et péremption » et « Déplacement », deux diagrammes de séquence nouveaux, classes et API complétées.
- [x] **6. ADR 0031** (D8) ; ADR 0030 : note sur `interactive-widget` (D4), qui complète son point sur le clavier virtuel. *Fait le 2026-11-14* : `0031-etat-derive-calcule-par-le-serveur.md` ; ADR 0030, point 3, complété.
- [x] **7. Feuille de route** : plan « en cours ». *Fait le 2026-11-14.*

### Lot B : part prise, péremption et recalcul, domaine et API (`feature/recalcul-api`)

- [x] **8. Migration `0010`** : `taken_amount` dans `shopping_items` (entier positif facultatif, absent pour un article coché, sans quantité ou ajouté, inférieur à la quantité), test de migration sur une base de `0009` ; requêtes sqlc, `make generate`. *Fait le 2026-11-14* : contrainte `CHECK` de la colonne, testée ; la coche s'écrit par `UpdateItem` (`SetChecked` du `Tx` retiré).
- [x] **9. Part prise dans l'ajout des courses des repas** (D9), tests en tableaux d'abord ; table de cas `internal/shopping/testdata/taken.json`, quantité affichée comprise ; `taken` dans l'API ; coche et décoche selon D9. *Fait le 2026-11-14* : `applyTaken` ; `taken.json` porte aussi `shown` et `remaining` (« encore 250 g ») pour `shopping.ts` (lot D).
- [x] **10. Recalcul et péremption purs** (D5, D6), tests en tableaux d'abord, à partir des scénarios C8 et des cas de D5 (ligne ajoutée, ligne changée de famille, ligne retirée, plat servi retiré, plat servi sans provenance restante, article acheté, article déplacé, quantité réécrite, parts d'un plat sans quantité, baisse jusqu'à la part prise, 300 articles). *Fait le 2026-11-14* : `recompute` et `stale` dans `recompute.go` ; la limite de 300 articles est vérifiée par `Lists.Recompute`, pour que `stale` dise périmée une liste dont le recalcul serait refusé.
- [x] **11. `shopping.Lists`** : `Recompute` et `Stale` derrière l'interface `Store`, dans une transaction qui lit le planning, les plats et les listes ; adaptateur `ShoppingStore`. *Fait le 2026-11-14* : `Store.Read` (transaction annulée) pour `Stale` ; `Tx` : `AllLists`, `Received`, `CurrentReceived`, `ReplaceReceived`, `DeleteSources`.
- [x] **12. API** : `GET shopping-lists/stale`, `POST shopping-lists/{id}/recompute` (D7) ; codes d'erreur dans la conception. *Fait le 2026-11-14.*
- [x] **13. Étapes godog** des scénarios « API » du lot, retirés de `pending.txt`. *Fait le 2026-11-14* : 17 scénarios retirés (`acceptance/steps_recompute_test.go`).
- [x] **14. `make seed`** (D20). *Fait le 2026-11-14* : « bœuf haché » coché, le chili du mardi passé à 5 parts puis la liste recalculée (part prise de 875 g), la soupe à l'oignon passée à 6 parts ensuite (« Courses » périmée).

### Lot C : déplacement et nouvelle liste, domaine et API (`feature/deplacement-api`)

- [ ] **15. Déplacement pur** (D10) : fusion, provenances sans doublon, part prise et coche (D9), plats servis reçus copiés, articles partis entre-temps, limites ; tests en tableaux d'abord.
- [ ] **16. `shopping.Lists.Move`**, vers une liste existante ou nouvelle, dans une transaction ; adaptateur.
- [ ] **17. API** : `POST shopping-lists/{id}/move` (D7), corps de 16 Kio, test à 300 identifiants.
- [ ] **18. Étapes godog** des scénarios « API » de C9 et C10 et des écritures croisées, retirés de `pending.txt`.

### Lot D : péremption, recalcul et points reportés à l'écran (`feature/ecran-recalcul`)

- [ ] **19. Copie des listes** : péremption lue après chaque vérification (D7), recalcul, liste recalculée retirée des listes périmées ; message nommé d'une liste supprimée entre-temps (D12), d'après la copie d'avant la vérification.
- [ ] **20. Écran d'une liste** : bandeau de liste périmée, « Recalculer », « Liste recalculée. » (D13) ; part prise (D9) ; pastille « Repas modifiés » dans le choix des listes.
- [ ] **21. Points reportés** (D15) : période par défaut qui suit la date ; provenance absente du dépliage.
- [ ] **22. Clavier virtuel** (D4) : `interactive-widget=resizes-content` dans `web/src/index.html` ; test d'écran qui vérifie la balise ; `layout.ts` inchangé, son gel du format prouvé par ses tests existants.
- [ ] **23. C8 sous les sept projets** (`make e2e-formats`), retirés de `pending.txt`.

### Lot E : sélection à l'écran, dans les trois formats (`feature/ecran-selection`)

- [ ] **24. `shopping.ts`** : état de sélection (sélectionnés, « Tout » et « Aucun », actions possibles, articles disparus retirés), libellés des bandeaux du déplacement, avec leurs tests.
- [ ] **25. Mode sélection** (D14) : en-tête, pastilles rondes, cartes sélectionnées, actions en compact et dans le volet de la liste en moyen et en étendu ; navigations par `#go`.
- [ ] **26. Feuilles « Déplacer vers… » et « Nouvelle liste »**, bandeaux après le déplacement, refus (D11).
- [ ] **27. C9, C10 et les scénarios de `formats-ecran.feature`** sous les sept projets, retirés de `pending.txt`.

### Recette et release

- [ ] **28. Recette** sur l'archive de la PR de release `release/v0.6.0` : check-list ci-dessous.
- [ ] **29. Release `v0.6.0`** selon `RELEASING.md`, déployée en production.
- [ ] **30. Clôture** : statut « terminé » ; feuille de route (plan terminé, points reportés « clavier virtuel » fermé et `received` mesuré) ; conception ; `CLAUDE.md` (métier, version en service) ; `CHANGELOG.md`.

### Séance de recette (étape 28)

Par le développeur, guidé pas à pas, sur téléphone Android, tablette Android et ordinateur (Chrome) ; deux appareils pour les écritures croisées. Check-list détaillée au lot E, d'après les écrans réalisés ; au moins :

- [ ] Déploiement de la PR en recette : la migration `0010` s'applique ; les listes existantes sont intactes.
- [ ] Liste périmée : changer les parts d'un plat servi, une quantité du plat, retirer un plat servi, ajouter une ligne au plat : bandeau et pastille ; « Recalculer » : quantités justes, article de la ligne ajoutée créé, article acheté non recréé, « Liste recalculée. ». Quantité réécrite en kg : pas de bandeau.
- [ ] Part prise : cocher le bœuf, passer le chili de 4 à 6 parts, recalculer : « 750 g · encore 250 g » ; recocher ; repasser à 4 parts : coché.
- [ ] Sélection en compact, en moyen et en étendu : déplacer vers une liste, créer une liste ; fusion avec un article de même ingrédient ; rotation pendant une sélection ; deux appareils : articles partis entre-temps, liste d'arrivée supprimée entre-temps.
- [ ] Liste supprimée entre-temps : le message nomme la liste et dit ce qui n'a pas été fait.
- [ ] Clavier virtuel sur la tablette en paysage : « Enregistrer le plat » visible, mise en page inchangée.
- [ ] Taille de `GET shopping-lists` relevée (D15).
- [ ] Retour arrière avec restauration (migration `0010`), puis redéploiement de la PR.
- [ ] `sudo tribe-menus-check report`.

## Critères de validation

- `make ci` vert à chaque PR, `make e2e-formats` vert sur la PR de release ; `pending.txt` ne garde de C8 à C10 que les scénarios `@C7`.
- En recette : une liste périmée se recalcule juste au gramme près, sans recréer ce qui est acheté ; une sélection se déplace sans rien perdre ni compter deux fois, sur les trois appareils.
- En production, le développeur recalcule une liste après avoir changé le planning de la semaine.

### Ce que ce plan ne prouve pas

- Les coches et ajouts sans réseau, et leur rejeu après un recalcul ou un déplacement (C7, Q18) : plan hors-ligne.
- Le vrai Safari, et `interactive-widget` sous Safari (qui l'ignore), faute d'appareil Apple.

## Questions ouvertes

- **Croissance de `received`** : à mesurer en recette (D15), à reprendre si la taille gêne.
- **Un article calculé qu'on voudrait retirer sans l'acheter** : toujours hors V1 ; le déplacement vers une liste « À jeter » le permet de fait, mais la liste se remplit. À rediscuter après usage.

## Notes d'exécution

**Lot A** (`feature/specs-courses-suite`), 2026-11-14, depuis une session cloud, à valider à la revue :

- **Décisions** : D5 à D22 tenues pour confirmées par la fusion de #114, avec l'accord du développeur au lancement du lot.
- **Scénarios gardés intacts quand ils passent déjà**, des scénarios nouveaux plutôt que des exemples ajoutés, pour ne pas remettre en attente ce qui est prouvé : « Un article coché dont la quantité augmente est décoché » (C4) reste, « … garde sa part prise » s'y ajoute ; « Une écriture sur une liste supprimée entre-temps est refusée » garde ses exemples, « Recalculer une liste supprimée entre-temps est refusé » s'y ajoute ; l'accès croisé d'ENF-02 aussi (« … ni recalculée ni vidée par un déplacement », « … vers une liste d'une autre tribu », « Les listes périmées d'une autre tribu ne sont jamais visibles »).
- **Scénarios C8 : « Quand je touche "Recalculer" » devient « Quand je recalcule la liste "Courses" »** dans les scénarios prouvés contre l'API ; les deux scénarios `@ui` gardent le geste. « Un changement sans effet sur les articles… » devient un plan du scénario (parts des « Restes », quantité réécrite en 0,5 kg, plat renommé) ; « Les changements qui rendent une liste périmée » gagne les parts de référence, une ligne ajoutée et une ligne retirée.
- **Scénarios nouveaux au-delà du tableau du plan**, à valider à la revue : « Une liste périmée est signalée dans le choix des listes » (`@ui`), « Les plats servis des articles déplacés deviennent reçus par la liste d'arrivée », « Une liste créée à partir d'une sélection vient après les autres », « Un article avec une part prise reste après les courses faites », « La part prise s'efface quand l'article est coché ou décoché ».
- **Maquettes**, interprétations validées par le développeur le 2026-11-14 : part prise en seconde ligne ambre sous le nom (« encore 375 g ») et phrase dans le dépliage ; « Liste recalculée. » en bandeau vert ; après un déplacement, le nom de la liste d'arrivée en lien dans le bandeau vert ; liste d'arrivée supprimée en avertissement ambre, 300 articles en erreur rouge, la sélection gardée dans les deux cas ; en moyen et en étendu, actions de 260 px au plus, centrées, en bas du volet de la liste. Message du recalcul au-delà de 300 articles écrit dans le README, sans maquette.
- **Conception** : noms du `Tx` donnés à titre indicatif, à fixer aux lots B et C comme l'a fait `courses` ; `applyTaken`, pas commun de l'ajout des courses et du recalcul pour la coche et la part prise.

**Lot B** (`feature/recalcul-api`), 2026-11-14, dans le Dev Container :

- **Liste qui n'est pas périmée** : `Recompute` n'écrit rien, pas même les lignes gardées, et rend des versions vides (D7). Une liste dont seules les lignes gardées diffèrent (quantité réécrite en kg, parts d'un plat servi sans quantité) garde donc ses lignes gardées d'avant, sans effet : la comparaison suivante donne le même résultat. Revers : la provenance d'un ingrédient sans quantité garde les parts d'avant (« 4 parts » dans le dépliage) tant que rien d'autre ne fait recalculer la liste.
- **Péremption et refus des 300 articles** : `stale` dit périmée une liste dont le recalcul serait refusé ; le bandeau reste, et « Recalculer » affiche le message des 300 articles.
- **Plats servis reçus réécrits** : le recalcul remplace les lignes gardées, les parts et les parts de référence de tous les plats servis vivants encore au planning, changés ou non.
- **En attente du lot C** : « Le recalcul ne recrée pas un article déplacé vers une autre liste » et l'exemple « recalculer » de « Une liste d'une autre tribu ne peut être ni recalculée ni vidée par un déplacement », qui passe déjà, le plan du scénario attendant son exemple du déplacement.
