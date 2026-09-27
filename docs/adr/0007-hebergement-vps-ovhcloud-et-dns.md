# 0007. Hébergement : VPS OVHcloud et DNS

- **Date** : 2026-09-27
- **Statut** : accepté

## Contexte

L'application est un binaire Go unique avec des bases SQLite locales (ADR 0002, 0003), exposé par Caddy (ADR 0006). SQLite impose une seule machine en écriture ; un VPS est donc le mode d'hébergement naturel. La charge est minime : la plus petite offre de n'importe quel fournisseur est largement suffisante. Le domaine `codingmatters.org` est déjà géré chez OVHcloud.

Critères : hébergement dans l'UE (ou juridiction équivalente), coût, simplicité, stabilité de l'offre.

## Décision

1. **Serveur** : VPS OVHcloud, gamme VPS-1 (2 vCores, 4 Go de RAM, 40 Go NVMe au moment du choix), dans un datacenter français. Trafic illimité, sauvegarde quotidienne et protection anti-DDoS incluses.
2. **DNS** : la zone `codingmatters.org` reste chez OVHcloud.
   - enregistrements A et AAAA `tribe-menus.codingmatters.org` vers le VPS ;
   - enregistrement **CAA** n'autorisant que Let's Encrypt (`letsencrypt.org`) à émettre des certificats pour le domaine ;
   - **DNSSEC** activé.
   Caddy valide le domaine par HTTP-01 ou TLS-ALPN-01, sans accès à l'API DNS : aucun plugin n'est nécessaire.
3. **La sauvegarde incluse par OVHcloud n'est pas la sauvegarde de référence.** Les données sont répliquées hors site, chez un autre fournisseur, par Litestream (ADR 0003) ; la cible sera fixée par un ADR dédié.

## Alternatives envisagées

- **Hetzner (CX23, CAX11 en ARM)** : excellente réputation technique et très bon outillage (API, CLI). Écarté : environ 2 € de plus par mois après la hausse de juin 2026, IPv4 et sauvegardes en supplément.
- **Scaleway** : petites instances moins compétitives (IPv4 et stockage facturés à part). Reste candidat pour le stockage objet et l'envoi d'e-mails.
- **Infomaniak (Suisse), Netcup (Allemagne), IONOS, PulseHeberg, Ikoula** : crédibles, sans avantage décisif.
- **Contabo** : écarté (réputation de sursouscription). **Outscale** : écarté (surdimensionné, orienté SecNumCloud).
- **PaaS (Clever Cloud, Scalingo)** : écartés, SQLite imposant un volume persistant et une seule instance, ce qui retire l'essentiel de leur intérêt.

## Conséquences

- **Positif** : coût minimal, fournisseur français, domaine et serveur sous le même compte.
- **Négatif** : un seul serveur, sans haute disponibilité ; une panne du VPS interrompt le service jusqu'à sa restauration (acceptable pour un usage familial). La réputation du support OVHcloud est inégale.
- **Risque fournisseur** : un incident de datacenter (précédent de Strasbourg en 2021) justifie la réplication hors site chez un autre fournisseur.
- **À vérifier à la commande** : les conditions d'engagement associées au prix affiché.
- **Restent à décider** : système d'exploitation et durcissement, mode de déploiement, cible de sauvegarde hors site, service d'envoi d'e-mails, supervision.
