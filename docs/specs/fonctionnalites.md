# Spécifications fonctionnelles : menus de la semaine (V1)

- **Date** : 2026-09-26 (listes de courses révisées le 2026-09-27, refondues le 2026-10-01 ; bibliothèque de plats précisée le 2026-11-03, plan `bibliotheque` ; planning précisé le 2026-11-05, plan `planning`)
- **Statut** : prêt

## Objectif

Permettre à une tribu (une famille, ou tout groupe qui partage ses repas) de planifier ses repas de la semaine en choisissant des plats, puis d'ajouter automatiquement les courses correspondantes à ses listes de courses, tenues au fil de l'eau. Usage principal : mobile.

## Concepts

| Concept | Description |
| --- | --- |
| **Tribu** | Groupe de membres qui partagent la bibliothèque de plats, le référentiel d'ingrédients, le planning et les listes de courses. Elle a une **taille** (4 par défaut, réglable de 1 à 50, T3, Q31), qui sert de nombre de parts initial. Rien n'est partagé entre tribus (voir `gestion-membres-et-sessions.md` et ENF-02). |
| **Ingrédient** | Élément du référentiel d'ingrédients de la tribu (ex. « tomate », « riz »), avec une unité par défaut, proposée à chaque saisie, absente tant qu'aucune ligne ne lui a donné de quantité (Q20, Q26). Son nom normalisé (Q29) est unique dans la tribu. Il est créé en saisissant un plat (P4) ou un article ajouté (C3), jamais seul. |
| **Bibliothèque de plats** | L'ensemble des plats de la tribu. Un plat y reste indépendamment de son utilisation dans les repas. |
| **Plat** | Un nom, unique dans la tribu (Q28), et une liste d'ingrédients, chacun une seule fois (Q27) et avec une quantité facultative (Q26), exprimée pour un **nombre de parts de référence** (la taille de la tribu par défaut). Il porte une **couleur**, attribuée automatiquement à sa création (Q19). Pas de recette (étapes) en V1. |
| **Repas** | Une case du planning : une date et un moment (**midi** ou **soir**). Elle contient zéro, un ou plusieurs plats servis, dans l'ordre où ils ont été ajoutés (Q35). Un repas n'a pas d'existence propre : il est vide tant qu'aucun plat n'y est servi. |
| **Plat servi** | Un plat de la bibliothèque placé dans un repas, avec son propre **nombre de parts** (la taille de la tribu au moment de l'ajout, de 1 à 50, Q35). Exemple : chili 3 parts et chili végétarien 1 part. Il a une **identité propre**, distincte de celle du plat : retiré puis ajouté de nouveau au même repas, c'est un nouveau plat servi (Q32). |
| **Liste de courses** | Liste nommée, **indépendante du planning** : elle existe par elle-même et reçoit des articles ajoutés à la main (C3) ou les courses des repas d'une période (C4). Une tribu a toujours **au moins une liste**, et exactement une **liste principale**, celle que montre l'onglet Courses (C1). Plusieurs listes peuvent exister en même temps. Une liste n'est jamais close : déclarer les courses faites en retire les articles cochés (C11) ; elle n'est supprimée que vide (C12). |
| **Liste principale** | La liste ouverte par défaut et proposée par défaut pour ajouter des courses. Créée avec la tribu ; tout membre peut désigner une autre liste comme principale ; elle ne peut pas être supprimée. |
| **Article** | Une ligne d'une liste de courses : un ingrédient, une quantité et une unité. Un article **calculé** provient de plats servis (ses **provenances**) et reste lié à eux : on voit à quels plats il participe (C5) ; une liste a au plus un article calculé par ingrédient et famille d'unités. Un article **ajouté** a été saisi à la main, hors planning. |
| **Plats servis reçus** | Les plats servis dont une liste a déjà reçu les courses, par ajout (C4) ou par déplacement (C9, C10). Un plat servi reçu ne peut pas être ajouté une seconde fois à la même liste, même après que ses articles ont été achetés ou déplacés (Q22). |

### Ajout des courses d'une période

L'ajout (C4) prend les plats servis des repas de la période (bornes incluses) qui ne sont pas déjà des plats servis reçus de la liste choisie, calcule leurs ingrédients (règle de calcul ci-dessous) et les **verse dans les articles calculés** de la liste :

- un ingrédient déjà présent dans la liste, dans la même famille d'unités, s'additionne à l'article existant, qui gagne de nouvelles provenances ; sinon un nouvel article calculé est créé ;
- un article coché dont la quantité augmente est décoché (il faut en racheter) ;
- les articles ajoutés à la main ne sont jamais fusionnés avec les articles calculés ;
- si des plats servis de la période étaient déjà reçus, ils sont ignorés et un message indique combien.

### Recalcul d'une liste

Un article calculé reste lié à ses provenances. Une liste est **périmée** (C8) quand l'une des provenances de ses articles a changé depuis le dernier calcul : parts du plat servi modifiées, ingrédients du plat modifiés (P2), plat retiré du repas (R4). Un plat ajouté ensuite à un repas ne rend pas la liste périmée : on l'ajoute en ajoutant de nouveau les courses de la période (C4).

Le recalcul reconstruit les articles calculés de la liste à partir de l'état actuel de leurs provenances, puis :

- les **articles ajoutés** à la main sont conservés tels quels ;
- un article calculé encore présent **reste coché** s'il l'était et que sa quantité n'a pas augmenté ; si sa quantité augmente, il est décoché (il faut en racheter). « Encore présent » signifie même ingrédient et même famille d'unités ;
- une provenance retirée du planning disparaît de ses articles ; un article qui n'a plus de provenance ou plus de quantité disparaît, même coché.

Le recalcul ne concerne que les articles présents : un article déjà retiré par les courses faites (C11) n'est pas recréé.

### Sélection d'articles

Dans une liste, on peut sélectionner des articles (calculés ou ajoutés) pour les **déplacer** vers une autre liste (C9) ou **créer une nouvelle liste** à partir d'eux (C10). Un article déplacé garde sa coche et ses provenances ; ses plats servis deviennent des plats servis reçus de la liste d'arrivée (la liste de départ les garde aussi). Dans la liste d'arrivée :

- un article calculé dont l'ingrédient y est déjà présent, dans la même famille d'unités, est fusionné avec l'article existant ; une provenance déjà présente dans cet article n'est pas comptée deux fois ; l'article fusionné n'est coché que si les deux l'étaient ;
- un article ajouté reste une ligne distincte.

### Règle de calcul

Pour chaque ingrédient d'un plat servi :

```
quantité à acheter = quantité du plat × (parts du plat servi ÷ parts de référence du plat)
```

Exemple : plat « Chili » défini pour 4 parts avec 500 g de bœuf, servi pour 6 parts, soit 750 g de bœuf.

### Unités, conversions et arrondis

- Les unités forment une liste fermée, regroupée par famille :
  - masse : g, kg ;
  - volume : ml, cl, l ;
  - pièce ;
  - cuillère à soupe ;
  - cuillère à café ;
  - pincée.

  Pièce, cuillère à soupe, cuillère à café et pincée forment chacune une famille à elles seules : aucune conversion entre elles (1 cuillère à soupe et 2 cuillères à café d'huile donnent deux lignes).
- Pour agréger un même ingrédient saisi dans des unités différentes de la même famille, on convertit (ex. 500 g + 1 kg = 1,5 kg).
- Entre familles différentes (ex. « 2 pièces » et « 300 g » d'oignon), pas de conversion : lignes distinctes.
- Dans la liste de courses, les quantités en pièces, cuillères et pincées sont arrondies à l'entier supérieur, après agrégation (1,5 oignon donne 2 ; 1,5 cuillère à soupe donne 2). Les masses et les volumes ne sont pas arrondis à l'entier.
- Une quantité agrégée est affichée dans la plus grande unité de sa famille qui donne une valeur d'au moins 1 (1 500 g s'affiche 1,5 kg ; 750 g reste 750 g).

## User stories

Les critères d'acceptation de chaque story sont écrits en Gherkin dans `features/`. Chaque scénario porte l'identifiant de sa story en tag (ex. `@C2`).

### Bibliothèque de plats (`features/bibliotheque-plats.feature`)

- **P1** : Je crée un plat avec un nom, que ne porte aucun autre plat de la tribu, et une liste d'ingrédients (ingrédient, puis quantité et unité facultatives ; chaque ingrédient une seule fois), pour un nombre de parts égal par défaut à la taille de la tribu.
- **P2** : Je modifie un plat ; la modification s'applique à tous les repas où il est servi. Si un autre membre l'a modifié entre-temps, mon enregistrement est refusé et le plat à jour m'est présenté (Q25). Pas de suppression en V1.
- **P3** : Je retrouve un plat en cherchant sur les mots de son nom et sur ses ingrédients (Q30).
- **P4** : La saisie d'un ingrédient propose ceux qui existent déjà (autocomplétion, sans tenir compte de la casse ni des accents) ; je peux en créer un nouveau. Un ingrédient déjà dans le plat n'est pas proposé (Q27). L'unité par défaut de l'ingrédient est proposée.

### Planning des repas (`features/planning-repas.feature`)

- **R1** : Je vois les repas de 7 jours à partir d'aujourd'hui (J à J+6), avec deux cases par jour : midi et soir. « Aujourd'hui » est la date de mon appareil (Q34).
- **R2** : J'ajoute un ou plusieurs plats de la bibliothèque à un repas ; chaque plat servi part d'un nombre de parts égal à la taille de la tribu au moment de l'ajout. Un même plat ne figure qu'une fois dans un repas : un plat déjà servi dans le repas ne m'est pas proposé (Q21), et un plat qu'un autre membre y a servi entre-temps est refusé (Q33).
- **R3** : Je modifie le nombre de parts d'un plat servi.
- **R4** : Je retire un plat d'un repas ; il reste dans la bibliothèque, mais sa configuration dans ce repas est perdue : ajouté de nouveau, il repart de la taille de la tribu (Q32).
- **R5** : Je navigue vers les semaines précédentes et suivantes. Les repas passés restent modifiables comme les autres.

### Liste de courses (`features/liste-courses.feature`)

- **C1** : La tribu a toujours une liste principale, que l'onglet Courses ouvre par défaut ; elle est créée avec la tribu, sous le nom « Courses ». Je peux désigner une autre liste comme principale ; il y en a toujours exactement une.
- **C2** : Je crée une liste vide en lui donnant un nom ; je renomme une liste. Je passe d'une liste à l'autre depuis l'onglet Courses.
- **C3** : J'ajoute à une liste des articles saisis à la main (ex. pain, lessive), quantité facultative ; je peux les retirer.
- **C4** : J'ajoute à une liste les courses des repas d'une période : je choisis la date de début et la date de fin (par défaut, de demain à J+7 inclus) et la liste (par défaut, la liste principale). Les ingrédients sont agrégés avec ajustement aux parts, conversions et arrondis ; les plats servis déjà reçus par cette liste sont ignorés.
- **C5** : Un article calculé indique de combien de plats servis il provient ; en touchant la ligne, elle se déplie et montre les plats et repas concernés, avec la quantité apportée par chacun.
- **C6** : Je coche les articles au fil des courses.
- **C7** : Une liste déjà affichée reste consultable et cochable sans réseau ; les coches et les articles ajoutés se synchronisent au retour du réseau.
- **C8** : Si des plats servis dont proviennent des articles ont changé depuis le dernier calcul, un bandeau en haut de la liste le signale, avec une action « Recalculer ».
- **C9** : Je sélectionne des articles d'une liste et les déplace vers une autre liste.
- **C10** : Je sélectionne des articles d'une liste et crée une nouvelle liste à partir d'eux ; ils quittent la liste de départ.
- **C11** : Je déclare les courses faites : les articles cochés sont retirés de la liste ; les articles non cochés y restent. La liste elle-même demeure.
- **C12** : Je supprime une liste, à condition qu'elle soit vide et qu'elle ne soit pas la liste principale.

### Tribu (`features/tribu.feature`)

- **T1** : Les membres de la tribu partagent la bibliothèque de plats, le référentiel d'ingrédients, le planning et les listes de courses ; les autres tribus n'y ont pas accès.
- **T2** : Tous les membres ont les mêmes droits.
- **T3** : Je règle la taille de la tribu (4 par défaut). Elle s'applique aux plats servis ajoutés ensuite et aux nouveaux plats ; les plats servis déjà placés et les plats existants ne changent pas.

### Membres, sessions et administration

Stories EF-01 à EF-11, détaillées dans `gestion-membres-et-sessions.md` ; critères dans `features/membres-et-sessions.feature` et `features/administration.feature`. Les exigences ENF-01 (connexion et session) et ENF-02 (compartimentage entre tribus) ont leurs critères dans `features/authentification.feature` et `features/compartimentage-tribus.feature`.

## Hors périmètre V1

- Recettes (étapes, temps de préparation, photos).
- Suppression d'un plat de la bibliothèque.
- Rôles et droits différenciés entre membres.
- Synchronisation en temps réel entre membres (un rafraîchissement suffit).
- Suggestions automatiques de menus.
- Gestion du stock ou du placard.
- Préférences et contraintes alimentaires par membre.
- Budget, statistiques, historique des repas ou des courses faites.
- Retrait d'un article calculé d'une liste autrement qu'en le cochant puis en déclarant les courses faites, ou en le déplaçant (C9, C10).
- Retrait d'un plat servi d'une liste ; réajout d'un plat servi déjà reçu par une liste.
- Écran de gestion du référentiel d'ingrédients : renommer, corriger, fusionner ou supprimer un ingrédient.

## Décisions

- **Q1** : deux moments par jour, midi et soir. Un repas peut contenir plusieurs plats, chacun avec son nombre de parts.
- **Q2** : le planning s'ouvre sur aujourd'hui et les 6 jours suivants (J à J+6) ; la période par défaut pour ajouter des courses reste de demain à J+7 (C4), les courses se faisant pour les jours à venir. Les repas passés sont modifiables (PT-09, 2026-09-28).
- **Q3** : conversion d'unités quand c'est nécessaire, au sein d'une même famille. Seules la masse et le volume comptent plusieurs unités ; pièce, cuillère à soupe, cuillère à café et pincée sont chacune leur propre famille (PT-01, 2026-09-28).
- **Q4** : les quantités en pièces, cuillères et pincées sont arrondies à l'entier supérieur (PT-01, 2026-09-28).
- **Q5** : les plats vivent dans une bibliothèque ; retirer un plat d'un repas ne le supprime pas de la bibliothèque, mais perd sa configuration dans ce repas.
- **Q5b** : pas de suppression de plat de la bibliothèque en V1.
- **Q6** : pas de droits différenciés en V1 ; pas de temps réel nécessaire.
- **Q7** : la liste de courses doit pouvoir être consultée et cochée hors ligne.
- **Q8** : chaque plat servi part de la taille de la tribu (4 par défaut, réglable par tout membre, T3), quel que soit le nombre de parts de référence du plat ; un nouveau plat est proposé pour ce même nombre de parts (PT-08, 2026-09-28).
- **Q9** : la bibliothèque de plats et le référentiel d'ingrédients sont propres à chaque tribu : partagés entre ses membres, jamais entre tribus.
- **Q10** : les articles calculés ne changent que sur action explicite (ajout de courses, « Recalculer », déplacement), jamais automatiquement quand le planning change : la liste ne bouge pas sous les yeux de celui qui fait les courses. Le changement est seulement signalé (C8) (révisé le 2026-10-01).
- **Q11** : au recalcul, les articles ajoutés à la main sont conservés, et les coches aussi, sauf quand la quantité d'un article augmente.
- **Q12** : plusieurs listes sont permises. Un même plat servi peut être reçu par plusieurs listes ; il ne l'est qu'une fois par liste (Q22) (révisé le 2026-10-01).
- **Q13** : un article ajouté à la main est un ingrédient du référentiel (autocomplétion et création comme en P4), avec quantité et unité facultatives. Il reste une ligne distincte, jamais fusionnée avec un article calculé du même ingrédient.
- **Q14** : « courses faites » est possible même s'il reste des articles non cochés : seuls les articles cochés sont retirés, la liste reste utilisable. Pas d'historique des courses faites (révisé le 2026-10-01).
- **Q15** : supprimer une liste n'est possible que si elle est vide et n'est pas la liste principale ; pas de confirmation, puisqu'il n'y a rien à perdre (révisé le 2026-10-01).
- **Q16** : sans réseau, on peut cocher et ajouter des articles (synchronisés au retour du réseau) ; ajouter les courses d'une période, recalculer, déplacer des articles, créer, renommer ou supprimer une liste, changer la liste principale et déclarer les courses faites nécessitent le réseau (révisé le 2026-10-01).
- **Q18** : au retour du réseau, les coches et les articles ajoutés hors ligne sont rejoués (PT-03, 2026-09-28 ; révisé le 2026-10-01) :
  - si la liste a été entre-temps supprimée, les articles ajoutés hors ligne sont ignorés et un message le signale ;
  - si un article coché hors ligne a été entre-temps déplacé vers une autre liste, la coche s'applique à l'article dans sa nouvelle liste ;
  - si un recalcul a entre-temps supprimé un article coché hors ligne, la coche est ignorée et un message le signale ; s'il a été retiré par les courses faites, il était déjà coché et rien n'est signalé ;
  - un ingrédient créé hors ligne par un article ajouté est rapproché d'un ingrédient existant de même nom normalisé plutôt que dupliqué.
- **Q19** : à sa création, un plat reçoit la teinte de la palette (`design/README.md`) la moins utilisée dans la bibliothèque, la première dans l'ordre de la palette en cas d'égalité ; elle est enregistrée avec le plat et ne change plus. Pas de choix de couleur par le membre en V1 (PT-05, 2026-09-28).
- **Q20** : référentiel d'ingrédients (PT-06, 2026-09-28) :
  - deux noms qui ne diffèrent que par la casse, les accents ou les espaces (en tête, en fin, répétés) désignent le même ingrédient ; le nom affiché est celui de la première saisie ;
  - l'unité par défaut d'un ingrédient est la première unité saisie avec lui ; elle est proposée à chaque saisie suivante et reste modifiable ligne par ligne ;
  - un ingrédient n'est créé qu'en saisissant un plat ou un article ajouté ; pas d'écran de gestion du référentiel en V1.
- **Q21** : un même plat ne peut figurer qu'une fois dans un repas ; pour en prévoir davantage, on augmente ses parts (PT-10, 2026-09-28).
- **Q17** : une liste est désignée par son nom, saisi à sa création et modifiable ; deux listes peuvent porter le même nom. La liste principale vient en premier, les autres suivent par ordre de création (révisé le 2026-10-01).
- **Q22** : un plat servi n'est reçu qu'une fois par liste. Une liste mémorise les plats servis qu'elle a reçus, par ajout ou par déplacement, y compris ceux dont les articles ont depuis été achetés ou déplacés : ajouter de nouveau les courses d'une période n'apporte que les plats servis nouveaux (2026-10-01).
- **Q23** : une tribu a toujours au moins une liste et exactement une liste principale. La liste principale est créée avec la tribu (nom « Courses ») ; en désigner une autre retire ce rôle à la précédente ; la liste principale ne peut pas être supprimée (2026-10-01).
- **Q24** : un article calculé agrège, pour un ingrédient et une famille d'unités, les quantités de tous ses plats servis : une seule ligne par liste, dépliable vers ses provenances, déplacée d'un bloc (2026-10-01).
- **Q25** : un plat porte une **révision**, augmentée à chaque modification. Un enregistrement fait sur une révision dépassée, parce qu'un autre membre a modifié le plat entre-temps, est refusé : rien n'est écrasé ; le membre voit « Ce plat a été modifié entre-temps » et le plat à jour, qu'il modifie de nouveau. Pas de fusion automatique des deux modifications (plan `bibliotheque`, D5, 2026-10-29).
- **Q26** : la quantité d'un ingrédient dans un plat est **facultative** (« sel », « poivre »), comme pour un article ajouté (Q13). Sans quantité, pas d'unité ; avec une quantité, une unité. L'unité par défaut d'un ingrédient est la première unité saisie avec lui ; tant qu'aucune ligne ne lui en a donné, il n'en a pas (précise Q20). Ce que devient une ligne sans quantité dans les courses (C4) sera fixé avec les listes de courses (plan `bibliotheque`, D4, 2026-10-29).
- **Q27** : un ingrédient ne figure qu'**une fois dans un plat**, quelle que soit l'unité : « 3 pièces » et « 200 g » de courgette sont refusés dans un même plat. Un ingrédient déjà dans le plat n'est plus proposé à la saisie (plan `bibliotheque`, D6, 2026-10-29).
- **Q28** : deux plats d'une tribu ne portent pas le **même nom**, au sens du nom normalisé (Q29) : « Soupe à l'oignon » et « soupe a l’oignon » sont le même nom ; le second est refusé avec « Un plat porte déjà ce nom ». Contrairement aux listes de courses (Q17) (plan `bibliotheque`, D7, 2026-10-29).
- **Q29** : **nom normalisé**, une seule règle pour l'identité d'un ingrédient (Q20), l'unicité d'un nom de plat (Q28), la recherche (P3) et l'autocomplétion (P4). Deux noms sont le même s'ils ne diffèrent que par (complète Q20) :
  - la casse ;
  - les ligatures œ et æ, qui valent oe et ae (« Bœuf » et « boeuf ») ;
  - les accents et autres signes diacritiques ;
  - l'apostrophe, droite ou typographique (« piment d’Espelette » et « piment d'Espelette ») ;
  - les espaces : en tête, en fin, répétées, insécables.

  La règle exacte (ordre des étapes, caractères concernés) est dans `docs/conception/bibliotheque-de-plats.md` ; le nom affiché reste celui de la première saisie (plan `bibliotheque`, D8, 2026-10-29).
- **Q30** : la **bibliothèque** est présentée par ordre alphabétique du nom des plats, les résultats d'une recherche aussi. Une recherche découpe le texte saisi en termes, et le nom et les ingrédients de chaque plat en mots, après normalisation (Q29) ; un terme trouve un mot qui commence par lui (« courg » trouve « courgette ») ; un plat est un résultat s'il contient tous les termes (« gratin courgette ») (plan `bibliotheque`, D22, 2026-10-29).
- **Q31** : **limites** (plan `bibliotheque`, D12 et D20, 2026-10-29) :
  - taille de la tribu de 1 à 50 ;
  - nom d'un plat de 60 caractères au plus, nom d'un ingrédient de 40, 40 ingrédients par plat ;
  - dans le nom d'un plat ou d'un ingrédient, les espaces sont réduites à une seule entre deux mots ; un caractère de contrôle est refusé ; un nom sans aucun caractère visible est refusé comme un nom vide (plan `bibliotheque`, D30, 2026-11-03) ;
  - quantité strictement positive, saisie avec une virgule ou un point ; au plus les décimales que l'unité permet : aucune en ml, une en cl, trois en l, g, kg, pièce, cuillère à soupe, cuillère à café et pincée (« 2,5 cl » et « 0,125 kg » acceptés, « 2,5 ml » refusé). Au-delà, la quantité est refusée (« Quantité trop précise pour cette unité ») plutôt qu'arrondie en silence.
- **Q32** : un plat **retiré puis ajouté de nouveau** à un repas est un **nouveau plat servi** : il repart de la taille de la tribu (R4) et passe en dernier dans le repas (Q35) ; rien ne le relie au plat servi retiré. Les listes de courses le traiteront comme un plat servi qu'elles n'ont pas encore reçu (Q22) (plan `planning`, D2, 2026-11-05).
- **Q33** : **écritures croisées sur un repas**, quand deux membres le modifient en même temps. Un plat servi n'a pas de révision, contrairement à un plat (Q25) : il n'y a pas de temps réel (Q6), et un réglage de parts se prête mal à un refus (plan `planning`, D3, 2026-11-05).
  - **parts** : le dernier enregistrement l'emporte, sans message ;
  - **ajout d'un plat qu'un autre membre a servi entre-temps** dans le repas : refusé (Q21) ; le membre voit « « Chili con carne » est déjà servi dans ce repas. » et le repas à jour ;
  - **parts d'un plat servi qu'un autre membre a retiré entre-temps** : refusées ; le membre voit « Ce plat a été retiré du repas entre-temps. » et le repas à jour ;
  - **retrait d'un plat servi déjà retiré** par un autre membre : rien n'est signalé, le retrait est fait.
- **Q34** : **« aujourd'hui » est la date de l'appareil** du membre ; le serveur ne la connaît pas. Deux membres dans des fuseaux horaires différents voient chacun leur « aujourd'hui », sur le même planning. Un repas est désigné par sa seule date, sans heure ni fuseau. Quand la date de l'appareil change pendant que le planning montre les 7 jours à partir d'aujourd'hui, il passe au nouveau jour au retour sur l'application ; une autre semaine affichée reste affichée (plan `planning`, D4 et D17, 2026-11-05).
- **Q35** : **limites et ordre du planning** (plan `planning`, D4 et D11, 2026-11-05) :
  - parts d'un plat servi de 1 à 50, comme la taille de la tribu (Q31) ;
  - repas du 1er janvier 2000 au 31 décembre 2099, passés comme futurs (Q2) ;
  - les plats d'un repas sont présentés dans l'ordre où ils ont été ajoutés, sur le planning comme dans le détail du repas ; un plat retiré puis ajouté de nouveau passe en dernier (Q32). L'ordre alphabétique est écarté : il déplacerait un plat sous le doigt.
