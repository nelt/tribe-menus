# 0021. Base de limitation des demandes de code

- **Date** : 2026-10-03
- **Statut** : accepté

## Contexte

ENF-01 limite les demandes de code de connexion : 3 par quart d'heure pour une adresse dans une tribu, 10 par heure pour une adresse IP, 30 par heure pour une tribu (adressées à ses membres actifs depuis le plan `revue-securite`, D2). La limite par adresse s'applique de la même façon à une adresse inconnue, révoquée ou d'une autre tribu, et à une tribu qui n'existe pas (ENF-02, ADR 0006, point 9) : il faut donc compter des demandes, et retenir des codes, pour des tribus et des adresses qui n'ont pas de base où les écrire.

Les codes des membres réels vivent dans la base de leur tribu (ADR 0003, point 1). Le registre ne contient aucune donnée personnelle (ADR 0003, point 2). La mémoire du processus est perdue au redémarrage et propre à une instance, alors qu'un redémarrage ne doit remettre à zéro ni les limites ni les essais restants (plan du 2026-10-03, D3).

## Décision

1. **Une troisième sorte de base SQLite**, à côté du registre et des bases de tribu : `ratelimit.db`, dans le répertoire de données. Elle a sa propre série de migrations (`migrations/ratelimit/`), appliquée au démarrage comme les autres (ADR 0003, point 6), et son paquet sqlc. Même configuration que les autres bases (ADR 0003, point 5).
2. **Elle contient** :
   - les **demandes de code** acceptées, horodatées : empreinte de l'adresse, empreinte de l'IP. Elles servent aux limites par adresse et par IP ;
   - les **codes fantômes** : pour une tribu inexistante ou une adresse qui n'est pas membre actif, un code qui n'est jamais envoyé et ne peut pas réussir, avec la même échéance (10 minutes) et les mêmes 3 essais qu'un vrai code. Les réponses de l'API sont ainsi identiques pour un membre actif et pour toute autre adresse. Comme un vrai code, un code fantôme garde l'empreinte du jeton de sa demande, posé dans un cookie du navigateur qui l'a demandé (2026-10-05, plan `revue-securite`, D5) : un essai venu d'un autre navigateur reçoit la même réponse, et ne consomme rien, pour toute adresse.
3. **Empreintes** : SHA-256, jamais de valeur en clair.
   - Une adresse est hachée avec l'identifiant d'URL de la tribu : une personne membre de deux tribus n'a pas la même empreinte dans les deux, et rien ne relie ses appartenances (ENF-02, D10).
   - Une IP est hachée seule : la limite par IP vaut pour toute l'instance. Elle est d'abord normalisée (2026-10-06, plan `production`, D4) : une IPv6 est réduite à son préfixe /64, une IPv4 encapsulée dans une IPv6 ramenée à l'IPv4.
   - L'identifiant d'URL n'est conservé que dans l'empreinte de l'adresse, quel que soit son format (plan `revue-securite`, D2). Il l'était auparavant en clair, ou en empreinte pour un segment hors du format d'EF-08 (revue du lot B, 2026-10-04), pour la limite par tribu.
   - Ces empreintes restent des données personnelles (une adresse se devine par essais) : elles sont effacées à la sortie de leur fenêtre d'une heure, par l'effacement automatique qui passe toutes les dix minutes : une heure et dix minutes au plus après la demande, tant que le serveur tourne (arrêté, il les efface à son redémarrage).
   - Dans les bases de tribu, le code de connexion, le jeton de sa demande et le jeton de session sont stockés de la même façon, en SHA-256. Un jeton (32 octets aléatoires) ne se retrouve pas à partir de son empreinte ; un code à 8 chiffres, si : sa protection tient à sa validité de 10 minutes, à ses 3 essais, et à ce qu'il ne se saisit que depuis le navigateur qui l'a demandé.
4. **La limite par tribu se compte dans la base de la tribu** (2026-10-05, plan `revue-securite`, D2). Elle ne porte que sur les demandes adressées aux membres actifs : si la base de limitation la comptait, elle apprendrait quelles empreintes d'adresse sont celles de membres. La base de la tribu garde l'heure de chacune de ces demandes, sans adresse, effacée à la sortie de la fenêtre d'une heure. Une demande est d'abord comptée par la base de limitation, comme toute autre, puis, pour un membre actif, par la base de la tribu : ce que la première enregistre ne dépend pas de l'appartenance, y compris quand la limite de la tribu refuse la demande. Une tribu inexistante n'a pas de base, et n'atteint jamais cette limite, comme une tribu existante visée par des adresses qui ne sont pas membres.
   - Précisé le 2026-10-06 (plan `production`, D11 ; ADR 0023) : ce point n'était pas tenu jusque-là. La base de limitation n'écrivait un code fantôme que pour une adresse qui n'était pas membre actif : une demande enregistrée sans code fantôme au même instant désignait, pendant les dix minutes de validité du code, l'empreinte d'un membre, à qui lisait le fichier. Depuis, elle écrit le code fantôme pour toute demande acceptée par les limites, membre ou non ; celui d'un membre n'est jamais vérifié et s'efface comme les autres. Elle garde aussi, sans empreinte, des compteurs par tranche de dix minutes (demandes refusées par la limite par adresse ou par IP, codes fantômes épuisés) et l'heure de la dernière alerte envoyée ; les événements qui supposent un membre (refus par la limite de la tribu, code réel épuisé) sont comptés dans la base de la tribu. Ce point ne tient donc qu'à la demande : qui lit la base de limitation peut encore apprendre si une adresse est membre d'une tribu, en demandant lui-même un code pour elle et en y saisissant un code faux, puis en lisant les essais restants du code fantôme (un membre garde ses trois essais, le sien n'étant jamais vérifié) ou le compteur `decoy_code_exhausted`. Limite assumée (plan `production`, D13 ; ADR 0023) : ce fichier ne se lit qu'avec le répertoire de données, où les bases des tribus portent les adresses des membres en clair.
5. **Interface** : le code métier ne voit que la décision de limitation (une fonction pure des demandes passées) et une interface de stockage ; la base de limitation en est la seule implémentation.
6. **Pas d'instantané avant déploiement** pour ce fichier (ADR 0016) : le perdre remet les compteurs à zéro et efface les codes fantômes, sans autre conséquence.
7. **Nom de la tribu** (D9) : une fois la session vérifiée, le nom affiché est lu dans la base de la tribu, qui fait foi ; le nom du registre ne sert qu'aux commandes d'administration, qui écriront les deux le jour où une story renommera une tribu.

## Alternatives envisagées

- **Mémoire du processus** : écartée. Perdue au redémarrage (un redémarrage rendrait 3 essais sur chaque code) et propre à une instance.
- **Registre** : écarté. Il ne contient aucune donnée personnelle (ADR 0003, point 2), et une empreinte d'adresse en est une.
- **Base de la tribu pour toutes les demandes** : écartée. Elle n'existe pas pour une tribu inconnue, et la limite par IP vaut pour toute l'instance.
- **Empreinte d'adresse sans l'identifiant d'URL** : écartée. Une même personne aurait la même empreinte dans toutes ses tribus.

## Conséquences

- **Positif** : limites et essais survivent au redémarrage ; aucune adresse ni IP en clair ; une tribu inexistante se comporte exactement comme une tribu existante ; l'interface laisse la porte ouverte à un stockage partagé.
- **Négatif** : une base de plus à migrer et à effacer périodiquement. SQLite suppose toujours une seule machine : un vrai fonctionnement multi-instance demandera de revoir ce stockage, comme celui des tribus.
- **Confidentialité** : la conservation des empreintes, une heure et dix minutes au plus, est mentionnée dans `gestion-membres-et-sessions.md` et dans la page Confidentialité.
- **Précise** l'ADR 0001 (limitation des demandes et des essais) et l'ADR 0003 (sortes de bases).
