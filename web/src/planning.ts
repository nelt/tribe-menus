// The meal plan of the tribe, in memory (plan planning, D1, D15): a copy of the collection
// meal_plan (ADR 0029), checked together with the copy of the library, whose dishes give the
// names and colours of the tiles, at each visit of the Planning and each return of the
// application to the foreground. Nothing is written on the device: the copy goes with a
// reload, the sign-out or a 401.
import type { Api, Outcome } from "./api.ts";
import { CachedCollection } from "./collections.ts";
import type { Library } from "./library.ts";
import { type ServedDish, mealtimes } from "./meal-plan.ts";

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

/** Returns the dish served described by a body of the API, or undefined if it is not one. */
export function servedDishOf(raw: unknown): ServedDish | undefined {
  if (!isRecord(raw)) {
    return undefined;
  }
  const { id, date, mealtime, dish, servings, rank } = raw;
  if (typeof id !== "string" || typeof date !== "string" || typeof dish !== "string" || typeof servings !== "number" || typeof rank !== "number") {
    return undefined;
  }
  const time = mealtimes.find((m) => m === mealtime);
  if (time === undefined) {
    return undefined;
  }
  return { id, date, mealtime: time, dish, servings, rank };
}

/** Planning holds the copy of the meal plan; it dispatches "change" after each check. */
export class Planning extends EventTarget {
  readonly served: CachedCollection<ServedDish>;
  readonly library: Library;

  constructor(api: Api, library: Library) {
    super();
    this.library = library;
    this.served = new CachedCollection("meal_plan", (after, condition) => api.getPage("meal-plan", after, condition), servedDishOf, (s) => s.id);
  }

  /** Whether the copy of the meal plan and that of the library were loaded once. */
  get loaded(): boolean {
    return this.served.items !== undefined && this.library.loaded;
  }

  /** Checks the meal plan and the library against the API, at the same time (D15). */
  async check(): Promise<Outcome<undefined>> {
    const [served, library] = await Promise.all([this.served.check(), this.library.check()]);
    this.dispatchEvent(new Event("change"));
    return served.kind !== "ok" ? served : library;
  }

  /** Forgets the copy of the meal plan; the library is cleared by its owner. */
  clear(): void {
    this.served.clear();
    this.dispatchEvent(new Event("change"));
  }
}
