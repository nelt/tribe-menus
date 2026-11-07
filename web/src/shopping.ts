// Rules of the shopping lists computed in the browser (plan courses, D13, D18): the order of
// the lists (C2) and of the items (Q40), the quantity shown of an item and of what a served
// dish brought to it (Q37), the progress, the summary of « Courses faites » (D9), why a list
// cannot be deleted (C12), the period of the meal shopping and its calendar (C4, D24), the
// served dishes already received (Q22), the unfolded lines of an item (C5) and the messages
// of the screens (docs/design/README.md, « Listes de courses »). Pure functions, tested by
// node:test; the order of the items and the quantities share their tables of cases with Go.
import { type Dish, hasControl, hasVisible, normalizeName, quantityInput, unitLabel, units } from "./dishes.ts";
import { type Day, type Mealtime, type ServedDish, addDays, firstDay, lastDay, mealtimes } from "./meal-plan.ts";

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

/** The factor of a unit, in base units (internal/dish): mg for a mass, ml for a volume, thousandths otherwise. */
function factorOf(unit: string): number | undefined {
  return units.find((u) => u.code === unit)?.factor;
}

/**
 * The amount in base units of a quantity of the API ("166.667" g): a dot, no useless zero.
 * Unlike a quantity typed (parseQuantity), its integer part is not bounded: a computed item
 * adds up to 50 servings of 999 999 kg (Q31).
 */
function baseAmount(quantity: string, unit: string): number | undefined {
  const factor = factorOf(unit);
  const m = /^([0-9]+)(?:\.([0-9]+))?$/.exec(quantity);
  if (factor === undefined || m === null) {
    return undefined;
  }
  const fraction = (m[2] ?? "").slice(0, String(factor).length - 1).padEnd(String(factor).length - 1, "0");
  return Number(m[1]) * factor + Number(`0${fraction}`);
}

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

/** An amount of base units, in the largest unit of the family of unit that gives at least 1 (Q37). */
function shownAmount(amount: number, unit: string): string {
  const family = families.find((f) => f.includes(unit)) ?? [unit];
  const shownUnit = family.find((u) => amount >= (factorOf(u) ?? 0)) ?? unit;
  const shown = amountText(amount, factorOf(shownUnit) ?? 1);
  return `${shown} ${unitLabel(shownUnit, shown)}`;
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
  const amount = item.manual ? undefined : baseAmount(quantity, unit);
  if (amount === undefined) {
    const shown = quantityInput(quantity);
    return `${shown} ${unitLabel(unit, shown)}`;
  }
  return shownAmount(amount, unit);
}

/**
 * The quantity a served dish brought to an item, as its unfolded line shows it (C5, Q37, D7):
 * without the rounding of the item, but rounded up to the gram for a mass, to the millilitre
 * for a volume (its base unit already), to the hundredth otherwise (« 0,75 pièce »); then in
 * the largest unit of its family, as an item. Empty without quantity (Q36).
 */
export function contributedQuantity(source: Pick<ItemSource, "quantity" | "unit">): string {
  const { quantity, unit } = source;
  const amount = quantity === undefined || unit === undefined ? undefined : baseAmount(quantity, unit);
  if (amount === undefined || unit === undefined) {
    return "";
  }
  const step = families[0]?.includes(unit) ? 1000 : families[1]?.includes(unit) ? 1 : 10;
  return shownAmount(Math.ceil(amount / step) * step, unit);
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

/** The period of the meal shopping (C4): from one day to another, both included. */
export interface Period {
  from: Day;
  to: Day;
}

/** At most this many days in a period (Q38, D12). */
export const maxPeriodDays = 31;

/** The period proposed (D24, Q2, Q34): from tomorrow to 7 days after today, by the date of the device. */
export function defaultPeriod(todayDay: Day): Period {
  return { from: addDays(todayDay, 1), to: addDays(todayDay, 7) };
}

const dayMs = 86_400_000;

/** The number of days of a period, both bounds included; 0 or less when the end comes before the start. */
export function periodDays(period: Period): number {
  return Math.round((Date.parse(`${period.to}T00:00:00Z`) - Date.parse(`${period.from}T00:00:00Z`)) / dayMs) + 1;
}

/** Why a period is refused, as ParsePeriod in Go: the end before the start, or more than 31 days (C4, Q38). */
export function periodError(period: Period): "invalid_period" | "period_too_long" | undefined {
  if (period.to < period.from) {
    return "invalid_period";
  }
  return periodDays(period) > maxPeriodDays ? "period_too_long" : undefined;
}

/** The served dishes of a period, whatever their dish. */
export function servedInPeriod(served: Iterable<ServedDish>, period: Period): ServedDish[] {
  return [...served].filter((s) => s.date >= period.from && s.date <= period.to);
}

/** Under the calendar: « 7 jours », and « 5 plats servis » or « aucun plat servi ». */
export function periodSummary(period: Period, served: ServedDish[]): { days: string; dishes: string } {
  const days = periodDays(period);
  const n = served.length;
  return { days: days > 1 ? `${days} jours` : `${days} jour`, dishes: n === 0 ? "aucun plat servi" : n > 1 ? `${n} plats servis` : "1 plat servi" };
}

/** How many served dishes of the period the list has already received: they will be ignored (Q22). */
export function alreadyReceived(list: Pick<ShoppingList, "received">, served: ServedDish[]): number {
  const received = new Set(list.received);
  return served.filter((s) => received.has(s.id)).length;
}

/** Under the target list: the served dishes it has already received; undefined without one (Q22). */
export function ignoredMessage(n: number): string | undefined {
  if (n === 0) {
    return undefined;
  }
  return n > 1 ? `${n} plats servis sont déjà passés par cette liste : ils seront ignorés.` : "1 plat servi est déjà passé par cette liste : il sera ignoré.";
}

/** The green banner after the meal shopping: « 4 plats servis ajoutés, 1 déjà reçu ignoré », « Aucun plat servi ajouté ». */
export function mealShoppingMessage(added: number, ignored: number): string {
  const done = added === 0 ? "Aucun plat servi ajouté" : added > 1 ? `${added} plats servis ajoutés` : "1 plat servi ajouté";
  if (ignored === 0) {
    return done;
  }
  return `${done}, ${ignored > 1 ? `${ignored} déjà reçus ignorés` : "1 déjà reçu ignoré"}`;
}

/** The message of a refusal of the meal shopping (docs/design/README.md, « Messages des listes de courses »). */
export function mealShoppingErrorMessage(error: string): string | undefined {
  switch (error) {
    case "invalid_period":
      return "La date de fin doit suivre la date de début.";
    case "period_too_long":
      return "La période ne peut pas dépasser 31 jours.";
    case "too_many_items":
      return "La liste ne peut pas dépasser 300 articles : rien n’a été ajouté.";
    default:
      return undefined;
  }
}

/** A month of the calendar; month from 0 (January) to 11. */
export interface Month {
  year: number;
  month: number;
}

const months = ["janvier", "février", "mars", "avril", "mai", "juin", "juillet", "août", "septembre", "octobre", "novembre", "décembre"];
const shortMonths = ["janv.", "févr.", "mars", "avr.", "mai", "juin", "juil.", "août", "sept.", "oct.", "nov.", "déc."];
const weekdays = ["dimanche", "lundi", "mardi", "mercredi", "jeudi", "vendredi", "samedi"];
const shortWeekdays = ["dim.", "lun.", "mar.", "mer.", "jeu.", "ven.", "sam."];

function utcOf(day: Day): Date {
  return new Date(`${day}T00:00:00Z`);
}

/** The month of a day. */
export function monthOf(day: Day): Month {
  const d = utcOf(day);
  return { year: d.getUTCFullYear(), month: d.getUTCMonth() };
}

/** The month n months after (before, if negative); undefined out of January 2000 to December 2099 (Q35). */
export function addMonths(month: Month, n: number): Month | undefined {
  const index = month.year * 12 + month.month + n;
  const shifted = { year: Math.floor(index / 12), month: index % 12 };
  const first = monthOf(firstDay);
  const last = monthOf(lastDay);
  return shifted.year < first.year || shifted.year > last.year ? undefined : shifted;
}

/** The title of a month: « Octobre 2026 ». */
export function monthTitle(month: Month): string {
  const name = months[month.month] ?? "";
  return `${name.charAt(0).toUpperCase()}${name.slice(1)} ${month.year}`;
}

function dateNumber(date: number): string {
  return date === 1 ? "1er" : String(date);
}

/** A day in short: « mar. 6 oct. », « jeu. 1er oct. », in the fields of the period and the unfolded lines. */
export function shortDay(day: Day): string {
  const d = utcOf(day);
  return `${shortWeekdays[d.getUTCDay()] ?? ""} ${dateNumber(d.getUTCDate())} ${shortMonths[d.getUTCMonth()] ?? ""}`;
}

/** A day of the calendar, as the grid shows it (Doux-CoursesRepas). */
export interface CalendarDay {
  date: Day;
  /** The number of the day in its month. */
  number: number;
  /** The start or the end of the period. */
  bound: boolean;
  /** Strictly between the bounds of a period in order. */
  between: boolean;
  today: boolean;
  past: boolean;
  /** At least one dish served on that day, lunch or dinner. */
  meals: boolean;
  /** « lundi 5 octobre, repas planifiés, aujourd’hui ». */
  label: string;
}

/**
 * The grid of a month (C4): a week per row, Monday first; undefined for the cells before the
 * first day of the month. mealDays are the days with at least one dish served.
 */
export function monthGrid(month: Month, period: Period, todayDay: Day, mealDays: ReadonlySet<Day>): (CalendarDay | undefined)[] {
  const first = new Date(Date.UTC(month.year, month.month, 1));
  const length = new Date(Date.UTC(month.year, month.month + 1, 0)).getUTCDate();
  const blanks = (first.getUTCDay() + 6) % 7;
  const ordered = period.from <= period.to;
  const cells: (CalendarDay | undefined)[] = Array.from({ length: blanks }, () => undefined);
  for (let n = 1; n <= length; n++) {
    const date = `${String(month.year).padStart(4, "0")}-${String(month.month + 1).padStart(2, "0")}-${String(n).padStart(2, "0")}`;
    const meals = mealDays.has(date);
    const today = date === todayDay;
    const weekday = weekdays[utcOf(date).getUTCDay()] ?? "";
    cells.push({
      date,
      number: n,
      bound: date === period.from || date === period.to,
      between: ordered && date > period.from && date < period.to,
      today,
      past: date < todayDay,
      meals,
      label: `${weekday} ${dateNumber(n)} ${months[month.month] ?? ""}${meals ? ", repas planifiés" : ""}${today ? ", aujourd’hui" : ""}`,
    });
  }
  return cells;
}

/** The days with at least one dish served, for the points of the calendar. */
export function mealDays(served: Iterable<ServedDish>): Set<Day> {
  return new Set([...served].map((s) => s.date));
}

/** A line of an item unfolded (C5): the meal, the dish and its servings, the quantity brought. */
export interface SourceRow {
  servedDish: string;
  dish: Dish;
  /** « 3 parts ». */
  servings: string;
  /** « mar. 6 oct. ». */
  day: string;
  mealtime: Mealtime;
  /** « 375 g », « 0,75 pièce »; empty without quantity (Q36). */
  quantity: string;
}

/**
 * The lines of an item unfolded, in the order of the meals (date, lunch before dinner), then
 * of the names of the dishes. A source whose dish the copy of the library does not hold yet
 * is left out until the next check, as on the planning (plan planning, D15).
 */
export function sourceRows(item: Pick<Item, "sources">, dishes: Iterable<Dish>): SourceRow[] {
  const library = new Map<string, Dish>();
  for (const d of dishes) {
    library.set(d.id, d);
  }
  const rows: SourceRow[] = [];
  for (const s of item.sources) {
    const dish = library.get(s.dish);
    if (dish !== undefined) {
      rows.push({ servedDish: s.servedDish, dish, servings: s.servings > 1 ? `${s.servings} parts` : `${s.servings} part`, day: shortDay(s.date), mealtime: s.mealtime, quantity: contributedQuantity(s) });
    }
  }
  const order = new Map(item.sources.map((s) => [s.servedDish, s]));
  return rows.sort((a, b) => {
    const x = order.get(a.servedDish);
    const y = order.get(b.servedDish);
    return (
      (x?.date ?? "").localeCompare(y?.date ?? "") ||
      mealtimes.indexOf(a.mealtime) - mealtimes.indexOf(b.mealtime) ||
      compareCodePoints(normalizeName(a.dish.name), normalizeName(b.dish.name)) ||
      compareCodePoints(a.servedDish, b.servedDish)
    );
  });
}

/** What a computed item reads, folded or not: « bœuf haché, 875 g, 2 plats. Afficher les repas ». */
export function itemRowLabel(item: Pick<Item, "ingredient" | "quantity" | "unit" | "manual" | "sources">, open: boolean): string {
  const quantity = itemQuantity(item);
  return `${item.ingredient}, ${quantity === "" ? "sans quantité" : quantity}, ${dishCount(item)}. ${open ? "Masquer les repas" : "Afficher les repas"}`;
}
