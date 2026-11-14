# 0031. État dérivé calculé par le serveur, hors des collections

- **Date** : 2026-11-14
- **Statut** : accepté

## Contexte

Le navigateur garde en mémoire une copie de chaque collection de la tribu (plats, référentiel, planning, listes de courses) et la vérifie par son ETag : un `304` dit qu'elle n'a pas changé (ADR 0029). La version d'une collection n'augmente que quand ses propres tables changent.

Le plan `courses-suite` doit dire qu'une liste de courses est **périmée** (C8) : des plats servis dont elle a reçu les courses ont changé depuis (parts, plat modifié, plat retiré du planning). C'est un état **dérivé** : il dépend de la liste, du planning et des plats à la fois, et il change quand le planning change, alors que la liste, elle, ne change pas. La copie des listes reçoit `304` et ne peut pas l'apprendre. Le calcul a besoin de données qui restent dans le serveur : les lignes du plat gardées au moment du calcul, qui ne sortent pas par l'API (plan `courses`, D5). Et il doit suivre la même règle que le recalcul, qui se fait dans le serveur (plan `courses`, D6).

Décidé avec le développeur le 2026-11-14 (plan `courses-suite`, D3, D7 et D8).

## Décision

1. **Un état dérivé de plusieurs collections est calculé par le serveur, à chaque lecture, par une route à part.** Pour les listes de courses : `GET shopping-lists/stale`, qui renvoie `{ lists: [ListID] }`, les identifiants des listes périmées, dans l'ordre des identifiants.
2. **Ni ETag, ni `304`, ni version** : la réponse est petite (au plus 20 identifiants) et son calcul lit une seule base de tribu ; `Cache-Control: no-store`, comme toute l'API. Elle ne passe pas par `internal/collection`.
3. **La copie la demande juste après chaque vérification de sa collection** (visite de l'onglet, retour au premier plan) et la garde à part de ses éléments. Une écriture de la collection qui change l'état dérivé de façon connue l'applique sans attendre : une liste recalculée sort des listes périmées.
4. **La règle est une fonction pure du domaine**, la même que pour l'action qui rétablit l'état : une liste est périmée si le recalcul y changerait quelque chose (`internal/shopping`, `stale` et `recompute`). La route n'est qu'un adaptateur (ADR 0002).
5. **La route reste derrière la session** et ne lit que la base de la tribu de la session (ENF-02, ADR 0003).

## Alternatives envisagées

- **Un drapeau par élément de la collection, avec un ETag qui combine les versions des collections dont il dépend** (`shopping_lists`, `meal_plan`, `dishes`) : écarté. Chaque modification du planning ferait recharger toutes les listes, avec leurs provenances, au lieu de quelques identifiants ; l'ETag d'une collection ne dirait plus seulement si elle a changé, ce qui précise l'ADR 0029 pour un seul cas.
- **Le calcul dans le navigateur, à partir des copies** : écarté. Il demande d'exposer les lignes gardées et d'écrire la règle du recalcul en TypeScript, en double du serveur, contre le plan `courses`, D6 ; et le navigateur ne voit des plats servis retirés du planning que leur absence.
- **Rendre la liste périmée en écrivant dans ses tables quand le planning ou un plat change** (un drapeau mis à jour par l'écriture du planning) : écarté. Chaque écriture du planning et des plats devrait connaître les listes, et un oubli laisserait un drapeau faux sans que rien ne le montre ; la péremption dépend aussi des parts de référence et des lignes, et deviendrait une règle dispersée.

## Conséquences

- **Positif** : la copie des collections (ADR 0029) reste inchangée ; la règle n'a qu'une source, testée en Go ; aucune donnée gardée par le serveur ne sort par l'API ; la péremption est toujours juste au moment de la vérification.
- **Négatif** : une requête de plus à chaque vérification de la copie des listes, toujours calculée, même quand rien n'a changé ; un état dérivé n'est pas tenu à jour entre deux vérifications (comme une copie, sans temps réel, Q6). Au plan hors-ligne, la réponse ne pourra pas être rejouée : sans réseau, la péremption affichée est celle de la dernière vérification.
- **Pour les plans suivants** : un autre état dérivé de plusieurs collections (membres et sessions, par exemple) suit la même forme, une route à part sans ETag, plutôt qu'une version combinée.
