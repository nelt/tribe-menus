// Rules of the shopping lists computed in the browser (plan courses, D13, D18): the order of
// the lists (C2) and of the items (Q40), the quantity shown of an item (Q37), the progress,
// the summary of « Courses faites » (D9), why a list cannot be deleted (C12) and the messages
// of the screens (docs/design/README.md, « Listes de courses »). Pure functions, tested by
// node:test; the order of the items shares its table of cases with Go.
import { hasControl, hasVisible, normalizeName, parseQuantity, quantityInput, unitLabel } from "./dishes.ts";
import type { Mealtime } from "./meal-plan.ts";

/** What a served dish brought to a computed item, at the time of the computation (C5, D5). */
export interface ItemSource {
  servedDish: string;
  dish: string;
  date: string;
  mealtime: Mealtime;
  servings: number;
  /** Absent for an ingredient without quantity (D3); in the smallest unit of its family. */
  quantity?: string;
  unit?: string;
}

/** An item of a list, as the API gives it (docs/conception/listes-de-courses.md, « API »). */
export interface Item {
  id: string;
  ingredient: string;
  /** Absent for an unquantified item (Q36); a computed item in the smallest unit of its family. */
  quantity?: string;
  unit?: string;
  checked: boolean;
  /** Entered by hand (C3); otherwise computed from the meal plan, with its sources (C4). */
  manual: boolean;
  sources: ItemSource[];
}

/** A shopping list, with its items and the served dishes it received (Q22). */
export interface ShoppingList {
  id: string;
  name: string;
  main: boolean;
  /** The rank of creation, which orders the lists after the main one (C2). */
  rank: number;
  items: Item[];
  received: string[];
}

/** At most this many characters in the name of a list, 20 lists, 300 items in a list (Q38, D12). */
export const maxListNameLength = 40;
export const maxLists = 20;
export const maxItems = 300;

/** Compares two texts code point by code point, as Go compares their UTF-8 bytes. */
function compareCodePoints(a: string, b: string): number {
  const x = Array.from(a);
  const y = Array.from(b);
  for (let i = 0; i < x.length && i < y.length; i++) {
    const d = (x[i]?.codePointAt(0) ?? 0) - (y[i]?.codePointAt(0) ?? 0);
    if (d !== 0) {
      return d;
    }
  }
  return x.length - y.length;
}

/** false before true. */
function compareBool(a: boolean, b: boolean): number {
  return Number(a) - Number(b);
}

/** The lists in the order of C2: the main list, then by order of creation. */
export function sortLists(lists: Iterable<ShoppingList>): ShoppingList[] {
  return [...lists].sort((a, b) => compareBool(b.main, a.main) || a.rank - b.rank || compareCodePoints(a.id, b.id));
}

/**
 * The items in the order of Q40, as SortItems in Go, kept in agreement by
 * internal/shopping/testdata/sort-items.json: unchecked first, checked at the bottom; in each
 * group, the normalized name of the ingredient; at equal ingredient, with quantity before
 * without, computed before manual; then the identifier.
 */
export function sortItems(items: Iterable<Item>): Item[] {
  return [...items].sort(
    (a, b) =>
      compareBool(a.checked, b.checked) ||
      compareCodePoints(normalizeName(a.ingredient), normalizeName(b.ingredient)) ||
      compareBool(a.quantity === undefined, b.quantity === undefined) ||
      compareBool(a.manual, b.manual) ||
      compareCodePoints(a.id, b.id),
  );
}

/** The units of a family, from the largest; a unit alone is its own family. */
const families: readonly (readonly string[])[] = [
  ["kg", "g"],
  ["l", "cl", "ml"],
];

/** The factors of the units, in base units (internal/dish): mg for a mass, ml for a volume. */
const factors: Record<string, number> = { g: 1000, kg: 1_000_000, ml: 1, cl: 10, l: 1000 };

/** An amount of base units in a unit, written with a decimal comma and no useless zero. */
function amountText(amount: number, factor: number): string {
  const whole = Math.floor(amount / factor);
  const rest = amount % factor;
  if (rest === 0) {
    return String(whole);
  }
  const decimals = String(factor).length - 1;
  return `${whole},${String(rest).padStart(decimals, "0").replace(/0+$/, "")}`;
}

/**
 * The quantity of an item as the list shows it (docs/design/README.md, « Affichage des
 * quantités »): « 1,5 kg », « 3 pièces »; empty without quantity (Q36). A computed item,
 * given in the smallest unit of its family, is shown in the largest unit that gives at least
 * 1 (Q37: « 1,5 kg », « 750 g », « 75 cl »); a manual item keeps the unit entered.
 */
export function itemQuantity(item: Pick<Item, "quantity" | "unit" | "manual">): string {
  const { quantity, unit } = item;
  if (quantity === undefined || unit === undefined) {
    return "";
  }
  const family = item.manual ? undefined : families.find((f) => f.includes(unit));
  const reading = parseQuantity(quantity, unit);
  if (family === undefined || "error" in reading) {
    const shown = quantityInput(quantity);
    return `${shown} ${unitLabel(unit, quantity)}`;
  }
  const shownUnit = family.find((u) => reading.amount >= (factors[u] ?? 0)) ?? unit;
  const shown = amountText(reading.amount, factors[shownUnit] ?? 1);
  return `${shown} ${unitLabel(shownUnit, shown)}`;
}

/** « 1 plat », « 3 plats »: the served dishes an item comes from (C5). */
export function dishCount(item: Pick<Item, "sources">): string {
  const n = item.sources.length;
  return n > 1 ? `${n} plats` : `${n} plat`;
}

function articles(n: number): string {
  return n > 1 ? `${n} articles` : `${n} article`;
}

/** The progress of a list: checked items out of all, as Doux-Liste and Doux-ChoixListes show it. */
export interface Progress {
  checked: number;
  total: number;
  /** « 3 sur 8 articles », under the header of a list. */
  label: string;
  /** « 5 à prendre ». */
  remaining: string;
  /** The width of the green bar, in percent. */
  percent: number;
}

export function progressOf(list: Pick<ShoppingList, "items">): Progress {
  const total = list.items.length;
  const checked = list.items.filter((i) => i.checked).length;
  return {
    checked,
    total,
    label: `${checked} sur ${articles(total)}`,
    remaining: `${total - checked} à prendre`,
    percent: total === 0 ? 0 : Math.round((checked * 100) / total),
  };
}

/** The summary of a card of the list picker: « 4 articles · 4 à prendre », or « Vide ». */
export function listSummary(list: Pick<ShoppingList, "items">): string {
  const { total, checked } = progressOf(list);
  return total === 0 ? "Vide" : `${articles(total)} · ${total - checked} à prendre`;
}

/** What « Courses faites » would do (C11, D9): the items seen checked, that leave, and those that stay. */
export interface Checkout {
  /** The identifiers of the items checked, sent with the confirmation (D9). */
  leaving: string[];
  staying: number;
  /** Under the button. */
  hint: string;
  /** In the sheet of confirmation. */
  message: string;
}

export function checkoutOf(list: Pick<ShoppingList, "items">): Checkout {
  const leaving = list.items.filter((i) => i.checked).map((i) => i.id);
  const staying = list.items.length - leaving.length;
  const n = leaving.length;
  const hint = n === 0 ? "Cochez les articles achetés pour les retirer de la liste." : `Retire ${n > 1 ? `les ${n} articles cochés` : "l’article coché"} de la liste ; les autres y restent.`;
  const leave = n > 1 ? `Les ${n} articles cochés seront retirés de la liste.` : "L’article coché sera retiré de la liste.";
  const stay = staying === 0 ? "La liste sera vide." : staying > 1 ? `Les ${staying} articles non cochés y restent.` : "L’article non coché y reste.";
  return { leaving, staying, hint, message: `${leave} ${stay}` };
}

/** The green banner after « Courses faites »: « 1 article retiré », « 3 articles retirés ». */
export function removedMessage(removed: number): string {
  return removed > 1 ? `${removed} articles retirés` : `${removed} article retiré`;
}

/** Why a list cannot be deleted (C12), or undefined if it can: empty and not the main one. */
export function deleteBlocked(list: Pick<ShoppingList, "main" | "items">): string | undefined {
  if (list.main) {
    return "Désignez d’abord une autre liste comme principale.";
  }
  return list.items.length > 0 ? "Une liste doit être vide pour être supprimée." : undefined;
}

/**
 * Checks the name of a list before sending it, as ParseListName in Go (Q38, D12): a control
 * character, then a name without visible character; the length is bounded by the field.
 */
export function checkListName(name: string): "list_name_invalid" | "list_name_empty" | undefined {
  if (hasControl(name)) {
    return "list_name_invalid";
  }
  return hasVisible(name) ? undefined : "list_name_empty";
}

/**
 * The message of a refusal of a list or an item (docs/design/README.md, « Messages des
 * listes de courses »); undefined for the codes the screens cannot produce, shown as an
 * unforeseen response. The codes of an item added are those of a line of a dish.
 */
export function shoppingErrorMessage(error: string): string | undefined {
  switch (error) {
    case "list_name_empty":
      return "Donnez un nom à la liste.";
    case "list_name_invalid":
    case "ingredient_name_invalid":
      return "Ce nom contient un caractère invalide.";
    case "too_many_lists":
      return "20 listes au plus : supprimez une liste vide pour en créer une autre.";
    case "ingredient_name_required":
      return "Indiquez un article.";
    case "invalid_quantity":
      return "Quantité invalide.";
    case "unit_required":
      return "Choisissez une unité.";
    case "quantity_too_precise":
      return "Quantité trop précise pour cette unité.";
    case "too_many_items":
      return "La liste ne peut pas dépasser 300 articles.";
    default:
      return undefined;
  }
}
