# Plan : recette en partie automatisée

- **Nom** : `recette-auto`
- **Date** : 2026-11-14
- **Statut** : en cours depuis le 2026-11-15 ; lots A (#122) et B (#123) fusionnés, lot C en cours. D1 à D4 et D14 prises par le développeur (2026-11-14 et 2026-11-15) ; les autres confirmées par la fusion de la PR du plan (#120, 2026-11-15). Place dans la feuille de route : après les lots de `courses-suite`, **avant la recette de `v0.6.0`**, qui l'attend (D14).

## Objectif

Rendre la recette d'une release en partie automatique, et la justifier par une analyse plutôt que par une check-list qui s'allonge à chaque plan (ADR 0032).

À la fin du plan :

- chaque recette **commence par une analyse des changements et des risques**, qui produit le plan de recette de la version (`docs/recettes/vX.Y.Z.md`) : pour chaque changement, ce que la CI prouve déjà, ce qui se prouve automatiquement sur la recette, et ce qui reste manuel ;
- `make recette-auto`, lancé depuis le Dev Container après le déploiement en recette, **se connecte par le code reçu par e-mail** et joue les parcours de base et les parcours ciblés contre `recette.meltingtribe.codingmatters.org`, **filmés** (vidéo, trace, rapport) ;
- la séance manuelle guidée ne garde que ce qui demande un vrai appareil, un jugement visuel ou une vraie messagerie ;
- une release au moins est passée en recette de cette façon.

## Contexte et contraintes

- **À lire avant de commencer** : ADR 0005 (points 2 et 4), 0014, 0016, 0021, 0022, 0026, 0028, 0030, 0032 ; `RELEASING.md` (étape 6, scénarios `@manuel`) ; `deploy/README.md` (déployer en recette) ; `web/playwright.config.ts` et `web/e2e/` ; les séances de recette des plans `planning`, `courses` et `grands-ecrans`, pour ce qui y a été trouvé et comment.
- **Ce que la CI prouve déjà** : les règles contre l'API (godog), l'interface sur un serveur de test local avec les e-mails écrits dans un fichier (`-mail-file`), tous les formats sur la PR de release (`make e2e-formats`). La recette apporte l'environnement réel : systemd, Caddy, TLS, cookies `__Host-`, front à empreintes, envoi SMTP par le MX Plan, migration sur des données accumulées, et les vrais appareils.
- **Instance publique** : rien ne s'ajoute à l'application pour les tests (ni route, ni mode, ni option) ; la recette fait tourner le même binaire que la production.
- **Limitation des demandes de code** (ADR 0021) : 3 par quart d'heure pour une adresse, 10 par heure pour une IP, 30 par heure pour une tribu. Une exécution n'en consomme qu'une (D8) ; `make probe` en consomme aussi et reste à part.
- **Aucune dépendance nouvelle**, ni en Go ni en npm : POP3 par la bibliothèque standard (D7), Playwright déjà présent.
- **Aucun secret ni valeur de l'instance dans le dépôt** (ADR 0011) : mot de passe de la boîte, identifiant de la tribu `recette-auto` et état de session restent dans un volume du Dev Container (D10).
- **Où se fait le travail** : le lot A depuis une session cloud ou le Dev Container ; les lots B et C dans le Dev Container, seul à pouvoir joindre la boîte et lancer les parcours avec le secret.
- **Git** : une branche `feature/…` et une PR par lot ; un commit par étape, `git commit -s` ; une ligne de `CHANGELOG.md` par PR, citant le lot (`recette-auto/A`) ; `make ci` avant de pousser.

## Décisions

Le 2026-11-14, avec le développeur.

- **D1. Recette en partie automatisée** : scénarios joués par Playwright contre l'instance de recette, depuis le Dev Container, enregistrés en vidéo pour le contrôle. Les vidéos sont stockées dans un volume du Dev Container, jamais archivées, et supprimées une fois la recette terminée (précisé par le développeur le 2026-11-15).
- **D2. Code de connexion relevé dans une boîte dédiée**, le Dev Container ne pouvant pas recevoir de courrier ; aucune route qui exposerait le code sur la recette.
- **D3. Une phase d'analyse en tête de chaque recette** détermine le plan de recette, en séparant ce qui peut passer par la recette automatique de ce qui doit rester manuel, à partir d'une analyse des changements et des risques.
- **D4. L'enregistrement vidéo permet un contrôle** : un jugement humain qui ne demande pas de vrai appareil (rendu, conformité à la maquette, enchaînement, messages) se fait en regardant la vidéo du parcours automatique, pas en séance sur l'appareil. Seul ce qui dépend de l'appareil physique ou des vraies messageries reste manuel. La séance manuelle elle-même n'est pas filmée (décision du développeur, 2026-11-14).

Proposées le 2026-11-14 ; détail et alternatives dans l'ADR 0032.

- **D5. Analyse** (ADR 0032, points 1 à 4) : inventaire d'après `CHANGELOG.md`, les migrations, `deploy/` et les dépendances ; trois questions par changement (gravité, probabilité d'un défaut échappé à la CI, dépendance à l'environnement réel) ; voie de preuve déduite par quatre règles appliquées dans l'ordre : manuel sur un vrai appareil, automatique avec contrôle de la vidéo, automatique, CI (ADR 0032, point 4). Proposée par la session qui prépare la PR de release, dans la PR, avant le déploiement en recette ; validée ou corrigée par le développeur avant la séance. Procédure dans `docs/recette.md`, gabarit `docs/recettes/_template.md`.
- **D6. Parcours** (ADR 0032, points 5 et 6) : tests Playwright ordinaires dans `web/recette/`, données créées par l'interface ; parcours de base toujours joués, parcours ciblés ajoutés quand l'analyse en demande un. Un parcours ciblé écrit pour une release reste dans la suite s'il garde du sens, sinon il est retiré dans la PR qui l'a rendu inutile.
- **D7. Boîte et relève** (ADR 0032, points 7 et 8) : tribu `recette-auto` sur la recette, membre unique `recette-auto@codingmatters.org`, vraie boîte du MX Plan ; relève POP3 sur TLS par `internal/tools/mailbox`, toutes les 5 secondes, 2 minutes au plus ; messages relevés supprimés.
- **D8. Une connexion par exécution** (ADR 0032, point 9) : projet `setup` filmé, `storageState` réutilisé par les quatre projets (ADR 0032, point 10).
- **D9. Points de contrôle de la vidéo** (ADR 0032, point 11) : le plan de recette désigne, pour chaque changement de la voie « contrôle de la vidéo », le parcours, le projet, l'étape (`test.step`) et ce qu'il faut regarder (« le bandeau vert « Liste recalculée. » sous l'en-tête, en compact et en étendu, comme sur `Doux-ListeSuite` ») ; le rapport de l'exécution liste ces points avec un lien vers la vidéo à l'instant de l'étape et la capture correspondante, pour que le contrôle tienne en quelques minutes. Les vidéos des autres parcours ne se regardent qu'en cas d'échec.
- **D10. Valeurs de l'instance et secrets dans un volume du Dev Container**, `tribe-menus-recette`, monté sur `/home/vscode/.config/tribe-menus` : `recette-auto.json` (URL de la tribu, adresse, serveur et port POP3, compte), `recette-auto.pop3` (mot de passe, en `0600`), et l'état de session de la dernière exécution. `make recette-auto` refuse de démarrer si l'un manque, sans afficher de valeur. Écartés : variables d'environnement dans `.claude/settings.local.json` (le mot de passe se retrouverait dans un fichier de réglages souvent copié) ; un fichier `.env` dans le dépôt de travail (un oubli de `.gitignore` le publierait).
- **D11. Vidéos dans un volume, supprimées en fin de recette** (ADR 0032, points 11 et 12) : volume `tribe-menus-recette-videos`, monté sur `/home/vscode/recette-videos`, distinct du volume des secrets ; un dossier `vX.Y.Z/<horodatage>/` par exécution (vidéos, traces, captures, page de contrôle), les exécutions relancées après un correctif s'y ajoutant ; ni archive ni empreinte. `make recette-clean VERSION=vX.Y.Z` supprime le dossier de la version quand la recette est terminée : plan de recette rempli, release décidée ou abandonnée ; `make recette-clean` sans version vide le volume.
- **D12. Le plan de recette d'une version** (`docs/recettes/vX.Y.Z.md`) : tableau des changements (entrée, gravité, probabilité, dépendance, voie, parcours ou point de la séance), liste des points de contrôle de la vidéo et check-list de la séance manuelle, tirées du tableau, puis les résultats : exécution automatique (date, PR déployée, parcours passés ou en échec), contrôle de la vidéo (verdict par point, remarque, instant de la vidéo et description d'un défaut, assez précise pour se passer de la vidéo une fois supprimée), séance manuelle (cases cochées, défauts et PR qui les corrigent), écarts acceptés. Un défaut corrigé relance une analyse réduite aux changements du correctif.
- **D13. `RELEASING.md`, étape 6** devient : analyse et plan de recette dans la PR de release ; déploiement en recette ; `make recette-auto` ; contrôle de la vidéo, point par point ; séance manuelle guidée d'après le plan, réduite au vrai appareil et aux vraies messageries ; résultats consignés ; `make recette-clean` une fois la recette terminée. Les scénarios `@manuel` y restent, toujours dans la voie manuelle.
- **D14. Placement** (décision du développeur, 2026-11-15) : la recette automatique est en place **pour la recette de `v0.6.0`**. Les lots A à C passent après les lots de `courses-suite` (tous fusionnés le 2026-11-14) et avant la PR de release `release/v0.6.0`, ouverte une fois le lot C fusionné : ses notes de version emportent donc les lignes de ce plan, qui ne changent pas l'application. Le plan de recette de `v0.6.0` est le premier ; ses parcours ciblés sont écrits au lot C, d'après une analyse anticipée des changements de `v0.6.0`. Remplace la proposition du 2026-11-14 (lot A seul pour `v0.6.0`, outillage après).

## Étapes

### Lot A : procédure d'analyse et plan de recette (`feature/procedure-de-recette`)

- [x] 1. `docs/recette.md` : rôle de la recette face à la CI, inventaire, grille des trois questions avec des exemples tirés des recettes passées (double toucher de `planning`, clavier virtuel de `grands-ecrans`, `504` au téléchargement de `v0.5.0`), les quatre règles de la voie de preuve et leur ordre, comment désigner un point de contrôle de la vidéo et ce qu'une vidéo d'un navigateur émulé ne prouve pas, parcours de base, déroulé, cas d'une version corrective (ADR 0032, point 14).
- [x] 2. `docs/recettes/_template.md`, et `docs/recettes/README.md` qui liste les plans de recette.
- [x] 3. `RELEASING.md`, étape 6 (D13) ; renvois dans les ADR 0005 (point 4) et 0016 (« Recette »).
- [x] 4. ADR 0032 accepté ; `CLAUDE.md` (recette, ligne « Déploiement »).
- [x] 5. Essai à blanc de l'analyse sur `v0.5.0` (`grands-ecrans`), comparé à ce que sa séance a réellement trouvé : l'analyse aurait-elle envoyé en manuel le clavier virtuel en paysage ? Résultat dans les notes d'exécution ; la grille est corrigée s'il le faut.

### Lot B : relève du code et outillage (`feature/outillage-recette-auto`)

- [x] 6. `internal/tools/mailbox` : client POP3 sur TLS (`USER`, `PASS`, `LIST`, `RETR`, `DELE`, `QUIT`), choix du message (destinataire, arrivée après l'heure de la demande, le plus récent), extraction du code d'après le texte du message (ADR 0014, point 4), délai et intervalle de relève ; rien d'autre sur la sortie standard que le code ; ni mot de passe ni contenu de message dans les erreurs. Tests en tableaux de cas contre un faux serveur POP3 (`net.Listener`, `httptest` n'ayant pas d'équivalent), dont : aucun message, message plus ancien que la demande, autre destinataire, plusieurs messages, coupure du serveur, délai dépassé.
- [x] 7. Volumes `tribe-menus-recette` (D10) et `tribe-menus-recette-videos` (D11) dans `.devcontainer/devcontainer.json` ; `deploy/recette-auto.example.json` (modèle, sans valeur réelle).
- [x] 8. `web/playwright.recette.config.ts` : `baseURL` lue dans `recette-auto.json`, projet `setup` puis les quatre projets (ADR 0032, point 10) qui en dépendent, `video` à la taille de la fenêtre de chaque projet, `trace: "on"`, `screenshot: "on"`, court temps d'arrêt après une action visible (ADR 0032, point 11), aucun `webServer`, aucune reprise automatique d'un test en échec (`retries: 0`), sortie dans le volume des vidéos (D11).
- [x] 9. Projet `setup` (`web/recette/connexion.setup.ts`) : demande de code, relève par `mailbox` (appelé par `execFile`), saisie, état de session écrit dans le volume.
- [x] 10. `make recette-auto` (ADR 0010) : contrôle des fichiers du volume, version servie par la recette lue et vérifiée contre `VERSION` ou `PR`, exécution dans le dossier de la version (D11), chemin de la page de contrôle affiché à la fin ; `make recette-clean [VERSION=vX.Y.Z]`.
- [x] 11. `make ci` ne lance ni les parcours ni la relève ; il compile et vérifie (`tsc`, `go vet`, `staticcheck`) `web/recette/` et l'outil.
- [x] 12. Rapport de contrôle : à la fin de l'exécution, une page HTML locale qui liste les points de contrôle de la version (lus dans `docs/recettes/vX.Y.Z.md`), chacun avec sa capture et un lien vers la vidéo à l'instant de son étape ; une étape désignée qui n'existe dans aucun parcours fait échouer `make recette-auto`.

### Lot C : parcours de base et première recette (`feature/parcours-recette`)

- [x] 13. Parcours de base (ADR 0032, point 5) : connexion (projet `setup`) ; chaque onglet dans chaque projet ; créer un plat suffixé de l'exécution et le retrouver par la recherche ; l'ajouter à un repas de demain et relire le planning ; ajouter les courses du repas à « Courses » et cocher un article ; recharger et retrouver chaque écriture.
- [x] 14. Contrôles du serveur visibles depuis le navigateur : la page de la tribu sert `Cache-Control: no-cache`, les fichiers à empreinte `immutable`, le cookie de session préfixé `__Host-` (lus dans les réponses enregistrées par Playwright, sans rien ajouter à l'application).
- [x] 15. Analyse anticipée des changements de `v0.6.0` (section `## Non publié` de `CHANGELOG.md`, migration `0010`) : brouillon de `docs/recettes/v0.6.0.md`, validé par le développeur avant l'écriture des parcours ciblés.
- [x] 16. Parcours ciblés de `v0.6.0` que l'analyse demande, avec leurs étapes nommées pour les points de contrôle de la vidéo (pistes : liste périmée et « Recalculer », part prise, sélection et déplacement dans les trois formats, écritures croisées à deux contextes, taille de `GET shopping-lists`).
- [ ] 17. Séance S1 (ci-dessous) ; répétition générale : archive de la PR du lot C déployée en recette (elle contient tout `courses-suite`), `make recette-auto`, contrôle de la vidéo, corrections des parcours ; `make recette-clean`.
- [ ] 18. Recette de `v0.6.0` par le déroulé complet (D13) sur l'archive de la PR de release ; plan de recette rempli ; release et déploiement en production selon `RELEASING.md` (étapes 28 et 29 de `courses-suite`).
- [ ] 19. Clôture : statut « terminé » ; feuille de route ; `CLAUDE.md` (commandes : `recette-auto` ; recette) ; `CHANGELOG.md`.

### Séance S1 : préparation (développeur)

- [x] Boîte `recette-auto@codingmatters.org` créée sur le MX Plan, avec un mot de passe propre ; POP3 sur TLS accessible (serveur et port relevés dans l'espace client OVHcloud, à reporter dans `recette-auto.json`) ; aucune redirection vers elle, aucune depuis elle.
- [x] Tribu `recette-auto` créée sur l'instance de recette (`admin init`, interactif, membre unique `recette-auto@codingmatters.org`) ; commande exacte vérifiée au lot B et écrite dans `deploy/README.md`.
- [x] Volume du Dev Container rempli : `recette-auto.json`, `recette-auto.pop3` en `0600`.
- [x] Première relève à la main : `go run ./internal/tools/mailbox …` après une demande de code depuis le navigateur.

## Critères de validation

- `make ci` vert à chaque PR, sans accès à la boîte.
- `make recette-auto` passe sur la recette depuis le Dev Container, en une seule demande de code, et laisse dans le volume les vidéos, traces et la page de contrôle ; un parcours qui échoue fait échouer la commande. Après `make recette-clean`, le volume ne garde rien de la version, et le plan de recette suffit à dire ce qui a été vu.
- La séance manuelle d'une release ne contient que des points qui demandent un vrai appareil ou une vraie messagerie, et chacun dit pourquoi ; le contrôle de la vidéo d'une release tient en un quart d'heure.
- Sur l'essai à blanc de `v0.5.0` (étape 5), l'analyse classe en manuel sur un vrai appareil ce que la séance avait trouvé à la main (double toucher, clavier virtuel), et en contrôle de la vidéo ce qui ne demandait que de regarder.

### Ce que ce plan ne prouve pas

- La réception du code sur Gmail, Outlook et iCloud, hors des indésirables : vérification manuelle ponctuelle (point reporté de la feuille de route) ; la boîte `recette-auto@` ne prouve que l'envoi par le MX Plan.
- Le vrai Safari et les vrais appareils : toujours dans la voie manuelle.
- La justesse de l'analyse elle-même : elle se juge release après release, aux défauts trouvés en production qu'elle n'avait pas vus.

## Questions ouvertes

- **Projets** : quatre suffisent-ils, ou faut-il la tablette en portrait (`moyen`) dans les parcours de base ?
- **Place des vidéos** pendant une recette (quatre projets, exécutions relancées) : à mesurer à la première exécution ; sans incidence durable, le volume étant vidé à la fin.
- **Tribu `recette-auto` qui s'emplit** : la recréer à chaque reconstruction de la recette, ou ajouter une commande d'administration qui la vide ?
- **Parcours de deux appareils** (écritures croisées, deux contextes Playwright sur la même session) : utiles en automatique, ou laissés à la séance manuelle ?

## Notes d'exécution

**Lot A** (`feature/procedure-de-recette`), 2026-11-15, depuis une session cloud :

- **Opérations sur le serveur en voie 1** : retour arrière avec restauration, provisionnement relancé et `tribe-menus-check report` demandent SSH et `sudo`, que le Dev Container n'a pas ; ni appareil ni messagerie, ils n'entraient dans aucune règle. Ajoutés à la règle 1, précision notée dans l'ADR 0032, point 4.
- **Un changement, plusieurs entrées** : un même changement peut relever de deux voies par des aspects différents (le rendu du semainier en voie 2, son défilement au doigt en voie 1) ; le plan de recette lui donne une entrée par aspect, plutôt qu'une voie « principale ».
- **Défaut du déploiement lui-même** (`504` au téléchargement de `v0.5.0`) : vu par le développeur en déployant, il n'est cherché par aucune voie ; il reste aux PR qui touchent `deploy/`.
- **`RELEASING.md`** : étape 6 découpée en cinq temps (plan de recette, déploiement, parcours et contrôle de la vidéo, séance manuelle, résultats), étape 12 « Fin de la recette » ajoutée. Tant que les lots B et C ne sont pas fusionnés, les voies 2 et 3 se vérifient à la main.

**Essai à blanc sur `v0.5.0`** (étape 5), à partir de la section 0.5.0 de `CHANGELOG.md` et de la séance du plan `grands-ecrans` (2026-11-09) :

| Entrée | G | P | Dépendance | Voie |
| --- | --- | --- | --- | --- |
| `grands-ecrans/B` : formats, rail, dialogues centrés, dans chaque format | moyenne | élevée | aucune | 2 (rail et dialogues comparés aux maquettes `GE-*`) |
| `grands-ecrans/B` : format gardé pendant une saisie quand le clavier virtuel ferait basculer | moyenne | élevée | appareil | **1** |
| `grands-ecrans/B` : fenêtre large mais basse (moins de 560 px) en compact | faible | moyenne | aucune | 3 (taille de fenêtre) |
| `grands-ecrans/C` : semainier et repas en panneau, rendu | moyenne | élevée | aucune | 2 |
| `grands-ecrans/C` : défilement au doigt calé sur un jour, cibles au doigt | moyenne | élevée | appareil | **1** |
| `grands-ecrans/C` : focus et Échap au clavier | faible | moyenne | aucune | 3 |
| `grands-ecrans/D` : deux volets, « Plat enregistré. » en étendu | moyenne | élevée | aucune | 2 |
| `grands-ecrans/D` : « Abandonner la modification ? » en moyen et en étendu, pas en compact | haute (saisie perdue) | élevée | aucune | 2 |
| `grands-ecrans/D` : rotation sans perte de saisie ni de défilement | haute | élevée | appareil | **1** (rotation de l'appareil) ; le changement de format seul, en 3 par une taille de fenêtre |
| Tests du semainier rendus stables (deux lignes) | faible | faible | aucune | 4 |
| Plans `courses` (clôture) et `grands-ecrans` (brouillon) | — | — | aucune | 4 |
| Connexion sur tablette et ordinateur, `@manuel` « La session survit au redémarrage du navigateur », `tribe-menus-check report` | — | — | — | parcours de base ; **1** ; **1** |

- **Ce que la séance a trouvé** : le clavier virtuel qui cache « Enregistrer le plat » en paysage, classé en voie 1 par la grille, donc vu par la séance réduite. Aucun défaut n'est tombé dans les voies 2 à 4 ; la grille n'en aurait raté aucun.
- **Séance manuelle** : de neuf points à quatre sur la tablette (clavier virtuel, rotation, défilement et cibles au doigt), plus le scénario `@manuel` et `tribe-menus-check report` ; le reste en contrôle de la vidéo (sept points environ) ou automatique.
- **Le double toucher de `v0.3.0`**, rejoué de même : toucher, voie 1. Aucune correction de la grille.

**Séance S1** (développeur), 2026-11-15 : boîte créée, POP3 sur TLS vérifié par `openssl s_client` ; tribu créée par `sudo tribe-menus-admin recette init`, membre unique, connexion vérifiée avec un code relevé à la main ; un e-mail de test laissé dans la boîte (la première relève le supprimera s'il est adressé à la boîte). Restent le volume et la première relève par `mailbox`, une fois le lot B fusionné et le Dev Container reconstruit avec ses volumes.

**Lot B** (`feature/outillage-recette-auto`), 2026-11-15, dans le Dev Container :

- **Configuration partagée** : `recette-auto.json` et le mot de passe sont lus par `internal/tools/recetteconfig`, commun à `mailbox` et à `internal/tools/recetteauto` (l'outil de `make recette-auto`, ADR 0010 : rien de long dans le Makefile) ; `web/recette/config.ts` relit le même fichier pour Playwright. Le fichier porte aussi `tribeURL`, absent de D10, qui en disait le contenu sans en fixer les clés.
- **Version servie** : lue dans le pied de page du site public de la recette (« Code source de cette version (pr-131) », `{{VERSION}}` remplacé par `internal/tools/dist`), sans rien ajouter à l'application. `VERSION` nomme le plan de recette et le dossier de l'exécution ; `PR`, quand il est donné, est ce que la recette doit servir (répétition générale du lot C : `VERSION=v0.6.0 PR=<lot C>`).
- **Choix du message** : par l'en-tête `Date`, écrit par le serveur de l'instance, avec 30 secondes de tolérance d'horloge ; l'en-tête `Received` du MX aurait demandé de lire un format moins régulier. Code extrait de la ligne « Votre code de connexion à … : 12345678 » du texte décodé, refusé si deux lignes en portent un. Mot de passe refusé : arrêt immédiat ; coupure ou erreur du serveur : nouvel essai jusqu'au délai.
- **Instant de l'étape dans la vidéo** : Playwright ne donne pas le début d'une vidéo ; le `test` des parcours (`web/recette/test.ts`) note l'heure de création de la page en annotation, et la page de contrôle ouvre la vidéo une seconde avant l'étape (`#t=`). Vérifié sur un essai local jetable (deux projets, deux étapes) : capture juste, instant juste, code de sortie 1 pour une étape désignée absente.
- **Captures** : prises à la fin de chaque étape nommée (`step`), après une pause de 800 ms ; seules celles des étapes désignées par le plan de recette sont écrites, sous `captures/<point>-<projet>.png`. `slowMo` de 250 ms en plus, pour la lisibilité de la vidéo.
- **Un seul worker** : la tribu `recette-auto` est partagée par les quatre projets, et une vidéo se lit mieux sans écritures croisées ; à revoir au lot C si l'exécution est trop longue.
- **Non vérifié dans ce lot** : le montage des volumes (le Dev Container de cette session n'a pas été reconstruit) et une exécution contre la recette, qui attendent le remplissage du volume ; ce sera la première tâche du lot C.
- **Revue de la PR #123, point 2 (à traiter au lot C, étape 17)** : un code d'une demande précédente peut être pris pour le bon. `pick` accepte un message daté jusqu'à 30 s avant la demande (`clockSkew`) : une exécution interrompue juste après sa demande, relancée dans les 30 s, peut relever le code de l'ancienne demande s'il arrive avant le nouveau ; ce code est lié au jeton de l'ancienne demande, la connexion échoue (`ErrNewCodeNeeded`) et l'exécution consomme une deuxième demande. Pistes : vider la boîte juste avant la demande dans `connexion.setup.ts`, ou, si le code est refusé, relever de nouveau avec le même `since` en écartant le code essayé.

**Lot C** (`feature/parcours-recette`), 2026-11-15, dans le Dev Container, S1 terminée (volume rempli, première relève faite par le développeur) :

- **Premières exécutions** contre la recette, qui servait `pr-111` (`v0.5.0`), `make recette-auto VERSION=v0.6.0 PR=111` : connexion par le code relevé en POP3 en 12 à 17 s ; parcours de base et contrôles du serveur passés dans les quatre projets après deux corrections des parcours (titre « Plats » de niveau 2 dans le volet en étendu ; ingrédient limité à 40 caractères par le champ, d'où des projets abrégés dans les noms). Environ 65 Mo pour une exécution de neuf tests. Les données de ces exécutions, écrites sous `v0.5.0`, seront celles que la migration `0010` trouvera à la répétition générale (entrée 1 du plan de recette).
- **Limites des demandes de code** : la troisième exécution en cinq minutes a été refusée (« Trop de demandes depuis cet appareil », limite par IP, avec les essais manuels de S1 depuis la même connexion). Le projet `setup` dit désormais ce refus ; `make recette-auto REUSE=1` reprend la session de la dernière exécution quand elle ouvre encore la tribu, sans demande de code, pour les essais pendant l'écriture des parcours. La recette d'une release se lance sans `REUSE` : la connexion par e-mail fait partie des parcours de base.
- **Revue #123, point 2** traité : le projet `setup` vide la boîte (`mailbox -drain`) avant la demande, ce qui supprime aussi l'e-mail de test laissé à S1.
- **Données de la tribu** : les plats et les plats servis s'accumulent (un par projet et par exécution) ; les articles des parcours sont retirés par « Courses faites », et le parcours de sélection proposé supprime la liste qu'il crée (limite de 20 listes).
- **Analyse anticipée de `v0.6.0`** (étape 15) : `docs/recettes/v0.6.0.md`, validée par le développeur le 2026-11-15, sans correction, après une seule demande : la voie écrite en toutes lettres, dans la colonne qui suit le changement (gabarit et `docs/recette.md` aussi). Elle tranche deux questions ouvertes du plan : la tablette en portrait reste hors des parcours, les écritures croisées se font à deux contextes dans `recalcul.spec.ts`.
- **Parcours ciblés** (étape 16, en cours) : `recalcul.spec.ts` (V2 à V5, second contexte sur la même session pour l'autre appareil) et `selection.spec.ts` (V6 à V9, la liste créée supprimée à la fin), écrits d'après l'interface et les étapes `@ui` ; aides communes dans `web/recette/helpers.ts`, reprises par le parcours de base. À éprouver contre l'archive de #124 déployée en recette. Dans `ARGS`, un fichier de parcours se place avant `--project`, qui prend tous les mots qui le suivent.
- **Archive de #124 déployée en recette** (2026-11-15, `pr-124`, migration `0010` appliquée aux données écrites sous `v0.5.0`). Première exécution avec `REUSE=1` : base, recalcul et serveur passés dans les quatre projets ; sélection en échec sur le texte du bandeau d'une nouvelle liste, « … vers la nouvelle liste « … ». » (`docs/design/README.md`), que le parcours attendait sans « la nouvelle liste ». Corrigé ; le parcours vide et supprime désormais d'abord les listes « Sélection … » laissées par une exécution arrêtée (quatre ici).
- **Répétition générale** (étape 17), `make recette-auto VERSION=v0.6.0 PR=124`, sans `REUSE` : 17 tests passés sur 17 en 4,4 minutes, 302 Mo, les quatorze captures des points V1 à V9 écrites. Reste le contrôle de la vidéo par le développeur, puis `make recette-clean VERSION=v0.6.0`.
