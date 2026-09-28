# 0017. Nom de marque Melting Tribe et nom d'hôte

- **Date** : 2026-09-28
- **Statut** : accepté

## Contexte

L'application a désormais un nom de marque, **Melting Tribe**, retenu avec son identité visuelle (`docs/design/identite.md`). Le nom d'hôte fixé par l'ADR 0006, `tribe-menus.codingmatters.org`, reprend le nom de travail du projet et ne correspond plus à ce que voient les utilisateurs (site public, écran de connexion, icône de l'application installée, e-mail du code).

Le changement est possible sans coût aujourd'hui : rien n'est encore déployé, aucun certificat n'a été émis et aucune application n'a été installée avec l'ancienne adresse.

## Décision

1. **Nom d'hôte** : `meltingtribe.codingmatters.org`, à la place de `tribe-menus.codingmatters.org`. Il remplace l'ancien partout où l'ADR 0006 le cite (site public à `/`, application sous `/tribes/<identifiant>/`) ; le découpage des chemins ne change pas.
2. **Recette** : `recette.meltingtribe.codingmatters.org`, à la place de `recette.tribe-menus.codingmatters.org` (ADR 0016).
3. **DNS** : enregistrements A et AAAA `meltingtribe.codingmatters.org` et `recette.meltingtribe.codingmatters.org` vers le VPS (ADR 0007).
4. **Inchangés** : le domaine `codingmatters.org`, l'adresse d'envoi `no-reply@codingmatters.org` (ADR 0014), le nom du dépôt `tribe-menus` et du module Go `github.com/nelt/tribe-menus` (ADR 0008), les noms techniques côté serveur (service, commande `tribe-menus-deploy`).
5. **Nom affiché** : « Melting Tribe » dans le manifeste servi sans session, le titre des pages publiques, l'écran de connexion et l'e-mail du code (ADR 0006 point 9, ADR 0014).

## Alternatives envisagées

- **Garder `tribe-menus.codingmatters.org`** : écarté, décalage durable entre la marque et l'adresse.
- **`melting-tribe.codingmatters.org`** : écarté au profit de la forme sans tiret, plus proche de la façon dont le nom se prononce et s'écrit d'un bloc dans le logotype.
- **Domaine dédié** (`meltingtribe.fr`, `.org`…) : écarté pour l'instant, un domaine de plus à acheter, renouveler et configurer (SPF, DKIM, DMARC) sans besoin avéré. Pourra être reconsidéré si le projet s'ouvre à d'autres tribus.
- **Renommer aussi le dépôt et le module Go** : écarté, le nom technique n'est pas visible des utilisateurs et un renommage touche chaque import.

## Conséquences

- **Positif** : l'adresse correspond à la marque ; aucun coût de migration puisque rien n'est déployé.
- **Négatif** : deux noms coexistent, la marque côté utilisateurs et `tribe-menus` côté technique ; la correspondance est notée ici et dans `CLAUDE.md`.
- **Documents alignés** : ADR 0006, 0007 et 0016 (notes renvoyant à cet ADR), `CLAUDE.md`.
