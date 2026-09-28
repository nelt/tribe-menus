# 0002. Back-end en Go, monolithe modulaire

- **Date** : 2026-09-27
- **Statut** : accepté

## Contexte

L'application sert quelques tribus de quelques membres : la charge est négligeable, mais la maintenance doit rester légère sur la durée, et la surface d'attaque réduite. Le serveur porte les sessions (ADR 0001), le compartimentage des tribus (ENF-02), l'API JSON consommée par le front (ADR 0004), les scripts d'administration (EF-08, EF-09) et le journal d'audit (EF-07).

Critères retenus :

- peu de dépendances tierces à l'exécution ;
- un déploiement simple (un artefact, peu de mémoire, démarrage rapide) ;
- une base de code lisible et stable dans le temps, conforme aux principes de clean code.

## Décision

1. **Langage : Go**, dernière version stable, en suivant les pratiques de la communauté : bibliothèque standard d'abord, code explicite, pas de framework englobant.
2. **Architecture : monolithe modulaire.** Un seul binaire, découpé en paquets par domaine métier aux frontières nettes :
   - tribu, membres et sessions (y compris l'authentification et l'audit) ;
   - plats et référentiel d'ingrédients ;
   - planning des repas ;
   - listes de courses.

   La logique métier de chaque domaine est pure (sans HTTP ni SQL) et testable isolément ; les handlers HTTP et l'accès aux données sont des adaptateurs minces autour d'elle.
3. **Socle technique** :
   - HTTP : `net/http` (routage par méthode et chemin de la bibliothèque standard) ;
   - accès aux données : `database/sql` et SQL écrit à la main, avec **sqlc** pour générer le code typé (outil de développement, pas de dépendance à l'exécution) ;
   - journalisation : `log/slog` ;
   - configuration : fichier non secret par environnement, secrets lus comme des fichiers (credentials systemd), jamais en variables d'environnement ; *précisé par l'ADR 0015, points 9 et 10* ;
   - câblage des dépendances explicite dans `main`, sans conteneur d'injection.
4. **Un seul binaire, plusieurs commandes** : le serveur et les commandes d'administration (EF-08, EF-09) font partie du même exécutable (sous-commandes). Le front compilé y est embarqué (`embed`) et servi sur la même origine que l'API, comme l'exige l'ADR 0001.
5. **Dépendances tierces** limitées à ce que la bibliothèque standard ne couvre pas (pilote SQLite, exécuteur Gherkin), chacune justifiée. Leur liste est tenue à jour dans `go.mod` et revue à chaque ajout.

## Alternatives envisagées

- **Java / Spring Boot** : écarté malgré l'expertise disponible. Empreinte mémoire plus forte sur un petit serveur, support SQLite faible côté Hibernate, et nombreuses dépendances transitives.
- **Kotlin / Ktor** : écarté. Compromis entre Java et Go sans avantage décisif ici.
- **TypeScript (Node, Bun)** : écarté. Aurait permis un seul langage, mais la culture de l'écosystème npm pousse à accumuler les dépendances, avec un risque de chaîne d'approvisionnement plus élevé côté serveur.
- **Rust** : écarté. Excellent back-end, mais courbe d'apprentissage et nombre de dépendances transitives (crates) comparables à npm.
- **Microservices** : écartés. À cette échelle, ils n'apportent que du coût d'exploitation ; les frontières de modules permettront d'en extraire un plus tard si besoin.

## Conséquences

- **Positif** : binaire statique unique, empreinte mémoire minimale, démarrage instantané, compilation croisée triviale ; très peu de dépendances à l'exécution ; code stable dans le temps (promesse de compatibilité de Go 1).
- **Négatif** : gestion d'erreurs verbeuse ; modélisation du domaine moins expressive qu'avec des types somme (les familles d'unités, par exemple, se modélisent par des types et des constantes) ; deux langages dans le projet (Go et TypeScript).
- **Discipline requise** : sans framework, les conventions (structure des paquets, gestion des erreurs, validation des entrées) doivent être posées tôt et documentées dans `CLAUDE.md`.
