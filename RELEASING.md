# Publier une version

Procédure de release de Melting Tribe (ADR 0012, point 5). Claude Code peut la suivre pour préparer la PR de release ; l'étiquette, la fusion et le déploiement reviennent au développeur.

Une **version** est une archive `tribe-menus-vX.Y.Z-linux-amd64.tar.gz`, son empreinte SHA-256 et ses notes, publiées dans une release GitHub par le workflow `release.yml` quand une étiquette `vX.Y.Z` est posée sur `main`. Publier une version ne la déploie pas : le déploiement est une action distincte (ADR 0016), décrite par le plan `recette`.

## Choisir le numéro

Versionnage sémantique adapté à une application (ADR 0012, point 4) :

- `0.x.y` pendant la construction de la V1, `1.0.0` quand la V1 est complète ;
- **version mineure** (`0.3.0`) : la section « Non publié » contient au moins une story ou un changement visible ;
- **correctif** (`0.3.1`) : elle ne contient que des corrections.

## Release ordinaire

1. **Branche** `release/vX.Y.Z`, partie de `main` à jour. Le push d'une branche `release/…` demande une confirmation à Claude Code (`.claude/settings.json`).
2. **`CHANGELOG.md`** : la section `## Non publié` devient `## X.Y.Z — AAAA-MM-JJ` (tiret cadratin, date du jour) ; une section `## Non publié` vide est ajoutée au-dessus.
3. **Migrations** (ADR 0012, point 6) : lister celles ajoutées depuis la version précédente :

   ```sh
   git diff --name-only --diff-filter=A vA.B.C..HEAD -- internal/storage/migrations/
   ```

   S'il y en a, ajouter à la fin de la section une sous-section `### Migrations` : une ligne par migration, avec la base touchée (registre, tribu, limitation) et ce qu'elle change. Une migration ajoute sans casser : la version précédente doit pouvoir tourner sur la base migrée, pour que le retour arrière reste possible. Si ce n'est pas le cas, l'écrire en tête de la sous-section : le retour arrière passera par l'instantané pris avant le déploiement (ADR 0016, point 2).
4. **Vérifier les notes**, telles que la release les publiera :

   ```sh
   go run ./internal/tools/relnotes vX.Y.Z
   ```

5. **PR** intitulée `Release X.Y.Z`, commit signé (`git commit -s`), `make ci` avant de pousser. La ligne de `CHANGELOG.md` de cette PR, c'est la section renommée : pas de ligne en plus.
6. **Recette** : le développeur déploie l'archive de la PR en recette (artefact `tribe-menus-pr-<numéro>` de la CI) et y vérifie, sur téléphone :
   - les scénarios `@manuel` (ADR 0005), dont la liste est tenue ci-dessous ;
   - les stories de la section, sur un vrai appareil.

   Un défaut trouvé se corrige par une PR ordinaire ; la PR de release est ensuite mise à jour avec `main`.
7. **Fusion** par le développeur.
8. **Étiquette annotée**, posée par le développeur (seul le propriétaire du dépôt peut créer une étiquette `v*`, et elle ne se déplace ni ne se supprime) sur le commit de fusion :

   ```sh
   git fetch origin
   git tag -a vX.Y.Z -m "Melting Tribe X.Y.Z" origin/main
   git push origin vX.Y.Z
   ```

   Vérifier avant le push que `origin/main` est bien le commit de fusion de la PR de release (`git log -1 origin/main`) : une étiquette mal placée ne se corrige pas, il faut passer au numéro suivant.
9. **Release** : le workflow `release.yml` vérifie l'étiquette (forme `vX.Y.Z`, commit sur `main`, section présente dans `CHANGELOG.md`), lance `make tools` puis `make ci` avec cette version, et publie la release avec l'archive, son empreinte et les notes. Suivre son exécution :

   ```sh
   gh run list --workflow release.yml --limit 1
   gh run watch <identifiant>
   ```

10. **Vérifier ce qui est publié** :

    ```sh
    gh release download vX.Y.Z --dir /tmp/vX.Y.Z
    cd /tmp/vX.Y.Z && sha256sum -c tribe-menus-vX.Y.Z-linux-amd64.tar.gz.sha256
    tar -xzf tribe-menus-vX.Y.Z-linux-amd64.tar.gz
    tribe-menus-vX.Y.Z-linux-amd64/tribe-menus version
    ```

    La dernière commande affiche `tribe-menus vX.Y.Z (<commit>)`, avec le commit de l'étiquette.

Si le workflow échoue, la release n'est pas publiée. Corriger par une PR ordinaire et publier le numéro suivant : l'étiquette posée ne se déplace pas.

## Version corrective d'une vulnérabilité

Aux niveaux accéléré et urgent (`docs/traitement-des-vulnerabilites.md`), la version corrective est préparée **dans la PR du correctif**, pour livrer en une seule séance :

1. Sur la branche locale du correctif, après les commits du correctif et `make ci` : une ligne dans la section `## Non publié` de `CHANGELOG.md`, qui dit ce que le code fait désormais, **en termes neutres**.
2. Dans la même branche, un dernier commit fait les étapes 2 à 4 de la release ordinaire, avec le numéro de correctif suivant (`0.3.0` → `0.3.1`).
3. Push et PR quand le développeur ouvre la séance de livraison ; titre de la PR neutre, suivi de ` ; release X.Y.Z`.
4. Recette ciblée sur l'archive de la PR (accéléré) ou après la production (urgent), selon le niveau ; puis fusion, étiquette et vérification, étapes 7 à 10 de la release ordinaire.

**Ce que la version emporte.** Une étiquette n'est posée que sur un commit de `main` (`release.yml` le vérifie) : la version corrective emporte tout ce qui a été fusionné depuis la version précédente, et la section renommée le dit. Des stories non publiées, jamais passées en recette, partent avec le correctif : c'est l'axe 2 de l'analyse de risques, à peser avant de choisir le niveau.

**Après la production**, et pas avant : la ligne de la version est complétée par une PR ordinaire (`docs/traitement-des-vulnerabilites.md`, « Publication »).

## Scénarios `@manuel`

Vérifiés sur téléphone en recette avant chaque release (ADR 0005). La liste suit les `.feature` : `grep -rn -A1 '@manuel' docs/specs/features/`.

| Scénario | Fichier | Comment |
| --- | --- | --- |
| La session survit au redémarrage du navigateur | `authentification.feature` | se connecter, fermer complètement le navigateur (pas seulement l'onglet), rouvrir l'URL de la tribu ; sous Safari et sous Chrome |
| La session survit à la fermeture de l'app installée | `authentification.feature` | dès que l'application est installable (plan du hors-ligne) : installer, se connecter, fermer l'app depuis le sélecteur d'applications, la rouvrir |

## Essai à blanc

`release.yml` lancé à la demande construit une version `essai-<numéro d'exécution>` sans rien publier : pas de contrôle de `CHANGELOG.md`, pas de notes, pas de release. Il sert à vérifier la chaîne après une modification du workflow ou de `make dist`.

```sh
gh workflow run release.yml --ref main
gh run list --workflow release.yml --limit 1
gh run download <identifiant> --name release-essai-<numéro> --dir /tmp/essai
```

Puis les vérifications de l'étape 10, avec ce nom de version, et **le binaire de l'archive lancé une fois** : c'est le seul moment où il tourne avec son front et sa liste de fichiers embarqués, hors des tests. Le binaire est en `linux/amd64` : sur un poste ARM, faire l'essai dans une machine ou un conteneur de cette architecture.

```sh
cd /tmp/essai && tar -xzf tribe-menus-essai-<numéro>-linux-amd64.tar.gz
bin=$PWD/tribe-menus-essai-<numéro>-linux-amd64/tribe-menus
mkdir -p data credentials && echo essai > credentials/smtp-password
cat > config.json <<EOF
{
  "data": "$PWD/data",
  "listen": "127.0.0.1:8090",
  "baseURL": "https://essai.example.org",
  "smtp": {"host": "smtp.example.org", "port": 465, "username": "no-reply@example.org", "from": "no-reply@example.org"},
  "alerts": {"to": "admin@example.org"}
}
EOF
$bin admin seed -config config.json
CREDENTIALS_DIRECTORY=$PWD/credentials $bin serve -config config.json
```

Le mot de passe SMTP est exigé au démarrage, mais n'importe quelle valeur convient : la vérification du compte échoue (« SMTP check failed » dans les logs) sans arrêter le serveur. `baseURL` doit être en `https://`, même si l'essai se fait en HTTP. Dans un autre terminal :

```sh
curl -sS -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8090/healthz
curl -sS -D - http://127.0.0.1:8090/tribes/demo/
```

Attendu : `200` pour `/healthz` ; la page de la tribu `demo` avec `Cache-Control: no-cache`, qui référence `main-<empreinte>.js` et `app-<empreinte>.css` ; l'un de ces fichiers, demandé sous `/tribes/demo/`, avec `Cache-Control: public, max-age=31536000, immutable`. Ctrl-C arrête le serveur.

## Workflows planifiés

Sur un dépôt public, GitHub désactive un workflow planifié après 60 jours sans activité dans le dépôt (`ci.yml` hebdomadaire, `devcontainer.yml`). Pour le réactiver : onglet *Actions*, choisir le workflow, « Enable workflow » ; ou depuis l'hôte :

```sh
gh workflow enable ci.yml
gh workflow enable devcontainer.yml
```

Le jeton du Dev Container n'a pas le droit de le faire (ADR 0020).
