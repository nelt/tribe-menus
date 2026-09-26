# Design V1 : menus de la semaine

- **Date** : 2026-09-26
- **Statut** : retenu (version D)
- **Canevas** : [Menus de la semaine](https://claude.ai/artifact/YFJvuxWWxS89coq6c5M6zH) (privé ; les versions A à D y sont visibles, la D est celle retenue)

Principe : fond blanc, cartes délimitées par des liserés fins, couleur portée par des liserés adoucis et par le texte. La seule couleur pleine est réservée aux actions (rouge tomate). Clair et coloré, sans être chargé.

## Écrans et stories

| Écran | Fichier de référence | Stories |
| --- | --- | --- |
| Planning de la semaine | `maquettes/Doux-Main.dc.html` | R1, R5 |
| Détail d'un repas | `maquettes/Doux-Repas.dc.html` | R2, R3, R4 |
| Bibliothèque de plats (recherche) | `maquettes/Doux-Bibliotheque.dc.html` | P3 |
| Création / édition d'un plat | `maquettes/Doux-Plat.dc.html` | P1, P4 |
| Liste de courses | `maquettes/Doux-Courses.dc.html` | C1, C2, C3, C4 |

Les fichiers `.dc.html` sont les sources des maquettes (format de l'outil de design, pas du code de production). Ils servent de référence précise pour la structure, les styles et les comportements ; la logique de recherche de `Doux-Bibliotheque.dc.html` illustre les règles de P3.

## Navigation

- Barre d'onglets en bas : **Planning**, **Plats**, **Courses**. Onglet actif en aplat rouge tomate, texte blanc.
- Planning → toucher une case de repas ouvre le détail du repas.
- Plats → « Nouveau » ouvre la création d'un plat.

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

Chaque plat a une couleur, utilisée en **texte** (initiale, badge de parts) dans sa teinte soutenue, et en **liseré** dans une teinte éclaircie de 60 % vers le blanc :

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
- **Bandeau hors ligne** : liseré indigo adouci, icône et texte indigo.
