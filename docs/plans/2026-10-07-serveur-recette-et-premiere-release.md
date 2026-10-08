# Plan : serveur, recette, ouverture de la production et première release

- **Nom** : `recette`
- **Date** : 2026-10-07
- **Statut** : prêt (décisions D1 à D18 confirmées par le développeur le 2026-10-08)

## Objectif

Mettre l'application en ligne, en recette **et en production**, sur un serveur provisionné par script, et publier la première version par la chaîne de release : tout ce que les plans précédents ont renvoyé « à la recette » est fait ou prouvé ici, et on se place au plus tôt dans les conditions de la production (D1).

À la fin du plan :

- le serveur est construit par `deploy/provision.sh`, sans geste manuel hors de la check-list, et peut être reconstruit de la même façon (ADR 0015) ;
- `https://recette.meltingtribe.codingmatters.org` sert les archives de PR, `https://meltingtribe.codingmatters.org` sert les versions publiées : le site public minimal à `/` (présentation, mentions légales, Confidentialité) et l'application sous `/tribes/` ;
- en production, une seule tribu, celle du développeur ; la famille est invitée plus tard, quand le planning existe (D1) ;
- un membre reçoit son code sur une vraie messagerie et se connecte depuis un téléphone, Safari compris ;
- `tribe-menus-deploy` installe l'archive d'une PR (recette) ou une version publiée (recette et production), prend ses instantanés, et revient en arrière seul quand la version ne démarre pas (ADR 0016) ;
- les alertes arrivent : celles de l'application (ADR 0023), relayées par le serveur, et celles du serveur lui-même (panne, disque, mises à jour) ;
- la version `v0.1.0` est publiée par une étiquette et déployée en production ;
- un traitement accéléré de vulnérabilité a été répété de bout en bout sur un faux constat, jusqu'à la production, et sa durée est mesurée (ADR 0022) ;
- les points que la feuille de route reporte à `recette` sont fermés, ou reportés avec leur raison.

**Qui fait quoi.**

- **Claude Code dans le Dev Container** écrit le code et les scripts, en cinq PR (lots A à E, D18), prépare les PR de release et lance les contrôles qui se font de l'extérieur du serveur.
- **Une session cloud**, qui n'a pas écrit le code, relit chaque lot selon `docs/revue-de-pr.md`.
- **Le développeur** arbitre, fusionne, et fait ce que personne d'autre ne peut faire : espace client OVHcloud, SSH sur le serveur, téléphones, étiquettes. Ces gestes sont **regroupés en cinq séances**, chacune avec sa check-list (section « Étapes manuelles », D13).

**Hors périmètre**, renvoyé à l'invitation d'un autre membre que le développeur : fermer une session à distance et révoquer un membre (`revue-securite`, constat 5). Renvoyé plus loin : durcissement de DMARC (ADR 0014, point 3), sauvegarde hors site (ADR 0007), sonde externe (ADR 0015), attestation de provenance (ADR 0012).

## Contexte et contraintes

- À lire avant de commencer : ADR 0006, 0007, 0011 (points 2 et 5), 0012, 0014, 0015, 0016, 0017, 0022, 0023 ; `docs/plans/2026-10-06-application-prete-pour-la-production.md` (« Ce que ce plan ne prouve pas », « Ce que ce plan demande à `recette` », notes des revues des PR #46 et #47) ; `docs/plans/2026-10-04-revue-de-securite-avant-mise-en-ligne.md` (constats 4, 6, 9 et 11, « Ce qui n'a pas pu être vérifié ») ; `docs/traitement-des-vulnerabilites.md` ; `RELEASING.md` ; `docs/securite-depot.md` ; `docs/design/README.md` (site public) et les maquettes `identite/Site`, `SiteMobile`, `Mentions` et `Confidentialite`.
- **Git** : une branche `feature/…` et une PR par lot ; un commit par étape, avec `git commit -s` ; une ligne dans `CHANGELOG.md` par PR, qui cite le lot (`recette/B`) ; `make ci` avant de pousser.
- **Rien de propre à l'instance dans le dépôt** (ADR 0011, point 5, précisé à l'étape 14) : ni adresse IP, ni compte SSH, ni adresse de l'administrateur dans `deploy/`, dont les exemples utilisent `example.org`. Les valeurs de l'instance vivent dans un fichier du serveur (D4). Les rapports des contrôles (étapes 20 et 21) sont écrits pour être copiés dans une PR : ils ne citent aucune de ces valeurs.
- **Aucun secret hors du serveur et du gestionnaire de mots de passe du développeur** : ni dans le dépôt, ni dans une conversation, ni sur une ligne de commande, ni dans l'historique du shell (D5).
- **Vulnérabilités** : jusqu'au déploiement de `v0.1.0` en production (séance 4), rien n'est en production et tout suit le flux public (ADR 0022, point 6). **Ensuite, le signal d'arrêt s'applique** : un soupçon de vulnérabilité sur une version en production ne s'écrit plus sur GitHub, il est remis au développeur dans la conversation (`docs/traitement-des-vulnerabilites.md`). Le lot E et la répétition sont concernés.
- **Aucune dépendance ajoutée au binaire** (ADR 0002). Sur le serveur, les paquets de Debian 13 et ceux du dépôt du projet Caddy (D15), rien de téléchargé puis exécuté hors d'un dépôt de paquets signé.
- **Workflows** : ce plan ne modifie pas `.github/workflows/`. Rien n'est donc à pousser depuis l'hôte, hors les étiquettes.
- **Sessions cloud** : elles n'atteignent ni le serveur ni les registres de paquets. Elles relisent ; les contrôles qui demandent le réseau sont lancés par Claude Code dans le Dev Container.
- **Un script de serveur ne se prouve que sur un serveur.** Les lots B et C sont relus et passent `shellcheck`, mais leur vraie preuve est la séance 2 : elle se fait sur l'archive de la PR du lot C **avant sa fusion**, et ce que le serveur révèle se corrige dans cette PR.
- **Dix demandes de code par heure et par adresse IP** (ENF-01), par instance : les séances et le contrôle externe partagent ce budget s'ils partent de la même connexion vers la même instance (check-list S3).

### Ce que ce plan reprend

| Origine | Sujet | Étape |
| --- | --- | --- |
| `revue-securite`, constat 6 | cookie sans préfixe `__Host-`, recette sous-domaine de la production | 1, 2 |
| `revue-securite`, constat 9 | tribu de démonstration en `@exemple.fr` | 3, S1 |
| `revue-securite`, constat 4 | dossier de données en 0700, `UMask=0077` | 10, 11 |
| `revue-securite`, constat 11 | HSTS, en-têtes du site, `Host`, `/healthz`, délais et tailles, journaux d'accès | 12, 21 |
| `revue-securite`, non vérifié | temps de réponse, membre ou non, tribu existante ou non | 21, 23 |
| `production`, étape 14 | envoi réel par le compte du MX Plan | S2, S3 |
| `production`, non prouvé | activation de socket, credentials, `X-Forwarded-For` de Caddy, relais des alertes, première release | 11, 12, 17, 21, S4 |
| `production`, revue de la PR #46 | réponses du MX Plan à un destinataire refusé, `EHLO localhost` | S3, 31 |
| `production`, revue de la PR #47 | fenêtre des compteurs de 50 à 60 minutes ; destinataire refusé compté comme échec d'envoi | 34, 31 |
| `revue-securite`, revue de la PR #40 | un seul cookie de demande de code par navigateur et par tribu | S3 |
| `socle`, D14 | connexion sous Safari, scénarios `@manuel` | S3 |
| `canal-prive` | déploiement tenant dans une séance, recette ciblée, retour arrière fiable, mesure d'attente, répétition d'un traitement accéléré | 16, 18, S4, S5 |
| ADR 0007, 0014 | VPS, DNS (A, AAAA, CAA, DNSSEC), SPF, DKIM, DMARC, boîtes `no-reply@` et `server@` | S1 |
| Feuille de route, « Ouverture de la production » | site public, boîte `contact@`, première release déployée en production | 25 à 28, S1, S4 |

### Constat fait en écrivant le plan

L'ADR 0012 (point 6) veut qu'une migration « ajoute sans casser », pour qu'un retour à la version précédente reste possible sur la base migrée. **Le binaire ne le permet pas** : `migrate` (`internal/storage/migrate.go`) refuse de démarrer sur une base dont la version de schéma dépasse la sienne (`schema version N is newer than this binary`). Un retour arrière après une version qui apporte une migration échoue donc, à moins de restaurer l'instantané. D7 tranche, et le script de déploiement en tient compte (étape 16).

## Étapes manuelles : cinq séances

Tout ce qui demande le développeur en personne est regroupé ici. Hors de ces séances, il ne lui reste que les gestes ordinaires : demander et arbitrer les revues, relire les textes du site public dans la PR du lot D, fusionner les PR.

| Séance | Sujet | Où | Quand | Durée visée |
| --- | --- | --- | --- | --- |
| **S1** | Comptes et DNS | espace client OVHcloud, GitHub, poste | dès maintenant, pendant les lots A et B | 1 h |
| **S2** | Serveur | SSH depuis l'hôte | PR du lot C ouverte, CI verte | 1 h 30 |
| **S3** | Recette sur appareils | téléphones, ordinateur, une commande SSH | PR `release/v0.1.0` ouverte | 1 h |
| **S4** | Première release, ouverture de la production, retours arrière | hôte, SSH, téléphone | S3 sans défaut ouvert | 2 h |
| **S5** | Répétition d'un traitement accéléré | hôte, SSH, téléphone | correctif prêt en local | 1 h, chronométrée |

Les durées sont des estimations, à corriger dans les notes d'exécution. S3 et S4 peuvent se suivre le même jour, S4 et S5 aussi.

**Règles communes.**

- **Avant chaque séance**, Claude Code donne ce qu'elle demande : numéro de PR, identifiant de l'exécution de la CI, commandes exactes à jour (`deploy/README.md`).
- **Rien ne se corrige à la main sur le serveur.** Ce qui manque ou échoue est un défaut d'un script : il est noté, corrigé dans la PR, redéployé. Si une correction manuelle a quand même été nécessaire pour avancer, et tant que la production n'a pas de données, le VPS est réinstallé depuis l'espace client et S2 est rejouée : c'est la preuve qu'il se reconstruit.
- **Après chaque séance**, le développeur remet à Claude Code le rapport de `tribe-menus-check` et ce qu'il a observé ; Claude Code coche la check-list dans ce plan et consigne le reste dans les notes d'exécution.
- **Les commandes sont écrites ici telles que le plan les propose** ; leur forme définitive est celle de `deploy/README.md`, que les lots B et C écrivent.

### S1. Comptes et DNS

La zone DNS et DNSSEC mettent des heures à se propager : c'est la séance à faire en premier.

Poste :

- [ ] Clé SSH ed25519 réservée au serveur, protégée par une phrase de passe, créée sur l'hôte (pas dans le Dev Container).

Serveur (ADR 0007) :

- [ ] VPS-1 commandé : datacenter français, Debian 13, clé publique déposée à la commande. Conditions d'engagement relues avant de valider.
- [ ] Sauvegarde automatique quotidienne active sur le VPS.
- [ ] Adresses IPv4 et IPv6 notées dans le gestionnaire de mots de passe, nulle part ailleurs.

E-mail (ADR 0014, 0015) :

- [ ] Boîte `no-reply@codingmatters.org` créée dans le MX Plan ; mot de passe généré, dans le gestionnaire.
- [ ] Boîte `server@codingmatters.org` créée de même.
- [ ] Boîte `contact@codingmatters.org` créée de même, avant la publication du site (`docs/design/README.md`).
- [ ] Adresse de l'administrateur choisie : elle recevra les alertes de l'application et du serveur, et les rapports DMARC.
- [ ] Adresses de recette (D3) : redirections `recette-alice@`, `recette-bruno@`, `recette-chloe@` et `recette-david@codingmatters.org` vers la boîte du développeur ; aucune redirection pour `recette-inconnu@codingmatters.org`, qui doit rester inexistante.

Zone `codingmatters.org` :

- [ ] Existant relevé avant toute modification (SPF, DKIM, DMARC, CAA) : le domaine porte déjà du courrier.
- [ ] Enregistrements A et AAAA de `meltingtribe.codingmatters.org` et de `recette.meltingtribe.codingmatters.org` vers le VPS (ADR 0017, point 3).
- [ ] Certificats déjà émis pour le domaine consultés (journaux Certificate Transparency, `crt.sh`), puis CAA n'autorisant que `letsencrypt.org`. Si un autre nom du domaine obtient ses certificats ailleurs : CAA posé sur `meltingtribe.codingmatters.org` seulement, et noté.
- [ ] DNSSEC activé.
- [ ] SPF autorisant les serveurs d'envoi du MX Plan.
- [ ] DKIM activé pour le MX Plan.
- [ ] DMARC en observation (`p=none`), rapports vers l'adresse de l'administrateur.

GitHub (ADR 0016, point 6) :

- [ ] Jeton à portée fine réservé au serveur : dépôt `tribe-menus` seul, *Actions* et *Metadata* en lecture, avec une date d'expiration ; dans le gestionnaire jusqu'à S2.

Ce que la séance ne vérifie pas elle-même : l'émission des certificats en S2 prouve A, AAAA et CAA ; le contrôle externe (étape 23) relit DNSSEC, SPF, DKIM et DMARC.

### S2. Serveur

Prérequis : S1 faite, lot B fusionné, PR du lot C ouverte avec une CI verte.

Amorçage :

- [ ] Archive de la PR du lot C téléchargée sur l'hôte (`gh run download`), empreinte vérifiée (`sha256sum -c`), copiée sur le serveur (`scp`).
- [ ] Première connexion SSH avec le compte de l'image ; empreinte de la clé d'hôte du serveur notée dans le gestionnaire.
- [ ] `/etc/tribe-menus/instance.conf` écrit à partir de `deploy/instance.example.conf` (D4).
- [ ] `sudo ./deploy/provision.sh production recette`, depuis l'archive déballée.

Accès (D14) :

- [ ] Dans un second terminal, **avant de fermer le premier** : connexion par clé avec le compte d'administration personnel ; connexion refusée pour `root` et par mot de passe.
- [ ] Mot de passe `sudo` de ce compte choisi et rangé dans le gestionnaire.
- [ ] `provision.sh` relancé depuis ce compte : il ne change rien d'autre que le retrait du compte de l'image.

Secrets (D5), saisis au clavier sur le serveur, jamais collés dans une commande :

- [ ] mot de passe de `no-reply@`, pour chaque environnement (`sudo tribe-menus-credential recette smtp-password`, puis `production`) ;
- [ ] mot de passe de `server@` ;
- [ ] jeton GitHub de S1.

Recette :

- [ ] `sudo tribe-menus-deploy --env recette --pr <numéro>`.
- [ ] Fichier des membres de la tribu de recette écrit sur le serveur (D3) : les adresses `recette-…@codingmatters.org`, une adresse directe à soi par messagerie à essayer en S3, et `recette-inconnu@codingmatters.org`.
- [ ] `sudo tribe-menus-admin recette seed -members <fichier>`.
- [ ] Depuis l'ordinateur : `https://recette.meltingtribe.codingmatters.org/tribes/demo/` s'ouvre avec un certificat valide ; un code demandé arrive, la connexion réussit (`production`, étape 14).

Production, sans version encore (D1) :

- [ ] `https://meltingtribe.codingmatters.org/` répond avec un certificat valide et la page d'attente : aucune version n'est installée avant S4.

Contrôle :

- [ ] `sudo reboot` ; au retour, la recette est servie sans rien relancer à la main.
- [ ] `sudo tribe-menus-check` : aucune ligne en échec ; l'e-mail d'essai du serveur est reçu.
- [ ] Rapport remis à Claude Code.

### S3. Recette sur appareils

Prérequis : PR `release/v0.1.0` ouverte, CI verte. C'est la recette de la première release (`RELEASING.md`, étape 6 ; D11).

**Budget** : dix demandes de code par heure depuis une même connexion. Téléphones en données mobiles, ordinateur sur la connexion du domicile ; l'essai des limites vient en dernier.

- [ ] `sudo tribe-menus-deploy --env recette --pr <numéro de la PR de release>`.

Réception du code (ADR 0014, point 6), sur les adresses directes, une ligne par messagerie réellement utilisée par la famille :

- [ ] Gmail : reçu en moins d'une minute, dans la boîte de réception ; SPF, DKIM et DMARC « pass » dans les en-têtes du message.
- [ ] Outlook : de même.
- [ ] iCloud : de même.
- [ ] Autre messagerie de la famille, s'il y en a une : de même.
- [ ] Une adresse `recette-…@codingmatters.org` : reçue par la redirection.
- [ ] Message lisible sur téléphone : objet, code à 8 chiffres, durée de validité, ni lien ni nom de tribu.
- [ ] Code demandé pour `recette-inconnu@codingmatters.org` : l'écran ne dit rien de plus que pour une autre adresse (ENF-02).

Connexion :

- [ ] iPhone, Safari : connexion complète ; les huit cases restent alignées (`socle`, D14).
- [ ] Android, Chrome : de même.
- [ ] Ordinateur, deux navigateurs dont Safari ou Firefox.
- [ ] Dans l'inspecteur de l'ordinateur : cookies préfixés `__Host-`, `Secure`, `HttpOnly`, `SameSite=Lax`, `Path=/` (D2).
- [ ] `@manuel` « La session survit au redémarrage du navigateur » : navigateur fermé complètement puis rouvert sur l'URL de la tribu, sous Safari et sous Chrome.
- [ ] Code demandé sur un appareil et saisi sur un autre : il est refusé sans essai consommé, et le message suffit à comprendre qu'il faut le redemander (`revue-securite`, D5).
- [ ] Deux demandes de code depuis deux onglets du même navigateur, pour deux adresses : comportement noté (revue de la PR #40).
- [ ] Déconnexion, puis retour à l'écran de connexion.
- [ ] Lien « code source » de l'écran de connexion : il mène à la version déployée.

Site public, sur le téléphone et l'ordinateur :

- [ ] Présentation, mentions légales et Confidentialité lisibles ; liens du pied de page justes, dont « Code source de cette version » (D17).

Limites et alertes, en dernier :

- [ ] « Renvoyer un code » jusqu'au refus : « Réessayez dans quelques minutes » à la quatrième demande.
- [ ] Dix refus de suite (commande fournie par Claude Code, lancée du poste) : e-mail d'alerte de l'application reçu dans les dix minutes, sans adresse, IP ni tribu ; puis le même signal relayé par le serveur.

Rapport :

- [ ] `sudo tribe-menus-check report` sur le serveur, remis à Claude Code : réponses SMTP, alertes, erreurs et avertissements de l'heure.

### S4. Première release, ouverture de la production et retours arrière

Prérequis : S3 sans défaut ouvert (lot E fusionné s'il y en avait), PR de release à jour avec `main` et verte, deux PR d'essai prêtes (étape 32). À partir du déploiement de `v0.1.0` en production, le signal d'arrêt des vulnérabilités s'applique.

Release (`RELEASING.md`, étapes 7 à 10) :

- [ ] PR `release/v0.1.0` fusionnée.
- [ ] `origin/main` est bien le commit de fusion ; étiquette annotée `v0.1.0` posée et poussée depuis l'hôte.
- [ ] `release.yml` au vert : c'est le premier passage du contrôle de l'étiquette, de l'extraction des notes et du job `publish`.
- [ ] Release vérifiée par Claude Code (empreinte, `tribe-menus version`).

Production :

- [ ] `sudo tribe-menus-deploy v0.1.0`, chronométré.
- [ ] `sudo tribe-menus-deploy --pr <numéro>` (sans `--env`, donc en production) : refusé.
- [ ] Site public servi à `https://meltingtribe.codingmatters.org/` ; « Code source de cette version » mène à l'étiquette `v0.1.0`.
- [ ] `sudo tribe-menus-admin production init` : votre tribu, avec votre adresse.
- [ ] Connexion à votre tribu depuis l'iPhone, en données mobiles.
- [ ] `make probe` lancé par Claude Code sur la production : rapport sans échec.

Retours arrière, en recette (ADR 0016 ; D7) :

- [ ] PR d'essai « migration » déployée, puis `sudo tribe-menus-deploy --env recette --rollback` : refusé tant que la restauration des données n'est pas demandée ; avec elle, la version précédente revient et la connexion fonctionne.
- [ ] PR d'essai « ne démarre pas » déployée : le script revient seul à la version précédente, le dit, et la connexion fonctionne.
- [ ] Les deux PR d'essai fermées sans fusion, branches supprimées.

Mesures d'attente et pannes (`canal-prive`) :

- [ ] `sudo tribe-menus-maintenance production on` : la page d'attente est servie, et une requête ne relance pas le service ; `off` : l'application répond de nouveau.
- [ ] `sudo tribe-menus-check drill` : e-mails de panne du service et d'essai du disque reçus.
- [ ] Durées notées : `release.yml`, déploiement, retour arrière.

### S5. Répétition d'un traitement accéléré

Prérequis : faux constat remis et correctif prêt sur une branche locale, non poussé (étape 34). La séance suit `docs/traitement-des-vulnerabilites.md`, niveau accéléré, étapes 5 à 9, jusqu'à la vraie production (D12).

Chronométré :

- [ ] Heure de début notée ; Claude Code prépare la version `v0.1.1` dans la branche, pousse et ouvre la PR, en termes neutres.
- [ ] Relecture demandée à la session cloud, en termes neutres ; arbitrage.
- [ ] CI verte : archive de la PR déployée en recette, et seul le scénario touché par le correctif est rejoué.
- [ ] Fusion, étiquette `v0.1.1`, `release.yml` au vert.
- [ ] `sudo tribe-menus-deploy v0.1.1` ; le défaut du faux constat n'est plus là.
- [ ] Heure de fin notée, avec la durée de chaque phase.

Hors chronomètre :

- [ ] `sudo tribe-menus-deploy --rollback` en production : `v0.1.0` revient sans demander de restauration, aucune migration n'ayant changé le schéma ; puis `sudo tribe-menus-deploy v0.1.1`.
- [ ] Publication demandée : commentaire « Analyse » sur la PR, marqué comme une répétition ; ligne de `CHANGELOG.md` complétée par une PR ordinaire.

## Étapes

### Lot A : ce que l'application doit changer avant la première session réelle (`feature/cookies-host-et-tribu-de-recette`)

Points d'attention de la revue : un nom de cookie pour toute valeur de l'identifiant d'URL, hors format compris ; réponses toujours identiques pour un membre, une autre adresse et une tribu inexistante, en-têtes `Set-Cookie` compris ; ce que `admin seed` accepte dans le fichier des membres.

- [x] **1. ADR 0024 et specs : cookies préfixés `__Host-`** (D2 ; `revue-securite`, constat 6).
  - L'ADR remplace l'ADR 0006 (point 5) : le cookie de session et celui de la demande de code portent le préfixe `__Host-`, sans attribut `Domain`, avec `Path=/` et un nom propre à la tribu. ADR 0001 et 0006 reçoivent un renvoi daté.
  - Le nom vaut pour toute valeur de l'identifiant d'URL, hors format comprise, sans rien révéler de plus (ENF-02). La forme du suffixe se fixe dans l'ADR ; proposée : une empreinte courte de l'identifiant.
  - ENF-01 (attributs du cookie) et `authentification.feature` : le scénario « Le cookie de session n'est pas lisible par le code de la page » est complété (préfixe, `Path=/`), et un scénario prouve que deux tribus ouvertes dans le même navigateur gardent chacune leur session.
- [x] **2. Cookies `__Host-`**, `internal/server/api.go`. Test d'abord : un cookie de l'ancien nom n'ouvre aucune session ; le cookie d'une tribu présenté à une autre reçoit toujours 401. Harnais godog et tests Playwright accordés.
- [x] **3. Tribu de recette** (D3 ; constat 9). `admin seed -members <fichier>` : une ligne par membre, adresse puis nom d'affichage, le premier étant le premier membre ; chaque adresse passe par `ParseEmail`. Avec `-config`, l'option est obligatoire : les adresses en `@exemple.fr` n'arrivent jamais sur une instance qui envoie de vrais e-mails. Sans `-config`, `make seed` ne change pas.
- [x] **4. Retour arrière et migrations** (D7). ADR 0012 (point 6) et `RELEASING.md` (étape 3) reçoivent une précision datée : le binaire refuse une base plus récente que lui ; revenir sur une version qui a migré restaure l'instantané. Un test fixe ce refus, s'il n'existe pas.
- [x] **5. Documentation du lot** : `CLAUDE.md`, `CHANGELOG.md`, `docs/design/README.md` si le texte de la page Confidentialité nomme les cookies ; feuille de route (points reportés des constats 6 et 9 retirés) ; cases cochées.

### Lot B : provisionnement du serveur (`feature/provisionnement-du-serveur`)

Points d'attention de la revue : ce qui tourne en `root` et sur quelles entrées (les valeurs de `instance.conf` ne sont jamais évaluées par le shell) ; origine et signature de chaque paquet ; droits de chaque fichier qui porte ou protège un secret ; risque de se fermer la porte SSH ; règles du pare-feu en IPv6 ; qui peut ouvrir le socket de chaque instance, donc écrire `X-Forwarded-For` ; ce que le service confiné peut encore écrire ; ce qu'une instance peut lire de l'autre.

- [ ] **6. Outillage des scripts** (D10). `shellcheck` dans l'image du Dev Container et dans `make lint`, sur les scripts de `deploy/` ; `make tools` l'installe s'il manque au runner. Scripts en `bash`, `set -euo pipefail`, marqués exécutables dans le dépôt : l'archive garde ce droit (ADR 0012, point 3).
- [ ] **7. Valeurs de l'instance** (D4). `deploy/instance.example.conf` : noms d'hôte de la production et de la recette, adresse de l'administrateur, serveur et comptes SMTP de l'application et du serveur, nom du compte d'administration. `provision.sh` lit `/etc/tribe-menus/instance.conf`, refuse une clé inconnue ou manquante en la nommant, et en tire le `config.json` de chaque environnement (`jq`), les blocs de site de Caddy et la configuration de `msmtp`. Ces fichiers sont produits, jamais retouchés à la main.
- [ ] **8. `deploy/provision.sh` : système** (ADR 0015, points 1 à 7). Idempotent, relançable, il prend en argument les environnements à créer (`production recette`).
  - Paquets : ceux de Debian 13 (`nftables`, `unattended-upgrades`, `msmtp-mta`, `sqlite3`, `jq`, `curl`, `unzip`) et Caddy (étape 12).
  - SSH : clé seule, ni mot de passe ni `root`, compte d'administration personnel seul admis, `sudo` avec mot de passe (D14) ; le compte de l'image est retiré au second lancement, fait depuis ce compte.
  - `nftables` : 22, 80 et 443 en TCP en entrée, ICMP et ICMPv6 nécessaires, IPv4 et IPv6 ; aucun port UDP (D16).
  - `unattended-upgrades` : correctifs de sécurité de Debian et paquets du dépôt de Caddy ; redémarrage à 3 h 30 si nécessaire, annoncé par e-mail.
  - journald persistant, plafonné, un mois de conservation (PT-07) ; synchronisation de l'heure.
- [ ] **9. Secrets et relais d'envoi** (D5 ; ADR 0015, points 6 et 10).
  - `tribe-menus-credential` : lit un secret au clavier, sans écho, et l'écrit chiffré par `systemd-creds` avec la clé de la machine. Secrets : `smtp-password` de chaque environnement, mot de passe de `server@`, jeton GitHub.
  - `msmtp` vers le MX Plan avec le compte `server@`, mot de passe lu dans son credential, jamais en clair sur le disque ; le courrier de `root` va à l'administrateur.
- [ ] **10. Utilisateurs et emplacements** (ADR 0015, points 8 et 9). Par environnement : utilisateur système `tribe-menus-<environnement>`, sans shell ; `/opt/tribe-menus/<environnement>/releases/`, `/etc/tribe-menus/<environnement>/`, `/var/lib/tribe-menus/<environnement>/` en 0700, `/var/www/tribe-menus/<environnement>/` (D8). Une instance ne lit rien de l'autre.
- [ ] **11. Unités systemd** (ADR 0015, points 10 à 12 ; `production`, « Ce que ce plan demande »).
  - `tribe-menus@.socket` : socket Unix, ouvert au seul groupe de Caddy.
  - `tribe-menus@.service` : `serve -config /etc/tribe-menus/%i/config.json`, credential `smtp-password`, `UMask=0077` (constat 4), `TimeoutStopSec` de 45 secondes au moins, redémarrage sur échec avec une limite, `OnFailure=` vers l'alerte de l'étape 17.
  - Confinement : aucun privilège ni capacité, système de fichiers en lecture seule hors du répertoire de données, `/home` invisible, `/tmp` privé, appels système `@system-service`, familles réseau Unix, IPv4 et IPv6. La note de `systemd-analyze security` est relevée ; chaque réglage laissé ouvert est justifié dans `deploy/README.md`.
  - Le service ne démarre pas tant que `config.json`, le credential ou une version installée manque, et le dit.
- [ ] **12. Caddy** (ADR 0006 ; constat 11 ; D15, D16). Paquet du dépôt du projet Caddy, clé de signature épinglée par son empreinte, sans plugin.
  - Options globales : `admin off`, adresse de contact ACME, HTTP/1.1 et HTTP/2 seuls, délais de lecture et d'inactivité, taille maximale du corps.
  - Un bloc par nom d'hôte, chacun vers le socket de son instance : `/tribes/*` transmis au socket, `Host` tel quel, adresse du client écrite par Caddy dans `X-Forwarded-For` à la place de ce que le client envoie ; `/healthz` jamais transmis ; `/` servi depuis le site de l'environnement (D8), avec ses en-têtes de sécurité, dont une CSP propre au site.
  - HSTS sur les deux noms d'hôte, une journée d'abord (à allonger après quelques semaines sans incident, point reporté) ; `X-Robots-Tag: noindex` et `Disallow: /` sur tout le nom d'hôte de la recette.
  - Page d'attente quand le socket ne répond pas, ou qu'aucune version n'est installée (étape 18).
  - Journaux d'accès vers journald, sans en-tête `Cookie` ni `Authorization` : ils donnent l'heure, l'adresse IP et l'URL, ce que l'e-mail d'alerte promet (ADR 0023, point 6) et ce que décrit la page Confidentialité.
  - `caddy validate` sur la configuration produite, dans `provision.sh`.
- [ ] **13. `tribe-menus-admin`** (ADR 0015, point 13) : `sudo tribe-menus-admin <environnement> init|seed …` lance la sous-commande sous l'utilisateur du service, avec `-config`.
- [ ] **14. Documentation du lot** : `deploy/README.md` (contenu du dossier, première installation, saisie des secrets) ; ADR 0015 précisé (D4, D5, D8, D14 à D16) ; ADR 0011 (point 5) précisé : les valeurs de l'instance vivent dans un fichier du serveur, pas chez GitHub, comme le veut l'ADR 0016 ; ADR 0006 (D15, D16) ; `CLAUDE.md` ; `CHANGELOG.md` ; cases cochées.
- [ ] **15. Séance 1**, par le développeur, en parallèle des lots A et B : check-list S1.

### Lot C : déploiement, supervision et contrôles (`feature/deploiement-et-supervision`)

Points d'attention de la revue : arguments validés avant de servir à un chemin ou à une URL ; jeton GitHub jamais dans une ligne de commande, l'environnement ou un journal ; à qui appartient la PR déployée ; refus d'une archive de PR en production ; déballage de l'archive (chemins absolus, `..`, liens) ; bascule du lien et retour arrière quand une étape échoue au milieu ; contenu du journal traité comme une donnée par le relais ; rapports sans valeur de l'instance.

- [ ] **16. `tribe-menus-deploy`** (ADR 0016 ; D1, D6, D7).
  - Environnement : `production` par défaut, `--env recette` sinon.
  - Sources : `<version>`, la release publiée, sans jeton, pour les deux environnements ; `--pr <numéro>`, l'artefact `tribe-menus-pr-<numéro>` de la CI, lu avec le jeton du serveur, **en recette seulement** (ADR 0016, point 4).
  - Une PR n'est déployée que si son auteur est le propriétaire du dépôt, que sa branche n'est pas celle d'un fork, et que l'artefact est celui de son dernier commit (ADR 0016, point 6).
  - Empreinte SHA-256 vérifiée avant le déballage ; pour une PR, elle ne prouve que l'intégrité du transfert, l'empreinte venant du même artefact.
  - Instantané `VACUUM INTO` du registre et de chaque base de tribu, sous l'utilisateur du service, dans `backups/avant-<version>/`, en 0700 ; pas d'instantané de la base de limitation (ADR 0021, point 6).
  - Installation dans `releases/<version>/` (pour une PR, le nom porte le commit), site de l'environnement mis à jour (D8), bascule atomique de `current`, redémarrage, `/healthz` appelé par le socket.
  - Échec de `/healthz` : retour à la version précédente, avec restauration de l'instantané si une version de schéma a changé, puis nouveau contrôle ; le script dit ce qu'il a fait et sort en erreur. Au premier déploiement, il n'y a pas de version précédente : le service reste arrêté et la page d'attente servie.
  - `--rollback` : version précédente ; si le schéma a changé depuis, refus sans `--restore-data`, qui restaure l'instantané et dit ce qui est perdu.
  - Cinq versions et leurs instantanés conservés.
  - Le script n'installe que l'application et son site. Unités, Caddy et scripts du serveur ne changent que par `provision.sh`, relancé depuis la version installée (D6).
- [ ] **17. Supervision** (ADR 0015, point 15 ; ADR 0023, point 1).
  - Relais des alertes de l'application (D9) : un minuteur lit le journal de chaque service depuis son dernier passage et envoie par `msmtp` chaque enregistrement portant l'attribut `alert`, avec le nom de l'environnement ; au plus un e-mail par nature et par heure.
  - Panne : `OnFailure=` envoie l'état du service et ses dernières lignes de journal.
  - Disque rempli à plus de 80 % : minuteur quotidien.
- [ ] **18. Mesures d'attente** (`canal-prive`). `tribe-menus-maintenance <environnement> on|off` arrête le socket **et** le service : arrêter le service seul ne suffit pas, la première requête le relancerait par le socket. Caddy sert alors la page d'attente, en 503, et le site public reste en ligne. `docs/traitement-des-vulnerabilites.md` (« Mesures d'attente ») reçoit les commandes exactes, retour arrière compris.
- [ ] **19. Procédures**, dans `deploy/README.md` : déployer, revenir en arrière, restaurer, mettre en attente, relancer le provisionnement, reconstruire le serveur. Avec des données en production, la reconstruction commence par un instantané copié hors du serveur et se termine par sa restauration. `RELEASING.md` y renvoie pour le déploiement, et demande aux notes de version de dire quand le provisionnement est à relancer, comme pour les migrations.
- [ ] **20. `tribe-menus-check`** (D10), le contrôle fait sur le serveur. Une ligne par point, réussi ou en échec, sans valeur de l'instance.
  - Sans argument : SSH effectif, règles du pare-feu et ports à l'écoute, état des unités et des minuteurs, droits des répertoires de données et des credentials, note de `systemd-analyze security`, configuration de Caddy, simulation d'`unattended-upgrades`, synchronisation de l'heure, envoi d'un e-mail d'essai.
  - `report` : sur l'heure écoulée et par environnement, réponses SMTP par étape et par code, alertes, erreurs, avertissements « adresse du client inconnue » (`production`, étape 6).
  - `drill` : provoque l'alerte de panne sur la recette et l'essai de l'alerte du disque, puis remet le service en route.
- [ ] **21. Contrôle externe** (D10) : outil du projet en Go, cible `make probe` avec l'URL en variable, lancé du Dev Container sur l'un ou l'autre nom d'hôte.
  - TLS et redirection depuis HTTP ; HSTS ; en-têtes du site et de l'application ; `noindex` en recette ; `/healthz` introuvable ; corps trop gros et requête lente refusés ; aucun autre port ouvert parmi les plus courants.
  - `X-Forwarded-For` forgé sans effet : onze demandes de code portant chacune une adresse inventée, la onzième est refusée. L'essai consomme le budget de l'heure de la connexion d'où il part.
  - DNS, par un résolveur public : CAA, DNSSEC, SPF, DKIM, DMARC.
  - Temps de réponse (`revue-securite`, « Ce qui n'a pas pu être vérifié »), en recette : requêtes alternées pour un membre et une autre adresse, une tribu existante et une autre, sur `POST sessions` puis `POST login-codes`, au rythme que les limites laissent à un attaquant ; médianes, dispersion et verdict.
- [ ] **22. Séance 2**, par le développeur, sur l'archive de cette PR : check-list S2.
- [ ] **23. Contrôles après la séance 2**, par Claude Code : `make probe` sur la recette ; lecture du rapport de `tribe-menus-check`. Les défauts se corrigent dans la PR, qui est redéployée. Les deux rapports sont joints à la PR : c'est le contrôle du serveur en place, que la session cloud relit avec le code.
- [ ] **24. Documentation du lot** : ADR 0016 précisé (D1, D6, D7, D8) ; `docs/securite-depot.md` (jeton du serveur, section 1) ; `CLAUDE.md` (`make probe`, scripts de `deploy/`) ; `CHANGELOG.md` ; feuille de route (points reportés des constats 4 et 11 et du temps de réponse retirés) ; cases cochées.

### Lot D : site public minimal (`feature/site-public`)

Peut avancer en parallèle des lots B et C. Points d'attention de la revue : aucun script ni ressource tierce sur le site ; CSP du site ; aucune mesure d'audience ni cookie posé par le site ; textes conformes à ce que fait réellement l'application.

- [ ] **25. Pages du site**, dans `site/`, d'après les maquettes `identite/Site`, `SiteMobile`, `Mentions` et `Confidentialite` et `docs/design/README.md` (« Site public ») : présentation, mentions légales, Confidentialité. HTML et CSS statiques, sans JavaScript ; polices et fichiers d'identité servis depuis le site ; balises Open Graph avec `partage.png` ; `robots.txt` gardé.
- [ ] **26. Version dans les pages** (D17 ; ADR 0011, point 2). `site/*.html` porte les marques `{{VERSION}}` et `{{COMMIT}}`, que `internal/tools/dist` remplace en construisant l'archive : le pied de page « Code source de cette version » mène à l'étiquette, ou au commit pour une archive de PR. L'archive reste reproductible. Test : aucune marque ne reste dans l'archive, et deux exécutions donnent la même empreinte. La date de mise à jour des textes reste écrite à la main.
- [ ] **27. Textes** : mentions légales (éditeur à titre personnel, directeur de la publication, `contact@codingmatters.org`, hébergeur, licences) et Confidentialité (données traitées, deux cookies, durées de conservation, journaux du serveur un mois, droits), accordés à ce que font l'application et le serveur au terme de ce plan. **Relus par le développeur dans la PR**, avant la fusion.
- [ ] **28. Documentation du lot** : `docs/design/README.md` si une maquette a dû être interprétée ; `CLAUDE.md` ; `CHANGELOG.md` ; feuille de route ; cases cochées.

### Première release : `v0.1.0`

- [ ] **29. PR `release/v0.1.0`**, préparée par Claude Code selon `RELEASING.md` (étapes 1 à 5), une fois les lots A à D fusionnés : section du `CHANGELOG.md`, migrations depuis le début du projet, notes relues.
- [ ] **30. Séance 3**, par le développeur, sur l'archive de cette PR : check-list S3.

### Lot E : suites de la recette (`feature/suites-de-la-recette`)

Ce que les séances 2 et 3 ont appris, en une PR ordinaire ; la PR de release est ensuite mise à jour avec `main`, et seuls les scénarios touchés sont rejoués. Le lot disparaît s'il n'y a rien à corriger.

- [ ] **31. Corrections.**
  - Réponses du MX Plan à un destinataire refusé : si elles citent la partie locale de l'adresse, ne garder au journal que le code SMTP et le code d'état étendu (revue de la PR #46, point 2).
  - `EHLO localhost` : si un filtre s'en formalise, se présenter avec l'hôte de `baseURL` (point 3).
  - Échecs d'envoi : ne compter que les étapes de connexion et d'authentification, ou garder le sens large et l'écrire dans l'ADR 0023 (revue de la PR #47, point 3).
  - Tout défaut relevé sur les appareils, avec le scénario qui le prouve.

### Première release, suite : ouverture de la production et retours arrière

- [ ] **32. PR d'essai des retours arrière**, jamais fusionnées, préparées par Claude Code avec une CI verte :
  - « migration » : une migration sans effet de plus sur la base d'une tribu ;
  - « ne démarre pas » : une clé de configuration obligatoire de plus, absente du serveur.
- [ ] **33. Séance 4**, par le développeur : check-list S4. Claude Code vérifie la release publiée (`RELEASING.md`, étape 10), lance `make probe` sur la production et coche le job `publish` dans `docs/securite-depot.md`.

### Répétition d'un traitement accéléré : `v0.1.1`

- [ ] **34. Faux constat et correctif** (D12). Le développeur remet à Claude Code, dans la conversation, un constat écrit selon le modèle de `docs/traitement-des-vulnerabilites.md` : la fenêtre des compteurs d'alerte, qui ne couvre que 50 à 60 minutes (revue de la PR #47, point 2), jouée comme une vulnérabilité de `v0.1.0`. Claude Code écrit le test puis le correctif sur une branche locale, en termes neutres, lance `make ci` et **ne pousse pas**.
- [ ] **35. Séance 5**, par le développeur : check-list S5.
- [ ] **36. Bilan de la répétition** : durées dans les notes d'exécution ; la question ouverte de `canal-prive` sur la durée d'une séance de livraison reçoit sa réponse ; `docs/traitement-des-vulnerabilites.md` corrigé là où le déroulé écrit s'est écarté du déroulé vécu.
- [ ] **37. Clôture** : statut « terminé » ; feuille de route (plan terminé, points reportés à jour, « Ce que ce plan ne prouve pas » inscrit) ; `docs/securite-depot.md` relu ; `CLAUDE.md` ; `CHANGELOG.md`.

## Critères de validation

- `make ci` passe à la fin de chaque lot ; aucun scénario nouveau ne reste dans `acceptance/pending.txt`.
- **Serveur** : construit par la seule check-list S2, sans correction manuelle ; `tribe-menus-check` sans échec après un redémarrage ; seuls les ports 22, 80 et 443 en TCP répondent ; ni `root` ni mot de passe en SSH ; `sudo` demande un mot de passe.
- **Services** : chaque instance lancée par l'activation de socket, sous son utilisateur, avec son credential déchiffré ; répertoires de données en 0700 et fichiers en 0600 ; une instance ne lit rien de l'autre ; note de `systemd-analyze security` relevée et justifiée.
- **Caddy** : certificats de Let's Encrypt sur les deux noms d'hôte ; HSTS ; `/healthz` introuvable de l'extérieur ; un `X-Forwarded-For` forgé ne déplace pas la limite par IP, et une demande venue d'une autre connexion reste acceptée.
- **E-mail** : code reçu hors des indésirables sur chaque messagerie de la famille, SPF, DKIM et DMARC « pass » ; les journaux ne contiennent ni adresse ni code après un envoi réussi et un envoi refusé.
- **Alertes** : dix refus déclenchent l'e-mail de l'application et son relais par le serveur ; une panne du service et l'essai du disque arrivent par e-mail.
- **Cookies** : `__Host-`, un nom par tribu ; scénarios ENF-01 et ENF-02 inchangés par ailleurs.
- **Déploiement** : une PR du propriétaire se déploie en recette, celle d'un fork est refusée, toute PR est refusée en production ; une version qui ne démarre pas laisse la précédente en service, sans geste ; un retour sur une version migrée restaure l'instantané et le dit ; un retour sans migration garde les données.
- **Production** : `v0.1.0` publiée par `release.yml`, empreinte vérifiée, déployée ; site public servi, « Code source de cette version » mène à l'étiquette ; la tribu du développeur s'ouvre depuis un iPhone.
- **Appareils** : connexion sous Safari sur iPhone et sous Chrome sur Android ; scénario `@manuel` vérifié.
- **Traitement accéléré** : répété de la PR à la version déployée en production en une séance, durée notée ; rien de ce qui a été poussé avant le déploiement ne décrivait le faux constat.
- **Temps de réponse** : mesuré, verdict écrit ; un écart exploitable devient un constat, traité selon `docs/traitement-des-vulnerabilites.md` (flux public avant S4, signal d'arrêt ensuite).
- **Feuille de route** : les points reportés à `recette` en sont retirés ; ceux que ce plan reporte y sont inscrits.

### Ce que ce plan ne prouve pas

À inscrire dans la feuille de route :

- le retour arrière automatique quand une migration échoue à mi-chemin : la grille range une migration en « risque de livrer sans recette » élevé, donc hors du niveau urgent, qui est le seul à s'appuyer sur ce filet ;
- la reconstruction du serveur avec des données à reprendre : la procédure est écrite (étape 19), elle n'est jouée qu'avant l'arrivée des données ;
- la délivrabilité dans la durée : rapports DMARC à lire après quelques semaines, avant de durcir la politique (ADR 0014, point 3) ;
- les seuils des alertes (`production`, D8), à revoir après les premières semaines ;
- la durée de HSTS, à allonger après quelques semaines sans incident ;
- tout ce qui suppose un autre membre que le développeur : révocation d'un membre et fermeture d'une session à distance (`revue-securite`, constat 5), à faire avant d'inviter quelqu'un.

## Questions ouvertes

- **Filtrage des sorties** du serveur : aucun en V1 ; les services confinés n'ont besoin que du SMTP. À rouvrir si la surface change.
- **Avis de sécurité publié** après une version corrective (`canal-prive`, question ouverte) : la répétition de S5 ne le tranche pas.
- **Textes légaux** : écrits d'après `docs/design/README.md` et relus par le développeur (étape 27) ; aucune relecture juridique n'est prévue.

## Décisions

D1 à D13 proposées par la session qui a écrit le plan le 2026-10-07 ; confirmées par le développeur le 2026-10-08, sauf D1, qu'il a remplacée, et D3, qu'il a complétée ; D14 à D18 tranchées le même jour, à partir des questions ouvertes du brouillon et d'un point apporté par D1.

- **D1. La production ouvre dans ce plan, avec la seule tribu du développeur** (choix du développeur : se mettre le plus tôt possible dans les conditions de la production). Deux instances sur le serveur, `production` et `recette`. La release `v0.1.0`, les retours arrière et la répétition du traitement accéléré se jouent sur la vraie production. Le site public minimal ouvre en même temps (lot D), avec ses mentions légales et sa page Confidentialité. La famille est invitée plus tard, quand le planning existe ; la révocation d'un membre et la fermeture d'une session à distance (`revue-securite`, constat 5) sont à faire avant. Révise deux décisions d'ordonnancement du 2026-10-04 (feuille de route). Écartés : une seule instance, la recette tenant le rôle de la production (proposition du brouillon) ; une instance `production` installée sans être exposée ; une ouverture à la famille dès `v0.1.0`, qui demandait le constat 5 d'abord pour une application qui ne fait encore que connecter.
- **D2. Cookies préfixés `__Host-`, un nom par tribu, `Path=/` ; la recette reste sous le nom d'hôte de la production** (constat 6). Le navigateur refuse à tout autre hôte du domaine de poser un cookie de ce nom : ni la recette, ni un autre nom de `codingmatters.org` ne peut substituer une session. Rien n'est déployé : le changement ne déconnecte personne. Écartés : sortir la recette du nom d'hôte de la production, qui ne suffit pas (un hôte voisin pose un cookie pour `codingmatters.org` entier, que la production reçoit aussi) ; les deux ensemble ; ne rien changer.
- **D3. La tribu de recette vient d'un fichier du serveur, sur des adresses de recette de `codingmatters.org`** (constat 9). `admin seed -members`, obligatoire en mode serveur. Les membres de démonstration sont des redirections `recette-…@codingmatters.org` vers la boîte du développeur (ajout du développeur), sur un domaine que le projet contrôle et stables d'un serveur reconstruit à l'autre ; `recette-inconnu@codingmatters.org` reste inexistante, pour lire les réponses du MX Plan. La délivrabilité se vérifie sur des adresses directes, une redirection pouvant changer ce que voient SPF et DMARC. Écartés : une tribu par adresse avec `admin init` ; des membres dérivés par sous-adressage d'une seule adresse ; changer le domaine des exemples dans tout le projet.
- **D4. Les valeurs de l'instance tiennent dans un seul fichier du serveur**, `/etc/tribe-menus/instance.conf`, d'où `provision.sh` tire les `config.json`, les sites de Caddy et la configuration de `msmtp`. Précise l'ADR 0011 (point 5), écrit avant l'ADR 0016 : ces valeurs ne vont pas chez GitHub. Écartés : écrire chaque fichier à la main avec les mêmes valeurs ; des variables GitHub, qu'il faudrait un jeton de plus pour lire.
- **D5. Les secrets se saisissent au clavier sur le serveur**, par `tribe-menus-credential`, et n'existent en clair que dans le gestionnaire de mots de passe du développeur. Un serveur reconstruit redemande leur saisie : la clé de chiffrement est celle de la machine. Écartés : un fichier copié par `scp` ; les secrets en clair dans `instance.conf` (ADR 0015, point 10).
- **D6. Le provisionnement est un geste distinct du déploiement.** Première installation depuis l'archive d'une PR copiée par `scp` ; ensuite, `provision.sh` est relancé depuis la version installée quand `deploy/` a changé, ce que les notes de version disent. Écarté : laisser `tribe-menus-deploy` relancer le provisionnement, ce qui ferait réécrire la configuration du serveur entier, production comprise, par l'archive d'une PR déployée en recette.
- **D7. Le binaire garde son refus d'une base plus récente que lui ; revenir sur une version qui a migré restaure l'instantané.** Automatique quand la nouvelle version n'a jamais répondu à `/healthz` (aucune requête n'a été servie, rien n'est perdu) ; sur demande explicite ensuite, le script disant ce qui est perdu. L'ADR 0012 (point 6) est précisé. Écartés : faire accepter au binaire un schéma plus récent, ce qui ferait tourner un code ancien sur des tables qu'il ne connaît pas ; une compatibilité déclarée par migration, non retenue et non reportée.
- **D8. Le site public suit la version de l'application et s'installe par environnement** (`/var/www/tribe-menus/<environnement>/`) ; la recette le sert aussi, en `noindex`. Site, application et lien « code source » viennent du même commit, et les en-têtes du site se vérifient en recette avant la production (constat 11). Corriger un texte demande un déploiement. Précise l'ADR 0006 (point 3 : « déployées indépendamment ») et les ADR 0015 (point 9) et 0016 (point 1). Écartés : ne pas servir le site en recette ; un déploiement du site seul.
- **D9. Les alertes de l'application sont relayées par un minuteur** qui lit le journal depuis son dernier passage, toutes les cinq minutes. Écarté : un processus qui suit le journal en continu, à surveiller lui aussi.
- **D10. Les contrôles sont des programmes, pas une liste à dérouler à la main** : `tribe-menus-check` sur le serveur, `make probe` de l'extérieur, `shellcheck` sur les scripts. Ils se rejouent à chaque version et après chaque reconstruction, et leurs rapports se lisent en revue. `shellcheck` est le seul outil ajouté au poste (ADR 0009).
- **D11. La recette sur appareils est celle de la PR de la première release.** `RELEASING.md` la demande à cet endroit : la faire plus tôt obligerait à la refaire.
- **D12. La répétition du traitement accéléré porte sur un correctif réel et mineur, jusqu'à la vraie production** : la fenêtre des compteurs d'alerte (revue de la PR #47, point 2). Les durées mesurées sont celles d'une vraie livraison, et `v0.1.1` n'est pas une version vide. Le faux constat est marqué comme tel dans l'analyse publiée.
- **D13. Cinq séances pour les gestes du développeur**, chacune avec sa check-list, et rien de manuel en dehors. Ce qui pouvait être confié à un script ou à Claude Code l'a été ; une séance qui déborde est un défaut du plan, à noter.
- **D14. `sudo` avec mot de passe pour le compte d'administration.** Une clé SSH ou un agent compromis ouvre le serveur, pas `root`. Coût : une saisie par séance. Écarté : `sudo` sans mot de passe, réglage par défaut des images cloud.
- **D15. Caddy vient du dépôt apt du projet Caddy**, clé de signature épinglée par son empreinte, suivi par `unattended-upgrades`. Version à jour et correctifs dès leur publication. Précise l'ADR 0006 (point 2 : « paquet officiel »). Écarté : le paquet de Debian 13, plus ancien et figé pour la durée de vie de la distribution.
- **D16. HTTP/3 coupé.** HTTP/1.1 et HTTP/2 seuls ; le pare-feu reste sur trois ports TCP (ADR 0015, point 4). Écarté : ouvrir 443 en UDP, pour un gain modeste à ce volume.
- **D17. La version des pages du site est écrite par `make dist`**, qui remplace des marques dans `site/` en construisant l'archive (ADR 0011, point 2). L'archive reste reproductible, et ce que sert Caddy est exactement ce que couvre l'empreinte. Écartés : les gabarits de Caddy (une surface de plus sur un composant exposé) ; le remplacement par `tribe-menus-deploy` (le contenu installé différerait de l'archive vérifiée).
- **D18. Cinq PR, une par lot** (A : cookies et tribu de recette ; B : provisionnement ; C : déploiement et supervision ; D : site public ; E : suites de la recette), plus les PR de release. Écarté : réunir B et C, plus long à relire.

## Notes d'exécution

- **2026-10-07** : plan écrit par une session cloud, à partir de la feuille de route, des ADR et des plans précédents. Aucune commande n'a été lancée : les noms des scripts et leurs options sont des propositions, que les lots B et C fixent.
- **2026-10-08** : décisions passées en revue une à une avec le développeur. D1 remplacée (la production ouvre dans ce plan), d'où un lot de plus (site public) et des séances S1, S2 et S4 complétées ; D3 complétée (adresses de recette sur `codingmatters.org`) ; questions ouvertes sur `sudo`, le paquet de Caddy et HTTP/3 tranchées (D14 à D16) ; version des pages du site (D17) et nombre de PR (D18). Étapes renumérotées : 37 au lieu de 33.
- **Lot A** (`feature/cookies-host-et-tribu-de-recette`), 2026-10-08, Claude Code dans le Dev Container :
  - **Étape 1** : ADR 0024. Suffixe du nom retenu : les 16 premiers caractères hexadécimaux de SHA-256 de l'identifiant, tel que lu dans le chemin, pour toute valeur ; noms `__Host-session-<suffixe>` et `__Host-code-request-<suffixe>`. Scénario ajouté à `authentification.feature` : deux tribus ouvertes dans le même navigateur, déconnexion de l'une, l'autre garde sa session ; trois formulations nouvelles (`je me déconnecte de la tribu`, `je ne suis plus connecté à la tribu`, `je suis toujours connecté à la tribu`), « je me déconnecte » sans tribu restant celle des scénarios en attente.
  - **Étape 2** : un essai en remplaçant le suffixe par une constante fait échouer quatre scénarios, dont le nouveau et « Une même adresse a des sessions distinctes dans chaque tribu ». Le harnais godog présente aussi, à l'ouverture d'une URL, les cookies des autres tribus sous les noms de celle-ci (au lieu de les envoyer en ignorant le chemin, devenu `/`). Le parcours Playwright vérifie que Chromium garde le cookie `__Host-` sur `http://localhost`, puis l'efface à la déconnexion.
  - **Étape 3** : format du fichier : une ligne par membre, adresse puis nom d'affichage facultatif, séparés par des espaces ou des tabulations ; lignes vides et lignes commençant par `#` ignorées (ajout, pour commenter le fichier du serveur) ; adresse en double refusée ; le fichier est refusé en entier à sa première ligne invalide, avant toute écriture. Sans `-config`, `-members` reste possible. Si la tribu `demo` existe déjà, rien ne change, fichier ou non. `RELEASING.md` (essai à blanc) et le test du mode serveur en TCP passent un fichier de membres. **Pour le lot B** : `tribe-menus-admin` lance la commande sous l'utilisateur du service, qui doit pouvoir lire le fichier des membres.
  - **Étape 4** : le refus était déjà fixé par `internal/storage/storage_test.go` (« database newer than the binary ») et `TestServeRefusesUnmigratedDatabases` ; aucun test ajouté.
  - **Étape 5** : la page Confidentialité ne nomme pas les cookies ; `docs/design/README.md` précise seulement qu'ils sont propres à une tribu.
