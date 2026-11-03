// Rules of the dish library computed in the browser (ADR 0027): the normalized name, the
// search (P3, Q30), the alphabetical order (Q30), the suggestions of ingredients (P4, Q27),
// the quantities (Q31, D20) and the merge of a dish saved (D26, D27). Pure functions, tested
// by node:test.

/** A line of a dish, as the API gives it (docs/conception/bibliotheque-de-plats.md). */
export interface DishLine {
  name: string;
  quantity?: string;
  unit?: string;
  defaultUnit?: string;
}

/** A dish of the library. */
export interface Dish {
  id: string;
  name: string;
  color: string;
  servings: number;
  revision: number;
  ingredients: DishLine[];
}

/** An ingredient of the catalog. */
export interface Ingredient {
  name: string;
  defaultUnit?: string;
}

/**
 * The spaces of the normalized name: the same set as isSpace in internal/dish. Neither \s
 * nor trim(), which take other characters (U+2003, U+FEFF…).
 */
const spaces = /[ \t\n\v\f\r  ]+/g;
const combiningMarks = /[̀-ͯ]/g;
const apostrophes = /[’‘ʼ]/g;
/**
 * The control characters (category Cc) a displayed name refuses, as hasControl in
 * internal/dish: those of the spaces above are reduced to a space before (D30).
 */
const controls = /[\u0000-\u0008\u000e-\u001f\u007f-\u009f]/;

/**
 * Returns the normalized name (Q29), with the same steps as NormalizeName in Go, kept in
 * agreement by internal/dish/testdata/normalized-names.json: lower case, then the ligatures
 * œ and æ, then NFD and the marks U+0300 to U+036F removed, then the apostrophes, then the
 * spaces.
 */
export function normalizeName(name: string): string {
  return name
    .toLowerCase()
    .replaceAll("œ", "oe")
    .replaceAll("æ", "ae")
    .normalize("NFD")
    .replace(combiningMarks, "")
    .replace(apostrophes, "'")
    .replace(spaces, " ")
    .replace(/^ | $/g, "");
}

/** The words of a name: its normalized name cut on anything but a-z and 0-9 (Q30). */
export function words(name: string): string[] {
  return normalizeName(name)
    .split(/[^a-z0-9]+/)
    .filter((w) => w !== "");
}

const graphemes = new Intl.Segmenter("fr", { granularity: "grapheme" });

/**
 * The initial of a dish, shown in its color: the first character as the reader sees it, an
 * emoji whole, composed or not, and not a half of a UTF-16 surrogate pair.
 */
export function initialOf(name: string): string {
  for (const { segment } of graphemes.segment(name.trimStart())) {
    return segment;
  }
  return "";
}

const collator = new Intl.Collator("fr");

/** The dishes by alphabetical order of their name (Q30). */
export function sortDishes(dishes: Iterable<Dish>): Dish[] {
  return [...dishes].sort((a, b) => collator.compare(a.name, b.name) || a.id.localeCompare(b.id));
}

/** A dish found by a search, with the names of its ingredients the search found. */
export interface Result {
  dish: Dish;
  /** The ingredients one of whose words starts with a term. */
  found: Set<string>;
}

function startsAWord(term: string, nameWords: string[]): boolean {
  return nameWords.some((w) => w.startsWith(term));
}

/**
 * Searches the dishes (P3, Q30): each term of the query must start a word of the name or of
 * an ingredient of the dish. An empty query finds every dish. The results come in
 * alphabetical order.
 */
export function searchDishes(dishes: Iterable<Dish>, query: string): Result[] {
  const terms = words(query);
  const results: Result[] = [];
  for (const dish of sortDishes(dishes)) {
    const nameWords = words(dish.name);
    const ingredientWords = dish.ingredients.map((l) => ({ name: l.name, words: words(l.name) }));
    const all = [...nameWords, ...ingredientWords.flatMap((i) => i.words)];
    if (!terms.every((t) => startsAWord(t, all))) {
      continue;
    }
    const found = new Set(ingredientWords.filter((i) => terms.some((t) => startsAWord(t, i.words))).map((i) => i.name));
    results.push({ dish, found });
  }
  return results;
}

/** The ingredients of the catalog the text typed finds, and whether to offer creating it. */
export interface Suggestions {
  /** At most maxSuggestions, by alphabetical order. */
  ingredients: Ingredient[];
  /** The text designates no ingredient of the catalog nor of the dish: « Créer « … » ». */
  create: boolean;
}

export const maxSuggestions = 6;

/**
 * Suggests ingredients for the text typed as the ingredient of a line (P4): those one of
 * whose words starts with each term, as the search (Q30), but not those already in the dish
 * (Q27, D6), given by the names of its lines. Creating is offered unless the text designates
 * an ingredient of the catalog or of the dish, in the sense of the normalized name.
 */
export function suggestIngredients(catalog: Iterable<Ingredient>, text: string, inDish: string[]): Suggestions {
  const normalized = normalizeName(text);
  const excluded = new Set(inDish.map(normalizeName));
  const terms = words(text);
  const ingredients: Ingredient[] = [];
  let known = excluded.has(normalized);
  for (const ingredient of catalog) {
    const name = normalizeName(ingredient.name);
    known ||= name === normalized;
    if (terms.length === 0 || excluded.has(name)) {
      continue;
    }
    const ingredientWords = words(ingredient.name);
    if (terms.every((t) => startsAWord(t, ingredientWords))) {
      ingredients.push(ingredient);
    }
  }
  ingredients.sort((a, b) => collator.compare(a.name, b.name));
  return { ingredients: ingredients.slice(0, maxSuggestions), create: normalized !== "" && !known };
}

/** A part of a name shown in a suggestion: found, the beginning of a word the text found. */
export interface Segment {
  text: string;
  found: boolean;
}

const displayWords = /[\p{L}\p{M}\p{N}]+/gu;

/**
 * Cuts a name into the beginnings of words the text typed found, as the mock-up Doux-Plat
 * shows them, and the rest. The beginning of a word is measured on its normalized name:
 * « boeu » finds « Bœu » in « Bœuf ».
 */
export function highlight(name: string, text: string): Segment[] {
  const terms = words(text);
  const segments: Segment[] = [];
  const push = (part: string, found: boolean) => {
    const last = segments.at(-1);
    if (last !== undefined && last.found === found) {
      last.text += part;
    } else if (part !== "") {
      segments.push({ text: part, found });
    }
  };
  let end = 0;
  for (const m of name.matchAll(displayWords)) {
    const word = m[0];
    const normalized = normalizeName(word);
    const term = terms.filter((t) => normalized.startsWith(t)).sort((a, b) => b.length - a.length)[0];
    if (term === undefined) {
      continue;
    }
    const chars = Array.from(word);
    let k = 1;
    while (k < chars.length && normalizeName(chars.slice(0, k).join("")).length < term.length) {
      k++;
    }
    const prefix = chars.slice(0, k).join("");
    push(name.slice(end, m.index), false);
    push(prefix, true);
    end = m.index + prefix.length;
  }
  push(name.slice(end), false);
  return segments;
}

/** A unit of the closed list (glossary), by its code in the API. */
export interface Unit {
  code: string;
  /** As shown: in the singular, the spoons abridged (docs/design/README.md). */
  label: string;
  /** From 2; the metric units and the spoons are invariable. */
  plural: string;
  /** The full name, for screen readers. */
  name: string;
  /** The number of base units in one unit, and the decimals an entry may have (Q31, D20). */
  factor: number;
  decimals: number;
}

/** The closed list of units, in the order of docs/design/README.md. */
export const units: readonly Unit[] = [
  { code: "g", label: "g", plural: "g", name: "gramme", factor: 1000, decimals: 3 },
  { code: "kg", label: "kg", plural: "kg", name: "kilogramme", factor: 1_000_000, decimals: 3 },
  { code: "ml", label: "ml", plural: "ml", name: "millilitre", factor: 1, decimals: 0 },
  { code: "cl", label: "cl", plural: "cl", name: "centilitre", factor: 10, decimals: 1 },
  { code: "l", label: "l", plural: "l", name: "litre", factor: 1000, decimals: 3 },
  { code: "piece", label: "pièce", plural: "pièces", name: "pièce", factor: 1000, decimals: 3 },
  { code: "tablespoon", label: "c. à soupe", plural: "c. à soupe", name: "cuillère à soupe", factor: 1000, decimals: 3 },
  { code: "teaspoon", label: "c. à café", plural: "c. à café", name: "cuillère à café", factor: 1000, decimals: 3 },
  { code: "pinch", label: "pincée", plural: "pincées", name: "pincée", factor: 1000, decimals: 3 },
];

function unitOf(code: string): Unit | undefined {
  return units.find((u) => u.code === code);
}

/** A quantity entered, read: its amount in the base unit, or the error code of the API. */
export type QuantityReading = { amount: number } | { error: "invalid_quantity" | "quantity_too_precise" | "invalid_unit" };

const quantityPattern = /^([0-9]+)(?:[.,]([0-9]+))?$/;
const maxIntegerDigits = 6;

/** The text without the spaces of the normalized name around it. */
function trimSpaces(text: string): string {
  return text.replace(/^[ \t\n\v\f\r\u00a0\u202f]+|[ \t\n\v\f\r\u00a0\u202f]+$/g, "");
}

/**
 * Reads a quantity entered in a unit, with a comma or a dot (Q31, D20), as ParseQuantity in
 * Go, kept in agreement by internal/dish/testdata/quantities.json: strictly positive, with
 * at most the decimals the base unit represents; beyond, refused rather than rounded.
 */
export function parseQuantity(text: string, unit: string): QuantityReading {
  const info = unitOf(unit);
  if (info === undefined) {
    return { error: "invalid_unit" };
  }
  const m = quantityPattern.exec(trimSpaces(text));
  if (m === null) {
    return { error: "invalid_quantity" };
  }
  const whole = (m[1] ?? "").replace(/^0+/, "");
  const fraction = (m[2] ?? "").replace(/0+$/, "");
  if (whole.length > maxIntegerDigits) {
    return { error: "invalid_quantity" };
  }
  if (fraction.length > info.decimals) {
    return { error: "quantity_too_precise" };
  }
  let scale = info.factor;
  for (let i = 0; i < fraction.length; i++) {
    scale /= 10;
  }
  const amount = Number(`0${whole}`) * info.factor + Number(`0${fraction}`) * scale;
  return amount > 0 ? { amount } : { error: "invalid_quantity" };
}

/**
 * The label of a unit after a quantity (docs/design/README.md, « Affichage des quantités »):
 * plural from 2, as in French; the singular without a readable quantity.
 */
export function unitLabel(unit: string, quantity: string): string {
  const info = unitOf(unit);
  if (info === undefined) {
    return "";
  }
  const n = Number(trimSpaces(quantity).replace(",", "."));
  return trimSpaces(quantity) !== "" && n >= 2 ? info.plural : info.label;
}

/** A quantity of the API ("2.5") as the field shows it, with a decimal comma ("2,5"). */
export function quantityInput(quantity: string | undefined): string {
  return quantity?.replace(".", ",") ?? "";
}

/** A dish refused, by the API or before sending it: the error code, the line in cause. */
export interface DishError {
  error: string;
  line?: number;
  ingredient?: string;
}

/** A dish as the screen holds it: lines named, the quantity as typed, the unit code. */
export interface DishForm {
  name: string;
  lines: { name: string; quantity: string; unit: string }[];
}

/**
 * Checks before sending what does not need the library, in the order of the server
 * (internal/dish, validate): the name, then each line, its name, its quantity and its unit
 * (Q26, Q31, D30), then whether its ingredient is already in the dish (Q27). A control
 * character pasted in a field is refused as by the server (D30): a text field only
 * removes the line breaks.
 */
export function checkDish(form: DishForm): DishError | undefined {
  if (controls.test(form.name)) {
    return { error: "name_invalid" };
  }
  if (normalizeName(form.name) === "") {
    return { error: "name_required" };
  }
  const seen = new Map<string, string>();
  for (const [i, line] of form.lines.entries()) {
    if (controls.test(line.name)) {
      return { error: "ingredient_name_invalid", line: i };
    }
    if (trimSpaces(line.quantity) !== "") {
      if (line.unit === "") {
        return { error: "unit_required", line: i };
      }
      const reading = parseQuantity(line.quantity, line.unit);
      if ("error" in reading) {
        return { error: reading.error, line: i };
      }
    }
    const normalized = normalizeName(line.name);
    const first = seen.get(normalized);
    if (first !== undefined) {
      return { error: "duplicate_ingredient", line: i, ingredient: first };
    }
    seen.set(normalized, trimSpaces(line.name));
  }
  return undefined;
}

/**
 * The message of a dish refused (docs/design/README.md, « Messages »); undefined for the
 * codes a bounded entry cannot produce, shown as an unforeseen response.
 */
export function dishErrorMessage(e: DishError): string | undefined {
  switch (e.error) {
    case "name_required":
      return "Donnez un nom au plat.";
    case "name_invalid":
    case "ingredient_name_invalid":
      return "Ce nom contient un caractère invalide.";
    case "name_taken":
      return "Un plat porte déjà ce nom.";
    case "duplicate_ingredient":
      return `« ${e.ingredient ?? ""} » figure déjà dans le plat.`;
    case "invalid_quantity":
      return "Quantité invalide.";
    case "unit_required":
      return "Choisissez une unité.";
    case "quantity_too_precise":
      return "Quantité trop précise pour cette unité.";
    default:
      return undefined;
  }
}

/**
 * Splits a dish returned by a write (D27): the dish as GET dishes gives it, without the
 * default units, and the ingredients of its lines, for the catalog.
 */
export function splitSaved(saved: Dish): { dish: Dish; ingredients: Ingredient[] } {
  return {
    dish: { ...saved, ingredients: saved.ingredients.map(({ defaultUnit: _, ...line }) => line) },
    ingredients: saved.ingredients.map((l) => (l.defaultUnit === undefined ? { name: l.name } : { name: l.name, defaultUnit: l.defaultUnit })),
  };
}
