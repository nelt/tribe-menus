// The meal plan of the tribe, in memory (plan planning, D1, D15): a copy of the collection
// meal_plan (ADR 0029), checked together with the copy of the library, whose dishes give the
// names and colours of the tiles, at each visit of the Planning or of a meal and each return
// of the application to the foreground. The writes of the member are taken with the version
// they give; a refusal that sends back the meal, or a dish served removed in between, makes
// the copy check itself (D3). Nothing is written on the device: the copy goes with a reload,
// the sign-out or a 401.
import type { Api, Failure, Outcome } from "./api.ts";
import { CachedCollection } from "./collections.ts";
import type { Library } from "./library.ts";
import { type Day, type Mealtime, type ServedDish, mealtimes } from "./meal-plan.ts";

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

/** The result of a dish served to a meal: served, or already served in it (Q21, D3). */
export type ServeResult = { kind: "served"; served: ServedDish } | { kind: "already-served" } | Failure;

/** The result of a setting of servings: saved, or the dish served removed in between (D3). */
export type ServingsResult = { kind: "saved"; served: ServedDish } | { kind: "removed" } | Failure;

/** Planning holds the copy of the meal plan; it dispatches "change" after each check and write. */
export class Planning extends EventTarget {
  readonly served: CachedCollection<ServedDish>;
  readonly library: Library;
  readonly #api: Api;

  constructor(api: Api, library: Library) {
    super();
    this.#api = api;
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

  /**
   * Serves a dish in a meal, for the size of the tribe (R2, D9). A dish already served, by
   * another member in between: the copy takes the meal as it is now, then checks itself,
   * the meal sent back having no version (review of PR #88, point 1).
   */
  async serve(date: Day, mealtime: Mealtime, dish: string): Promise<ServeResult> {
    const outcome = await this.#api.serveDish({ date, mealtime, dish });
    if (outcome.kind !== "ok") {
      return outcome;
    }
    const write = outcome.value;
    if (write.kind === "already-served") {
      const meal = write.meal.map(servedDishOf);
      if (meal.some((s) => s === undefined)) {
        return { kind: "network-error" };
      }
      const stale = (this.served.items ?? []).filter((s) => s.date === date && s.mealtime === mealtime).map((s) => s.id);
      this.served.put(meal as ServedDish[], stale);
      this.dispatchEvent(new Event("change"));
      await this.check();
      return { kind: "already-served" };
    }
    const served = write.kind === "written" ? servedDishOf(write.servedDish) : undefined;
    if (write.kind !== "written" || served === undefined) {
      return { kind: "network-error" };
    }
    await this.served.applyWrite([served], write.version);
    this.dispatchEvent(new Event("change"));
    return { kind: "served", served };
  }

  /** Sets the servings of a dish served (R3): the last save wins (D3). */
  async setServings(id: string, servings: number): Promise<ServingsResult> {
    const outcome = await this.#api.setServings(id, servings);
    if (outcome.kind !== "ok") {
      return outcome;
    }
    const write = outcome.value;
    if (write.kind === "not-found") {
      await this.#removedInBetween(id);
      return { kind: "removed" };
    }
    const served = write.kind === "written" ? servedDishOf(write.servedDish) : undefined;
    if (write.kind !== "written" || served === undefined) {
      return { kind: "network-error" };
    }
    await this.served.applyWrite([served], write.version);
    this.dispatchEvent(new Event("change"));
    return { kind: "saved", served };
  }

  /** Removes a dish served from its meal (R4); one already removed is removed too (D19). */
  async remove(id: string): Promise<Outcome<undefined>> {
    const outcome = await this.#api.removeServedDish(id);
    if (outcome.kind !== "ok") {
      return outcome;
    }
    const write = outcome.value;
    if (write.kind === "not-found") {
      await this.#removedInBetween(id);
      return { kind: "ok", value: undefined };
    }
    if (write.kind !== "written") {
      return { kind: "network-error" };
    }
    await this.served.applyWrite([], write.version, [id]);
    this.dispatchEvent(new Event("change"));
    return { kind: "ok", value: undefined };
  }

  /** A dish served removed by another member: dropped from the copy, which checks itself (D3). */
  async #removedInBetween(id: string): Promise<void> {
    this.served.put([], [id]);
    this.dispatchEvent(new Event("change"));
    await this.check();
  }

  /** Forgets the copy of the meal plan; the library is cleared by its owner. */
  clear(): void {
    this.served.clear();
    this.dispatchEvent(new Event("change"));
  }
}
