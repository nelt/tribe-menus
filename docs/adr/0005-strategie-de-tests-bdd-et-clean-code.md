# 0005. Stratégie de tests : BDD et clean code

- **Date** : 2026-09-27
- **Statut** : accepté

## Contexte

Les critères d'acceptation sont écrits en Gherkin français dans `docs/specs/features/*.feature`, chaque scénario portant l'identifiant de sa story en tag (`@C2`, `@ENF-02`…). Le projet doit suivre les principes du BDD et du clean code. Le back-end est en Go (ADR 0002), le front en TypeScript avec Lit (ADR 0004), et le souci de limiter les dépendances vaut aussi pour l'outillage de test.

## Décision

1. **Les fichiers `.feature` sont la source unique des critères d'acceptation** et sont exécutés tels quels ; ils ne sont pas recopiés dans le code de test. Un scénario modifié dans la spec change le test.
2. **Chaque scénario est exécuté au niveau le plus bas qui le prouve** :
   - **par défaut, contre l'API** avec **godog** (Gherkin officiel pour Go, compatible `# language: fr`). Le serveur tourne dans le processus de test (`httptest`), sur des bases SQLite temporaires. Les règles métier (calculs, recalcul, compartimentage, sessions) sont prouvées à ce niveau ;
   - **dans le navigateur** avec **Playwright**, pour les seuls scénarios dont le sens tient à l'interface : hors-ligne (C4), dépliage d'un article (C9), bandeaux, parcours de connexion dans la PWA. Ces scénarios portent le tag `@ui`. Playwright pilote aussi WebKit, moteur proche de Safari sur iOS.
3. **Tests unitaires** :
   - Go : `go test`, tests en tableaux de cas, sur la logique métier pure de chaque domaine (conversion et agrégation des quantités, règles de recalcul…) ;
   - TypeScript : `node:test`, sur les modules purs sans DOM (file d'opérations hors-ligne, fusion des coches, formatage des quantités).
   Les composants Lit ne sont pas testés isolément dans un DOM simulé : leur logique est extraite dans des modules purs, et leur comportement est couvert par les scénarios `@ui`.
4. **Scénarios difficiles à automatiser** (PT-15, 2026-09-28) :
   - les scénarios négatifs (« aucune action ne permet de supprimer le plat », « aucune fonction de l'interface ne permet de modifier le journal ») sont prouvés contre l'API : la route correspondante n'existe pas et la requête est refusée (`405 Method Not Allowed` ou `404`) ;
   - les scénarios qui exigent un vrai appareil (survie de la session après fermeture du navigateur ou de l'app installée) portent le tag `@manuel` : exclus de `make ci`, ils sont vérifiés sur téléphone dans l'environnement de recette avant chaque release, selon la liste tenue dans `RELEASING.md` (ADR 0012).
5. **Tests d'accès croisé entre tribus (ENF-02)** : exécutés avec godog contre l'API, avec deux tribus réelles, pour la lecture, la modification et la suppression.
6. **Principes de clean code appliqués** :
   - logique métier sans dépendance à HTTP, SQL ou au DOM, et testable isolément ;
   - vocabulaire du code aligné sur celui des spécifications, traduit en anglais selon `docs/specs/glossaire.md` (ADR 0008) ;
   - fonctions courtes, une seule responsabilité, erreurs traitées explicitement ;
   - pas d'abstraction sans deuxième usage réel ;
   - formatage et analyse statique automatiques : `gofmt`, `go vet` et `staticcheck` côté Go ; `tsc` en mode strict côté TypeScript.
7. **Intégration continue** : chaque PR exécute les tests unitaires, les scénarios godog, les scénarios `@ui` et l'analyse statique. Une PR ne fusionne que si tout est vert. Les commits citent l'identifiant de la story concernée.

## Alternatives envisagées

- **Tout exécuter dans le navigateur (Playwright ou godog avec chromedp / playwright-go)** : écarté. Tests lents et fragiles pour des règles métier qui se prouvent mieux contre l'API ; chromedp ne pilote que Chrome, et playwright-go est un portage communautaire.
- **Cucumber-js pour tous les scénarios** : écarté. Ajoute des dépendances npm et éloigne les tests métier du code Go qui les implémente.
- **Tests unitaires des composants dans un DOM simulé (jsdom, happy-dom)** : écartés. Dépendances supplémentaires pour un gain faible, dès lors que la logique est extraite dans des modules purs.

## Conséquences

- **Positif** : les specs restent la référence vivante et exécutable ; les règles métier sont prouvées rapidement et sans navigateur ; peu de dépendances de test côté Go.
- **Négatif** : deux exécuteurs de scénarios (godog et Playwright), dont les définitions d'étapes ne sont pas partagées ; Playwright est une dépendance de développement lourde, limitée à la CI et au poste de développement.
- **À préciser à l'implémentation** :
  - le moyen de faire lire les `.feature` par Playwright (par exemple `playwright-bdd`, ou des tests Playwright qui référencent les scénarios par leur tag) ;
  - l'ajout du tag `@ui` aux scénarios concernés dans `docs/specs/features/`, au fil des stories implémentées ;
