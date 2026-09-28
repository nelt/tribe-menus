# 0004. Front-end : SPA TypeScript avec Lit

- **Date** : 2026-09-27
- **Statut** : accepté

## Contexte

L'application est une PWA utilisée surtout sur téléphone, y compris installée sur iOS. Les maquettes (`docs/design/`) sont riches en interactions : stepper de parts, autocomplétion des ingrédients, calendrier de période, articles dépliables, feuilles de confirmation, bandeaux (liste périmée, hors-ligne). La liste de courses doit rester consultable et cochable sans réseau (C4, Q16).

Critères retenus : une surface d'attaque réduite, peu de dépendances, pas de framework englobant exposé aux ruptures de compatibilité, et un code pérenne.

## Décision

1. **Une SPA en TypeScript**, qui consomme l'API JSON du back-end Go (ADR 0002). Le front compilé est embarqué dans le binaire et servi sur la même origine.
2. **Lit** comme unique bibliothèque d'exécution. Les composants sont des *custom elements* standard du navigateur.
3. **Règle d'usage du shadow DOM** :
   - **shadow DOM** pour les composants visuels terminaux, sans champ de formulaire : pilule de plat, badge de parts, initiale, avatar, pastilles de statut, bandeaux, feuille de confirmation (contenu fourni par des slots) ;
   - **DOM classique** (`createRenderRoot()` renvoie l'élément lui-même) pour les écrans et les formulaires : saisie de l'e-mail et du code, création de plat, ajout d'article, autocomplétion (motif ARIA combobox), écrans complets ;
   - **champs sur mesure** (stepper de parts) : shadow DOM et déclaration comme champ de formulaire via `ElementInternals`.
4. **Styles** : CSS natif. Les tokens de `docs/design/README.md` sont déclarés comme propriétés personnalisées sur `:root`, ce qui les rend visibles dans les shadow roots. Pas de framework CSS.
5. **Polices auto-hébergées** (Bricolage Grotesque, Figtree) : pas d'appel à Google Fonts, pour le fonctionnement hors-ligne, la vie privée et une CSP stricte.
6. **Routage** : un petit routeur écrit dans le projet, fondé sur l'API History, avec le préfixe de tribu (`/tribes/<identifiant>/…`).
7. **Hors-ligne** :
   - **service worker écrit dans le projet** : pré-cache des ressources versionnées, cache d'abord pour les ressources statiques, réseau d'abord pour l'API ;
   - **IndexedDB** pour les listes de courses affichées et pour une **file d'opérations** (coches, articles ajoutés) rejouée au retour du réseau. Les opérations sont idempotentes : identifiants générés côté client pour les articles ajoutés, état final plutôt que bascule pour les coches ; un ingrédient créé hors ligne est transmis par son nom et rapproché côté serveur d'un ingrédient existant de même nom normalisé ; les opérations devenues sans objet (liste close, article retiré par un recalcul) sont ignorées et signalées (Q18) ;
   - l'écran de liste s'affiche à partir de ces données locales, jamais d'un HTML en cache.
8. **Outillage** :
   - `typescript` pour la vérification des types (`tsc --noEmit`) ;
   - **esbuild** pour transpiler et assembler ;
   - pas de Vite, de Webpack ni de Workbox.
9. **Sécurité** :
   - CSP stricte : `script-src 'self'`, aucun script en ligne, aucune ressource tierce ;
   - interdiction, vérifiée par lint ou revue, des échappatoires au rendu échappé (`unsafeHTML`, `innerHTML`) ;
   - versions de dépendances épinglées, installation par `npm ci --ignore-scripts`, mises à jour suivies par Dependabot.

## Alternatives envisagées

- **HTML rendu par Go avec des îlots de TypeScript** : sérieusement envisagé, pour son minimum de JavaScript et l'échappement contextuel de `html/template`. Écarté : la densité d'interactions des maquettes multiplie les îlots, et l'écran de liste devant s'afficher hors-ligne depuis IndexedDB, plusieurs composants auraient existé deux fois (gabarit Go et TypeScript).
- **Preact** : envisagé. Un seul paquet sans dépendance, CSS global et formulaires sans friction. Écarté au profit de Lit : les composants Lit sont des éléments HTML standard qui survivraient à la bibliothèque, sans JSX ni transformation de syntaxe, et avec un modèle orienté classes.
- **TypeScript sans bibliothèque (Web Components natifs)** : écarté. Aucune dépendance, mais beaucoup de code répétitif pour le rendu et les mises à jour.
- **React, Angular, SvelteKit, Next** : écartés. Surface de dépendances importante et ruptures de modèle ou d'API fréquentes.
- **Workbox** : écarté. Le besoin de cache et de synchronisation est limité et se traite en une centaine de lignes maîtrisées.

## Conséquences

- **Positif** : une seule bibliothèque d'exécution, maintenue par une seule équipe ; composants fondés sur des standards pérennes ; un seul système de rendu, y compris hors-ligne ; frontière nette entre front et back (contrat de l'API JSON).
- **Négatif** :
  - la frontière du shadow DOM impose un choix par composant ; la règle ci-dessus doit être appliquée et, en cas de doute, discutée en revue ;
  - un état client, un routeur et un service worker à écrire et à maintenir ;
  - le routeur de Lit étant expérimental, le routage reste à notre charge.
- **Sécurité** : l'échappement des données repose sur `lit-html` ; il reste sûr tant que les échappatoires sont proscrites.
