# 0011. Dépôt public et licence AGPL

- **Date** : 2026-09-27
- **Statut** : accepté

## Contexte

Le dépôt, jusqu'ici privé, doit devenir public. Son code source ne contient pas de secret, mais :

- l'historique expose les adresses e-mail des auteurs des commits ;
- la configuration de déploiement (Caddyfile, unité systemd) et les ADR décrivent l'infrastructure ;
- sur un dépôt public, n'importe qui peut proposer une PR depuis un fork, et donc faire exécuter du code par la CI.

L'intention est qu'une version modifiée de l'application, même seulement exploitée en ligne, reste libre. Toutes les dépendances sont sous licences permissives (BSD, MIT, Apache-2.0) ; les polices sont sous OFL.

## Décision

1. **Dépôt public**, sous licence **`AGPL-3.0-or-later`** : fichier `LICENSE` à la racine, texte intégral de la licence.
2. **Lien vers le code source dans l'application et sur le site public**, pointant vers la version exacte déployée (étiquette ou commit), comme l'exige la clause réseau de l'AGPL.
3. **Polices** : distribuées avec leur licence OFL, dans le dossier des polices.
4. **Avant la publication** :
   - Git configuré avec l'adresse « noreply » de GitHub pour tous les commits futurs ; l'historique existant, qui contient des adresses personnelles et professionnelles, est conservé tel quel (risque accepté) ;
   - détection de secrets et blocage des pushes qui en contiennent activés (*secret scanning*, *push protection*) ;
   - `SECURITY.md` et signalement privé des vulnérabilités activé.
5. **Rien de propre à l'instance dans le dépôt** : IP du serveur, utilisateur SSH, clés et toute valeur de configuration de production vivent dans les variables et secrets GitHub, jamais dans les fichiers versionnés. Aucun fichier `.env` versionné. Aucun secret dans le code front.
6. **Toutes les protections disponibles sont appliquées avant la publication**, selon la liste tenue dans `docs/securite-depot.md` (accès, règles de branches et d'étiquettes, contributions externes, Actions, sécurité du code et des dépendances). Les points essentiels de la CI sont rappelés ci-dessous.
   **Durcissement de la CI** :
   - approbation requise avant l'exécution des workflows déclenchés par des contributeurs externes ;
   - jamais de déclencheur `pull_request_target` ;
   - droits du jeton de CI en lecture seule par défaut, élargis workflow par workflow ;
   - actions tierces épinglées par empreinte de commit ;
   - pas de runner auto-hébergé ;
   - secrets de déploiement dans un environnement GitHub protégé, accessible aux seuls workflows de release, jamais aux PR.
7. **Contributions sous DCO** (*Developer Certificate of Origin*) : chaque commit porte une ligne `Signed-off-by`, par laquelle son auteur certifie avoir le droit de le contribuer sous la licence du projet. Pas de CLA : aucune double licence commerciale n'est envisagée. Règle décrite dans un `CONTRIBUTING.md` et vérifiée en CI sur les PR externes.
8. **Sourcemaps publiées** avec l'application (exclues du pré-cache du service worker) : le code étant public, elles n'exposent rien de plus et facilitent le débogage sur téléphone.

## Alternatives envisagées

- **Dépôt privé** : écarté. Pas d'exposition, mais minutes de CI limitées et fonctions de sécurité (détection de secrets, analyse de code, attestation de provenance) payantes ou réservées aux offres Entreprise.
- **GPL-3.0** : écartée. Le copyleft ne s'applique qu'à la distribution ; une version modifiée exploitée en ligne échapperait à l'obligation de publier.
- **EUPL-1.2** : envisagée (copyleft couvrant l'usage en ligne, droit européen, texte français faisant foi). Écartée : sa clause de compatibilité peut affaiblir le copyleft dans certaines combinaisons.
- **CeCILL v2.1** : écartée, sans clause réseau.
- **MIT, Apache-2.0** : écartées, ne correspondant pas à l'intention copyleft.
- **CLA (accord de cession de droits aux contributeurs)** : écarté. Utile seulement pour garder la liberté de changer de licence ou de proposer une double licence, ce qui n'est pas prévu ; plus lourd et parfois mal perçu.
- **Réécriture de l'historique pour masquer les adresses e-mail** : écartée, disproportionnée.

## Conséquences

- **Positif** : minutes de CI illimitées ; détection de secrets et analyse de code gratuites ; attestation de provenance possible sans offre Entreprise (reportée, ADR 0012) ; code auditable.
- **Négatif** : infrastructure et code lisibles par des attaquants potentiels ; adresses e-mail de l'historique exposées ; CI exposée aux PR externes, d'où les règles du point 6.
- **Contributions externes** : sans CLA, tout changement de licence ultérieur exigera l'accord de chaque contributeur ; accepté, puisqu'aucune double licence n'est prévue.
- **Point ouvert** : licence de la documentation et des maquettes (AGPL comme le reste, ou CC BY-SA 4.0).
