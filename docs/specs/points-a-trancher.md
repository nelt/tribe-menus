# Points à trancher avant de coder

- **Date** : 2026-09-28
- **Origine** : analyse de cohérence des specs, du design et des ADR.

Chaque point est à décider puis reporté dans le document de référence cité (specs, `.feature`, glossaire, ADR ou maquette). Un point tranché est coché, avec la décision en une ligne et le lien vers la modification.

## Bloquants

- [x] **PT-01. Unités de la famille « autres »** (`fonctionnalites.md`, `glossaire.md`, ADR 0003 point 7)
  - Pièce, cuillère à soupe, cuillère à café et pincée sont rangées dans une même famille, sans conversion définie entre elles, alors que l'ADR 0003 prévoit une unité de base unique (« millièmes de pièce »).
  - À décider : chaque unité forme-t-elle sa propre famille, ou définit-on des conversions (1 c. à soupe = 3 c. à café) ? L'arrondi à l'entier supérieur s'applique-t-il aux cuillères et aux pincées, ou seulement à la pièce ?
  - Conséquence : règle « encore présent = même famille » du recalcul, unité de base de stockage, scénarios C2.
  - **Décision (2026-09-28)** : pièce, cuillère à soupe, cuillère à café et pincée forment chacune leur propre famille, sans conversion ; toutes sont arrondies à l'entier supérieur après agrégation. Reporté dans `fonctionnalites.md` (unités, Q3, Q4), `glossaire.md`, ADR 0003 point 7 et `liste-courses.feature` (C2).

- [x] **PT-02. Paramètres de connexion et limitation des demandes** (ENF-01, `authentification.feature`, ADR 0006 point 7, ADR 0014 point 5, `identite.md`)
  - Valeurs absentes : durée de validité du code, nombre d'essais, seuils de limitation des demandes.
  - Les ADR 0006 et 0014 renvoient à une limitation « par adresse, par IP et par tribu » prévue par ENF-01, qui ne la décrit pas ; l'état « trop de demandes » des maquettes n'a pas de scénario.
  - À décider aussi : une nouvelle demande invalide-t-elle le code précédent ? La limitation par adresse s'applique-t-elle de la même façon aux adresses inconnues (pour ne rien révéler, ENF-02) ?
  - **Décision (2026-09-28)** : code valable 10 minutes, 3 essais ; une nouvelle demande invalide le code précédent ; limites de 3 demandes par quart d'heure par adresse et par tribu (adresses inconnues comprises), 10 par heure par IP, 30 par heure par tribu. Reporté dans ENF-01 et `authentification.feature`.

- [x] **PT-03. Conflits de la file hors ligne** (C4, Q16, ADR 0004 point 7)
  - Sort d'une coche ou d'un article ajouté rejoué au retour du réseau si, entre-temps, la liste a été recalculée (article disparu), déclarée faite (figée, Q14) ou abandonnée.
  - Un article ajouté hors ligne peut créer un ingrédient (C8, Q13) : l'ADR 0004 ne prévoit d'identifiants générés côté client que pour les articles ; risque de doublons dans le référentiel si deux membres créent le même ingrédient.
  - **Décision (2026-09-28)** : liste faite ou abandonnée entre-temps → modifications ignorées et signalées ; article retiré par un recalcul → coche ignorée et signalée ; ingrédient créé hors ligne → rapproché par nom normalisé (normalisation fixée par PT-06). Reporté dans `fonctionnalites.md` (Q18), `liste-courses.feature` (C4) et ADR 0004 point 7.

- [ ] **PT-04. Périmètre du hors-ligne hors listes de courses** (ENF-01, `identite.md` écran de chargement, ADR 0004)
  - ENF-01 promet le planning en cache ; l'ADR 0004 ne stocke que les listes en IndexedDB (le reste via le cache du service worker).
  - À préciser : écrans consultables sans réseau (planning, plats, autocomplétion), comportement d'une modification du planning ou d'un plat hors ligne (refus avec message, comme C4 ?).
  - À préciser : purge des données locales (IndexedDB, caches) à la déconnexion, à la révocation d'une session ou d'un membre (EF-03, EF-05).

- [ ] **PT-05. Couleur des plats** (`design/README.md` tokens, P1)
  - Le design attribue une couleur à chaque plat ; aucune story ne dit comment elle est choisie (par le membre, automatiquement dans la palette) ni si elle est stockée.

- [ ] **PT-06. Référentiel d'ingrédients** (P4, C8, T1, ENF-02)
  - L'unité par défaut d'un ingrédient est définie mais n'intervient dans aucune story (pré-remplissage de l'unité à la saisie ?).
  - Des scénarios créent un ingrédient seul, avec une unité (`tribu.feature` T1, `compartimentage-tribus.feature`), sans story ni écran correspondant.
  - Normalisation des noms non spécifiée (casse, accents, espaces) : la recherche P3 ignore les accents, l'autocomplétion P4 ne le dit pas.
  - Correction d'une faute de frappe ou fusion de doublons : hors périmètre V1 à acter, ou story à ajouter.

- [ ] **PT-07. Conservation et effacement des données** (EF-03, EF-07, `identite.md` page Confidentialité)
  - La page Confidentialité exige des durées de conservation et une modalité de suppression sur demande.
  - Or le journal d'audit est en ajout seul et un membre révoqué n'est jamais supprimé : son e-mail est conservé sans limite. Codes expirés et sessions expirées : purge non spécifiée.

## Mineurs

- [ ] **PT-08. Parts initiales d'un plat servi** (Q8, R2) : un plat servi part toujours de 4 parts, même si le plat est défini pour 6 parts (P1). À confirmer, ou partir des parts de référence du plat.
- [ ] **PT-09. Planning glissant** (R1, R5, Q2) : les repas du jour ne sont visibles que via « semaine précédente ». Les repas passés sont-ils modifiables ?
- [ ] **PT-10. Même plat deux fois dans un repas** (R2) : autorisé ou refusé ?
- [ ] **PT-11. Révocation affichée sur la fiche membre** (EF-02, `Doux-Membre.dc.html`) : la maquette affiche « Révoqué par … · date », qu'EF-02 ne prévoit pas (seuls la date et l'auteur de l'ajout y figurent).
- [ ] **PT-12. Migrations et contrôle de santé** (ADR 0003 point 6, ADR 0016) : les migrations s'appliquent à l'ouverture de chaque base ; `/healthz` après déploiement ne détecte pas l'échec de migration d'une tribu ouverte plus tard. Migrer toutes les bases au démarrage ?
- [ ] **PT-13. Contrôle des échappatoires au rendu** (ADR 0004 point 9, ADR 0010) : l'interdiction de `unsafeHTML` et `innerHTML` est « vérifiée par lint ou revue », mais `make lint` n'a pas d'outil pour cela (pas de linter JavaScript). Petit contrôle écrit dans le projet, ou revue seule ?
- [ ] **PT-14. Harmonisation des étapes Gherkin** (`features/`, ADR 0005) : même idée écrite de plusieurs façons (dates avec ou sans année, « dans la période », « je suis connecté … en tant que » / « dans mon navigateur » / « Alice se connecte »). À normaliser avant d'écrire les définitions d'étapes godog.
- [ ] **PT-15. Scénarios non automatisables tels quels** (ADR 0005) : scénarios négatifs (« aucune action ne permet de supprimer le plat », « aucune fonction … ne permet de modifier le journal »), survie de la session après fermeture de l'app installée. Décider de leur preuve (absence de route dans l'API, test manuel en recette…).
- [ ] **PT-16. Glossaire** : ajouter nom de session (EF-04), tentative et limitation (ENF-01), noms des opérations d'audit (EF-07), provenance d'un article et quantité apportée (C9).
