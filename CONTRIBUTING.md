# Contribuer

Merci de votre intérêt pour Melting Tribe.

## Licence

Tout le dépôt est sous licence `AGPL-3.0-or-later`, documentation comprise, polices exceptées (ADR 0011). Une contribution est publiée sous cette même licence.

## Certificat d'origine (DCO)

Chaque commit porte une ligne `Signed-off-by` au nom et à l'adresse de son auteur :

```
Signed-off-by: Prénom Nom <adresse@example.org>
```

Par cette ligne, vous certifiez avoir le droit de contribuer ce changement sous la licence du projet, dans les termes du [Developer Certificate of Origin](https://developercertificate.org/). Git l'ajoute avec `git commit -s`. Pour signer après coup les commits d'une branche : `git rebase --signoff main`.

La CI vérifie chaque commit d'une PR (`internal/tools/dcocheck`), sauf les commits de fusion et ceux de Dependabot, qui ne font que monter des versions.

## Avant d'ouvrir une PR

- Travailler sur une branche `feature/<sujet>`.
- Lancer `make ci` : c'est ce que lance la CI (ADR 0010, 0013). `make help` liste les cibles.
- Ajouter une ligne à `CHANGELOG.md`, sous « Non publié » (ADR 0012).
- Toute décision d'architecture significative fait l'objet d'un ADR dans `docs/adr/`.

L'environnement de développement de référence est le Dev Container décrit dans `docs/poste-de-developpement.md`.

## Signaler une vulnérabilité

Pas d'issue publique : utilisez le signalement privé de vulnérabilités de GitHub (onglet *Security*).
