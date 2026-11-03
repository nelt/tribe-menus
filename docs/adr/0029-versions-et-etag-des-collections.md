# 0029. Versions et ETag des collections

- **Date** : 2026-11-03
- **Statut** : accepté

## Contexte

Le navigateur garde en mémoire la bibliothèque de plats et le référentiel d'ingrédients de la tribu, chargés par pages (ADR 0027). Un membre qui revient sur l'application doit voir ce qu'un autre a modifié entre-temps, sans temps réel (Q6) mais sans copie de la veille ; recharger toutes les pages à chaque retour coûterait cher sur une grande bibliothèque, pour un changement rare. Le navigateur doit donc savoir, en une requête, si une collection a changé, et le serveur le dire sans relire la collection.

Les plans suivants ajouteront d'autres collections (planning, listes de courses, membres, sessions), avec le même besoin et le même risque : une écriture nouvelle qui oublierait de signaler le changement laisserait des copies périmées sans que rien ne le montre.

Décidé avec le développeur le 2026-10-29 (plan `bibliotheque`, D25, D26, D28, D29).

## Décision

1. **Une version par collection**, un entier dans la table `collection_versions` de la base de la tribu (`name`, `version`). Une collection regroupe une ou plusieurs tables : `dishes` pour `dishes` et `dish_ingredients`, `ingredients` pour `ingredients`.
2. **La version est augmentée par des déclencheurs SQLite** `AFTER INSERT`, `AFTER UPDATE` et `AFTER DELETE` sur chaque table de la collection, écrits dans la migration qui crée la table. Aucune écriture ne peut oublier de l'augmenter, quel que soit le code qui écrit. **Un test lit le schéma** et vérifie que chaque table déclarée dans une collection a ses trois déclencheurs. C'est de la mécanique de stockage, pas une règle métier : l'ADR 0002 (logique métier en Go, pure) n'est pas touché.
3. **Un paquet générique `internal/collection`**, par lequel passe **toute route de collection** de l'API, de ce plan et des suivants :
   - pages bornées par le serveur, dans l'ordre des identifiants, avec un curseur vers la page suivante (paramètre `after`), absent à la dernière ;
   - `ETag: "<collection>-<version>"` sur chaque page ;
   - première page avec `If-None-Match` : `304` sans corps si la version n'a pas changé ;
   - pages suivantes avec `If-Match` : `412 collection_changed` si la version a changé pendant le chargement ;
   - la réponse d'une écriture porte les versions, après l'écriture, des collections qu'elle a touchées (`"versions": { "dishes": 43, "ingredients": 18 }`) ;
   - la version et la page sont lues dans une même transaction de lecture, pour qu'une page ne porte jamais l'ETag d'une autre version.
4. **Côté navigateur, une copie générique** (`web/src/collections.ts`), sur laquelle s'appuient les copies de chaque domaine (`library.ts` pour les plats et le référentiel) :
   - elle garde les éléments et l'ETag d'une collection, **en mémoire seulement** : rien n'est écrit sur l'appareil (ni `localStorage`, ni IndexedDB, ni cache), la copie disparaît au rechargement de la page, à la déconnexion et sur un `401` ;
   - **vérification** à chaque visite de l'onglet du domaine et à chaque retour de l'application au premier plan (`visibilitychange`) : la première page en conditionnel ; `304`, la copie reste ; `200`, toutes les pages sont rechargées avec `If-Match` ; `412`, le chargement reprend depuis la première page, trois fois au plus avant d'afficher une erreur ;
   - **après une écriture** du membre : la copie prend l'élément renvoyé ; si la version renvoyée suit immédiatement la sienne, elle la prend aussi ; sinon, un autre membre a écrit entre-temps, et la copie se vérifie aussitôt ;
   - la décision (garder, recharger, reprendre, prendre la version) est une fonction pure, testée par `node:test` (ADR 0005, point 3).
5. **Les routes restent derrière la session**, et la version d'une tribu n'est visible que de ses membres (ENF-02). L'API garde `Cache-Control: no-store` : l'ETag sert la copie en mémoire, jamais le cache HTTP du navigateur.

## Alternatives envisagées

- **Augmenter la version dans le code Go de chaque écriture** : écarté. Une écriture nouvelle peut l'oublier, et rien ne le verrait.
- **Un ETag calculé par empreinte du contenu** : écarté. Il oblige à lire toute la collection à chaque vérification.
- **Un horodatage de dernière modification** : écarté. Ni monotone (horloge du serveur) ni unique (deux écritures dans la même milliseconde).
- **Une version par tribu, toutes collections confondues** : écartée. Ajouter un article à une liste ferait recharger la bibliothèque.
- **Recharger à chaque retour, ou après un délai** : écarté. Inutilement coûteux dans le premier cas, copie périmée dans le second ; une vérification sans changement ne coûte qu'une requête et un `304`.
- **Temps réel** (WebSocket, événements du serveur) : hors périmètre V1 (Q6).
- **Garder la copie sur l'appareil** (IndexedDB, cache du service worker) : laissé au plan hors-ligne, avec l'effacement à la déconnexion (ADR 0004, point 7).

## Conséquences

- **Positif** : un retour sur l'application coûte une requête quand rien n'a changé ; aucune écriture ne peut laisser une copie périmée sans le signaler ; un seul mécanisme pour toutes les collections, que les plans suivants n'ont qu'à déclarer.
- **Négatif** : une seule modification fait encore recharger toute la collection ; un delta (`?since=<version>`, chaque élément gardant la version qui l'a écrit) est laissé au plan hors-ligne, et demandera des marques de suppression quand des éléments se supprimeront. Chaque migration qui crée une table de collection doit écrire ses trois déclencheurs, ce que le test de schéma vérifie.
- **Les déclencheurs comptent toute écriture**, y compris une écriture sans effet visible (par exemple, l'unité par défaut fixée sur un ingrédient) : la copie se recharge alors pour rien, ce qui reste correct.
- **Obligatoire pour les plans suivants** : planning, listes de courses, membres et sessions servent leurs collections par `internal/collection` (feuille de route, « Métier, suite »). Le hors-ligne des listes de courses (IndexedDB, file d'opérations) s'y ajoutera sans le remplacer.
