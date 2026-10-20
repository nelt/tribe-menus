# 0026. Demande de code traitée en arrière-plan

- **Date** : 2026-10-20
- **Statut** : accepté

## Contexte

ENF-02 veut que rien de ce que répond l'application avant la connexion ne permette de savoir si une adresse est membre d'une tribu. Les réponses étaient identiques (ADR 0021, codes fantômes), mais pas leur temps : la mesure faite en recette le 2026-10-18 (plan `recette`, étape 23) a trouvé, pour `POST login-codes`, des réponses plus lentes pour un membre, avec une vingtaine de demandes par cas. Pour un membre seulement, la demande ouvrait une transaction d'écriture dans la base de la tribu (limite par tribu, code de connexion) et lançait l'envoi de l'e-mail avant de répondre.

La réponse elle-même n'était pas toujours identique : au-delà de la limite par tribu (30 demandes par heure pour ses membres), une adresse de membre recevait `429` (« Réessayez dans quelques minutes ») et une autre adresse `202`. Cette limite ne peut pas être appliquée avant de savoir si l'adresse est membre.

Rien n'était en production : le constat suit le flux public (ADR 0022, point 6).

## Décision

1. **La réponse à une demande de code ne dépend que de la base de limitation.** Le chemin synchrone de `POST login-codes` vérifie l'adresse, applique les limites par adresse et par IP, écrit la demande et le code fantôme dans la base de limitation (ADR 0021), pose le cookie de la demande et répond. Il n'ouvre pas la base de la tribu, ni même le registre : le gestionnaire de la route ne résout plus la tribu.
2. **La suite est traitée en arrière-plan**, dans une file bornée (`tribe.CodeRequests`, 256 demandes) vidée par un seul traitement : ouverture de la base de la tribu, recherche du membre actif, limite par tribu, écriture du code, envoi de l'e-mail. Une tribu inconnue ou une adresse qui n'est pas membre actif ne donne rien de plus.
3. **La limite par tribu refuse en silence.** Au-delà, aucun code n'est écrit ni envoyé ; le refus est compté pour les alertes, comme avant (ADR 0023). La réponse, écrite plus tôt, est celle de toute adresse. La limite garde son rôle : protéger les boîtes des membres et le quota du compte d'envoi (ADR 0014).
4. **File pleine** : la demande est abandonnée, avec un avertissement au journal qui ne nomme pas l'adresse ; la réponse ne change pas. Il faudrait des centaines de demandes acceptées par les limites dans le temps de traitement de quelques-unes.
5. **Arrêt du serveur** : la file est fermée une fois que le serveur ne reçoit plus de requêtes, les demandes en attente sont traitées jusqu'au bout, puis les envois se terminent, avant la fermeture des bases.

Ce que la réponse peut encore dire : une adresse mal formée (`400`) ; une limite par adresse ou par IP atteinte (`429`), qui se compte de la même façon pour toute adresse et toute tribu ; sinon `202`, avec le cookie de la demande.

## Alternatives envisagées

- **Durée plancher des réponses** (attendre, par exemple, 200 ms avant de répondre) : écartée par le développeur le 2026-10-18. Un traitement plus long que le plancher, sous charge ou pour un envoi lent, trahirait encore l'appartenance, et chaque demande coûterait ce temps.
- **`429` pour toute la tribu une fois sa limite atteinte**, membre ou non : écarté. Le chemin synchrone devrait lire la base de la tribu, avec un écart de temps entre une tribu existante et une autre, et la réponse dirait à tout visiteur que la tribu existe pendant l'attaque.
- **Supprimer la limite par tribu** : écarté. Avec des adresses de membres connues et de nombreuses adresses IP, rien ne bornerait plus les e-mails envoyés aux membres d'une tribu.
- **Alerte sur file pleine** : écartée pour l'instant. Une file pleine suppose un afflux que les alertes sur les demandes refusées signalent déjà.

## Conséquences

- **Positif** : le temps de réponse d'une demande de code ne dépend plus de l'appartenance, ni de l'existence de la tribu ; la réponse non plus, limite par tribu comprise. Le chemin synchrone est plus court.
- **Négatif** : pendant une attaque qui atteint la limite d'une tribu, un membre attend un code qui ne vient pas, sans message ; l'administrateur est alerté. Un code arrive un peu après la réponse au lieu d'en même temps, ce qui ne se voit pas, l'e-mail prenant déjà plusieurs secondes.
- **Limite assumée** : le traitement d'un membre occupe un peu le processeur et le disque après la réponse ; une demande suivante pourrait en être ralentie de façon mesurable, à condition d'arriver dans les quelques millisecondes qui suivent. La nouvelle mesure en recette (`make probe … -timing`) dit si cet effet se voit.
- `POST sessions` cherche toujours le membre de façon synchrone : la mesure du 2026-10-18 n'y trouve aucun écart, sans demande de code préalable. Un essai qui suit une vraie demande n'a pas été mesuré ; il ne vient qu'après la réception d'un code, donc d'un e-mail que seul le membre reçoit.
- **Précise** l'ADR 0021 (point 4 : où et quand la limite par tribu est comptée) et ENF-01 et ENF-02 (`docs/specs/exigences-non-fonctionnelles.md`).
