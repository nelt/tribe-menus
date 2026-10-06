# 0023. Alertes de l'application

- **Date** : 2026-10-06
- **Statut** : accepté

## Contexte

ENF-01 demande que les limites de demandes de code atteintes de façon répétée soient signalées à l'administrateur par e-mail (plan `revue-securite`, D1) : une force brute ou un blocage ciblé est visible de ses victimes, il doit l'être aussi de l'administrateur. L'ADR 0015 (point 15) prévoit des alertes par e-mail depuis le serveur, par `msmtp`, dont les échecs répétés d'envoi SMTP « signalés par l'application », sans dire comment l'application les signale.

Jusqu'au plan `production`, rien n'était compté : une demande refusée par une limite recevait un 429, un code épuisé était effacé, sans trace. La base de limitation ne doit pas apprendre quelles empreintes d'adresse sont celles de membres (ADR 0021, point 4), et ce qu'elle garde est borné : les demandes refusées n'y sont pas enregistrées.

Le plan `production` (D6 à D9, D11, D12) a tranché les points ci-dessous ; cet ADR les rassemble.

## Décision

1. **Une alerte est un enregistrement du journal** de niveau erreur portant un attribut `alert`, dont la valeur nomme sa nature : `code_request_limits` ou `smtp_failures`. Le serveur relaie par `msmtp`, avec le compte `server@`, tout enregistrement qui le porte (ADR 0015, points 6 et 15 ; plan `recette`). Un seul mécanisme, qui fonctionne encore quand le compte d'envoi de l'application est en panne.
2. **Limites atteintes de façon répétée** (`code_request_limits`) : trois signaux, chacun sur la dernière heure et pour toute l'instance (D8, D12) :
   - **demandes refusées** : dix demandes refusées par une limite, quelle qu'elle soit (adresse, IP, tribu). Trace d'une inondation ;
   - **codes épuisés** : cinq codes invalidés par leur dernier essai erroné, réels ou fantômes. Trace d'une force brute menée au rythme des limites, qui n'est jamais refusée ;
   - **demandes répétées** : une même empreinte d'adresse à neuf demandes acceptées ou plus. Trace d'un blocage ciblé mené au rythme exact de la limite par adresse (douze par heure), qui ne produit ni refus ni code épuisé.
3. **Échecs répétés d'envoi** (`smtp_failures`) : trois envois échoués en une heure, vérification du compte au démarrage comprise (D9).
4. **Pas de répétition** : au plus une alerte par signal (et, pour `smtp_failures`, par nature) en six heures, tant que la situation dure. Seuils et délais sont des constantes du code, pas de la configuration (`socle`, D15).
5. **L'alerte sur les limites part aussi par e-mail depuis l'application** (D6), à l'adresse de la clé `alerts.to` du fichier de configuration, obligatoire en mode serveur, par le même `Mailer` que les codes : elle n'attend pas le relais du serveur. L'alerte sur les échecs d'envoi ne part pas par l'application, dont l'envoi est justement ce qui échoue. En développement, l'e-mail d'alerte part dans les logs, comme les codes.
6. **Contenu de l'e-mail** : en français, en texte brut ; le nom de l'instance (`baseURL`), la fenêtre, les signaux franchis, les compteurs de l'heure par nature d'événement, et un renvoi vers les journaux d'accès de Caddy, qui donnent l'heure, les IP et les URL. **Ni adresse e-mail, ni adresse IP, ni tribu** (nom ou identifiant d'URL), ni empreinte : les empreintes ne se relisent pas, et nommer la tribu visée écrirait dans une boîte aux lettres qu'elle existe et qu'on s'en prend à ses membres. L'enregistrement du journal suit la même règle.
7. **Compteurs** (D7) : une ligne par tranche de dix minutes et par nature d'événement, incrémentée dans la transaction qui constate l'événement, sans empreinte ni identifiant. Leur taille ne dépend pas du nombre de demandes. Chaque événement est compté dans la base qui le constate :
   - base de limitation : demandes refusées par la limite par adresse, par la limite par IP, codes fantômes épuisés ; et l'heure de la dernière alerte envoyée, par signal ;
   - base de la tribu : demandes refusées par la limite de la tribu, codes réels épuisés. Ces événements supposent un membre : comptés dans la base de limitation, à côté de demandes horodatées, ils laisseraient deviner quelles empreintes sont celles de membres.

   Une demande refusée par plusieurs limites compte une fois. Les compteurs survivent au redémarrage, comme les limites (`socle`, D3), et sont effacés avec le reste à la sortie de leur fenêtre d'une heure (PT-07). Les échecs d'envoi se comptent en mémoire : c'est un état de supervision, pas de sécurité, et les compter dans une base y daterait des envois, donc des demandes faites pour des membres.
8. **Évaluation** : avec l'effacement automatique, toutes les dix minutes. Elle additionne les compteurs des six dernières tranches de la base de limitation et de toutes les tribus, et compte les empreintes d'adresse à neuf demandes ou plus parmi les demandes que la base de limitation garde déjà. La décision (compteurs, dernière alerte par signal, heure) est une fonction pure. Les échecs d'envoi sont évalués à chaque échec.
9. **Un code fantôme pour toute demande** (D11) : la base de limitation écrit le code fantôme de l'empreinte d'adresse pour toute demande acceptée par les limites, membre ou non, avant de chercher le membre. Celui d'un membre n'est jamais vérifié ; il expire et s'efface comme les autres. Une demande enregistrée sans code fantôme ne désigne plus l'empreinte d'un membre.

## Alternatives envisagées

- **Tout envoyer par l'application** : écarté, elle est muette quand le SMTP tombe.
- **Tout laisser au serveur** : écarté, ENF-01 lie l'alerte à l'envoi réel des e-mails, apporté par l'application.
- **Lancer `msmtp` depuis l'application** : écarté, un programme externe exécuté par un service confiné (ADR 0015, point 12).
- **Déduire les alertes des seules demandes déjà enregistrées** : sans table de plus, mais les codes épuisés n'y laissent aucune trace, et les demandes refusées n'y sont pas enregistrées.
- **Enregistrer chaque demande refusée** : la base grossirait avec l'inondation qu'elle doit signaler.
- **Nommer la tribu visée dans l'e-mail** : écarté, voir le point 6.
- **Seuils réglables dans le fichier de configuration** : écartés, pas de paramètre de sécurité réglable au lancement (`socle`, D15).

## Conséquences

- **Positif** : les trois attaques étudiées par la revue de sécurité (force brute, inondation, blocage ciblé) laissent chacune un signal ; l'administrateur est prévenu sans que l'e-mail ni le journal n'apprennent qui est visé ; les tables de comptage ont une taille bornée.
- **Négatif** : des seuils fixés sans mesure, à revoir après les premières semaines de recette ; une alerte de six heures en six heures pendant une attaque durable ; un redémarrage remet à zéro le compte des échecs d'envoi et retarde leur alerte (chaque échec reste dans le journal).
- **Limite assumée** (2026-10-06, plan `production`, D13) : la base de limitation dit, à qui la lit, si une adresse est membre d'une tribu, pour l'adresse et la tribu de son choix. Il suffit de demander un code pour cette adresse depuis son propre navigateur et d'y saisir un code faux : le code fantôme d'une adresse qui n'est pas membre active perd un essai, celui d'un membre, jamais vérifié, garde ses trois essais ; et un code épuisé incrémente `decoy_code_exhausted` dans la base de limitation pour une autre adresse, `login_code_exhausted` dans la base de la tribu pour un membre. D11 ne rend les lignes identiques qu'à la demande (point 9), pas après un essai. C'est accepté : lire la base de limitation suppose de lire le répertoire de données, qui contient aussi les bases des tribus, où les adresses des membres sont en clair. Écarté : reporter sur le code fantôme chaque essai d'un membre et compter tout code épuisé dans la seule base de limitation, ce qui changeait D7 et ne protégeait que d'une base de limitation lue seule.
- **Précise** l'ADR 0015 (point 15 : comment l'application signale ses alertes au serveur) et l'ADR 0021 (points 2 et 4 : codes fantômes pour toute demande, compteurs et dernière alerte dans la base de limitation, compteurs dans la base de la tribu).
