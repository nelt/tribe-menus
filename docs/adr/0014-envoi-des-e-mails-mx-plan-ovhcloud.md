# 0014. Envoi des e-mails par le MX Plan OVHcloud

- **Date** : 2026-09-27
- **Statut** : accepté

## Contexte

Le seul e-mail envoyé par l'application est le code de connexion (ENF-01, ADR 0001). Sa délivrabilité conditionne la première connexion sur chaque appareil. Le volume est très faible : quelques messages par jour.

Le domaine `codingmatters.org` dispose déjà d'un MX Plan OVHcloud actif, qui fournit un serveur SMTP authentifié et prend en charge DKIM. Le souci de limiter les dépendances à des tiers s'applique (ADR 0002, 0009).

## Décision

1. **Envoi par le SMTP du MX Plan** : `ssl0.ovh.net`, port 465, TLS, authentification obligatoire.
2. **Compte dédié** `no-reply@codingmatters.org`, utilisé uniquement par l'application. Son mot de passe fait partie de la configuration du serveur, jamais du dépôt (ADR 0011).
3. **Authentification du domaine**, dans la zone DNS OVHcloud (ADR 0007) :
   - **SPF** autorisant les serveurs d'envoi d'OVHcloud ;
   - **DKIM** activé pour le MX Plan ;
   - **DMARC** d'abord en observation (`p=none`, avec adresse de réception des rapports), puis durci (`quarantine`, puis `reject`) une fois les rapports vérifiés.
4. **Côté application** :
   - envoi par `net/smtp` de la bibliothèque standard (connexion TLS directe sur le port 465), sans dépendance ;
   - l'envoi est caché derrière une petite interface (`Mailer`), avec une implémentation SMTP pour la production et une implémentation qui écrit dans les logs pour le développement et les tests (ADR 0009) ; changer de fournisseur ne touche que cette implémentation ;
   - message en texte brut, court, en français, avec le code, sa durée de validité et le nom de l'application ; sans lien cliquable ni nom de tribu (ENF-02).
5. **Protection du quota** : la limitation des demandes de code (par adresse, par IP et par tribu) prévue par ENF-01 protège aussi le quota d'envoi horaire du compte.
6. **Validation avant la mise en service** : réception d'un code vérifiée sur les messageries réellement utilisées par les membres (Gmail, Outlook, iCloud…), y compris hors du dossier des indésirables.

## Alternatives envisagées

- **Service transactionnel (Brevo, Scaleway TEM, Mailjet…)** : meilleure délivrabilité, suivi des rebonds et journaux d'envoi. Écarté pour la V1 : un fournisseur et un compte de plus, pour un volume de quelques messages par jour. Reste l'option de repli si la délivrabilité déçoit.
- **Serveur SMTP sur le VPS** : écarté. Réputation d'IP à construire, port 25 souvent bloqué, exploitation lourde.

## Conséquences

- **Positif** : aucun coût ni fournisseur supplémentaire ; aucune dépendance de code ; changement de fournisseur isolé derrière l'interface `Mailer`.
- **Négatif** : l'offre est pensée pour des boîtes aux lettres humaines, pas pour de l'envoi applicatif. Pas de suivi des rebonds ni de tableau de bord de délivrabilité ; quotas non documentés officiellement (de l'ordre de 200 messages par heure et par compte selon des retours d'utilisateurs) ; réputation de l'infrastructure mutualisée variable auprès de certains fournisseurs.
- **À surveiller** : les rapports DMARC et les signalements des membres qui ne reçoivent pas leur code.
