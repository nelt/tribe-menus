# 0012. Chaîne de build et procédure de release

- **Date** : 2026-09-27
- **Statut** : accepté

## Contexte

Chaque version déployée doit pouvoir être rattachée à un commit précis et reconstruite à l'identique. Le binaire Go embarque le front (ADR 0002, 0004) ; Caddy sert le site public depuis le disque (ADR 0006) ; le serveur est un VPS OVHcloud en `linux/amd64` (ADR 0007). Les commandes passent par le Makefile (ADR 0010). Le dépôt est public (ADR 0011).

## Décision

1. **Ce qui tourne en production est construit par la CI, jamais sur un poste de développement.**
2. **Étapes de `make build`** :
   - **front** : esbuild produit des fichiers minifiés dont le nom contient une empreinte de leur contenu (`app.3f9a1c.js`) ; sourcemaps générées et publiées (ADR 0011) ;
   - **service worker** : un script de build (API d'esbuild) écrit la liste des fichiers à pré-cacher et la version du service worker ; les sourcemaps en sont exclues ;
   - **binaire** : `CGO_ENABLED=0 go build -trimpath`, pour `linux/amd64`, avec la version et le commit injectés (`-ldflags -X`), affichés par `tribe-menus version`, dans les logs de démarrage et dans le lien vers le code source (ADR 0011).
3. **Artefact livré** : une archive contenant le binaire (front embarqué), le site public (`site/`), les fichiers de `deploy/`, et une empreinte SHA-256 publiée à côté de l'archive.
4. **Versions** : versionnage sémantique adapté à une application.
   - `0.x` pendant la construction de la V1, `1.0.0` quand la V1 est complète ;
   - version mineure : nouvelles stories ; correctif : corrections seules.
5. **Procédure de release** (décrite dans `RELEASING.md`, que Claude Code peut suivre pour préparer la PR de release) :
   1. chaque PR ajoute une ligne à la section « Non publié » de `CHANGELOG.md`, en citant la story ;
   2. une PR `release/vX.Y.Z` renomme cette section en `X.Y.Z — date` ;
   3. l'archive de cette PR est déployée en recette, où les scénarios `@manuel` (ADR 0005) sont vérifiés sur téléphone ;
   4. après sa fusion, une étiquette annotée `vX.Y.Z` est posée sur ce commit de `main` ;
   5. l'étiquette déclenche le workflow de release : `make ci`, `make build` avec la version de l'étiquette, empreinte, puis release GitHub avec l'archive et les notes extraites de `CHANGELOG.md` ;
   6. le déploiement est une action distincte (ADR 0016) : publier une version ne la déploie pas, et revenir en arrière consiste à redéployer une version antérieure.
6. **Garde-fous** :
   - une règle GitHub protège les étiquettes `v*` : seul le propriétaire du dépôt peut les créer, et elles ne peuvent être ni déplacées ni supprimées ;
   - les migrations de base (appliquées au démarrage, ADR 0003) sont signalées dans les notes de version ; d'une version à la suivante, elles ajoutent sans casser, pour qu'un retour à la version précédente reste possible.

## Alternatives envisagées

- **Une release à chaque fusion sur `main`** (identifiée par date et commit) : plus fluide pour tester souvent, écartée au profit d'une release comme acte volontaire, au numéro porteur de sens.
- **Sourcemaps non publiées**, jointes à la release : écartées, le code étant public.
- **Build sur le poste puis copie sur le serveur** : écarté, pas de traçabilité entre binaire et commit.

## Conséquences

- **Positif** : binaire reproductible et traçable jusqu'au commit ; front jamais servi périmé depuis le cache ; release et déploiement découplés, retour arrière simple.
- **Négatif** : une PR de release à chaque version (quelques lignes) ; discipline de migrations compatibles d'une version à l'autre.
- **Reporté à une version ultérieure** : attestation de provenance des archives (gratuite sur un dépôt public, vérifiable sur le serveur avant installation) ; SBOM (inventaire des dépendances de chaque release).
