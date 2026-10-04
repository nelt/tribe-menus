# Polices

Polices auto-hébergées de l'application (ADR 0004, point 5 ; plan du 2026-10-03, D7), en `woff2` variables, prises telles quelles dans les dépôts officiels de leurs auteurs. Aucun paquet npm. Elles sont sous licence SIL Open Font License 1.1 (fichiers `OFL-*.txt`), seule exception à l'AGPL du dépôt (ADR 0011).

| Fichier | Police | Version | Source | SHA-256 |
| --- | --- | --- | --- | --- |
| `BricolageGrotesque[opsz,wdth,wght].woff2` | Bricolage Grotesque, axes `opsz` 12–96, `wdth` 75–100, `wght` 200–800 | 1.001 (table `name`) | [ateliertriay/bricolage](https://github.com/ateliertriay/bricolage), `fonts/webfonts/`, commit `84745e5b96261ae5f8c6c856e262fe78d1d6efdd` (le dépôt n'a pas d'étiquette) | `b51a8ebd169637e47cb7db430431ab3e122d2f09b03ee2a03ea06f4cb46f1a8e` |
| `Figtree[wght].woff2` | Figtree, axe `wght` 300–900 | 2.001 (table `name`) | [erikdkennedy/figtree](https://github.com/erikdkennedy/figtree), `fonts/variable/`, étiquette `v2.0.3` | `f80864f632039e7c94f4ec471de1f029b2c8bef05e894c559044c6dee6580d8a` |

`OFL-BricolageGrotesque.txt` et `OFL-Figtree.txt` sont les fichiers `OFL.txt` de ces mêmes versions. Ils sont copiés dans `web/dist` avec les polices.

Bricolage Grotesque a des chiffres tabulaires (`tnum`), dont se sert la saisie du code de connexion (vérifié le 2026-10-04 dans Chromium : les dix chiffres ont la même chasse avec `font-variant-numeric: tabular-nums`).

Pour mettre à jour une police : télécharger le fichier de la nouvelle version et son `OFL.txt` depuis le dépôt de l'auteur, remplacer ceux d'ici, mettre ce tableau à jour (`sha256sum`).
