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
| Listes en cours (onglet Courses) | `maquettes/Doux-ListesCourses.dc.html` | C7, C10 |
| Nouvelle liste / modifier les dates | `maquettes/Doux-NouvelleListe.dc.html` (réglage `mode`) | C1, C5 |
| Liste de courses | `maquettes/Doux-Courses.dc.html` (réglage `etat` : périmée, à jour, hors ligne) | C2, C3, C4, C5, C6, C8, C9, C10, C11 |
| Historique | `maquettes/Doux-Historique.dc.html` | C10 |
| Liste faite (lecture seule) | `maquettes/Doux-CoursesFaite.dc.html` | C10 |
| Tribu : membres | `maquettes/Doux-Tribu.dc.html` | EF-02, EF-03 (quitter la tribu), EF-06 |
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
- Courses → l'onglet présente les listes en cours et un accès à l'historique ; « Nouvelle liste » ouvre le choix des dates ; toucher une liste l'ouvre. Dans une liste, toucher « Du » ou « Au » ouvre la modification des dates ; « Courses faites » mène à l'historique, « Abandonner » revient aux listes (chacun après confirmation).
- Planning → la pastille de l'en-tête (nom de la tribu et nombre de membres) ouvre l'écran Tribu. Pas d'onglet dédié : la gestion de la tribu est occasionnelle.
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
- **Bandeau de liste périmée** (C6) : sous les dates, liseré ambre `#F2D27A`, icône et texte `#8A6100` ; bouton « Recalculer » en aplat rouge tomate. La pastille « Repas modifiés » (mêmes couleurs) signale une liste périmée dans l'index des listes.
- **Article calculé / ajouté** (C8, C9) : la case à cocher et la ligne sont deux cibles distinctes. Un article calculé porte une pastille neutre « N repas » et un chevron ; toucher la ligne la déplie : initiale du plat, plat et parts, jour et moment (couleurs midi/soir), quantité apportée. Un article ajouté porte une pastille en pointillé « ajouté », une croix pour le retirer, et ne se déplie pas.
- **Ajout d'un article hors repas** (C8) : bouton en pointillé en bas de la liste, qui s'ouvre en formulaire (article, quantité facultative) au liseré de saisie `#E39A89`.
- **Calendrier de période** (C1, C5) : grille du mois, lundi en premier ; bornes au contour rouge tomate 2 px, jours intermédiaires au liseré `#E39A89`, aujourd'hui en pointillé, point sous les jours ayant des repas. Date de fin avant la date de début : message au liseré `#E39A89` et bouton désactivé en pointillé.
- **Abandon d'une liste** (C11) : action destructive, confirmée par la feuille de confirmation.
- **Bandeau hors ligne** : liseré indigo adouci, icône et texte indigo.
- **Avatar de membre** : rond 40 px (72 px sur la fiche), liseré adouci et initiale dans la teinte soutenue, comme l'initiale de plat. Membre révoqué : liseré en pointillé `#CFC6DE`, initiale `#6E6680`.
- **Statut de membre** : pastille « Actif » (liseré `#A9D3B1`, texte `#2F7A3E`) ou « Révoqué » (pointillé `#CFC6DE`, texte `#6E6680`). La carte d'un membre révoqué est en pointillé.
- **Carte de session** : icône d'appareil, nom donné ou appareil détecté, dates ; la session courante porte la pastille verte « cet appareil » et un liseré vert.
- **Action destructive** (révoquer, déconnecter, quitter) : texte ou contour rouge tomate, jamais en aplat sur l'écran. L'aplat n'apparaît que dans la feuille de confirmation.
- **Feuille de confirmation** : panneau blanc en bas d'écran (coins 26 px), voile `rgba(34, 25, 58, 0.45)`, titre sous forme de question, conséquences en une phrase, action en aplat puis « Annuler ».
- **Fond d'accueil** (`maquettes/FondTribu.dc.html`, composant importé par les écrans d'accueil) : *abandonné, à remplacer par le symbole de l'identité visuelle (voir `identite.md`) lors de la refonte des écrans d'accueil.* Description de la version d'origine : une foule de pictogrammes naïfs pleins, cernés d'un trait `#22193A`, qui se chevauchent sans ordre et couvrent tout l'écran (fond `#22193A` derrière). Moitié personnages de la tribu en buste (peaux, coiffures et vêtements variés), moitié ingrédients et plats. Couleurs vives de la palette des plats. Une sous-couche de grands pictogrammes sur une grille serrée garantit la couverture ; la foule est posée par-dessus. Le texte ne se pose jamais directement sur le fond : il est dans un médaillon ou une carte blanche au contour foncé de 2 px. C'est le seul endroit de l'application où la couleur est en aplat hors des actions.
