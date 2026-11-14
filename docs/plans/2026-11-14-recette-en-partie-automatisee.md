# Plan : recette en partie automatisée

- **Nom** : `recette-auto`
- **Date** : 2026-11-14
- **Statut** : brouillon. D1 à D3 demandées par le développeur le 2026-11-14 ; les autres proposées, confirmées par la fusion de la PR du plan. Place dans la feuille de route : lot A pendant la fin de `courses-suite`, pour servir dès la recette de `v0.6.0` ; lots B et C après `courses-suite`, avant le plan hors-ligne (D12).

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
- **Limitation des demandes de code** (ADR 0021) : 3 par quart d'heure pour une adresse, 10 par heure pour une IP, 30 par heure pour une tribu. Une exécution n'en consomme qu'une (D7) ; `make probe` en consomme aussi et reste à part.
- **Aucune dépendance nouvelle**, ni en Go ni en npm : POP3 par la bibliothèque standard (D6), Playwright déjà présent.
- **Aucun secret ni valeur de l'instance dans le dépôt** (ADR 0011) : mot de passe de la boîte, identifiant de la tribu `recette-auto` et état de session restent dans un volume du Dev Container (D8).
- **Où se fait le travail** : le lot A depuis une session cloud ou le Dev Container ; les lots B et C dans le Dev Container, seul à pouvoir joindre la boîte et lancer les parcours avec le secret.
- **Git** : une branche `feature/…` et une PR par lot ; un commit par étape, `git commit -s` ; une ligne de `CHANGELOG.md` par PR, citant le lot (`recette-auto/A`) ; `make ci` avant de pousser.

## Décisions

Le 2026-11-14, avec le développeur.

- **D1. Recette en partie automatisée** : scénarios joués par Playwright contre l'instance de recette, depuis le Dev Container, enregistrés en vidéo comme preuve de recette.
- **D2. Code de connexion relevé dans une boîte dédiée**, le Dev Container ne pouvant pas recevoir de courrier ; aucune route qui exposerait le code sur la recette.
- **D3. Une phase d'analyse en tête de chaque recette** détermine le plan de recette, en séparant ce qui peut passer par la recette automatique de ce qui doit rester manuel, à partir d'une analyse des changements et des risques.

Proposées le 2026-11-14 ; détail et alternatives dans l'ADR 0032.

- **D4. Analyse** (ADR 0032, points 1 à 4) : inventaire d'après `CHANGELOG.md`, les migrations, `deploy/` et les dépendances ; trois questions par changement (gravité, probabilité d'un défaut échappé à la CI, dépendance à l'environnement réel) ; voie de preuve déduite : CI, automatique sur la recette, manuel. Proposée par la session qui prépare la PR de release, dans la PR, avant le déploiement en recette ; validée ou corrigée par le développeur avant la séance. Procédure dans `docs/recette.md`, gabarit `docs/recettes/_template.md`.
- **D5. Parcours** (ADR 0032, points 5 et 6) : tests Playwright ordinaires dans `web/recette/`, données créées par l'interface ; parcours de base toujours joués, parcours ciblés ajoutés quand l'analyse en demande un. Un parcours ciblé écrit pour une release reste dans la suite s'il garde du sens, sinon il est retiré dans la PR qui l'a rendu inutile.
- **D6. Boîte et relève** (ADR 0032, points 7 et 8) : tribu `recette-auto` sur la recette, membre unique `recette-auto@codingmatters.org`, vraie boîte du MX Plan ; relève POP3 sur TLS par `internal/tools/mailbox`, toutes les 5 secondes, 2 minutes au plus ; messages relevés supprimés.
- **D7. Une connexion par exécution** (ADR 0032, point 9) : projet `setup` filmé, `storageState` réutilisé par les quatre projets (ADR 0032, point 10).
- **D8. Valeurs de l'instance et secrets dans un volume du Dev Container**, `tribe-menus-recette`, monté sur `/home/vscode/.config/tribe-menus` : `recette-auto.json` (URL de la tribu, adresse, serveur et port POP3, compte), `recette-auto.pop3` (mot de passe, en `0600`), et l'état de session de la dernière exécution. `make recette-auto` refuse de démarrer si l'un manque, sans afficher de valeur. Écartés : variables d'environnement dans `.claude/settings.local.json` (le mot de passe se retrouverait dans un fichier de réglages souvent copié) ; un fichier `.env` dans le dépôt de travail (un oubli de `.gitignore` le publierait).
- **D9. Preuves** (ADR 0032, point 11) : `web/recette-results/vX.Y.Z-<horodatage>/` (vidéos, traces, captures, rapport HTML), archive et empreinte produites par `make recette-auto` ; l'empreinte va dans le plan de recette, l'archive chez le développeur.
- **D10. Le plan de recette d'une version** (`docs/recettes/vX.Y.Z.md`) : tableau des changements (entrée, gravité, probabilité, dépendance, voie, parcours ou point de la séance), check-list de la séance manuelle tirée du tableau, puis les résultats : exécution automatique (date, PR déployée, parcours passés ou en échec, empreinte des preuves), séance manuelle (cases cochées, défauts et PR qui les corrigent), écarts acceptés. Un défaut corrigé relance une analyse réduite aux changements du correctif.
- **D11. `RELEASING.md`, étape 6** devient : analyse et plan de recette dans la PR de release ; déploiement en recette ; `make recette-auto` ; séance manuelle guidée d'après le plan ; résultats consignés. Les scénarios `@manuel` y restent, toujours dans la voie manuelle.
- **D12. Placement** : le lot A (procédure, gabarit) ne demande aucun outil et sert dès la recette de `v0.6.0`, avec la voie automatique encore vide ; il peut fusionner pendant `courses-suite` sans rien changer à l'application. Les lots B et C viennent après `v0.6.0`, avant le plan hors-ligne, dont la recette (réseau coupé, application installée) gagnera à s'appuyer dessus.

## Étapes

### Lot A : procédure d'analyse et plan de recette (`feature/procedure-de-recette`)

- [ ] 1. `docs/recette.md` : rôle de la recette face à la CI, inventaire, grille des trois questions avec des exemples tirés des recettes passées (double toucher de `planning`, clavier virtuel de `grands-ecrans`, `504` au téléchargement de `v0.5.0`), règle de la voie de preuve, parcours de base, déroulé, cas d'une version corrective (ADR 0032, point 13).
- [ ] 2. `docs/recettes/_template.md`, et `docs/recettes/README.md` qui liste les plans de recette.
- [ ] 3. `RELEASING.md`, étape 6 (D11) ; renvois dans les ADR 0005 (point 4) et 0016 (« Recette »).
- [ ] 4. ADR 0032 accepté ; `CLAUDE.md` (recette, ligne « Déploiement »).
- [ ] 5. Essai à blanc de l'analyse sur `v0.5.0` (`grands-ecrans`), comparé à ce que sa séance a réellement trouvé : l'analyse aurait-elle envoyé en manuel le clavier virtuel en paysage ? Résultat dans les notes d'exécution ; la grille est corrigée s'il le faut.

### Lot B : relève du code et outillage (`feature/outillage-recette-auto`)

- [ ] 6. `internal/tools/mailbox` : client POP3 sur TLS (`USER`, `PASS`, `LIST`, `RETR`, `DELE`, `QUIT`), choix du message (destinataire, arrivée après l'heure de la demande, le plus récent), extraction du code d'après le texte du message (ADR 0014, point 4), délai et intervalle de relève ; rien d'autre sur la sortie standard que le code ; ni mot de passe ni contenu de message dans les erreurs. Tests en tableaux de cas contre un faux serveur POP3 (`net.Listener`, `httptest` n'ayant pas d'équivalent), dont : aucun message, message plus ancien que la demande, autre destinataire, plusieurs messages, coupure du serveur, délai dépassé.
- [ ] 7. Volume `tribe-menus-recette` dans `.devcontainer/devcontainer.json` (D8) ; `deploy/recette-auto.example.json` (modèle, sans valeur réelle).
- [ ] 8. `web/playwright.recette.config.ts` : `baseURL` lue dans `recette-auto.json`, projet `setup` puis les quatre projets (ADR 0032, point 10) qui en dépendent, `video: "on"`, `trace: "on"`, `screenshot: "on"`, aucun `webServer`, aucune reprise automatique d'un test en échec (`retries: 0`), sortie dans `web/recette-results/`.
- [ ] 9. Projet `setup` (`web/recette/connexion.setup.ts`) : demande de code, relève par `mailbox` (appelé par `execFile`), saisie, état de session écrit dans le volume.
- [ ] 10. `make recette-auto` (ADR 0010) : contrôle des fichiers du volume, version servie par la recette lue et vérifiée contre `VERSION` ou `PR`, exécution, archive et empreinte des preuves (D9) ; `/web/recette-results/` dans `.gitignore`.
- [ ] 11. `make ci` ne lance ni les parcours ni la relève ; il compile et vérifie (`tsc`, `go vet`, `staticcheck`) `web/recette/` et l'outil.

### Lot C : parcours de base et première recette (`feature/parcours-recette`)

- [ ] 12. Parcours de base (ADR 0032, point 5) : connexion (projet `setup`) ; chaque onglet dans chaque projet ; créer un plat suffixé de l'exécution et le retrouver par la recherche ; l'ajouter à un repas de demain et relire le planning ; ajouter les courses du repas à « Courses » et cocher un article ; recharger et retrouver chaque écriture.
- [ ] 13. Contrôles du serveur visibles depuis le navigateur : la page de la tribu sert `Cache-Control: no-cache`, les fichiers à empreinte `immutable`, le cookie de session préfixé `__Host-` (lus dans les réponses enregistrées par Playwright, sans rien ajouter à l'application).
- [ ] 14. Séance S1 (ci-dessous), puis première exécution complète sur l'archive de la PR d'un lot ou de `main`, avant une release.
- [ ] 15. Première release passée par le déroulé complet (D11), avec ses parcours ciblés ; plan de recette rempli.
- [ ] 16. Clôture : statut « terminé » ; feuille de route ; `CLAUDE.md` (commandes : `recette-auto` ; recette) ; `CHANGELOG.md`.

### Séance S1 : préparation (développeur)

- [ ] Boîte `recette-auto@codingmatters.org` créée sur le MX Plan, avec un mot de passe propre ; POP3 sur TLS accessible (serveur et port relevés dans l'espace client OVHcloud, à reporter dans `recette-auto.json`) ; aucune redirection vers elle, aucune depuis elle.
- [ ] Tribu `recette-auto` créée sur l'instance de recette (`admin init`, interactif, membre unique `recette-auto@codingmatters.org`) ; commande exacte vérifiée au lot B et écrite dans `deploy/README.md`.
- [ ] Volume du Dev Container rempli : `recette-auto.json`, `recette-auto.pop3` en `0600`.
- [ ] Première relève à la main : `go run ./internal/tools/mailbox …` après une demande de code depuis le navigateur.

## Critères de validation

- `make ci` vert à chaque PR, sans accès à la boîte.
- `make recette-auto` passe sur la recette depuis le Dev Container, en une seule demande de code, et produit une archive de preuves avec son empreinte ; un parcours qui échoue fait échouer la commande et laisse sa vidéo et sa trace.
- La séance manuelle d'une release ne contient que des points classés manuels par le plan de recette, et chacun dit pourquoi.
- Sur l'essai à blanc de `v0.5.0` (étape 5), l'analyse classe en manuel ce que la séance avait trouvé à la main.

### Ce que ce plan ne prouve pas

- La réception du code sur Gmail, Outlook et iCloud, hors des indésirables : vérification manuelle ponctuelle (point reporté de la feuille de route) ; la boîte `recette-auto@` ne prouve que l'envoi par le MX Plan.
- Le vrai Safari et les vrais appareils : toujours dans la voie manuelle.
- La justesse de l'analyse elle-même : elle se juge release après release, aux défauts trouvés en production qu'elle n'avait pas vus.

## Questions ouvertes

- **Lot A pour `v0.6.0`** (D12) : le développeur veut-il l'analyse dès la recette de `v0.6.0`, quitte à fusionner le lot A avant la PR de release de `courses-suite` ?
- **Projets** : quatre suffisent-ils, ou faut-il la tablette en portrait (`moyen`) dans les parcours de base ?
- **Taille des preuves** : vidéo de chaque test dans quatre projets ; à mesurer à la première exécution, quitte à ne garder la vidéo que du projet `setup` et des échecs.
- **Tribu `recette-auto` qui s'emplit** : la recréer à chaque reconstruction de la recette, ou ajouter une commande d'administration qui la vide ?
- **Parcours de deux appareils** (écritures croisées, deux contextes Playwright sur la même session) : utiles en automatique, ou laissés à la séance manuelle ?

## Notes d'exécution
