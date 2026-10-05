# language: fr
Fonctionnalité: Connexion et session persistante
  En tant que membre d'une tribu,
  je veux me connecter une seule fois par appareil avec un code reçu par e-mail,
  afin d'utiliser l'application sans mot de passe ni reconnexion fréquente.

  Contexte:
    Étant donné la tribu "Les Martin" d'identifiant "martin"
    Et "alice@exemple.fr" est membre actif de la tribu "martin"

  @ENF-01
  Scénario: Se connecter avec un code reçu par e-mail
    Étant donné j'ouvre l'URL de la tribu "martin" sans être connecté
    Quand je saisis "alice@exemple.fr"
    Alors un code à 8 chiffres est envoyé à "alice@exemple.fr"
    Quand je saisis ce code
    Alors je suis connecté à la tribu "martin"
    Et une session est ouverte pour cet appareil

  @ENF-01
  Scénario: Même réponse pour une adresse inconnue
    Étant donné j'ouvre l'URL de la tribu "martin" sans être connecté
    Quand je saisis "inconnu@exemple.fr"
    Alors l'application affiche le même message que pour une adresse membre
    Et aucun e-mail n'est envoyé

  @ENF-01
  Scénario: Un membre d'une autre tribu ne reçoit pas de code
    Étant donné "david@exemple.fr" est membre actif de la tribu "durand" uniquement
    Et j'ouvre l'URL de la tribu "martin" sans être connecté
    Quand je saisis "david@exemple.fr"
    Alors l'application affiche le même message que pour une adresse membre
    Et aucun e-mail n'est envoyé

  @ENF-01
  Scénario: Un membre révoqué ne reçoit pas de code
    Étant donné "bruno@exemple.fr" est membre révoqué de la tribu "martin"
    Quand "bruno@exemple.fr" demande un code pour la tribu "martin"
    Alors l'application affiche le même message que pour une adresse membre
    Et aucun e-mail n'est envoyé

  @ENF-01
  Scénario: Un code erroné est refusé
    Étant donné j'ai demandé un code pour "alice@exemple.fr"
    Quand je saisis un code erroné
    Alors je ne suis pas connecté
    Et un message m'indique que le code est incorrect

  @ENF-01
  Scénario: Le nombre de tentatives est limité
    Étant donné j'ai demandé un code pour "alice@exemple.fr"
    Quand je saisis un code erroné 3 fois
    Alors le code est invalidé
    Et même le bon code n'est plus accepté

  @ENF-01
  Scénario: Un code expiré est refusé
    Étant donné j'ai demandé un code pour "alice@exemple.fr"
    Et plus de 10 minutes se sont écoulées depuis la demande
    Quand je saisis ce code
    Alors je ne suis pas connecté
    Et un message m'invite à demander un nouveau code

  @ENF-01
  Scénario: Il reste des essais après un code erroné
    Étant donné j'ai demandé un code pour "alice@exemple.fr"
    Quand je saisis un code erroné 2 fois
    Et je saisis le bon code
    Alors je suis connecté à la tribu "martin"

  @ENF-01
  Scénario: Une nouvelle demande remplace le code précédent
    Étant donné j'ai demandé un code pour "alice@exemple.fr"
    Et j'ai demandé un nouveau code pour "alice@exemple.fr"
    Quand je saisis le premier code
    Alors je ne suis pas connecté
    Mais le second code est accepté

  @ENF-01
  Scénario: Le nombre de demandes de code par adresse est limité
    Étant donné "alice@exemple.fr" a demandé 3 codes pour la tribu "martin" dans le dernier quart d'heure
    Quand elle demande un nouveau code
    Alors l'application affiche « Réessayez dans quelques minutes »
    Et aucun e-mail n'est envoyé

  @ENF-01
  Scénario: La limite par adresse ne révèle pas si l'adresse est membre
    Étant donné "inconnu@exemple.fr" a demandé 3 codes pour la tribu "martin" dans le dernier quart d'heure
    Quand il demande un nouveau code
    Alors l'application affiche « Réessayez dans quelques minutes »

  @ENF-01
  Plan du scénario: Le nombre de demandes est aussi limité par IP et par tribu
    Étant donné <demandes> ont été faites dans la dernière heure
    Quand une nouvelle demande de code est faite <origine>
    Alors l'application affiche « Réessayez dans quelques minutes »
    Et aucun e-mail n'est envoyé

    Exemples:
      | demandes                                             | origine                                   |
      | 10 demandes de code depuis la même adresse IP        | depuis cette adresse IP, pour une autre adresse e-mail |
      | 30 demandes de code pour la tribu "martin"           | pour la tribu "martin", depuis une autre adresse IP |

  @ENF-01
  Scénario: Un code ne sert qu'une fois
    Étant donné je me suis connecté à la tribu "martin" en tant que "alice@exemple.fr" avec un code
    Quand je saisis à nouveau ce code sur un autre appareil
    Alors je ne suis pas connecté sur cet autre appareil

  @ENF-01 @manuel
  Scénario: La session survit au redémarrage du navigateur
    Étant donné je suis connecté à la tribu "martin" en tant que "alice@exemple.fr" dans mon navigateur
    Quand je ferme complètement le navigateur puis le rouvre sur l'URL de la tribu "martin"
    Alors je suis toujours connecté, sans ressaisir de code

  @ENF-01 @manuel
  Scénario: La session survit à la fermeture de l'app installée
    Étant donné je suis connecté à la tribu "martin" en tant que "alice@exemple.fr" dans l'app installée
    Quand je ferme l'app puis la rouvre
    Alors je suis toujours connecté, sans ressaisir de code

  @ENF-01
  Scénario: L'expiration est glissante
    Étant donné je me suis connecté il y a 80 jours
    Et j'ai utilisé l'application hier
    Quand j'utilise l'application aujourd'hui
    Alors je suis toujours connecté
    Et ma session expire 90 jours après aujourd'hui

  @ENF-01
  Scénario: Une session inutilisée expire
    Étant donné je n'ai pas utilisé l'application depuis plus de 90 jours
    Quand j'ouvre l'URL de la tribu "martin"
    Alors je vois l'écran de connexion

  @ENF-01
  Scénario: Le cookie de session n'est pas lisible par le code de la page
    Étant donné je suis connecté à la tribu "martin" en tant que "alice@exemple.fr"
    Alors le cookie de session est marqué HttpOnly, Secure et SameSite=Lax
    Et le jeton de session n'est pas stocké en clair côté serveur

  @ENF-01
  Scénario: Consultation hors ligne
    Étant donné je suis connecté à la tribu "martin" en tant que "alice@exemple.fr" et j'ai déjà affiché le planning
    Quand je perds le réseau et rouvre l'application
    Alors je vois le planning en cache
    Et ma session est revérifiée au retour du réseau

  @ENF-01
  Scénario: Consultation hors ligne de la bibliothèque de plats
    Étant donné je suis connecté à la tribu "martin" en tant que "alice@exemple.fr" et j'ai déjà affiché la bibliothèque de plats
    Quand je perds le réseau et rouvre l'application
    Alors je vois la bibliothèque de plats en cache

  @ENF-01
  Plan du scénario: Les modifications autres que les listes de courses nécessitent le réseau
    Étant donné je suis connecté à la tribu "martin" en tant que "alice@exemple.fr"
    Et le réseau est coupé
    Quand je tente de <action>
    Alors un message m'indique que le réseau est nécessaire
    Et rien n'est modifié

    Exemples:
      | action                              |
      | ajouter un plat à un repas          |
      | modifier le nombre de parts d'un plat servi |
      | créer un plat                       |
      | modifier un plat                    |
      | ajouter un membre                   |
      | révoquer une de mes sessions        |

  @ENF-01
  Scénario: La déconnexion efface les données de la tribu sur l'appareil
    Étant donné je suis connecté à la tribu "martin" en tant que "alice@exemple.fr" et j'ai déjà affiché le planning
    Quand je me déconnecte
    Alors aucune donnée de la tribu "martin" ne reste stockée sur l'appareil

  @ENF-01
  Scénario: Une session révoquée hors ligne est fermée au retour du réseau
    Étant donné je suis hors ligne sur mon téléphone
    Et ma session sur ce téléphone est révoquée depuis un autre appareil
    Quand le réseau revient
    Alors le téléphone affiche l'écran de connexion
    Et aucune donnée de la tribu "martin" ne reste stockée sur le téléphone
