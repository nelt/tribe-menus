<p align="center"><img src="docs/design/identite/symbole.svg" alt="" width="160"><br><img src="docs/design/identite/logotype.svg" alt="Melting Tribe" width="320"></p>

# Melting Tribe

Les menus de la semaine, en tribu : chacun ajoute ses plats au planning, la liste de courses se fait toute seule. Application web progressive, logiciel libre sous licence AGPL-3.0-or-later ([`LICENSE`](LICENSE)), servie sur `meltingtribe.codingmatters.org` (ADR 0017).

Le projet sert aussi à expérimenter un flux de travail hybride avec Claude :

- **Claude (app web / mobile)** pour le cadrage, l'architecture et la préparation, avec les résultats déposés dans `docs/`.
- **Claude Code** (terminal, IDE ou cloud) pour l'implémentation, en s'appuyant sur `CLAUDE.md` et `docs/`.

## Organisation

| Chemin | Rôle |
| --- | --- |
| `CLAUDE.md` | Instructions chargées automatiquement par Claude Code à chaque session |
| `docs/specs/` | Spécifications fonctionnelles (le *quoi*) |
| `docs/design/` | Design dans son état actuel : identité, site public, connexion, application ; maquettes de référence |
| `docs/plans/` | Plans de travail préparés en amont (un fichier par sujet) |
| `docs/adr/` | Architecture Decision Records : décisions techniques et leur justification |

## Flux de travail

1. Préparer un sujet dans Claude (chat) et produire un plan dans `docs/plans/AAAA-MM-JJ-sujet.md`.
2. Consigner toute décision structurante dans un ADR (`docs/adr/NNNN-titre.md`, modèle `docs/adr/0000-template.md`).
3. Dans Claude Code : « lis `docs/plans/<fichier>.md` et implémente l'étape 1 ».
4. Travailler sur une branche dédiée, puis revue via pull request.

## Licence

Copyright © 2026 Nel Taurisson et les contributeurs de Melting Tribe.

Ce programme est un logiciel libre : vous pouvez le redistribuer et le modifier selon les termes de la GNU Affero General Public License publiée par la Free Software Foundation, en version 3 ou (à votre choix) toute version ultérieure. Il est distribué sans aucune garantie ; voir [`LICENSE`](LICENSE). Les polices restent sous leur propre licence (OFL), dont le texte est dans leur dossier, [`web/src/fonts/`](web/src/fonts/).

Contribuer : [`CONTRIBUTING.md`](CONTRIBUTING.md). Signaler une vulnérabilité : [`SECURITY.md`](SECURITY.md).
