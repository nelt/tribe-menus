# 0027. Recherche et autocomplétion dans le navigateur

- **Date** : 2026-11-03
- **Statut** : accepté

## Contexte

La bibliothèque de plats se consulte et se cherche depuis un téléphone (P3) ; la saisie d'un ingrédient propose ceux du référentiel de la tribu, sans tenir compte de la casse ni des accents (P4). La maquette `Doux-Bibliotheque` cherche à chaque frappe, sans attente. Le hors-ligne viendra avec son propre plan : la bibliothèque et le référentiel devront alors rester consultables sans réseau (ADR 0004, point 7 ; PT-04).

Une tribu compte quelques dizaines de plats, quelques centaines au plus : bibliothèque et référentiel tiennent sans peine dans la mémoire d'un téléphone. Le serveur, lui, décide de l'identité d'un ingrédient et de l'unicité d'un nom de plat, que les index uniques de la base figent (Q20, Q28) : la règle de comparaison des noms existe forcément côté serveur.

Décidé avec le développeur le 2026-10-29 (plan `bibliotheque`, D1, D2, D8, D9).

## Décision

1. **La recherche (P3) et les propositions d'ingrédients (P4) se calculent dans le navigateur**, sur la bibliothèque et le référentiel de la tribu chargés en entier. L'API ne les sert pas : elle n'a ni route de recherche ni route de propositions.
2. **Bibliothèque et référentiel sont transférés par pages**, bornées par le serveur (50 plats, 500 ingrédients), dans l'ordre des identifiants, avec un curseur vers la page suivante. L'écran enchaîne les pages jusqu'à la dernière, puis garde tout en mémoire (`web/src/library.ts`) ; une collection qui change pendant le chargement le fait recommencer, pour que la copie soit toujours celle d'une même version (ADR 0029). Pas de plafond de plats ni d'ingrédients par tribu.
3. **Une seule règle de nom normalisé** pour l'identité d'un ingrédient, l'unicité d'un nom de plat, la recherche et les propositions (Q29). Elle est écrite deux fois :
   - **en Go** (`dish.NormalizeName`), qui décide de l'identité et de l'unicité ;
   - **en TypeScript** (`normalizeName` de `web/src/dishes.ts`), qui cherche et propose.

   Les deux passent **une même table de cas**, un fichier JSON lu par `go test` et par `node:test` ; un cas ajouté pour l'un vaut pour l'autre. Ses étapes et ses caractères sont fixés dans `docs/conception/bibliotheque-de-plats.md` : minuscules, puis ligatures œ et æ, puis décomposition NFD et retrait des seuls signes diacritiques U+0300 à U+036F, apostrophes typographiques ramenées à `'`, espaces insécables comptées comme des espaces, espaces réduites.
4. **La décomposition Unicode** se fait en Go par `golang.org/x/text/unicode/norm`, et dans le navigateur par `String.prototype.normalize`. `golang.org/x/text`, du projet Go, déjà dans `go.sum` par les outils, devient une **dépendance d'exécution**, justifiée selon l'ADR 0002, point 5 : la bibliothèque standard n'a pas de décomposition Unicode.
5. **Précise l'ADR 0005, point 2** : P3 et P4 sont des règles métier, mais elles s'exécutent dans le navigateur ; leurs scénarios sont donc prouvés dans le navigateur (`@ui`, ADR 0028), et la règle de normalisation, en plus, par les tests unitaires des deux langages sur la table commune. Contre l'API, « une recherche ne renvoie rien » et « ne fait pas partie des propositions » se prouvent par l'absence de l'élément dans `GET dishes` et `GET ingredients`, qui suffit au compartimentage entre tribus (ENF-02).

## Alternatives envisagées

- **Routes de recherche et de propositions dans l'API**, prouvées par godog : écartées. Hors ligne, elles ne rendraient que les recherches déjà faites, et seraient à réécrire au plan hors-ligne ; chaque frappe coûterait une requête.
- **Une recherche côté serveur qui rend une page de résultats** : écartée, pour la même raison.
- **Chaque liste en une seule réponse, sans pagination** : proposée d'abord, écartée par le développeur, qui veut des réponses bornées.
- **Un plafond de plats et d'ingrédients par tribu** : écarté ; rien ne le justifie à cette échelle.
- **Une seule implémentation de la règle, côté serveur** : impossible avec le point 1.
- **Une table des lettres latines accentuées écrite dans le projet**, sans dépendance : écartée. Toute lettre absente de la table resterait telle quelle, et deux noms qui ne diffèrent que par elle feraient deux ingrédients.
- **Retrait de toute la catégorie Unicode `Mn`** plutôt que du seul bloc U+0300 à U+036F : écarté. Les deux langages ne la découpent pas forcément de la même façon selon leur version d'Unicode ; le bloc suffit aux langues latines.

## Conséquences

- **Positif** : une recherche instantanée, sans requête ; le hors-ligne n'aura qu'à garder les pages déjà chargées, sans réécrire la recherche. Les réponses de l'API restent bornées.
- **Négatif** : la règle existe en deux langages ; seule la table commune les garde d'accord, et un écart non couvert par un cas ferait trouver par la recherche ce que le serveur tient pour deux noms, ou l'inverse. Le premier chargement d'une grande bibliothèque coûte plusieurs requêtes.
- **Corriger la règle après coup** demanderait de fusionner des ingrédients et de renommer des plats devenus égaux : les index uniques de la migration `0007` la figent. D'où une table de cas fournie dès le départ (ligatures, majuscules accentuées, apostrophes, espaces insécables).
- Une dépendance d'exécution de plus, du projet Go, revue à chaque montée de version comme les autres.
