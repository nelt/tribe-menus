# 0006. Reverse proxy Caddy, TLS et découpage des URL

- **Date** : 2026-09-27
- **Statut** : accepté

## Contexte

L'application doit être servie en HTTPS : la PWA et son service worker exigent un contexte sécurisé, et le cookie de session est `Secure` (ADR 0001). Le certificat doit être obtenu et renouvelé automatiquement, avec le moins possible de dépendances mouvantes.

Deux besoins s'ajoutent à l'application elle-même :

- quelques **pages publiques** sur le même nom d'hôte, l'application répondant un niveau en dessous ;
- la possibilité d'**héberger plus tard d'autres applications** sur le même serveur, avec d'autres noms d'hôte.

Le domaine `codingmatters.org` est disponible.

## Décision

1. **Nom d'hôte** : `tribe-menus.codingmatters.org`.
2. **Caddy en reverse proxy**, seul processus exposé sur les ports 80 et 443 :
   - certificats obtenus et renouvelés automatiquement auprès de Let's Encrypt ; redirection de HTTP vers HTTPS ; en-tête HSTS ;
   - paquet officiel, **sans plugin** (pas de compilation sur mesure) ;
   - API d'administration désactivée (`admin off`) ;
   - chaque application écoute sur un **socket Unix** (pas de port local), accessible au seul utilisateur de Caddy ; ajouter une application revient à ajouter un bloc de configuration avec son nom d'hôte.
3. **Découpage des chemins** sur `tribe-menus.codingmatters.org` :
   - `/` : **site public statique**, servi directement par Caddy. Première itération : trois pages (présentation, mentions légales, politique de confidentialité). Sources dans le dépôt, dossier `site/`, déployées indépendamment de l'application ;
   - `/tribes/<identifiant>/…` : **l'application**, transmise au binaire Go. Toutes ses ressources sont servies sous ce préfixe : pages, fichiers statiques, API (`/tribes/<identifiant>/api/…`), manifeste et service worker. Aucun chemin applicatif n'est partagé entre tribus, donc aucun identifiant de tribu n'a besoin d'être réservé.
4. **PWA** : le manifeste est généré par le serveur pour chaque tribu, avec la portée (`scope`) et l'URL de démarrage `/tribes/<identifiant>/`. Il ne contient le nom de la tribu que pour une requête authentifiée (voir point 9). Le service worker est enregistré avec la même portée : chaque tribu est une application installée distincte, avec ses propres caches. Le site public n'est jamais capturé par l'application installée.
5. **Cookie de session** : `Path=/tribes/<identifiant>/`. Cela tranche le point laissé ouvert par l'ADR 0001 : un même navigateur peut être connecté à plusieurs tribus sans que les sessions se mélangent.
6. **Indexation** : en-tête `X-Robots-Tag: noindex` sur toutes les réponses sous `/tribes/`, et `Disallow: /tribes/` dans `robots.txt`. Les pages publiques restent indexables.
7. **IP du client** : l'application ne tient compte de `X-Forwarded-For` que pour les requêtes arrivées par son socket, c'est-à-dire venant de Caddy. Cette IP sert à limiter les demandes et les saisies de code (ENF-01).
8. **Mode sans proxy** : le binaire peut aussi écouter sur une adresse TCP (développement, tests). Le choix se fait par configuration, sans effet sur le code métier.
9. **Rien n'est révélé avant connexion (ENF-02)** : sans session valide, toute URL sous `/tribes/<identifiant>/` renvoie le même écran de connexion générique, que la tribu existe ou non. Le nom de la tribu n'apparaît qu'une fois la connexion réussie. Les maquettes d'accueil ont été modifiées en conséquence. Le manifeste servi sans session porte un nom générique ; point à vérifier à l'implémentation : les navigateurs téléchargent le manifeste sans cookie par défaut, il faudra donc le déclarer avec `crossorigin="use-credentials"` pour obtenir le nom de la tribu sous l'icône.

## Alternatives envisagées

- **`autocert` dans le binaire Go** : envisagé sérieusement (une seule dépendance maintenue par l'équipe Go, pas d'intermédiaire). Écarté : l'application prendrait seule les ports 80 et 443, ce qui empêche de servir le site public indépendamment et d'héberger d'autres applications ; la clé privée serait détenue par le processus applicatif.
- **nginx** : très robuste, mais TLS automatique historiquement via certbot (pièce supplémentaire) ; son module ACME natif est trop récent.
- **HAProxy** : surdimensionné, client ACME intégré récent.
- **Traefik** : orienté conteneurs, et ruptures de configuration entre versions majeures.
- **Apache avec `mod_md`** : solide, mais configuration lourde pour ce besoin.
- **Client ACME séparé (certbot, lego) et TLS dans le binaire** : une pièce mobile de plus sans avantage sur Caddy.

## Conséquences

- **Positif** :
  - TLS entièrement automatique et tolérant aux incidents (relances, autorité de secours) ;
  - la clé privée n'est détenue que par Caddy, jamais par l'application ;
  - le site public reste en ligne même quand l'application redémarre ;
  - ajouter une application ou un nom d'hôte ne touche pas aux applications existantes.
- **Négatif** : un composant de plus à installer, mettre à jour et superviser, avec un arbre de dépendances plus large que celui de l'application.
- **Visibilité** : le nom d'hôte est publié dans les journaux Certificate Transparency dès la première émission de certificat et sera sondé rapidement ; la limitation des demandes de code doit être active dès la mise en ligne.
- **Énumération des tribus** : grâce au point 9, essayer des identifiants au hasard ne permet pas de savoir lesquels existent. Les options « accepter la fuite » et « identifiants difficiles à deviner » ont été écartées au profit de cette règle.
- **Documents alignés** : les exemples d'URL passent de `/t/<identifiant>/` à `/tribes/<identifiant>/` (ADR 0003 et 0004, `gestion-membres-et-sessions.md`, maquette `Doux-AjoutMembre.dc.html`). Les scénarios Gherkin parlent de « l'URL de la tribu » et ne changent pas.
