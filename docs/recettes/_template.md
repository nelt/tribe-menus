# Plan de recette : vX.Y.Z

- **Version** : `vX.Y.Z`, depuis `vA.B.C`
- **PR de release** : #N, archive du commit `<commit>`
- **Analyse** : proposée le AAAA-MM-JJ par <session> ; validée le AAAA-MM-JJ par le développeur
- **Statut** : analyse proposée | analyse validée | en recette | terminée (release décidée ou abandonnée, vidéos supprimées)

Procédure : [`docs/recette.md`](../recette.md) (ADR 0032).

## Changements et voies de preuve

Une ligne par entrée de l'inventaire (section `## Non publié` de `CHANGELOG.md`, migrations, `deploy/`, dépendances). Gravité et probabilité : haute/moyenne/faible et élevée/moyenne/faible. Dépendance : serveur, appareil, messagerie, aucune. Voie, en toutes lettres (`docs/recette.md`, section 3) : **manuel** (vrai appareil, vraie messagerie, `@manuel`, SSH), **auto + vidéo** (parcours puis contrôle de la vidéo), **auto** (parcours seul), **CI** (rien de plus que les parcours de base).

| # | Changement | Voie | Couvert par | Gravité | Probabilité | Dépendance |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `<lot>` : … |  | parcours `…` / point de contrôle V1 / point manuel M1 / CI |  |  |  |

Parcours ciblés demandés par l'analyse :

- `web/recette/….spec.ts` : … (entrées n°…)

## Points de contrôle de la vidéo

| Point | Entrée | Parcours, projet | Étape (`test.step`) | Ce qu'il faut regarder |
| --- | --- | --- | --- | --- |
| V1 | 1 | `….spec.ts`, `phone-chromium` | « … » | … comme sur `<maquette>` |

## Séance manuelle

Appareils : …

- [ ] M1 (entrée n°…) : …
- [ ] Scénarios `@manuel` (`RELEASING.md`) : …

## Résultats

### Exécutions automatiques

| Date | Archive déployée | Parcours passés | Parcours en échec | Remarques |
| --- | --- | --- | --- | --- |
| | | | | |

### Contrôle de la vidéo

| Point | Verdict | Instant et description d'un défaut |
| --- | --- | --- |
| V1 | | |

### Séance manuelle

Date, appareils, résultat de chaque point ; défauts trouvés.

### Défauts et écarts

| Défaut | Trouvé par | PR du correctif | Rejoué |
| --- | --- | --- | --- |
| | | | |

Écarts acceptés par le développeur : …

### Fin de la recette

Date, release décidée ou abandonnée ; `make recette-clean VERSION=vX.Y.Z` lancé.
