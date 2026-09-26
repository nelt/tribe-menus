# personal-sandbox

Bac à sable personnel pour expérimenter, notamment un flux de travail hybride avec Claude :

- **Claude (app web / mobile)** pour le cadrage, l'architecture et la préparation, avec les résultats déposés dans `docs/`.
- **Claude Code** (terminal, IDE ou cloud) pour l'implémentation, en s'appuyant sur `CLAUDE.md` et `docs/`.

## Organisation

| Chemin | Rôle |
| --- | --- |
| `CLAUDE.md` | Instructions chargées automatiquement par Claude Code à chaque session |
| `docs/plans/` | Plans de travail préparés en amont (un fichier par sujet) |
| `docs/adr/` | Architecture Decision Records : décisions techniques et leur justification |

## Flux de travail

1. Préparer un sujet dans Claude (chat) et produire un plan dans `docs/plans/AAAA-MM-JJ-sujet.md`.
2. Consigner toute décision structurante dans un ADR (`docs/adr/NNNN-titre.md`, modèle `docs/adr/0000-template.md`).
3. Dans Claude Code : « lis `docs/plans/<fichier>.md` et implémente l'étape 1 ».
4. Travailler sur une branche dédiée, puis revue via pull request.
