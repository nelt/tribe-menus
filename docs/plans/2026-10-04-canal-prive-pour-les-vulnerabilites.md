# Plan : canal privé pour les vulnérabilités

- **Nom** : `canal-prive`
- **Date** : 2026-10-04
- **Statut** : en cours (avance en parallèle de `socle/C` : il ne touche pas au code de l'application)

## Objectif

Le dépôt est public : ce que les sessions Claude s'écrivent par les PR (revues, constats, correctifs) l'est aussi. C'est voulu pour le travail courant, mais une vulnérabilité ne doit pas être publiée avant que son correctif soit déployé.

À la fin du plan :

- un canal privé existe pour une vulnérabilité, **du constat au correctif déployé**, utilisable par la session cloud, par Claude Code dans le Dev Container et par le développeur ;
- une règle écrite dit ce qui passe par ce canal et ce qui reste public ;
- le canal a été répété de bout en bout sur un faux constat ;
- ce que `SECURITY.md` promet aux tiers est réellement en place.

**Hors périmètre** : la revue de sécurité elle-même (plan `revue-securite`, qui se servira du canal).

## Contexte et contraintes

- À lire avant de commencer : `docs/revue-de-pr.md`, `SECURITY.md`, `docs/securite-depot.md`, ADR 0011, 0018, 0019, 0020.
- **Le correctif révèle la faille autant que le constat** : une PR publique qui corrige une vulnérabilité l'expose jusqu'au déploiement. Le canal doit donc couvrir aussi la préparation du correctif.
- **Le flux public reste la règle.** Revues de conception, bugs, durcissement, défauts d'un code qui n'est déployé nulle part : rien ne change. Le canal privé est l'exception.
- **Qui fait quoi.** Une session cloud prépare et rédige ; le développeur fait les réglages GitHub et élargit les jetons, que les sessions ne peuvent ni ne doivent modifier elles-mêmes ; Claude Code dans le Dev Container fait sa part des essais.
- **Droits au strict nécessaire** (ADR 0019, 0020) : tout droit ajouté à un jeton est justifié dans l'ADR de ce plan.
- **Constat du 2026-10-04**, depuis une session cloud : l'API du dépôt répond que le signalement privé de vulnérabilités n'est **pas activé**, alors que `SECURITY.md` y renvoie. La liste des avis en brouillon revient vide, ce qui ne dit pas si la session y a accès. À confirmer à l'étape 1.

### Canaux envisagés

| Canal | Fonctionnement | Limites connues |
| --- | --- | --- |
| **A. Avis de sécurité GitHub en brouillon** (préféré) | Le constat est écrit dans un avis privé du dépôt ; le correctif est préparé dans le fork privé temporaire associé, puis fusionné dans `main` ; l'avis est publié après le déploiement. | La CI ne tourne pas dans le fork temporaire : `make ci` se lance dans le Dev Container. Accès des sessions à vérifier. |
| **B. Dépôt privé compagnon** (repli) | Un second dépôt privé reçoit les constats et les branches de correctif, avec sa propre CI ; le correctif est poussé sur le dépôt public une fois prêt. | Un dépôt à tenir synchronisé, des jetons à élargir, pas d'analyse CodeQL sans offre payante. |
| **C. Le développeur comme relais** (dernier recours) | Le relecteur donne le constat au développeur dans la conversation, qui le transmet à l'autre session. | Ne couvre pas le correctif, et remet le développeur en position de copiste. |

## Étapes

- [x] **1. Essais d'accès**, avec un avis factice créé puis supprimé par le développeur.
  - Signalement privé de vulnérabilités : activé ou non dans les réglages du dépôt.
  - Session cloud : lire un avis en brouillon, y écrire, lire le fork privé temporaire.
  - Claude Code dans le Dev Container, avec son jeton actuel puis avec le droit sur les avis de sécurité du dépôt : mêmes essais, plus cloner le fork temporaire, y pousser une branche, y ouvrir une PR.
  - Résultat consigné dans les notes d'exécution : ce qui marche, ce qui est refusé, avec quel droit.
- [ ] **2. Choix du canal**, avec le développeur, d'après l'étape 1 : A si les deux sessions y accèdent ; A avec C pour le seul constat si une session n'y accède pas ; B sinon.
- [ ] **3. Règle de tri.** Passe par le canal privé : un défaut exploitable sur une version déployée, en recette comme en production, et tout constat dont on ne sait pas encore s'il l'est. Reste public : tout le reste. En cas de doute, privé : rendre public plus tard est toujours possible, l'inverse non. Préciser qui tranche (le développeur) et sur quel signal le relecteur s'arrête d'écrire en public.
- [ ] **4. Déroulé privé de bout en bout**, écrit pour le canal retenu : constat (même contenu qu'un point « à corriger » de `docs/revue-de-pr.md`) ; correctif et test ; vérification sans la CI du dépôt (`make ci` dans le Dev Container, résultat rapporté dans le canal) ; relecture ; fusion par le développeur ; déploiement ; publication de l'avis et ligne dans `CHANGELOG.md`, qui ne décrit la faille qu'à ce moment.
- [ ] **5. Droits** : jeton du Dev Container élargi par le développeur selon l'étape 2 ; `docs/poste-de-developpement.md` et `docs/securite-depot.md` mis à jour ; signalement privé activé si l'étape 1 confirme qu'il ne l'est pas.
- [ ] **6. Documents** : un ADR (canal retenu, règle de tri, droits ajoutés, alternatives écartées) ; section « Vulnérabilités » dans `docs/revue-de-pr.md` ; `SECURITY.md` accordé à ce qui est réellement en place ; `CLAUDE.md` (une ligne dans les conventions) ; `.claude/settings.json` si une commande `gh` doit être autorisée.
- [ ] **7. Répétition** sur un faux constat sans conséquence (par exemple un commentaire à corriger), de bout en bout : constat par la session cloud, correctif par le Dev Container, vérification, fusion, publication ou suppression de l'avis. Les frottements constatés corrigent le déroulé de l'étape 4.
- [ ] **8. Clôture** : statut « terminé », `docs/feuille-de-route.md` et `CHANGELOG.md` mis à jour.

## Critères de validation

- La répétition de l'étape 7 s'est déroulée sans que rien du faux constat n'apparaisse dans une PR, un commit, une issue ou un commentaire publics avant la fusion.
- Chaque session sait, par `CLAUDE.md` et `docs/revue-de-pr.md`, quand quitter le flux public et où écrire.
- Aucun jeton n'a plus de droits que ce que l'ADR justifie.
- `SECURITY.md` ne promet que ce qui est en place.

## Questions ouvertes

- **Vérification sans CI** : si le canal A est retenu, le correctif n'est vérifié que par `make ci` dans le Dev Container avant la fusion ; la CI du dépôt ne passe qu'après. Suffisant, ou faut-il que le développeur relance `make ci` de son côté ?
- **Historique Git** : le message de commit d'un correctif est public dès la fusion, donc avant le déploiement. Convention à fixer : message neutre à la fusion, détail dans l'avis publié ensuite.
- **Recette** : une vulnérabilité qui ne touche que la recette (données de démonstration) justifie-t-elle le canal privé ? La règle de l'étape 3 dit oui par défaut ; à confirmer.

## Notes d'exécution

- **Étape 1, essais d'accès** (2026-10-04), sur l'avis factice `GHSA-6658-5pv8-wrwf`, en brouillon, créé par le développeur. Son fork privé temporaire est `nelt/tribe-menus-ghsa-6658-5pv8-wrwf` (champ `private_fork` de l'avis).
  - **Signalement privé de vulnérabilités** : désactivé (`gh api repos/nelt/tribe-menus/private-vulnerability-reporting` renvoie `{"enabled":false}`, vu depuis les deux sessions), alors que `SECURITY.md` y renvoie. À activer à l'étape 5.
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
