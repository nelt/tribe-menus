# Traitement des vulnérabilités

Ce qu'on fait d'une vulnérabilité, du constat à sa publication. Le dépôt est public : un constat écrit dans une PR, comme le correctif lui-même, expose la faille tant que la version corrigée n'est pas en production. Il n'y a pas de canal privé sur GitHub (ADR 0022) : **une analyse de risques décide, pour chaque vulnérabilité, du traitement proportionné**.

Mis en place le 2026-10-05 par le plan `canal-prive`.

## Quand ce document s'applique

À tout constat qui pourrait être une vulnérabilité d'une version **en production** : un défaut qu'un tiers peut exploiter, ou dont on ne sait pas encore s'il le peut.

Il ne s'applique pas, et le flux public habituel suffit (`docs/revue-de-pr.md`), pour :

- un défaut d'un code qui n'est pas en production : PR en cours, code fusionné mais pas encore publié, recette seule (données de démonstration) ;
- un bug sans scénario d'attaque, un durcissement, une revue de conception.

Tant qu'aucune version n'est en production, tout relève donc du flux public. Les trois évaluations ci-dessous restent utiles pour classer un constat de sécurité.

## Rôles

- **Le relecteur** : la session ou la personne qui trouve le défaut. Il propose l'analyse.
- **Le développeur** : il décide du niveau, porte le constat à l'auteur, fusionne et déploie.
- **L'auteur** : Claude Code dans le Dev Container, qui écrit le correctif.
- **Un tiers** : il signale par le signalement privé de GitHub (`SECURITY.md`), que seul le développeur lit. Le développeur tient alors le rôle du relecteur.

## Le signal d'arrêt

Une session s'arrête d'écrire sur GitHub **dès qu'elle soupçonne** une vulnérabilité d'une version en production, avant de savoir si elle est exploitable. Elle n'écrit le constat ni dans une PR, ni dans un commentaire, ni dans un commit, ni dans un plan.

Elle remet le constat au développeur **hors de GitHub** : dans la conversation, pour une session cloud comme pour une session du Dev Container. En cas de doute, elle s'arrête : rendre public plus tard est toujours possible, l'inverse non.

Si la revue en cours porte sur une PR, son commentaire « Revue » ne mentionne pas ce point ; les autres points suivent leur cours.

## Le constat remis au développeur

```markdown
## Constat de sécurité

**Où** : `chemin/du/fichier.go`, fonction ; version concernée.

**Scénario d'attaque** : qui, avec quels prérequis, fait quoi, en une ou deux phrases.

**Conséquence** : ce que l'attaquant obtient.

**Comment il a été constaté** : lecture, test, reproduction ; ce qui n'a pas pu être vérifié.

**Évaluations**
1. Gravité : faible | moyenne | élevée, parce que …
2. Risque de livrer sans recette : faible | moyen | élevé, parce que …
3. Risque d'exploitation une fois publiée : faible | moyen | élevé, parce que …

**Niveau proposé** : normal | accéléré | urgent. Mesure d'attente possible : …
```

## Les trois évaluations

| Axe | Ce qu'on regarde | Faible | Élevé |
| --- | --- | --- | --- |
| **1. Gravité de la faille** | Ce que l'attaquant obtient, et ce qu'il lui faut pour y arriver | gêne mineure, ou accès au serveur requis | données d'une autre tribu ou session d'un membre, à distance et sans compte |
| **2. Risque de livrer sans recette** | Étendue du correctif, couverture par les tests automatiques, réversibilité, et ce que `main` contient de non publié | quelques lignes, couvertes par un test, retour arrière automatique, rien d'autre en attente | authentification ou migration de base touchées, retour arrière incertain, ou stories non publiées qui partiraient avec |
| **3. Risque d'exploitation une fois publiée** | Lisibilité de la faille dans le diff, compétence requise, durée d'exposition | faille difficile à déduire du correctif | exploitation évidente à la lecture du diff |

« Moyen » est tout ce qui n'est franchement ni l'un ni l'autre.

**L'axe 2 compte ce qui attend sur `main`.** Seule une release étiquetée va en production (ADR 0012, 0016) : une version corrective emporte tout ce qui a été fusionné depuis la dernière release. Des stories non publiées, jamais passées en recette, élèvent le risque de livrer sans elle.

## Les trois niveaux

| Niveau | Constat | PR du correctif | Relecture du correctif | Recette | Exposition publique |
| --- | --- | --- | --- | --- | --- |
| **Normal** | écrit dans la PR ou le plan | flux habituel | `docs/revue-de-pr.md` | complète | sans contrainte |
| **Accéléré** | hors de GitHub jusqu'au déploiement | ouverte quand le correctif est prêt, en termes neutres | sur la PR, en termes neutres | ciblée sur le correctif | une séance de travail |
| **Urgent** | hors de GitHub jusqu'au déploiement | ouverte le temps de livrer, en termes neutres | avant le push, hors de GitHub | aucune avant la production, complète après | le temps du build et du déploiement |

## Règle de décision

- **Gravité faible** : normal, quels que soient les deux autres axes.
- **Gravité élevée et exploitation probable** (axe 3 élevé) : urgent si livrer sans recette est peu risqué (axe 2 faible) ; sinon accéléré, avec une mesure d'attente si elle existe.
- **Tous les autres cas** : accéléré.

Le relecteur propose, **le développeur décide**. Il peut changer une évaluation, et le niveau avec elle.

### Exemples

Hypothétiques, pour donner l'échelle ; aucun ne décrit un défaut constaté.

| Constat | Gravité | Sans recette | Une fois publiée | Niveau |
| --- | --- | --- | --- | --- |
| Un en-tête de sécurité manque sur une réponse, sans scénario d'attaque connu | faible | faible | faible | normal |
| La limite des demandes de code se contourne par une variante d'écriture de l'adresse : la force brute du code est facilitée, mais reste longue | moyenne | faible | moyen | accéléré |
| Le cookie de session d'une tribu est accepté par une autre : un membre lit les données d'une tribu qui n'est pas la sienne ; correctif de quelques lignes, couvert par les scénarios ENF-02 | élevée | faible | élevé | urgent |
| Même défaut, mais le correctif demande une migration des sessions | élevée | élevé | élevé | accéléré, application coupée le temps du correctif |

## Déroulé

### Normal

Le flux habituel : le constat est écrit, le correctif passe par une PR relue selon `docs/revue-de-pr.md`, et part avec la release suivante.

### Accéléré

1. **Le relecteur** remet le constat au développeur, hors de GitHub.
2. **Le développeur** décide du niveau et, s'il y a lieu, applique une mesure d'attente.
3. **Le développeur** porte le constat à l'auteur, dans sa session du Dev Container.
4. **L'auteur** écrit, sur une branche locale, le test qui échoue puis le correctif ; `make ci` ; **il ne pousse pas**. Nom de branche, messages de commit et commentaires du code sont neutres (voir plus bas).
5. **Le développeur** ouvre la séance de livraison quand il a le temps de l'enchaîner jusqu'au bout. L'auteur prépare la version corrective dans la même branche (`RELEASING.md`), pousse et ouvre la PR, en termes neutres.
6. **Le relecteur** relit la PR selon `docs/revue-de-pr.md`, sans décrire la faille : son commentaire dit si le correctif est conforme à ce qui était convenu, et ce qui touche à la faille est dit au développeur, hors de GitHub.
7. **Le développeur** déploie l'archive de la PR en recette et déroule les scénarios touchés par le correctif, pas la recette complète.
8. **Le développeur** fusionne, pose l'étiquette, attend la release et déploie en production (ADR 0012, 0016). Il vérifie que la faille n'est plus exploitable.
9. **Publication** (voir plus bas).

### Urgent

Les étapes 1 à 4 sont celles du niveau accéléré. Ensuite :

5. **Relecture avant le push**, hors de GitHub : par une seconde session Claude Code du Dev Container, qui n'a pas vu le correctif s'écrire et lit le diff local, ou par le développeur. Les corrections se font avant le push.
6. **Le développeur** ouvre la séance de livraison. L'auteur prépare la version corrective, pousse et ouvre la PR, en termes neutres.
7. **Le développeur** fusionne dès que la CI est verte, pose l'étiquette, attend la release et déploie en production. Le retour arrière automatique (ADR 0016) est le filet. Il vérifie que la faille n'est plus exploitable.
8. **Recette après coup** : la même version est déployée en recette et la recette complète est déroulée. Un défaut trouvé là se traite comme un bug ordinaire, ou par un retour arrière.
9. **Publication** (voir plus bas).

### Écrire en termes neutres

Tant que la version corrigée n'est pas en production, rien de ce qui est poussé ne décrit la faille :

- **nom de branche, titre de PR, messages de commit** : ce que le code fait désormais (« Vérifie la tribu du jeton de session »), jamais ce qu'il empêchait ni les mots « faille », « vulnérabilité » ou « sécurité » ;
- **description de la PR** : le changement, les tests ajoutés, sans scénario d'attaque ;
- **ligne de `CHANGELOG.md`** : le changement, dans les mêmes termes ;
- **test** : il prouve le comportement attendu ; son nom et ses commentaires ne racontent pas l'attaque.

Le diff reste lisible par qui sait lire : la neutralité ne cache pas la faille, elle évite de la signaler. C'est la durée d'exposition qui protège.

### Mesures d'attente

Ce qui réduit le risque pendant que le correctif se prépare, à la main du développeur :

- arrêter l'application en production (`systemctl stop`, ADR 0015) : acceptable pour une application familiale, et toujours possible ;
- revenir à une version antérieure qui n'a pas le défaut (`tribe-menus-deploy --rollback`, ADR 0016).

Le plan `recette` documentera les commandes exactes, et ce qui peut être coupé sans tout arrêter.

## Publication

Une fois la version corrigée en production, et pas avant :

1. **Le relecteur**, à la demande du développeur, poste sur la PR du correctif un commentaire « Analyse » : le constat tel qu'il a été remis, les évaluations, le niveau décidé et pourquoi, la chronologie (constat, PR, déploiement).
2. **La ligne de `CHANGELOG.md`** de la version est complétée par une PR ordinaire : ce que la faille permettait, et le numéro de la PR du correctif.
3. **Le signalement d'un tiers** est clos dans GitHub, avec la mention de la version corrigée ; le tiers est cité s'il le souhaite (`SECURITY.md`).

## Phrases utiles

Au relecteur, quand il s'arrête :

```
Rédige le constat de sécurité selon docs/traitement-des-vulnerabilites.md, ici, sans rien écrire sur GitHub.
```

À l'auteur :

```
Traite ce constat de sécurité au niveau <accéléré|urgent> selon docs/traitement-des-vulnerabilites.md. Ne pousse rien avant que je te le dise.

<constat>
```

À la seconde session du Dev Container, au niveau urgent :

```
Relis le correctif de la branche locale <branche> contre ce constat, selon docs/traitement-des-vulnerabilites.md (niveau urgent). Réponds ici, n'écris rien sur GitHub.

<constat>
```

Au relecteur, après le déploiement :

```
La version <vX.Y.Z> est en production. Publie l'analyse sur la PR #<n> selon docs/traitement-des-vulnerabilites.md.
```
