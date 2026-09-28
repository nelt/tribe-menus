# 0003. Stockage SQLite, une base par tribu

- **Date** : 2026-09-27
- **Statut** : accepté

## Contexte

L'exigence ENF-02 impose un compartimentage strict des données entre tribus. Il a été choisi de faire porter cette ségrégation par la base de données elle-même, et non seulement par le code applicatif.

Toutes les données sont propres à une tribu : membres, sessions, codes de connexion, bibliothèque de plats, référentiel d'ingrédients, planning, listes de courses, journal d'audit. Rien n'est partagé, et une même adresse e-mail a une identité distincte dans chaque tribu. Le volume et la concurrence sont très faibles : quelques membres par tribu, quelques écritures par minute au plus.

## Décision

1. **SQLite, avec un fichier de base par tribu.** Chaque fichier contient toutes les données de la tribu, y compris ses membres, ses sessions, ses codes de connexion et son journal d'audit.
2. **Un registre global**, lui aussi en SQLite, ne contient que ce qui doit être connu avant de savoir de quelle tribu il s'agit : l'identifiant d'URL de chaque tribu, son nom et l'emplacement de son fichier. Il ne contient aucune donnée métier ni aucune donnée personnelle.
3. **Le point central du filtrage est le choix de la connexion.** La tribu est déterminée par l'identifiant d'URL (`/tribes/<identifiant>/…`) ; le serveur ouvre la base de cette tribu, puis y vérifie la session. Une session n'existe que dans la base de sa tribu : un cookie présenté à une autre tribu n'y trouve rien. Le code métier reçoit une connexion déjà liée à une tribu et n'a jamais de filtre par tribu à écrire.
4. **Pilote** : `modernc.org/sqlite`, en Go pur (sans cgo), qui inclut FTS5.
5. **Configuration de chaque base** : mode WAL, `foreign_keys = ON` et `busy_timeout` à chaque connexion, tables déclarées `STRICT`. Les écritures d'une tribu passent par une seule connexion.
6. **Migrations** : fichiers SQL embarqués dans le binaire, version suivie par `PRAGMA user_version`, appliquées à l'ouverture de chaque base. Une commande d'administration permet de les appliquer à toutes les tribus d'un coup.
7. **Quantités stockées en entiers**, dans l'unité de base de leur famille (milligrammes, millilitres, millièmes de pièce, de cuillère ou de pincée : pièce, cuillère à soupe, cuillère à café et pincée forment chacune leur propre famille), pour éviter les erreurs d'arrondi des flottants dans les calculs de la liste de courses (C2). La conversion pour l'affichage suit les règles de `docs/specs/fonctionnalites.md`.
8. **Sauvegarde et réplication continue** avec Litestream vers un stockage objet, pour chaque base et pour le registre. **Reporté à une version ultérieure** (voir ADR 0007) ; les modalités précises (cible, prise en compte des nouvelles tribus, restauration) seront fixées à ce moment-là.

## Alternatives envisagées

- **PostgreSQL avec Row Level Security** : écarté. Le filtrage est bien appliqué par la base, mais les données des tribus cohabitent dans les mêmes tables ; une politique mal écrite ou un rôle qui la contourne expose tout. Serveur à exploiter.
- **PostgreSQL avec un schéma par tribu** : écarté. Isolation franche, mais migrations à rejouer par schéma et exploitation d'un serveur de base, pour un volume qui ne le justifie pas.
- **libSQL / Turso** : écarté. Même modèle en service managé, mais dépendance à un fournisseur.
- **Cloudflare Durable Objects** : écarté. Isolation idéale, mais enfermement fort chez un fournisseur et modèle d'exécution particulier.
- **CouchDB avec une base par tribu et PouchDB côté client** : écarté. Synchronisation hors-ligne native, mais sessions, codes et audit devraient vivre ailleurs, et la gestion des conflits et de l'authentification ajoute une complexité disproportionnée.

## Conséquences

- **Positif** :
  - ségrégation physique : une fuite entre tribus exige d'ouvrir le mauvais fichier, et non d'oublier un filtre ;
  - exporter, sauvegarder, restaurer ou supprimer une tribu revient à manipuler un fichier ;
  - aucun serveur de base à exploiter ; tests rapides sur des bases temporaires ;
  - lectures très rapides (pas d'aller-retour réseau).
- **Négatif** :
  - une seule machine en écriture : pas de haute disponibilité ni de répartition horizontale (suffisant à cette échelle) ;
  - typage plus faible que PostgreSQL (atténué par les tables `STRICT`), `ALTER TABLE` limité (certaines migrations recréent la table) ;
  - pas de requête transverse entre tribus : les statistiques globales, si un jour elles sont souhaitées, passeront par un script qui parcourt les bases ;
  - inspection des données uniquement depuis le serveur.
- **Tests** : les tests d'accès croisé exigés par ENF-02 portent sur le routage (identifiant d'URL, session, fichier ouvert) et vérifient qu'aucune requête d'une tribu n'atteint la base d'une autre.
- **À surveiller** : le nombre de fichiers ouverts si le nombre de tribus devient important (fermeture des bases inactives).
