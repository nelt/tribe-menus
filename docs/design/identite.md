# Identité visuelle : Melting Tribe

- **Date** : 2026-09-28
- **Statut** : retenu (symbole, icônes, logotype, écrans de connexion et de chargement, site public).
- **Canevas** :
  - [Melting Tribe, identité visuelle](https://claude.ai/artifact/VQKHQXpkeWw3DmP6wBaUvg) (privé) : l'exploration, pistes A à D10 ;
  - [Melting Tribe, déclinaisons](https://claude.ai/artifact/PnDbsVYrEjfZondTPHUVe1) (privé) : la version retenue et ses déclinaisons. Sources dans `maquettes/identite/`.

Périmètre : pages publiques (site statique, ADR 0006) et fenêtre de connexion. Le design interne de l'application (`README.md`) ne change pas pour l'instant.

## Nom

**Melting Tribe**, en référence au *melting pot* : la tribu qui se mélange autour de la marmite. Le nom sert de marque sur le site et l'écran de connexion ; le nom d'hôte devient `meltingtribe.codingmatters.org` (ADR 0017) ; le dépôt et les noms techniques restent `tribe-menus`. La question d'un dépôt de marque est reportée.

Le nom de marque apparaît aussi dans le manifeste servi sans session et dans l'e-mail du code : jamais le nom d'une tribu avant connexion (ENF-02, ADR 0006, ADR 0014).

## Symbole

Trois personnages émergent d'une marmite dont la soupe déborde ; au-dessus, des ingrédients s'envolent en arc, de plus en plus gros de droite à gauche (cosse de petits pois, aubergine, part de pizza, tomate, carotte). La tomate touche le cercle, la carotte en sort franchement : ce débordement est une signature du symbole.

- **Personnages** : silhouettes pleines, sans visage ni main ; bras droits levés. Couleurs : ocre `#B36A0B`, bleu `#2F6FB0`, vert `#3F7F25`.
- **Volume** : lumière venant d'en haut à gauche, traitée en aplats (pas de dégradés) : croissant d'ombre dans un ton plus foncé en bas à droite, petit reflet clair en haut à gauche.
- **Marmite** : demi-disque rouge tomate `#C63D24`, bord `#22193A`, soupe et coulures jaunes `#F2C94C`.
- **Cercle** : anneau rouge tomate `#C63D24`, intérieur blanc. Il remplace le disque crème des premières pistes (le crème est abandonné). Il est présent dans toutes les versions ; la carotte passe par-dessus.

| Fichier | Usage |
| --- | --- |
| `identite/symbole.svg` | Version principale, avec cercle. |
| `identite/symbole-sans-cercle.svg` | Quand le cadre est déjà rond ou chargé. |
| `identite/symbole-reduit.svg` | 48 px et moins (favicon, onglets) : sans ombres, carotte et tomate seulement, formes épaissies, anneau deux fois plus épais. |
| `identite/symbole-mono.svg` | Une couleur (encre `#22193A`), formes séparées par un liseré de la couleur du fond (blanc). |
| `identite/symbole-mono-inverse.svg` | Blanc sur fond `#22193A`. |

Sur fond sombre, le symbole garde son intérieur blanc et son anneau rouge.

## Petites tailles

- Au-dessus de 48 px : symbole complet.
- 48 px et moins : symbole réduit. Son anneau épais reste lisible jusqu'à 16 px ; l'anneau fin du symbole complet disparaît vers 32 px.

## Icônes d'application

- **Icône PWA** (192 et 512 px) : symbole complet avec cercle sur fond blanc ; variante sur fond `#22193A` si le blanc se perd.
- **Android « maskable »** : plein cadre blanc, symbole dans la zone sûre (cercle central de 80 %). Les fanes de la carotte sortent de la zone et sont rognées par les lanceurs à masque rond : accepté.
- **Apple** (180 px) : fond blanc, iOS arrondit les coins.
- **Favicon** : symbole réduit, en SVG.
- **Manifeste** : nom générique « Melting Tribe » sans session (ADR 0006, point 9).

## Logotype

- **Lettrage** : « Melting Tribe » en Bricolage Grotesque 800, « Melting » en rouge tomate `#C63D24`, « Tribe » en `#22193A`.
- **Coulée** : une coulée de soupe jaune `#F2C94C` sous « Melting ». Elle reste jaune partout : jamais rouge (effet de coulée de sang).
- **Sur fond sombre** (`#22193A`) : « Melting » en tomate éclaircie `#EE7A5F`, « Tribe » en blanc, coulée jaune.
- **Assemblages** : horizontal (symbole à gauche) et vertical (symbole au-dessus, avec l'accroche « Les menus de la semaine, en tribu »).
- Fichiers : `identite/logotype.svg` (fond clair) et `identite/logotype-sombre.svg` (fond sombre), texte converti en tracés : ils ne dépendent d'aucune police. Obtenus à partir de Bricolage Grotesque (instance `wght` 800, `opsz` 72, `wdth` 100) avec approche de −1,8 px et crénage de la police, comme le rendu du navigateur à 72 px. Source de maquette : `maquettes/identite/Wordmark.dc.html`.

## Écrans de connexion

Remplacent `Doux-Connexion.dc.html` et `Doux-Code.dc.html` (ENF-01). Sources : `maquettes/identite/Connexion.dc.html`, `ConnexionBureau.dc.html`, `Code.dc.html`.

- Fond blanc ; en haut le symbole avec cercle, le logotype et l'accroche ; le formulaire en bas (téléphone) ou à droite (ordinateur, symbole et logotype en grand à gauche).
- Jamais le nom de la tribu (ENF-02) ; même message pour une adresse inconnue, révoquée ou d'une autre tribu : « Si … fait partie de la tribu, un code à 6 chiffres vient d'y être envoyé. »
- États (réglage `etat`) :
  - e-mail : saisie ; adresse mal formée (champ cerclé de rouge) ; trop de demandes (bouton désactivé en pointillés, « Réessayez dans quelques minutes ») ;
  - code : saisie ; code erroné (essais restants) ; essais épuisés et code expiré (seule action : « Recevoir un nouveau code »).
- Messages d'erreur en rouge tomate `#C63D24`, avec icône, annoncés aux lecteurs d'écran (`role="alert"`).
- Pied de page : « Melting Tribe · logiciel libre, code source », lien vers la version déployée (ADR 0011, point 2).

## Écran de chargement

Affiché au démarrage quand la session est valide, le temps que l'application démarre. Source : `maquettes/identite/Chargement.dc.html`.

- Fond blanc, symbole au centre qui « respire » (±3 %), trois gouttes de soupe jaunes qui tombent en décalé. Aucun texte ni nom de tribu.
- N'apparaît que si le chargement dépasse environ 300 ms, pour éviter un flash.
- Animations coupées avec `prefers-reduced-motion`.
- Réseau lent ou absent : message et bouton « Réessayer » ; le planning en cache reste accessible (ENF-01, consultation hors ligne).
- Fond blanc partout (`background_color` du manifeste, écran de lancement du système, chargement) pour un enchaînement sans saut de couleur. Sur iOS, pas d'images de démarrage : l'écran blanc du système suffit.

## Site public

Servi statiquement à la racine de `meltingtribe.codingmatters.org` (ADR 0006, ADR 0017), sources à venir dans `site/`. Maquettes : `maquettes/identite/Site.dc.html` (ordinateur), `SiteMobile.dc.html` (téléphone), `Mentions.dc.html`, `Confidentialite.dc.html` ; le texte des maquettes fait référence.

- **Ton** : vouvoiement ; neutre (pas d'histoire de l'auteur) ; pas d'aperçus d'écrans, seulement le symbole.
- **Présentation** : titre « Les menus de la semaine, en tribu. », sous-titre, boutons « Comment ça marche » et « Voir le code source », phrase d'accès (« sur invitation d'un membre d'une tribu », pas d'inscription) ; deux messages mis en avant, les courses calculées et la semaine construite à plusieurs, dans des cartes aux liserés midi et soir ; une ligne « Fonctionne hors ligne · Vos données restent à votre tribu » ; bandeau « Un logiciel libre » sur fond `#22193A` avec lien vers le dépôt.
- **Pied de page** (toutes les pages) : symbole, « logiciel libre sous licence AGPL-3.0 », liens Mentions légales, Confidentialité et « Code source de cette version » pointant vers l'étiquette déployée (ADR 0011, point 2).
- **Mentions légales** : éditeur Nel Taurisson, à titre personnel ; contact `contact@codingmatters.org` ; hébergeur OVH SAS ; licences (AGPL pour le code, la documentation et les éléments graphiques, OFL pour les polices).
- **Confidentialité** : données traitées, un seul cookie strictement nécessaire, aucune mesure d'audience ni cookie sur le site public, hébergement en France, durées de conservation, droits et CNIL.
- **À compléter avant publication** : `[DATE]` et `[VERSION]` (au déploiement), durée de conservation des journaux du serveur (`[DURÉE]`), modalité de suppression des données d'une tribu sur demande ; créer la boîte `contact@codingmatters.org`.

## Image de partage

`identite/partage.png` (1200 × 630, source `identite/partage.svg`), pour les balises Open Graph et Twitter du site public et l'aperçu social du dépôt GitHub : symbole avec cercle à gauche, logotype, accroche « Les menus de la semaine, en tribu. » et adresse `meltingtribe.codingmatters.org` à droite, sur fond blanc. Textes en tracés.

## Couleurs ajoutées

En plus des tokens de `README.md` :

| Rôle | Valeur |
| --- | --- |
| Soupe, coulée | `#F2C94C` |
| Carotte | `#E07B1F` |
| Personnage ocre (lumière / ombre) | `#B36A0B` (`#D9953A` / `#8A5008`) |
| Personnage bleu (lumière / ombre) | `#2F6FB0` (`#5C93CC` / `#1F4E80`) |
| Personnage vert (lumière / ombre) | `#3F7F25` (`#67A24A` / `#2B5A18`) |
| Marmite (lumière / ombre) | `#C63D24` (`#E0654C` / `#952B17`) |
| « Melting » sur fond sombre | `#EE7A5F` |

Sur les pages publiques, l'aplat de couleur n'est plus réservé aux actions : le symbole et les grands blocs (bandeau `#22193A` du site) en portent.

## Reste à faire

- Fichiers d'icônes (PNG 192, 512, maskable, Apple 180 ; favicon SVG), à générer à l'initialisation du projet.
- Supprimer `Doux-Connexion.dc.html`, `Doux-Code.dc.html` et `FondTribu.dc.html` une fois les nouveaux écrans implémentés.
- Licence : le symbole et le logotype sont couverts, comme tout le dépôt, par l'AGPL (ADR 0011). Protéger le nom et le logo (usage de la marque par des versions modifiées) demanderait une décision dédiée, non prise à ce jour.
