# Design : Melting Tribe

- **Date** : 2026-10-02
- **Statut** : état actuel du design retenu, sur tout le périmètre : identité, site public, connexion, application.
- **Portée** : ce document est la référence unique du design. Il décrit ce qui est retenu aujourd'hui, pas le chemin pour y arriver ; les pistes écartées restent dans les canevas et dans l'historique Git.

## Sources

| Source | Contenu |
| --- | --- |
| `maquettes/` | Écrans de l'application. Le préfixe `Doux-` est le nom de la version graphique retenue ; il n'a pas d'autre sens. |
| `maquettes/identite/` | Symbole, logotype, icônes, écrans de connexion et de chargement, pages du site public. |
| `identite/` | Fichiers livrables de l'identité : symbole et logotype en SVG, image de partage. |
| Canevas [Melting Tribe · design V1 · 2026-10-02](https://claude.ai/artifact/XmpkhmHJCLzd3Eyp9piVxy) (privé) | **Référence visuelle** : toutes les maquettes retenues, et elles seules, à la date du titre. Copie datée ; elle est refaite, avec un nouveau titre, quand le design change. |
| Canevas [Menus de la semaine](https://claude.ai/artifact/YFJvuxWWxS89coq6c5M6zH) (privé) | Canevas de travail des écrans de l'application, y compris les versions écartées. |
| Canevas [Melting Tribe, déclinaisons](https://claude.ai/artifact/PnDbsVYrEjfZondTPHUVe1) (privé) | Canevas de travail de l'identité et de ses déclinaisons. |

Les fichiers `.dc.html` sont les sources des maquettes (format de l'outil de design, pas du code de production). Ils font référence pour la structure, les styles, les textes et les comportements. Certains ont un réglage (`etat`, `mode`) qui montre leurs différents états.

Une maquette est d'abord dessinée ou modifiée dans son canevas de travail, validée, puis reportée ici ; ce document est mis à jour dans la même PR, et le canevas de référence est refait à partir du dépôt.

## Principes

1. **Dans l'application** : fond blanc, cartes délimitées par des liserés fins, couleur portée par des liserés adoucis et par le texte. La seule couleur pleine est le rouge tomate des actions. Clair et coloré, sans être chargé.
2. **Sur les pages publiques et à la connexion** : fond blanc aussi, mais l'aplat n'est plus réservé aux actions ; le symbole et les grands blocs (bandeau `#22193A` du site) en portent.
3. **Jamais le nom d'une tribu avant la connexion** : site public, écrans de connexion, écran de chargement, manifeste sans session et e-mail du code ne montrent que la marque (ENF-02, ADR 0006, ADR 0014).
4. **Usage principal sur téléphone** : cibles tactiles de 44 px au moins ; les maquettes de l'application sont en 390 × 844.
5. **Accessibilité, telle que les maquettes la pratiquent** : boutons, liens et champs natifs, avec libellé ; messages d'erreur accompagnés d'une icône et annoncés aux lecteurs d'écran (`role="alert"`) ; animations de l'écran de chargement coupées avec `prefers-reduced-motion`. Aucune exigence d'accessibilité n'est écrite dans les specs à ce jour.

## Identité

### Nom

**Melting Tribe**, en référence au *melting pot* : la tribu qui se mélange autour de la marmite. C'est la marque côté utilisateurs ; le nom d'hôte est `meltingtribe.codingmatters.org` ; le dépôt et les noms techniques restent `tribe-menus` (ADR 0017). Accroche : « Les menus de la semaine, en tribu. »

### Symbole

Trois personnages émergent d'une marmite dont la soupe déborde ; au-dessus, des ingrédients s'envolent en arc, de plus en plus gros de droite à gauche (cosse de petits pois, aubergine, part de pizza, tomate, carotte). La tomate touche le cercle, la carotte en sort franchement : ce débordement est une signature du symbole.

- **Personnages** : silhouettes pleines, sans visage ni main, bras droits levés ; ocre, bleu, vert.
- **Volume** : lumière venant d'en haut à gauche, traitée en aplats, sans dégradé : croissant d'ombre plus foncé en bas à droite, petit reflet clair en haut à gauche.
- **Marmite** : demi-disque rouge tomate, bord `#22193A`, soupe et coulures jaunes.
- **Cercle** : anneau rouge tomate, intérieur blanc, présent dans toutes les versions ; la carotte passe par-dessus. Sur fond sombre, le symbole garde son intérieur blanc et son anneau rouge.

| Fichier | Usage |
| --- | --- |
| `identite/symbole.svg` | Version principale, avec cercle. Au-dessus de 48 px. |
| `identite/symbole-sans-cercle.svg` | Quand le cadre est déjà rond ou chargé. |
| `identite/symbole-reduit.svg` | 48 px et moins (favicon, onglets) : sans ombres, carotte et tomate seulement, formes épaissies, anneau deux fois plus épais, lisible jusqu'à 16 px. |
| `identite/symbole-mono.svg` | Une couleur (encre `#22193A`), formes séparées par un liseré blanc. |
| `identite/symbole-mono-inverse.svg` | Blanc sur fond `#22193A`. |

### Logotype

- **Lettrage** : « Melting Tribe » en Bricolage Grotesque 800, « Melting » en rouge tomate `#C63D24`, « Tribe » en `#22193A`.
- **Coulée** : une coulée de soupe jaune `#F2C94C` sous « Melting ». Elle reste jaune partout, jamais rouge.
- **Sur fond sombre** (`#22193A`) : « Melting » en tomate éclaircie `#EE7A5F`, « Tribe » en blanc, coulée jaune.
- **Assemblages** : horizontal (symbole à gauche) et vertical (symbole au-dessus, avec l'accroche).
- **Fichiers** : `identite/logotype.svg` (fond clair) et `identite/logotype-sombre.svg` (fond sombre), texte converti en tracés : ils ne dépendent d'aucune police. Tracés obtenus à partir de Bricolage Grotesque (`wght` 800, `opsz` 72, `wdth` 100), approche de −1,8 px et crénage de la police.

### Icônes d'application

- **Icône PWA** (192 et 512 px) : symbole complet avec cercle sur fond blanc ; variante sur fond `#22193A` si le blanc se perd.
- **Android « maskable »** : plein cadre blanc, symbole dans la zone sûre (cercle central de 80 %). Les fanes de la carotte sortent de la zone et sont rognées par les lanceurs à masque rond : accepté.
- **Apple** (180 px) : fond blanc, iOS arrondit les coins.
- **Favicon** : symbole réduit, en SVG.
- **Manifeste** : nom générique « Melting Tribe » sans session (ADR 0006, point 9) ; `background_color` blanc.

### Image de partage

`identite/partage.png` (1200 × 630, source `identite/partage.svg`), pour les balises Open Graph et Twitter du site public et l'aperçu social du dépôt GitHub : symbole avec cercle à gauche ; logotype, accroche et adresse `meltingtribe.codingmatters.org` à droite ; fond blanc ; textes en tracés.

### Licence

Le symbole et le logotype sont couverts, comme tout le dépôt, par l'AGPL ; les polices par l'OFL (ADR 0011). Aucune protection de marque n'est décidée à ce jour.

## Tokens

### Couleurs d'interface

| Rôle | Valeur |
| --- | --- |
| Fond | `#FFFFFF` |
| Texte principal, contour foncé, encre | `#22193A` |
| Texte secondaire | `#5E5470` |
| Texte atténué (inactif, révoqué, croix de retrait) | `#6E6680` |
| Liseré neutre (cartes, séparateurs) | `#E7E1EE` |
| Pointillé (case vide, ajout, bouton inactif) | `#CFC6DE` |
| Action (aplat, texte blanc) ; erreur (texte) | `#C63D24` ; survol `#9E2E19` |
| Champ en cours de saisie, message d'erreur (liseré) | `#E39A89` |
| Midi, avertissement : liseré / texte | `#F2D27A` / `#8A6100` |
| Soir, hors ligne : liseré / texte | `#BDBAF0` / `#3D3A8C` |
| Coché, réussite, actif : liseré / texte et barre de progression | `#A9D3B1` / `#2F7A3E` |
| Voile derrière une feuille | `rgba(34, 25, 58, 0.45)` |

### Couleurs des plats

Chaque plat a une couleur, attribuée automatiquement à sa création parmi les teintes ci-dessous (règle Q19 de `specs/fonctionnalites.md`). Elle s'utilise en **texte** (initiale, badge de parts) dans sa teinte soutenue, et en **liseré** dans une teinte éclaircie de 60 % vers le blanc : `c + (255 − c) × 0,6` sur chaque composante RVB. Les avatars de membres suivent la même règle.

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

### Couleurs de l'identité

| Rôle | Valeur (lumière / ombre) |
| --- | --- |
| Anneau, marmite, « Melting » | `#C63D24` (`#E0654C` / `#952B17`) |
| « Melting » sur fond sombre | `#EE7A5F` |
| Soupe, coulée | `#F2C94C` |
| Carotte | `#E07B1F` |
| Personnage ocre | `#B36A0B` (`#D9953A` / `#8A5008`) |
| Personnage bleu | `#2F6FB0` (`#5C93CC` / `#1F4E80`) |
| Personnage vert | `#3F7F25` (`#67A24A` / `#2B5A18`) |

### Typographie

- Titres : **Bricolage Grotesque** 700/800. H1 de 32 à 36 px (40 px pour le moment d'un repas) ; titres de section 20 px ; titre de feuille 24 px.
- Texte : **Figtree** 400 à 800. Corps 15–16 px, libellés 12–13 px.
- Libellés de moment (MIDI, SOIR) : 11 px, 800, espacement 1 px, en capitales.
- Polices auto-hébergées, sans appel à Google Fonts (ADR 0004, point 5) ; seules les maquettes les chargent depuis Google Fonts.

### Formes

- Liserés : 1,5 px (colorés et neutres) ; 2 px foncé (`#22193A`) pour les champs de saisie, les champs de choix et l'élément sélectionné.
- Rayons : 999 px (pilules, badges, pastilles), 16 px (champs, cases de repas, boutons, articles), 18 px (boutons principaux), 20–22 px (cartes), 24 px (en-tête de repas), 26 px (haut d'une feuille).
- Cibles tactiles : 44 px minimum.

## Écrans

### Site public

Servi statiquement à la racine de `meltingtribe.codingmatters.org` (ADR 0006, ADR 0017), sources dans `site/`. Le texte des maquettes fait référence.

| Page | Maquette |
| --- | --- |
| Présentation, ordinateur | `maquettes/identite/Site.dc.html` |
| Présentation, téléphone | `maquettes/identite/SiteMobile.dc.html` |
| Mentions légales | `maquettes/identite/Mentions.dc.html` |
| Confidentialité | `maquettes/identite/Confidentialite.dc.html` |

- **Ton** : vouvoiement ; neutre, sans histoire de l'auteur ; pas d'aperçus d'écrans, seulement le symbole.
- **Présentation** : titre « Les menus de la semaine, en tribu. », sous-titre, boutons « Comment ça marche » et « Voir le code source », phrase d'accès (« sur invitation d'un membre d'une tribu », pas d'inscription) ; deux messages mis en avant, les courses calculées et la semaine construite à plusieurs, dans des cartes aux liserés midi et soir ; une ligne « Fonctionne hors ligne · Vos données restent à votre tribu » ; bandeau « Un logiciel libre » sur fond `#22193A` avec lien vers le dépôt.
- **Pied de page** (toutes les pages) : symbole, « logiciel libre sous licence AGPL-3.0 », liens Mentions légales, Confidentialité et « Code source de cette version », qui pointe vers l'étiquette déployée (ADR 0011, point 2).
- **Mentions légales** : éditeur Nel Taurisson, à titre personnel ; contact `contact@codingmatters.org` ; hébergeur OVH SAS ; licences (AGPL pour le code, la documentation et les éléments graphiques, OFL pour les polices).
- **Confidentialité** : données traitées ; un seul cookie, strictement nécessaire ; aucune mesure d'audience ni cookie sur le site public ; hébergement en France ; durées de conservation (empreintes d'adresse et d'IP pour la limitation des demandes de code 1 heure et 10 minutes au plus, journaux du serveur 1 mois, journal d'audit 12 mois) et suppression sur demande (EF-10, EF-11) ; droits et CNIL.
- **Valeurs posées au déploiement** : `[DATE]` et `[VERSION]` dans les maquettes.

### Connexion et chargement

| Écran | Maquette | Stories |
| --- | --- | --- |
| Saisie de l'e-mail, téléphone | `maquettes/identite/Connexion.dc.html` (réglage `etat`) | ENF-01 |
| Saisie de l'e-mail, ordinateur | `maquettes/identite/ConnexionBureau.dc.html` (réglage `etat`) | ENF-01 |
| Saisie du code | `maquettes/identite/Code.dc.html` (réglage `etat`) | ENF-01 |
| Chargement (session valide) | `maquettes/identite/Chargement.dc.html` (réglage `etat`) | ENF-01 |

**Connexion**

- Fond blanc ; en haut le symbole avec cercle, le logotype et l'accroche ; le formulaire en bas (téléphone) ou à droite (ordinateur, symbole et logotype en grand à gauche).
- Même message pour une adresse inconnue, révoquée ou d'une autre tribu : « Si … fait partie de la tribu, un code à 8 chiffres vient d'y être envoyé. »
- États de l'e-mail : saisie ; adresse mal formée (champ cerclé de rouge) ; trop de demandes (bouton inactif en pointillé, « Réessayez dans quelques minutes »).
- États du code : saisie ; code erroné (essais restants) ; essais épuisés et code expiré (seule action : « Recevoir un nouveau code »).
- Messages d'erreur en rouge tomate, avec icône.
- Pied de page : « Melting Tribe · logiciel libre, code source », lien vers la version déployée (ADR 0011, point 2).

**Chargement**

- Affiché au démarrage quand la session est valide, seulement si le chargement dépasse environ 300 ms, pour éviter un flash.
- Fond blanc, symbole au centre qui « respire » (±3 %), trois gouttes de soupe jaunes qui tombent en décalé. Aucun texte ni nom de tribu.
- Réseau lent ou absent : message et bouton « Réessayer » ; le planning en cache reste accessible (ENF-01).
- Fond blanc partout (manifeste, écran de lancement du système, chargement), pour un enchaînement sans saut de couleur. Sur iOS, pas d'images de démarrage.

**Écarts de la réalisation avec les maquettes** (lot C du plan du 2026-10-03, 2026-10-04)

- **Champ du code** : un seul champ (`inputmode="numeric"`, `autocomplete="one-time-code"`, `maxlength="6"`), dessiné en six cases par le CSS, chiffres tabulaires de Bricolage Grotesque. La case en cours de saisie n'a pas son propre liseré : au focus, le champ entier est entouré du liseré de saisie `#E39A89` (D17). Un code collé n'en garde que les chiffres.
- **Renvoi refusé** : « Je n'ai rien reçu : renvoyer un code » refusé pour trop de demandes affiche, sous le champ du code, le message de l'écran de l'e-mail (« Trop de demandes depuis cet appareil… ») ; le code déjà reçu reste utilisable.
- **Trop de demandes**, écran de l'e-mail : le bouton reste inactif jusqu'à ce que l'adresse soit modifiée.
- **Erreur de réseau** sur un formulaire : « Le réseau est lent ou absent. Réessayez. », avec l'icône d'erreur ; le formulaire reste en l'état. Une réponse imprévue du serveur (5xx) est traitée de la même façon.
- **Chargement, réseau lent** : le message et « Réessayer », sans la phrase sur le planning en cache, qui attend le hors-ligne.
- **« Modifier »** est un bouton présenté comme un lien. L'adresse saisie n'est gardée qu'en mémoire : un rechargement de la saisie du code ramène à la saisie de l'e-mail.
- **Accueil provisoire** après connexion, sans maquette : symbole, nom de la tribu, « Se déconnecter » (bouton au contour rouge tomate), pied de page ; il sera remplacé par le planning.
- **URL** : la connexion est servie sous `/tribes/<identifiant>/connexion`, l'accueil sous `/tribes/<identifiant>/`.

### Application

| Écran | Maquette | Stories |
| --- | --- | --- |
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

### Planches de référence de l'identité

Dans `maquettes/identite/` : `Symbole`, `SymboleMini`, `SymboleMono` et `Wordmark` sont les composants ; `Main` (versions du symbole), `Tailles` (petites tailles), `Icones` et `Logotype` les présentent.

## Navigation

- Site public → liens vers les pages légales et le dépôt. Il ne mène pas à l'application, qui est sous `/tribes/<identifiant>/` (ADR 0006).
- Connexion → « Recevoir un code » ouvre la saisie du code → « Se connecter » ouvre le planning. « Se déconnecter » et « Quitter la tribu » ramènent à la connexion.
- Barre d'onglets en bas : **Planning**, **Plats**, **Courses**. Onglet actif en aplat rouge tomate, texte blanc.
- Planning → toucher une case de repas ouvre le détail du repas.
- Planning → la pastille de l'en-tête (nom de la tribu et nombre de membres) ouvre l'écran Tribu. Pas d'onglet dédié : la gestion de la tribu est occasionnelle.
- Plats → « Nouveau » ouvre la création d'un plat.
- Courses → l'onglet ouvre la liste principale. Son nom, en tête, ouvre le choix des listes ; « Ajouter les courses des repas » ouvre le choix de la période et de la liste ; « Sélectionner » fait passer la liste en mode sélection ; « Courses faites » retire les articles cochés, après confirmation.
- Choix des listes → toucher une liste l'ouvre ; le bouton « ⋮ » d'une liste ouvre ses actions (renommer, définir comme principale, supprimer) ; « Nouvelle liste » crée une liste vide.
- Courses des repas → « Ajouter à « … » » revient à la liste.
- Tribu → le réglage **Taille de la tribu** (stepper) fixe les parts proposées pour les plats servis ajoutés et les nouveaux plats (T3).
- Tribu → toucher un membre ouvre sa fiche ; « Ajouter un membre » ouvre l'ajout ; « Mes appareils » ouvre la liste des sessions.

## Composants

### Communs

- **Bouton principal** : aplat rouge tomate, texte blanc 800, hauteur 54–56 px.
- **Bouton secondaire** : contour foncé 2 px, ou texte seul pour « Annuler ».
- **Bouton inactif** : contour en pointillé `#CFC6DE`, texte atténué ; il garde sa place.
- **Action destructive** (révoquer, déconnecter, quitter, supprimer une liste) : texte ou contour rouge tomate, jamais en aplat sur l'écran. L'aplat n'apparaît que dans la feuille de confirmation.
- **Champ de saisie** : contour foncé 2 px, libellé de 12 px au-dessus ; un formulaire ouvert dans la page est entouré du liseré de saisie `#E39A89`.
- **Feuille** : panneau blanc en bas d'écran (coins 26 px) sur un voile. Feuille de confirmation : titre sous forme de question, conséquences en une phrase, action en aplat puis « Annuler ». Feuille de choix : une ligne par option, puis « Annuler » ou « Fermer ».
- **Bandeaux** : une ligne d'icône et de texte dans un liseré. Avertissement en ambre, hors ligne en indigo, confirmation en vert, erreur en rouge tomate sur liseré `#E39A89`.
- **Pastille** : petite pilule de 12 px. Neutre (liseré `#E7E1EE`), en pointillé (« ajouté », « Révoqué »), colorée (« Repas modifiés », « Actif », « cet appareil »), foncée (« principale »).

### Planning et plats

- **Case de repas** : liseré de la couleur du moment (midi ou soir), libellé en capitales, plats en pilules ; case vide en pointillé avec « + Ajouter ».
- **Pilule de plat** : nom tronqué et badge rond du nombre de parts.
- **Stepper de parts** : − / valeur / +, contour foncé, « + » en rouge tomate.
- **Initiale de plat** : carré arrondi 40–44 px (32 px dans le détail d'un article), liseré adouci, lettre dans la teinte soutenue.
- **Ajout d'un plat à un repas** : recherche puis suggestions ; un plat déjà dans le repas n'est pas proposé, on augmente ses parts (Q21).

### Listes de courses

- **Article** (C6) : case à cocher native, nom, quantité en rouge tomate ; coché = barré, liseré vert, déplacé en bas. La case à cocher et la ligne sont deux cibles distinctes.
- **Article calculé** (C5) : pastille neutre « N plats » et chevron ; toucher la ligne la déplie : initiale du plat, plat et parts, jour et moment (couleurs midi/soir), quantité apportée.
- **Article ajouté** (C3) : pastille en pointillé « ajouté », croix pour le retirer ; il ne se déplie pas.
- **Ajout d'un article à la main** (C3) : bouton en pointillé en bas de la liste, qui s'ouvre en formulaire (article, quantité facultative).
- **Pastille « principale »** (C1) : contour foncé, à côté du nom de la liste (en-tête, choix des listes, choix de la destination). Dans le choix des listes, la carte de la liste principale a un liseré foncé.
- **Bandeau de liste périmée** (C8) : sous le nom de la liste, en ambre, avec le bouton « Recalculer » en aplat. La pastille « Repas modifiés » signale une liste périmée dans le choix des listes.
- **Courses faites** (C11) : bouton principal en bas de la liste, inactif tant qu'aucun article n'est coché ; la feuille de confirmation dit combien d'articles partent et combien restent. Un bandeau vert confirme ensuite le retrait.
- **Mode sélection** (C9, C10) : la case à cocher laisse place à une pastille ronde au contour foncé, pleine `#22193A` quand l'article est sélectionné ; la carte sélectionnée prend un liseré foncé de 2 px. L'en-tête devient « Annuler », le nombre d'articles sélectionnés, « Tout ». La barre d'onglets laisse place à deux actions : « Déplacer vers… » (aplat) et « Nouvelle liste » (contour foncé), inactives tant que rien n'est sélectionné. Chacune ouvre une feuille : choix de la liste d'arrivée, ou nom de la nouvelle liste.
- **Actions d'une liste** (C2, C12) : feuille ouverte par « ⋮ ». « Supprimer la liste » agit sans confirmation, pour une liste vide et non principale ; sinon elle est inactive, avec la raison.
- **Calendrier de période** (C4) : grille du mois, lundi en premier ; bornes au contour rouge tomate 2 px, jours intermédiaires au liseré `#E39A89`, aujourd'hui en pointillé, point sous les jours ayant des repas. Date de fin avant la date de début : message d'erreur et bouton inactif.
- **Destination des courses** (C4) : champ au contour foncé « Ajouter à la liste », qui ouvre une feuille de choix ; sous le champ, une ligne indique les plats servis déjà passés par la liste, qui seront ignorés (Q22).

### Tribu, membres et sessions

- **Avatar de membre** : rond 40 px (72 px sur la fiche), liseré adouci et initiale dans la teinte soutenue, comme l'initiale de plat. Membre révoqué : liseré en pointillé, initiale atténuée.
- **Statut de membre** : pastille « Actif » ou « Révoqué ». La carte d'un membre révoqué est en pointillé ; sa fiche indique la date et l'auteur de la révocation (EF-02).
- **Carte de session** : icône d'appareil, nom donné ou appareil détecté, dates ; la session courante porte la pastille verte « cet appareil » et un liseré vert.

## Reste à faire

- Générer les fichiers d'icônes (PNG 192, 512, maskable, Apple 180 ; favicon SVG).
- Réaliser le site public dans `site/` à partir des maquettes ; aujourd'hui, `site/index.html` n'est qu'une page d'attente.
- Créer la boîte `contact@codingmatters.org` avant la publication du site.
