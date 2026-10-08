# 0024. Cookies préfixés `__Host-` et nommés par tribu

- **Date** : 2026-10-08
- **Statut** : accepté

## Contexte

L'application pose deux cookies : celui de la session (ADR 0001, point 2) et celui de la demande de code, valable 10 minutes, sans lequel un code ne se saisit pas (ENF-01 ; plan `revue-securite`, D5). Chacun était propre à sa tribu par son chemin, `Path=/tribes/<identifiant>/` (ADR 0006, point 5), sous les noms `session` et `code_request`.

La revue de sécurité (plan `revue-securite`, constat 6) a relevé qu'un chemin ne protège pas un cookie des autres hôtes du domaine. Une page servie par `recette.meltingtribe.codingmatters.org`, qui sert les archives de PR (ADR 0016), ou par un autre hôte de `codingmatters.org`, peut poser un cookie `session` avec `Domain=meltingtribe.codingmatters.org` ou `Domain=codingmatters.org` : le navigateur de la victime présente alors en production la session de l'attaquant, ou perd la sienne.

Le préfixe `__Host-` règle ce point dans le navigateur : un cookie de ce nom n'est accepté que s'il est `Secure`, sans attribut `Domain` et avec `Path=/`, donc posé par l'hôte même qui le reçoit. Il impose `Path=/` : le chemin ne peut plus séparer les tribus, et le nom doit le faire à sa place, comme l'ADR 0001 le laissait ouvert (« nom de cookie par tribu »).

Rien n'est encore déployé (plan `recette`, D2) : changer les noms ne déconnecte personne.

## Décision

1. **Les deux cookies portent le préfixe `__Host-`** : `Secure`, sans attribut `Domain`, `Path=/`. Les autres attributs ne changent pas : `HttpOnly`, `SameSite=Lax` ; 90 jours à expiration glissante pour la session (ADR 0001, point 2), 10 minutes pour la demande de code.
2. **Un nom par tribu** : `__Host-session-<suffixe>` et `__Host-code-request-<suffixe>`, où le suffixe est formé des **16 premiers caractères hexadécimaux de l'empreinte SHA-256 de l'identifiant d'URL**, tel que le serveur le lit dans le chemin, après décodage.
   - Le nom se calcule pour toute valeur de l'identifiant, hors format comprise, qu'une tribu existe ou non : ses réponses ne disent rien de plus (ENF-02, ADR 0006, point 9).
   - Le suffixe a toujours la même longueur et ne contient que des caractères admis dans un nom de cookie, quel que soit l'identifiant.
   - Il ne révèle rien que l'URL ne donne déjà ; il n'a pas besoin d'être secret, puisqu'aucun autre hôte ne peut poser un cookie `__Host-`.
3. **Le serveur ne lit que les cookies de la tribu de l'URL.** Avec `Path=/`, le navigateur envoie à chaque tribu les cookies de toutes celles qu'il a ouvertes : ils sont ignorés. La valeur du cookie d'une tribu présentée sous le nom d'une autre ne trouve aucune session dans la base de celle-ci (401, ADR 0003, point 3).
4. **Les anciens noms ne sont plus lus**, ni effacés : aucune session réelle ne les porte.
5. **La recette reste sous le nom d'hôte de la production** (`recette.meltingtribe.codingmatters.org`, ADR 0017) : le préfixe suffit à ce qu'elle ne puisse pas poser un cookie que la production lirait.

Remplace l'ADR 0006, point 5, et tranche autrement le point laissé ouvert par l'ADR 0001 (« Une session par tribu »).

## Alternatives envisagées

- **Sortir la recette du nom d'hôte de la production** : ne suffit pas. Tout hôte de `codingmatters.org` peut poser un cookie pour le domaine entier, que la production reçoit aussi.
- **Les deux à la fois** : la seconde mesure n'ajoute rien au préfixe.
- **Ne rien changer** : écarté avant la première session réelle (constat 6).
- **L'identifiant lui-même comme suffixe** : un identifiant hors format devrait être échappé pour former un nom de cookie valide, la longueur du nom varierait, et le nom de la tribu se lirait en clair dans le stockage du navigateur.
- **Une empreinte avec une clé secrète** : n'apporte rien, le nom n'ayant pas à être secret (point 2).
- **Un seul cookie pour toutes les tribus**, qui contiendrait un jeton par tribu : plus compliqué, et une tribu lirait les jetons des autres.

## Conséquences

- **Positif** :
  - aucun autre hôte du domaine, recette comprise, ne peut substituer une session ni la faire perdre ;
  - un même navigateur garde une session par tribu, comme avant.
- **Négatif** :
  - chaque requête vers le nom d'hôte porte les cookies de toutes les tribus ouvertes dans le navigateur, y compris vers le site public servi par Caddy. Les journaux d'accès ne gardent donc pas l'en-tête `Cookie` (plan `recette`, étape 12). La taille ajoutée, environ 80 octets par tribu, reste négligeable ;
  - deux tribus dont les suffixes coïncideraient (une chance sur 2⁶⁴ pour une paire) partageraient un nom : la session de l'une remplacerait celle de l'autre dans ce navigateur, sans mélange de données, le jeton n'étant cherché que dans la base de la tribu de l'URL.
- **Développement** : sous `make dev` (`http://localhost:8080`), Chromium accepte les cookies `Secure` et `__Host-` venant de `localhost`, comme le parcours Playwright le vérifie ; WebKit les refuse, comme il refusait déjà le cookie `Secure` (plan `socle`, D14).
- **Documents alignés** : ADR 0001 et 0006 (renvois datés), ENF-01, scénarios d'`authentification.feature`.
