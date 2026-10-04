# 0021. Base de limitation des demandes de code

- **Date** : 2026-10-03
- **Statut** : accepté

## Contexte

ENF-01 limite les demandes de code de connexion : 3 par quart d'heure pour une adresse dans une tribu, 10 par heure pour une adresse IP, 30 par heure pour une tribu. La limite par adresse s'applique de la même façon à une adresse inconnue, révoquée ou d'une autre tribu, et à une tribu qui n'existe pas (ENF-02, ADR 0006, point 9) : il faut donc compter des demandes, et retenir des codes, pour des tribus et des adresses qui n'ont pas de base où les écrire.

Les codes des membres réels vivent dans la base de leur tribu (ADR 0003, point 1). Le registre ne contient aucune donnée personnelle (ADR 0003, point 2). La mémoire du processus est perdue au redémarrage et propre à une instance, alors qu'un redémarrage ne doit remettre à zéro ni les limites ni les essais restants (plan du 2026-10-03, D3).

## Décision

1. **Une troisième sorte de base SQLite**, à côté du registre et des bases de tribu : `ratelimit.db`, dans le répertoire de données. Elle a sa propre série de migrations (`migrations/ratelimit/`), appliquée au démarrage comme les autres (ADR 0003, point 6), et son paquet sqlc. Même configuration que les autres bases (ADR 0003, point 5).
2. **Elle contient** :
   - les **demandes de code** acceptées, horodatées : identifiant d'URL de la tribu (existante ou non), empreinte de l'adresse, empreinte de l'IP ;
   - les **codes fantômes** : pour une tribu inexistante ou une adresse qui n'est pas membre actif, un code qui n'est jamais envoyé et ne peut pas réussir, avec la même échéance (10 minutes) et les mêmes 3 essais qu'un vrai code. Les réponses de l'API sont ainsi identiques pour un membre actif et pour toute autre adresse.
3. **Empreintes** : SHA-256, jamais de valeur en clair.
   - Une adresse est hachée avec l'identifiant d'URL de la tribu : une personne membre de deux tribus n'a pas la même empreinte dans les deux, et rien ne relie ses appartenances (ENF-02, D10).
   - Une IP est hachée seule : la limite par IP vaut pour toute l'instance.
   - L'identifiant d'URL est conservé en clair s'il a le format d'EF-08 : ce n'est pas une donnée personnelle, et il figure déjà dans l'URL. Un segment d'URL qui n'a pas ce format, qu'aucune tribu ne peut porter, est un texte libre choisi par le client : seule son empreinte est conservée, précédée de `#` (revue du lot B, 2026-10-04).
   - Ces empreintes restent des données personnelles (une adresse se devine par essais) : elles sont effacées à la sortie de leur fenêtre d'une heure, par l'effacement automatique qui passe toutes les dix minutes : une heure et dix minutes au plus après la demande, tant que le serveur tourne (arrêté, il les efface à son redémarrage).
   - Dans les bases de tribu, le code de connexion et le jeton de session sont stockés de la même façon, en SHA-256. Le jeton (32 octets aléatoires) ne se retrouve pas à partir de son empreinte ; un code à 6 chiffres, si : sa protection tient à sa validité de 10 minutes et à ses 3 essais.
4. **Interface** : le code métier ne voit que la décision de limitation (une fonction pure des demandes passées) et une interface de stockage ; la base de limitation en est la seule implémentation.
5. **Pas d'instantané avant déploiement** pour ce fichier (ADR 0016) : le perdre remet les compteurs à zéro et efface les codes fantômes, sans autre conséquence.
6. **Nom de la tribu** (D9) : une fois la session vérifiée, le nom affiché est lu dans la base de la tribu, qui fait foi ; le nom du registre ne sert qu'aux commandes d'administration, qui écriront les deux le jour où une story renommera une tribu.

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
