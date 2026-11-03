// The dish library and the ingredient catalog of the tribe, in memory (plan bibliotheque,
// D25): two copies of collections (ADR 0029), checked at each visit of the tab Plats and
// each return of the application to the foreground, shared by the screens of the domain.
// Nothing is written on the device: the copy goes with a reload, the sign-out or a 401.
import type { Api, Outcome } from "./api.ts";
import { CachedCollection } from "./collections.ts";
import type { Dish, DishLine, Ingredient } from "./dishes.ts";

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function optionalString(value: unknown): value is string | undefined {
  return value === undefined || typeof value === "string";
}

function lineOf(raw: unknown): DishLine | undefined {
  if (!isRecord(raw)) {
    return undefined;
  }
  const { name, quantity, unit, defaultUnit } = raw;
  if (typeof name !== "string" || !optionalString(quantity) || !optionalString(unit) || !optionalString(defaultUnit)) {
    return undefined;
  }
  return {
    name,
    ...(quantity === undefined ? {} : { quantity }),
    ...(unit === undefined ? {} : { unit }),
    ...(defaultUnit === undefined ? {} : { defaultUnit }),
  };
}

/** Returns the dish described by a body of the API, or undefined if it is not one. */
export function dishOf(raw: unknown): Dish | undefined {
  if (!isRecord(raw) || !Array.isArray(raw["ingredients"])) {
    return undefined;
  }
  const { id, name, color, servings, revision } = raw;
  if (typeof id !== "string" || typeof name !== "string" || typeof color !== "string" || typeof servings !== "number" || typeof revision !== "number") {
    return undefined;
  }
  const ingredients: DishLine[] = [];
  for (const l of raw["ingredients"] as unknown[]) {
    const line = lineOf(l);
    if (line === undefined) {
      return undefined;
    }
    ingredients.push(line);
  }
  return { id, name, color, servings, revision, ingredients };
}

/** Returns the ingredient described by a body of the API, or undefined if it is not one. */
export function ingredientOf(raw: unknown): Ingredient | undefined {
  if (!isRecord(raw)) {
    return undefined;
  }
  const { name, defaultUnit } = raw;
  if (typeof name !== "string" || !optionalString(defaultUnit)) {
    return undefined;
  }
  return defaultUnit === undefined ? { name } : { name, defaultUnit };
}

/** Library holds the copies; it dispatches "change" after each check. */
export class Library extends EventTarget {
  readonly dishes: CachedCollection<Dish>;
  readonly catalog: CachedCollection<Ingredient>;

  constructor(api: Api) {
    super();
    this.dishes = new CachedCollection("dishes", (after, condition) => api.getPage("dishes", after, condition), dishOf, (d) => d.id);
    this.catalog = new CachedCollection("ingredients", (after, condition) => api.getPage("ingredients", after, condition), ingredientOf, (i) => i.name);
  }

  /** Whether the copy was loaded once. */
  get loaded(): boolean {
    return this.dishes.items !== undefined && this.catalog.items !== undefined;
  }

  /** Checks both copies against the API, at the same time. */
  async check(): Promise<Outcome<undefined>> {
    const [dishes, catalog] = await Promise.all([this.dishes.check(), this.catalog.check()]);
    this.dispatchEvent(new Event("change"));
    return dishes.kind !== "ok" ? dishes : catalog;
  }

  /** Forgets the copies (D25). */
  clear(): void {
    this.dishes.clear();
    this.catalog.clear();
    this.dispatchEvent(new Event("change"));
  }
}
