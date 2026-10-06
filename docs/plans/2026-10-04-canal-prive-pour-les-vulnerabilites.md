# Plan : traitement des vulnérabilités (canal privé écarté)

- **Nom** : `canal-prive` (fixé à la création : le nom reste, bien que le plan n'installe plus de canal privé)
- **Date** : 2026-10-04
- **Statut** : terminé le 2026-10-06 (il ne touche pas au code de l'application ; la répétition d'un traitement accéléré est inscrite au plan `recette`)

## Objectif

Le dépôt est public : ce que les sessions Claude s'écrivent par les PR (revues, constats, correctifs) l'est aussi. C'est voulu pour le travail courant, mais une vulnérabilité d'une version en production ne doit pas rester publique et non corrigée plus longtemps que nécessaire.

La première version de ce plan cherchait un canal privé couvrant une vulnérabilité du constat au correctif déployé. Les essais de l'étape 1 et l'examen des variantes ont conduit à l'écarter (D1, D2) : chaque canal ajoute un circuit parallèle complet pour un risque qui se résume au délai entre un correctif public et son déploiement.

À la place, **une analyse de risques décide du traitement de chaque vulnérabilité**, du flux normal de développement à une livraison d'urgence sans recette préalable.

À la fin du plan :

- la grille d'analyse, les trois niveaux de traitement et le déroulé de chacun sont écrits dans `docs/` ;
- chaque session sait quand s'arrêter d'écrire en public, et à qui remettre un constat ;
- ce que `SECURITY.md` promet aux tiers est réellement en place ;
- aucun jeton ne garde un droit ajouté pour les essais ;
- les plans `revue-securite` et `recette` savent ce que ce traitement leur demande.

**Hors périmètre** : la revue de sécurité elle-même (plan `revue-securite`) ; les scripts de déploiement (plan `recette`).

## Contexte et contraintes

- À lire avant de commencer : `docs/revue-de-pr.md`, `SECURITY.md`, `docs/securite-depot.md`, ADR 0011, 0012, 0016, 0019, 0020.
- **Le correctif révèle la faille autant que le constat** : une PR publique qui corrige une vulnérabilité l'expose jusqu'au déploiement. La recette est alimentée par les archives de PR (ADR 0016) : dans le flux normal, la durée de la recette fait donc partie de cette exposition.
- **Le flux public reste la règle.** Revues de conception, bugs, durcissement, défauts d'un code qui n'est pas en production : rien ne change.
- **Le build reste en CI** (ADR 0012) et la PR publique existe toujours : ce qui varie d'un niveau à l'autre, c'est le moment où elle apparaît et ce qui se passe entre son ouverture et le déploiement.
- **Qui fait quoi.** Une session cloud prépare et rédige ; le développeur fait les réglages GitHub et gère les jetons, que les sessions ne peuvent ni ne doivent modifier elles-mêmes ; Claude Code dans le Dev Container écrit les correctifs.
- **Droits au strict nécessaire** (ADR 0019, 0020).
- **Rien n'est en production à ce jour** : tant que c'est le cas, toute vulnérabilité relève du traitement normal (D8).

## Décisions

Prises le 2026-10-04 avec le développeur.

- **D1. Pas de canal privé sur GitHub.**
  - *Avis de sécurité en brouillon et fork privé temporaire* : la session cloud n'accède ni à l'avis ni au fork ; le Dev Container accède à l'avis avec un droit supplémentaire, pas au fork (notes d'exécution, étape 1). La préparation privée du correctif n'est pas possible.
  - *Dépôt privé compagnon* : un miroir à synchroniser, des jetons à étendre, pas de CodeQL, et la fenêtre entre fusion et déploiement reste entière, sauf à déployer depuis le dépôt privé, ce qui demande une chaîne de release parallèle.
  - *Le développeur comme seul relais* : ne couvre pas le correctif. Le relais reste, mais comme premier pas de l'analyse (D6).
- **D2. Variantes écartées** : un build local réservé aux correctifs de sécurité (entorse à l'ADR 0012) ; un développement en dépôt privé avec un miroir public à chaque release (on perd le développement à ciel ouvert).
- **D3. Trois évaluations** pour chaque vulnérabilité :

  | Axe | Ce qu'on regarde | Faible | Élevé |
  | --- | --- | --- | --- |
  | **1. Gravité de la faille** | Ce que l'attaquant obtient, et ce qu'il lui faut pour y arriver | gêne mineure, ou accès au serveur requis | données d'une autre tribu ou session d'un membre, à distance et sans compte |
  | **2. Risque de livrer sans recette** | Étendue du correctif, couverture par les tests automatiques, réversibilité | quelques lignes, couvertes par un test, retour arrière automatique | authentification ou migration de base touchées, retour arrière incertain |
  | **3. Risque d'exploitation une fois la faille publiée dans une PR** | Lisibilité de la faille dans le diff, compétence requise, durée d'exposition | faille difficile à déduire du correctif | exploitation évidente à la lecture du diff |

- **D4. Trois niveaux de traitement** :

  | Niveau | Constat | PR | Recette | Durée d'exposition publique |
  | --- | --- | --- | --- | --- |
  | **Normal** | écrit dans la PR ou le plan | flux habituel | complète | sans contrainte |
  | **Accéléré** | hors de GitHub jusqu'au déploiement | ouverte quand le correctif est prêt, titre et messages neutres | ciblée sur le correctif | une séance de travail |
  | **Urgent** | hors de GitHub jusqu'au déploiement | ouverte le temps de livrer | aucune avant la production, faite après | le temps du build et du déploiement |

- **D5. Règle de décision.**
  - Gravité faible : traitement normal, quels que soient les deux autres axes.
  - Gravité élevée et exploitation probable : urgent si livrer sans recette est peu risqué ; sinon accéléré, avec une mesure d'attente si elle existe (couper la fonction touchée, par exemple).
  - Tous les autres cas : accéléré.
- **D6. L'analyse précède toute écriture publique.** Un constat de sécurité arrive d'abord hors de GitHub : dans la conversation avec le développeur si le relecteur est une session cloud, directement dans le Dev Container sinon. Le développeur le porte à la session qui corrige.
- **D7. Le relecteur propose les trois évaluations, le développeur décide du niveau.**
- **D8. Ce qui ne touche pas la production relève du traitement normal** : une faille de la seule recette (données de démonstration), ou d'un code qui n'est pas encore déployé.
- **D9. L'analyse est publiée après le déploiement**, avec le constat, pour garder la trace de la décision ; `CHANGELOG.md` ne décrit la faille qu'à ce moment.

Le 2026-10-06, avec le développeur, après le rodage de la grille par `revue-securite` (étape 7).

- **D10. Une faille qui se déduit d'un document déjà public se traite sans les précautions de neutralité.** Elle est exposée avant tout correctif (constat 1 de `revue-securite`, lisible dans ENF-01) : les précautions de neutralité ne protègent rien et ralentissent. Le niveau reste celui que donne la gravité ; seul le délai compte. Le signal d'arrêt ne change pas : la session remet d'abord son constat au développeur, et c'est lui qui constate que la faille est déjà publique. Une phrase l'écrit dans `docs/traitement-des-vulnerabilites.md`, sous les trois évaluations ; ni les niveaux ni la règle de décision ne changent.
- **D11. La grille ne change pas pour une fonction manquante** (constat 5 de `revue-securite`, fermeture d'une session à distance). Ce n'est pas un défaut à corriger mais un sujet de la feuille de route : il s'y ordonne, sans passer par les trois évaluations.

## Étapes

- [x] **1. Essais d'accès** aux avis de sécurité en brouillon et au fork privé temporaire, avec un avis factice créé par le développeur (notes d'exécution).
- [x] **2. Choix du traitement**, avec le développeur : canaux écartés, analyse de risques retenue (D1 à D9).
- [x] **3. `docs/traitement-des-vulnerabilites.md`**, sur le modèle de `docs/revue-de-pr.md` :
  - la grille (D3), les niveaux (D4), la règle (D5), avec un ou deux exemples tirés de ce code ;
  - le signal d'arrêt : ce qui fait qu'un relecteur cesse d'écrire en public, et ce qu'il remet au développeur (fichier et fonction, scénario d'attaque, conséquence, les trois évaluations, le niveau proposé) ;
  - le déroulé de chaque niveau, pas à pas : qui écrit le correctif, où il vit avant la PR, qui le relit et comment, quand la PR s'ouvre, ce que disent son titre et ses messages de commit, ce qui est fait en recette, quand l'analyse est publiée ;
  - le modèle de l'analyse publiée (D9) et les phrases utiles à donner aux sessions.
- [x] **4. ADR 0022** : traitement des vulnérabilités par analyse de risques ; alternatives écartées (D1, D2).
- [x] **5. Documents accordés** :
  - `docs/revue-de-pr.md` : une section « Vulnérabilités » qui renvoie au signal d'arrêt ;
  - `SECURITY.md` : ce qui est réellement en place pour un tiers ;
  - `CLAUDE.md` : une ligne dans les conventions ;
  - `docs/securite-depot.md` et `docs/poste-de-developpement.md` : état des réglages et du jeton après l'étape 6.
- [x] **6. Réglages, par le développeur** :
  - activer le signalement privé de vulnérabilités, auquel `SECURITY.md` renvoie ;
  - retirer du jeton du Dev Container le droit *Repository security advisories* ajouté pour les essais ;
  - fermer l'avis factice `GHSA-6658-5pv8-wrwf` et supprimer son fork temporaire.
- [x] **7. Mise à l'épreuve.**
  - La grille est appliquée à chaque constat de `revue-securite`, même si tous relèvent du traitement normal (D8) : c'est le rodage de l'analyse, et ce qui frotte corrige le document de l'étape 3.
  - Un traitement accéléré ne peut être répété que sur une version déployée : la répétition, sur un faux constat, est inscrite au plan `recette`.
  - Fait le 2026-10-05 par le plan `revue-securite` : dix des onze constats portent leurs trois évaluations et un niveau (trois accélérés, sept normaux) ; le onzième, la liste de ce qui revient à Caddy, n'est pas un défaut. Trois remarques en sortent (« Retour sur la grille » de ce plan). La première est tranchée : « accès au serveur requis » reste une gravité faible (`revue-securite`, D6). Les deux autres le sont le 2026-10-06 : une faille lisible dans un document public se traite sans les précautions de neutralité (D10) ; la grille ne change pas pour une fonction manquante (D11).
- [x] **8. Clôture** : statut « terminé », `docs/feuille-de-route.md` et `CHANGELOG.md` mis à jour.
  - Fait le 2026-10-06 : critères de validation relus (notes d'exécution).

## Ce que ce plan demande aux autres plans

- **`revue-securite`** : les constats suivent le flux public, chacun avec ses trois évaluations (étape 7).
- **`production` et `recette`** :
  - un déploiement en production assez court pour tenir dans une séance, de la fusion à la vérification ;
  - une recette ciblée possible : déployer en recette l'archive d'une PR et ne dérouler que les scénarios touchés ;
  - un retour arrière automatique fiable (ADR 0016), puisque le niveau urgent s'appuie dessus ;
  - une mesure d'attente documentée : comment couper l'application ou une fonction le temps d'un correctif ;
  - la répétition d'un traitement accéléré sur un faux constat.

## Critères de validation

- Une session qui lit `CLAUDE.md` et `docs/revue-de-pr.md` sait quand quitter le flux public et à qui remettre son constat.
- Chaque niveau a un déroulé écrit que le développeur peut suivre sans le reconstruire.
- Chaque constat de `revue-securite` porte ses trois évaluations.
- Aucun jeton n'a plus de droits que ce que les ADR justifient.
- `SECURITY.md` ne promet que ce qui est en place.

## Questions ouvertes

- **Avis de sécurité publié** : en plus de `CHANGELOG.md` et de l'analyse sur la PR, publier un avis GitHub après le déploiement ? Utile s'il existe un jour d'autres instances que la nôtre (ADR 0022, conséquences).
- **Durée réelle d'une séance de livraison** : inconnue tant que la chaîne de release et le déploiement n'existent pas. Si elle dépasse l'heure, les niveaux accéléré et urgent se rapprochent ; à mesurer à la répétition du plan `recette`.

## Notes d'exécution

- **Étape 1, essais d'accès** (2026-10-04), sur l'avis factice `GHSA-6658-5pv8-wrwf`, en brouillon, créé par le développeur. Son fork privé temporaire est `nelt/tribe-menus-ghsa-6658-5pv8-wrwf` (champ `private_fork` de l'avis).
  - **Signalement privé de vulnérabilités** : désactivé (`gh api repos/nelt/tribe-menus/private-vulnerability-reporting` renvoie `{"enabled":false}`, vu depuis les deux sessions), alors que `SECURITY.md` y renvoie. À activer à l'étape 6.
  - **Session cloud** :

    | Essai | Résultat |
    | --- | --- |
    | Lire l'avis | refusé : 403 « Resource not accessible by integration » |
    | Lister les avis en brouillon | liste vide sans erreur : l'avis lui est caché |
    | Écrire dans l'avis | non tenté : écriture bloquée par les garde-fous de la session (la lecture étant refusée, l'écriture l'aurait été aussi) |
    | Lire le fork temporaire | refusé : dépôt introuvable ou inaccessible |

    Conséquence pour l'étape 2 : la session cloud ne peut ni écrire le constat ni relire le correctif par le canal A.
  - **Dev Container, passe 1, jeton actuel** (jeton à portée fine des ADR 0019 et 0020) :

    | Essai | Commande | Résultat |
    | --- | --- | --- |
    | 1. Lire l'avis | `gh api repos/nelt/tribe-menus/security-advisories/GHSA-6658-5pv8-wrwf` | refusé : 403 « Resource not accessible by personal access token » |
    | 2. Lister les avis en brouillon | `gh api "repos/nelt/tribe-menus/security-advisories?state=draft"` | liste vide sans erreur : l'avis est caché |
    | 3. Écrire dans l'avis | `gh api -X PATCH …/GHSA-6658-5pv8-wrwf -f state=draft` (sans effet sur le contenu, la description n'ayant pu être lue) | refusé : 403 « Resource not accessible by personal access token » |
    | 4. Lire le fork par l'API | `gh api repos/nelt/tribe-menus-ghsa-6658-5pv8-wrwf` | refusé : 404 « Not Found » |
    | 5. Cloner le fork | `git clone https://github.com/nelt/tribe-menus-ghsa-6658-5pv8-wrwf.git` | refusé : « remote: Write access to repository not granted. », HTTP 403 |
    | 6. Pousser une branche | `git push <fork> feature/essai-canal-prive` (commit `-s` dans un dépôt local, faute de clone) | refusé : même message que l'essai 5 |
    | 7. Ouvrir une PR | `gh api -X POST repos/nelt/tribe-menus-ghsa-6658-5pv8-wrwf/pulls …` | refusé : 404 « Not Found » |

  - **Dev Container, passe 2, jeton élargi** (droit *Repository security advisories* en lecture et écriture ajouté) :

    | Essai | Commande | Résultat |
    | --- | --- | --- |
    | 1. Lire l'avis | comme en passe 1 | réussi : marqueur de lecture `canal-prive-essai-1` lu |
    | 2. Lister les avis en brouillon | comme en passe 1 | réussi : l'avis apparaît |
    | 3. Écrire dans l'avis | `gh api -X PATCH …/GHSA-6658-5pv8-wrwf --input <description lue + ligne ajoutée>` | réussi : « Marqueur d'écriture : dev-container-passe-2 » ajouté en fin de description, le reste conservé |
    | 4. Lire le fork par l'API | comme en passe 1 ; aussi `gh repo view` | refusé : 404 « Not Found » ; `gh repo view` : « Could not resolve to a Repository » |
    | 5. Cloner le fork | comme en passe 1 | refusé : « remote: Write access to repository not granted. », HTTP 403 |
    | 6. Pousser une branche | comme en passe 1 | refusé : même message |
    | 7. Ouvrir une PR | comme en passe 1 | refusé : 404 « Not Found » |

  - **Droit qui manque pour le fork** : les messages n'en nomment aucun. Le fork existe (l'avis le désigne), mais le jeton ne le voit pas du tout : 404 par l'API, et un refus de Git qui parle d'écriture même pour un clone. Hypothèse à vérifier par le développeur, non confirmée par les messages : le fork, créé après le jeton, est hors de la liste des dépôts auxquels un jeton à portée fine donne accès.
  - **Bilan pour l'étape 2** : avec le droit sur les avis, le Dev Container lit et écrit le constat. Ni l'une ni l'autre session n'accède au fork temporaire, donc la préparation privée du correctif n'est pas acquise par le canal A.
- **Étape 2, choix du traitement** (2026-10-04), avec le développeur. Les trois canaux envisagés sont écartés, ainsi que deux variantes examinées en séance ; les raisons sont dans les décisions D1 et D2. Le plan est réécrit autour d'une analyse de risques (D3 à D9) : les étapes 3 à 8 remplacent celles de la première version (règle de tri, déroulé privé, droits, documents, répétition, clôture).
- **Étape 6, réglages** (2026-10-05), faits par le développeur avant les étapes 3 à 5. Signalement privé de vulnérabilités activé : l'API du dépôt répond `{"enabled":true}`, vérifié depuis une session cloud. Avis factice fermé et droit *Repository security advisories* retiré du jeton du Dev Container : déclarés par le développeur, non vérifiables depuis une session.
- **Étapes 3 à 5, documents** (2026-10-05). Questions ouvertes tranchées avec le développeur :
  - *relecture du correctif* : sur la PR en termes neutres au niveau accéléré, avant le push et hors de GitHub au niveau urgent ;
  - *signalement d'un tiers* : lu par le seul développeur, le droit sur les avis de sécurité est retiré du jeton du Dev Container.
  - `docs/poste-de-developpement.md` n'est pas modifié : le jeton du Dev Container a retrouvé les droits qui y sont décrits.
- **Constat fait en écrivant le déroulé** : seule une release étiquetée va en production (ADR 0012, 0016), donc une version corrective emporte tout ce que `main` contient de non publié. L'axe 2 de la grille (risque de livrer sans recette) en tient compte, et le plan `production` doit décrire dans `RELEASING.md` une version corrective préparée dans la PR du correctif.
- **Étape 8, clôture** (2026-10-06) : critères de validation relus.
  - **Signal d'arrêt et déroulés** : écrits dans `CLAUDE.md`, `docs/revue-de-pr.md` et `docs/traitement-des-vulnerabilites.md` ; aucun n'a encore servi, rien n'étant en production.
  - **Constats de `revue-securite`** : dix portent leurs trois évaluations, le onzième n'est pas un défaut.
  - **Droits des jetons** : le jeton du Dev Container a les droits que décrivent les ADR 0019 et 0020, sans les avis de sécurité. L'application GitHub de Claude, elle, a des permissions plus larges que ce que l'ADR 0020 décrit (écriture des workflows et des hooks), fixées par son éditeur : c'est écrit dans `docs/securite-depot.md` depuis le 2026-10-05, avec ce qui en limite la portée. Le critère est tenu pour ce qui se règle, et l'écart est documenté pour le reste.
  - **`SECURITY.md`** : ne promet que le signalement privé, activé, et un correctif par une PR publique.
