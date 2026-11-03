// The dish library and the ingredient catalog of the tribe, in memory (plan bibliotheque,
// D25): two copies of collections (ADR 0029), checked at each visit of the tab Plats and
// each return of the application to the foreground, shared by the screens of the domain.
// Nothing is written on the device: the copy goes with a reload, the sign-out or a 401.
import type { Api, DishBody, Failure, Outcome } from "./api.ts";
import { CachedCollection } from "./collections.ts";
import { type Dish, type DishError, type DishLine, type Ingredient, splitSaved } from "./dishes.ts";

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

/** The versions of the collections after a write (ADR 0029): those it changed. */
export interface Versions {
  dishes: number;
  ingredients?: number;
}

/** The result of a save of a dish, as the screen of a dish shows it. */
export type SaveResult = { kind: "saved"; dish: Dish } | { kind: "refused"; error: DishError } | { kind: "changed"; dish: Dish } | Failure;

/** Library holds the copies; it dispatches "change" after each check and each write. */
export class Library extends EventTarget {
  readonly dishes: CachedCollection<Dish>;
  readonly catalog: CachedCollection<Ingredient>;
  readonly #api: Api;

  constructor(api: Api) {
    super();
    this.#api = api;
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

  /**
   * Takes a dish saved by the member and the ingredients of its lines, with the versions of
   * the collections after the write (D26, D27, D29): nothing is reloaded, unless another
   * member wrote in between.
   */
  async applySaved(saved: Dish, versions: Versions): Promise<void> {
    const { dish, ingredients } = splitSaved(saved);
    const writes: Promise<unknown>[] = [this.dishes.applyWrite([dish], versions.dishes)];
    if (versions.ingredients !== undefined) {
      writes.push(this.catalog.applyWrite(ingredients, versions.ingredients));
    } else {
      this.catalog.put(ingredients);
    }
    await Promise.all(writes);
    this.dispatchEvent(new Event("change"));
  }

  /** Takes the current dish sent back with a refusal of a crossed change (D5, D26). */
  applyCurrent(current: Dish): void {
    const { dish, ingredients } = splitSaved(current);
    this.dishes.put([dish]);
    this.catalog.put(ingredients);
    this.dispatchEvent(new Event("change"));
  }

  /**
   * Saves a dish, new or read in the revision given, and takes the response into the copy:
   * the dish saved (D26, D27), or the current dish of a crossed change refused (D5).
   */
  async save(existing: { id: string; revision: number } | undefined, body: DishBody): Promise<SaveResult> {
    const outcome = await this.#api.saveDish(existing, body);
    if (outcome.kind !== "ok") {
      return outcome;
    }
    const response = outcome.value;
    if (response.kind === "refused") {
      return response;
    }
    const dish = dishOf(response.dish);
    if (dish === undefined) {
      return { kind: "network-error" };
    }
    if (response.kind === "changed") {
      this.applyCurrent(dish);
      return { kind: "changed", dish };
    }
    const { dishes, ingredients } = response.versions;
    if (dishes === undefined) {
      return { kind: "network-error" };
    }
    await this.applySaved(dish, ingredients === undefined ? { dishes } : { dishes, ingredients });
    return { kind: "saved", dish };
  }

  /** Forgets the copies (D25). */
  clear(): void {
    this.dishes.clear();
    this.catalog.clear();
    this.dispatchEvent(new Event("change"));
  }
}
