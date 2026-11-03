// Rules of the dish library computed in the browser (ADR 0027): the normalized name, the
// search (P3, Q30) and the alphabetical order (Q30). Pure functions, tested by node:test.

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
