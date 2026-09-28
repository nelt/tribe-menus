# 0016. Déploiement, retour arrière et environnement de recette

- **Date** : 2026-09-27
- **Statut** : accepté

## Contexte

Les releases sont construites par la CI et publiées sur GitHub (ADR 0012). Le serveur fait tourner l'application sous systemd, avec activation de socket (ADR 0015). Les migrations de base s'appliquent au démarrage (ADR 0003) et il n'y a pas de sauvegarde hors site en V1 (ADR 0007). Les tests sur téléphone se font par déploiement sur le VPS (ADR 0009). La sécurité prime : aucun secret capable d'entrer sur le serveur ne doit être stocké à l'extérieur si on peut l'éviter.

## Décision

### Script de déploiement

1. **Un script unique sur le serveur**, `tribe-menus-deploy`, versionné dans `deploy/`, qui :
   1. télécharge l'archive de la version demandée ;
   2. vérifie son empreinte SHA-256 ;
   3. prend un instantané `VACUUM INTO` du registre et de chaque base de tribu, dans `/var/lib/tribe-menus/<environnement>/backups/avant-<version>/` ;
   4. installe la version dans `releases/<version>/` et, en production, met à jour le site public ;
   5. bascule le lien `current` de façon atomique et redémarre le service (les requêtes attendent dans le socket) ;
   6. vérifie la santé du service (`/healthz` appelé par le socket) et, en cas d'échec, **revient automatiquement** à la version précédente ;
   7. conserve les cinq dernières versions et leurs instantanés.
2. **Retour arrière manuel** : `tribe-menus-deploy --rollback`, ou déploiement de la version précédente. Après une migration non compatible, restauration depuis l'instantané pris avant le déploiement.
3. **`/healthz`** n'est pas exposé publiquement : Caddy ne le transmet pas.

### Production

4. **Déploiement manuel** : connexion SSH par l'administrateur, puis `sudo tribe-menus-deploy <version>`. Seules les releases étiquetées (`vX.Y.Z`) sont déployables en production. **Aucune clé de déploiement n'est stockée chez GitHub.**

### Recette

5. **Environnement de recette sur le même VPS**, sur `recette.tribe-menus.codingmatters.org` (*remplacé par `recette.meltingtribe.codingmatters.org`, ADR 0017*), pour tester sur téléphone avant release :
   - seconde instance de l'unité systemd modèle (`tribe-menus@recette`, la production étant `tribe-menus@production`), avec son utilisateur, ses données, son socket et sa configuration ;
   - **données de démonstration uniquement**, jamais de copie de la production ;
   - envoi d'e-mails réels par le compte `no-reply@` (ADR 0014), pour recevoir les codes sur téléphone ;
   - marquée `noindex`, comme l'application en production.
6. **Déploiement en recette** : `sudo tribe-menus-deploy --env recette --pr <numéro>` installe l'archive construite par la CI pour la PR. Le téléchargement des artefacts de workflow exige un jeton même sur un dépôt public : jeton à portée fine, limité à la lecture des Actions de ce dépôt, stocké en credential systemd sur le serveur. **Seules les PR du propriétaire du dépôt** sont déployées en recette, jamais celles venant d'un fork.
7. La CI produit l'archive (`make build`) pour chaque PR et la conserve comme artefact quelques jours (ADR 0013).
8. **Caddy** sert les deux noms d'hôte, chacun vers son socket (ADR 0006).

## Alternatives envisagées

- **Déploiement tiré par le serveur** (minuteur qui suit un fichier `deploy/production-version` modifié par PR) : automatisé et tracé, sans secret chez GitHub. Reporté : plus de mécanique, et le script actuel s'y prête sans changement.
- **Déploiement poussé par GitHub Actions en SSH** : le plus fluide, mais une clé capable d'entrer sur le serveur serait stockée chez GitHub, sans restriction d'IP possible. Écarté.
- **Versions candidates (`vX.Y.Z-rc.N`) en production pour les tests** : écartées, la famille verrait les versions de test.
- **Recette protégée par mot de passe HTTP** : écartée, fonctionne mal avec une PWA installée sur iOS.

## Conséquences

- **Positif** : aucun secret d'accès au serveur hors du serveur ; retour arrière automatique en cas d'échec au démarrage ; instantané local des données avant chaque déploiement ; tests sur téléphone sans toucher à la production.
- **Négatif** : chaque déploiement demande une connexion SSH ; deux instances à entretenir sur le même serveur ; un jeton GitHub (lecture seule) stocké sur le serveur.
- **Risques acceptés** : la recette est joignable publiquement et fait tourner du code non encore fusionné (mais écrit par le propriétaire du dépôt) ; son nom d'hôte est publié dans les journaux Certificate Transparency.
- **Les instantanés locaux ne remplacent pas une sauvegarde hors site** : ils sont sur le même disque que les données.
