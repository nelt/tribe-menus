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
    Alors un code à 6 chiffres est envoyé à "alice@exemple.fr"
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
    Quand je saisis un code erroné autant de fois que le maximum autorisé
    Alors le code est invalidé
    Et même le bon code n'est plus accepté

  @ENF-01
  Scénario: Un code expiré est refusé
    Étant donné j'ai demandé un code pour "alice@exemple.fr"
    Et la durée de validité du code est dépassée
    Quand je saisis ce code
    Alors je ne suis pas connecté
    Et un message m'invite à demander un nouveau code

  @ENF-01
  Scénario: Un code ne sert qu'une fois
    Étant donné je me suis connecté avec un code
    Quand je saisis à nouveau ce code sur un autre appareil
    Alors je ne suis pas connecté sur cet autre appareil

  @ENF-01
  Scénario: La session survit au redémarrage du navigateur
    Étant donné je suis connecté à la tribu "martin" dans mon navigateur
    Quand je ferme complètement le navigateur puis le rouvre sur l'URL de la tribu "martin"
    Alors je suis toujours connecté, sans ressaisir de code

  @ENF-01
  Scénario: La session survit à la fermeture de l'app installée
    Étant donné je suis connecté à la tribu "martin" dans l'app installée
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
    Étant donné je suis connecté à la tribu "martin"
    Alors le cookie de session est marqué HttpOnly, Secure et SameSite=Lax
    Et le jeton de session n'est pas stocké en clair côté serveur

  @ENF-01
  Scénario: Consultation hors ligne
    Étant donné je suis connecté à la tribu "martin" et j'ai déjà affiché le planning
    Quand je perds le réseau et rouvre l'application
    Alors je vois le planning en cache
    Et ma session est revérifiée au retour du réseau

  @ENF-01
  Scénario: Une session révoquée hors ligne est fermée au retour du réseau
    Étant donné je suis hors ligne sur mon téléphone
    Et ma session sur ce téléphone est révoquée depuis un autre appareil
    Quand le réseau revient
    Alors le téléphone affiche l'écran de connexion
