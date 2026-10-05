# 0022. Traitement des vulnérabilités par analyse de risques

- **Date** : 2026-10-05
- **Statut** : accepté

## Contexte

Le dépôt est public (ADR 0011) et les sessions Claude se parlent par les PR (ADR 0020, `docs/revue-de-pr.md`) : un constat de vulnérabilité écrit dans une revue est public, et le correctif l'est aussi dès qu'il est poussé. Entre ce moment et le déploiement de la version corrigée, la faille est connue et encore exploitable.

Cette durée est structurelle :

- seule une release étiquetée, construite par la CI, va en production (ADR 0012, 0016) ;
- la recette est alimentée par les archives de PR (ADR 0016) : la PR est publique avant le premier essai en recette.

Le plan `canal-prive` a d'abord cherché un canal privé couvrant une vulnérabilité du constat au correctif déployé. Les essais du 2026-10-04 (notes d'exécution du plan) ont montré que les avis de sécurité GitHub ne le permettent pas avec nos sessions : la session cloud ne lit ni l'avis ni son fork privé temporaire ; le Dev Container lit l'avis avec un droit supplémentaire, mais pas le fork.

Le projet est mené par une personne, pour une instance et quelques tribus. Le risque à couvrir est le délai entre un correctif public et son déploiement, pas une divulgation coordonnée entre plusieurs éditeurs.

## Décision

1. **Pas de canal privé sur GitHub.** La PR publique reste le seul chemin d'un correctif vers `main`, et le build reste en CI.
2. **Une analyse de risques décide du traitement de chaque vulnérabilité**, sur trois évaluations : la gravité de la faille, le risque de livrer sans recette, le risque d'exploitation une fois la faille publiée dans une PR.
3. **Trois niveaux de traitement** :
   - *normal* : le flux public habituel ;
   - *accéléré* : constat hors de GitHub, PR ouverte quand le correctif est prêt, relecture sur la PR en termes neutres, recette ciblée, déploiement dans la séance ;
   - *urgent* : constat hors de GitHub, relecture avant le push, PR ouverte le temps de livrer, production sans recette préalable, recette complète après.
4. **Règle** : gravité faible, normal ; gravité élevée et exploitation probable, urgent si livrer sans recette est peu risqué, sinon accéléré avec une mesure d'attente ; accéléré dans les autres cas.
5. **L'analyse précède toute écriture publique.** Une session qui soupçonne une vulnérabilité d'une version en production cesse d'écrire sur GitHub et remet son constat au développeur, dans la conversation. Le relecteur propose les évaluations, le développeur décide du niveau et porte le constat à la session qui corrige.
6. **Ce qui ne touche pas la production relève du traitement normal** : recette seule, code non déployé.
7. **L'analyse est publiée après le déploiement**, sur la PR du correctif, et `CHANGELOG.md` ne décrit la faille qu'à ce moment.
8. **Signalement privé de vulnérabilités activé** sur le dépôt, pour les tiers (`SECURITY.md`). Seul le développeur le lit : aucun jeton de session n'a de droit sur les avis de sécurité.

Le détail (grille, déroulé de chaque niveau, modèles) est dans `docs/traitement-des-vulnerabilites.md`.

## Alternatives envisagées

- **Avis de sécurité GitHub en brouillon, correctif dans le fork privé temporaire** : le chemin prévu par GitHub. Écarté : inaccessible à nos sessions (voir le contexte), et la CI ne tourne pas dans le fork.
- **Dépôt privé compagnon**, miroir de `main`, où vivent constats et correctifs : accessible aux deux sessions. Écarté : un miroir à synchroniser, des jetons à étendre, pas de CodeQL, et la durée entre fusion et déploiement reste entière, sauf à déployer depuis le dépôt privé, ce qui demande une chaîne de release parallèle et un script de déploiement qui accepte une archive locale.
- **Build local réservé aux correctifs de sécurité** : déployer une archive construite dans le Dev Container, puis publier. Supprime l'exposition, sans dépôt supplémentaire. Écarté : entorse à l'ADR 0012, et le binaire en production n'est plus celui de la CI.
- **Développement dans un dépôt privé, miroir public à chaque release** : plus rien ne passe par des PR publiques. Écarté : on perd le développement à ciel ouvert, CodeQL, et le ruleset de `main` sans offre payante (ADR 0020).
- **Tout traiter dans le flux public** : aucun coût. Écarté comme règle unique : pour une faille grave et évidente, la durée d'exposition couvre la relecture et la recette. C'est le niveau normal, réservé aux cas où la gravité est faible.
- **Toujours livrer en urgence** : exposition minimale. Écarté : livrer sans recette un correctif qui touche l'authentification ou une migration est un risque en soi, que l'analyse doit peser.

## Conséquences

- Aucun dépôt, aucun jeton, aucune synchronisation de plus ; le jeton du Dev Container garde les droits de l'ADR 0020.
- Le développeur redevient un relais pour les constats de sécurité : c'est voulu, et rare.
- Une faille traitée au niveau accéléré ou urgent reste publique, lisible dans le diff, le temps de la séance de livraison. Le risque est réduit, pas supprimé.
- Le niveau urgent repose sur le retour arrière automatique (ADR 0016) et sur une chaîne de release rapide.
- Une version corrective emporte tout ce que `main` contient de non publié : l'analyse en tient compte, et garder `main` proche de la dernière release réduit ce risque.
- Les plans `production` et `recette` doivent fournir : une version corrective préparée dans la PR du correctif (`RELEASING.md`), un déploiement en production qui tient dans une séance, une recette ciblée sur l'archive d'une PR, des mesures d'attente documentées, et la répétition d'un traitement accéléré sur un faux constat.
- Si d'autres instances que la nôtre existent un jour, elles sont exposées dès qu'un correctif est public : la publication d'avis de sécurité GitHub sera à reprendre.
