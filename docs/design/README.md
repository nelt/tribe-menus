# Design V1 : menus de la semaine

- **Date** : 2026-09-26
- **Statut** : retenu (version D)
- **Canevas** : [Menus de la semaine](https://claude.ai/artifact/YFJvuxWWxS89coq6c5M6zH) (privé ; les versions A à D y sont visibles, la D est celle retenue)

Principe : fond blanc, cartes délimitées par des liserés fins, couleur portée par des liserés adoucis et par le texte. La seule couleur pleine est réservée aux actions (rouge tomate). Clair et coloré, sans être chargé.

L'identité visuelle (nom **Melting Tribe**, symbole, logotype, icônes), qui s'applique aux pages publiques et à l'écran de connexion, est décrite dans [`identite.md`](identite.md).

## Écrans et stories

| Écran | Fichier de référence | Stories |
| --- | --- | --- |
| Chargement (session valide) | `maquettes/identite/Chargement.dc.html` (réglage `etat`) | ENF-01 |
| Accueil : saisie de l'e-mail | `maquettes/identite/Connexion.dc.html`, `maquettes/identite/ConnexionBureau.dc.html` (réglage `etat`) | ENF-01 |
| Accueil : saisie du code | `maquettes/identite/Code.dc.html` (réglage `etat`) | ENF-01 |
| Planning de la semaine | `maquettes/Doux-Main.dc.html` | R1, R5 |
| Détail d'un repas | `maquettes/Doux-Repas.dc.html` | R2, R3, R4 |
| Bibliothèque de plats (recherche) | `maquettes/Doux-Bibliotheque.dc.html` | P3 |
| Création / édition d'un plat | `maquettes/Doux-Plat.dc.html` | P1, P4 |
| Liste de courses (onglet Courses) | `maquettes/Doux-Courses.dc.html` (réglage `etat` : périmée, à jour, hors ligne, sélection) | C1, C3, C5, C6, C7, C8, C11 |
| Sélection d'articles | `maquettes/Doux-CoursesSelection.dc.html` (l'écran précédent, réglage `etat` sur sélection) | C9, C10 |
| Choix des listes | `maquettes/Doux-ListesCourses.dc.html` | C1, C2, C12 |
| Courses des repas (période et liste) | `maquettes/Doux-AjoutCourses.dc.html` | C4 |
| Tribu : membres | `maquettes/Doux-Tribu.dc.html` | EF-02, EF-03 (quitter la tribu), EF-06, T3 |
| Ajouter un membre | `maquettes/Doux-AjoutMembre.dc.html` | EF-01, EF-06 |
| Fiche d'un membre | `maquettes/Doux-Membre.dc.html` | EF-03, EF-06 |
| Mes appareils | `maquettes/Doux-Appareils.dc.html` | EF-04, EF-05 |

Les fichiers `.dc.html` sont les sources des maquettes (format de l'outil de design, pas du code de production). Ils servent de référence précise pour la structure, les styles et les comportements ; la logique de recherche de `Doux-Bibliotheque.dc.html` illustre les règles de P3.

## Navigation

- Accueil → « Recevoir un code » ouvre la saisie du code → « Se connecter » ouvre le planning. « Se déconnecter » et « Quitter la tribu » ramènent à l'accueil.
- Les écrans d'accueil n'affichent pas le nom de la tribu : il n'apparaît qu'une fois la connexion réussie (ENF-02, ADR 0006). La pastille « Les Martin » des versions antérieures des maquettes a été retirée.
- Barre d'onglets en bas : **Planning**, **Plats**, **Courses**. Onglet actif en aplat rouge tomate, texte blanc.
- Planning → toucher une case de repas ouvre le détail du repas.
- Plats → « Nouveau » ouvre la création d'un plat.
- Courses → l'onglet ouvre la liste principale. Son nom, en tête, ouvre le choix des listes ; « Ajouter les courses des repas » ouvre le choix de la période et de la liste ; « Sélectionner » fait passer la liste en mode sélection ; « Courses faites » retire les articles cochés, après confirmation.
- Choix des listes → toucher une liste l'ouvre ; le bouton « ⋮ » d'une liste ouvre ses actions (renommer, définir comme principale, supprimer) ; « Nouvelle liste » crée une liste vide.
- Courses des repas → « Ajouter à « … » » revient à la liste.
- Planning → la pastille de l'en-tête (nom de la tribu et nombre de membres) ouvre l'écran Tribu. Pas d'onglet dédié : la gestion de la tribu est occasionnelle.
- Tribu → le réglage **Taille de la tribu** (stepper) fixe les parts proposées pour les plats servis ajoutés et les nouveaux plats (T3).
- Tribu → toucher un membre ouvre sa fiche ; « Ajouter un membre » ouvre l'ajout ; « Mes appareils » ouvre la liste des sessions.

## Tokens

### Couleurs

| Rôle | Valeur |
| --- | --- |
| Fond | `#FFFFFF` |
| Texte principal | `#22193A` |
| Texte secondaire | `#5E5470` |
| Liseré neutre (cartes, séparateurs) | `#E7E1EE` |
| Pointillé (case vide, ajout) | `#CFC6DE` |
| Action (aplat, texte blanc) | `#C63D24` ; survol `#9E2E19` |
| Midi : liseré / texte | `#F2D27A` / `#8A6100` |
| Soir : liseré / texte | `#BDBAF0` / `#3D3A8C` |
| Coché : liseré / barre de progression | `#A9D3B1` / `#2F7A3E` |
| Champ en cours de saisie (liseré) | `#E39A89` |

Chaque plat a une couleur, attribuée automatiquement à sa création parmi les teintes ci-dessous (règle Q19 de `fonctionnalites.md`), utilisée en **texte** (initiale, badge de parts) dans sa teinte soutenue, et en **liseré** dans une teinte éclaircie de 60 % vers le blanc :

| Teinte soutenue | Exemple |
| --- | --- |
| `#C2502E` | Chili con carne |
| `#3F7F25` | Chili végétarien |
| `#2F6FB0` | Gratin de courgettes |
| `#B0306A` | Gratin dauphinois |
| `#B36A0B` | Ratatouille |
| `#8A45B3` | Soupe à l'oignon |
| `#2E8A6E` | Pâtes pesto |
| `#9C7400` | Pizza maison |
| `#6E6680` | Restes |

Liseré = `c + (255 − c) × 0,6` sur chaque composante RVB.

### Typographie

- Titres : **Bricolage Grotesque** 700/800 (Google Fonts). H1 36 px, titres de section 20 px.
- Texte : **Figtree** 400 à 800. Corps 15–16 px, libellés 12–13 px.
- Libellés de moment (MIDI, SOIR) : 11 px, 800, espacement 1 px, en capitales.

### Formes

- Liserés : 1,5 px (colorés et neutres) ; 2 px foncé (`#22193A`) pour les champs de saisie.
- Rayons : 999 px (pilules, badges), 16 px (champs, cases de repas, boutons), 20–22 px (cartes), 24 px (en-tête de repas).
- Cibles tactiles : 44 px minimum.

## Composants récurrents

- **Case de repas** : liseré de la couleur du moment (midi/soir), libellé en capitales, plats en pilules ; case vide en pointillé avec « + Ajouter ».
- **Pilule de plat** : nom tronqué + badge rond du nombre de parts.
- **Stepper de parts** : − / valeur / +, contour foncé, « + » en rouge tomate.
- **Initiale de plat** : carré arrondi 40–44 px, liseré adouci, lettre dans la teinte soutenue.
- **Article de liste de courses** : case à cocher native, nom, quantité en rouge tomate ; coché = barré, liseré vert, déplacé en bas.
- **Bandeau de liste périmée** (C8) : sous le nom de la liste, liseré ambre `#F2D27A`, icône et texte `#8A6100` ; bouton « Recalculer » en aplat rouge tomate. La pastille « Repas modifiés » (mêmes couleurs) signale une liste périmée dans le choix des listes.
- **Article calculé / ajouté** (C3, C5) : la case à cocher et la ligne sont deux cibles distinctes. Un article calculé porte une pastille neutre « N plats » et un chevron ; toucher la ligne la déplie : initiale du plat, plat et parts, jour et moment (couleurs midi/soir), quantité apportée. Un article ajouté porte une pastille en pointillé « ajouté », une croix pour le retirer, et ne se déplie pas.
- **Ajout d'un article hors repas** (C3) : bouton en pointillé en bas de la liste, qui s'ouvre en formulaire (article, quantité facultative) au liseré de saisie `#E39A89`.
- **Calendrier de période** (C4) : grille du mois, lundi en premier ; bornes au contour rouge tomate 2 px, jours intermédiaires au liseré `#E39A89`, aujourd'hui en pointillé, point sous les jours ayant des repas. Date de fin avant la date de début : message au liseré `#E39A89` et bouton désactivé en pointillé.
- **Courses faites** (C11) : bouton en aplat en bas de la liste, en pointillé et inactif tant qu'aucun article n'est coché ; confirmé par la feuille de confirmation, qui dit combien d'articles partent et combien restent. Un message au liseré vert confirme ensuite le retrait.
- **Pastille « principale »** (C1) : contour foncé `#22193A`, à côté du nom de la liste (en-tête, choix des listes, choix de la destination). Dans le choix des listes, la carte de la liste principale a un liseré foncé.
- **Mode sélection** (C9, C10) : la case à cocher laisse place à une pastille ronde au contour foncé, pleine `#22193A` quand l'article est sélectionné ; la carte sélectionnée prend un liseré foncé de 2 px. L'en-tête devient « Annuler », le nombre d'articles sélectionnés, « Tout ». La barre d'onglets laisse place à deux actions : « Déplacer vers… » (aplat) et « Nouvelle liste » (contour foncé), en pointillé tant que rien n'est sélectionné. Chacune ouvre une feuille : choix de la liste d'arrivée, ou nom de la nouvelle liste.
- **Actions d'une liste** (C2, C12) : feuille ouverte par « ⋮ ». « Supprimer la liste » est en contour rouge tomate, sans confirmation, pour une liste vide et non principale ; sinon en pointillé, avec la raison.
- **Destination des courses** (C4) : champ au contour foncé « Ajouter à la liste », qui ouvre une feuille de choix ; sous le champ, une ligne indique les plats servis déjà passés par la liste, qui seront ignorés (Q22).
- **Bandeau hors ligne** : liseré indigo adouci, icône et texte indigo.
- **Avatar de membre** : rond 40 px (72 px sur la fiche), liseré adouci et initiale dans la teinte soutenue, comme l'initiale de plat. Membre révoqué : liseré en pointillé `#CFC6DE`, initiale `#6E6680`.
- **Statut de membre** : pastille « Actif » (liseré `#A9D3B1`, texte `#2F7A3E`) ou « Révoqué » (pointillé `#CFC6DE`, texte `#6E6680`). La carte d'un membre révoqué est en pointillé.
- **Carte de session** : icône d'appareil, nom donné ou appareil détecté, dates ; la session courante porte la pastille verte « cet appareil » et un liseré vert.
- **Action destructive** (révoquer, déconnecter, quitter) : texte ou contour rouge tomate, jamais en aplat sur l'écran. L'aplat n'apparaît que dans la feuille de confirmation.
- **Feuille de confirmation** : panneau blanc en bas d'écran (coins 26 px), voile `rgba(34, 25, 58, 0.45)`, titre sous forme de question, conséquences en une phrase, action en aplat puis « Annuler ».
- **Fond d'accueil** (`maquettes/FondTribu.dc.html`, composant importé par les écrans d'accueil) : *abandonné, à remplacer par le symbole de l'identité visuelle (voir `identite.md`) lors de la refonte des écrans d'accueil.* Description de la version d'origine : une foule de pictogrammes naïfs pleins, cernés d'un trait `#22193A`, qui se chevauchent sans ordre et couvrent tout l'écran (fond `#22193A` derrière). Moitié personnages de la tribu en buste (peaux, coiffures et vêtements variés), moitié ingrédients et plats. Couleurs vives de la palette des plats. Une sous-couche de grands pictogrammes sur une grille serrée garantit la couverture ; la foule est posée par-dessus. Le texte ne se pose jamais directement sur le fond : il est dans un médaillon ou une carte blanche au contour foncé de 2 px. C'est le seul endroit de l'application où la couleur est en aplat hors des actions.
