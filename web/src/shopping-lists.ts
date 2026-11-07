// The shopping lists of the tribe, in memory (plan courses, D4, D18): a copy of the collection
// shopping_lists (ADR 0029), checked at each visit of the tab Courses and each return of the
// application to the foreground. A write of the member takes the list sent back and the
// version it gives, or checks the copy if another member wrote in between; a list or an item
// unknown (404) makes it check itself, then tells whether the list is still there (D8, D11).
// Nothing is written on the device (D2): the copy goes with a reload, the sign-out or a 401.
import type { Api, Failure, ItemBody, Outcome, ShoppingWrite } from "./api.ts";
import { CachedCollection } from "./collections.ts";
import type { Library } from "./library.ts";
import { mealtimes } from "./meal-plan.ts";
import { type Item, type ItemSource, type ShoppingList, sortItems } from "./shopping.ts";

/** The name of the collection, and of its version in the answers to the writes. */
const collection = "shopping_lists";

/** The refusals of a write that another member's write may cause (409). */
const conflicts = new Set(["main_list", "list_not_empty", "too_many_lists", "too_many_items"]);

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function optionalString(value: unknown): value is string | undefined {
  return value === undefined || typeof value === "string";
}

function sourceOf(raw: unknown): ItemSource | undefined {
  if (!isRecord(raw)) {
    return undefined;
  }
  const { servedDish, dish, date, mealtime, servings, quantity, unit } = raw;
  const time = mealtimes.find((m) => m === mealtime);
  if (typeof servedDish !== "string" || typeof dish !== "string" || typeof date !== "string" || time === undefined || typeof servings !== "number" || !optionalString(quantity) || !optionalString(unit)) {
    return undefined;
  }
  return { servedDish, dish, date, mealtime: time, servings, ...(quantity === undefined ? {} : { quantity }), ...(unit === undefined ? {} : { unit }) };
}

function itemOf(raw: unknown): Item | undefined {
  if (!isRecord(raw) || !Array.isArray(raw["sources"])) {
    return undefined;
  }
  const { id, ingredient, quantity, unit, checked, manual } = raw;
  if (typeof id !== "string" || typeof ingredient !== "string" || !optionalString(quantity) || !optionalString(unit) || typeof checked !== "boolean" || typeof manual !== "boolean") {
    return undefined;
  }
  const sources: ItemSource[] = [];
  for (const s of raw["sources"] as unknown[]) {
    const source = sourceOf(s);
    if (source === undefined) {
      return undefined;
    }
    sources.push(source);
  }
  return { id, ingredient, ...(quantity === undefined ? {} : { quantity }), ...(unit === undefined ? {} : { unit }), checked, manual, sources };
}

/** Returns the list described by a body of the API, or undefined if it is not one. */
export function listOf(raw: unknown): ShoppingList | undefined {
  if (!isRecord(raw) || !Array.isArray(raw["items"]) || !Array.isArray(raw["received"])) {
    return undefined;
  }
  const { id, name, main, rank } = raw;
  if (typeof id !== "string" || typeof name !== "string" || typeof main !== "boolean" || typeof rank !== "number") {
    return undefined;
  }
  const items: Item[] = [];
  for (const i of raw["items"] as unknown[]) {
    const item = itemOf(i);
    if (item === undefined) {
      return undefined;
    }
    items.push(item);
  }
  const received = raw["received"] as unknown[];
  if (!received.every((r) => typeof r === "string")) {
    return undefined;
  }
  return { id, name, main, rank, items, received: received as string[] };
}

/**
 * The result of a write on a list, as the screens show it: done, with the list up to date
 * when the API sends it back and the number of items removed by a checkout; refused, with
 * its error code (the list taken into the copy when the refusal sends it back); the list
 * deleted in between (D11); or a failure.
 */
export type WriteResult =
  | { kind: "done"; list: ShoppingList | undefined; removed: number }
  | { kind: "refused"; error: string }
  | { kind: "deleted" }
  | Failure;

/** ShoppingLists holds the copy of the lists; it dispatches "change" after each check and write. */
export class ShoppingLists extends EventTarget {
  readonly lists: CachedCollection<ShoppingList>;
  readonly library: Library;
  readonly #api: Api;

  constructor(api: Api, library: Library) {
    super();
    this.#api = api;
    this.library = library;
    this.lists = new CachedCollection(collection, (after, condition) => api.getPage("shopping-lists", after, condition), listOf, (l) => l.id);
  }

  /** Whether the copy was loaded once. */
  get loaded(): boolean {
    return this.lists.items !== undefined;
  }

  /** The list of an identifier, or the main list without one; undefined if the copy has none. */
  find(id: string | undefined): ShoppingList | undefined {
    return this.lists.items?.find((l) => (id === undefined ? l.main : l.id === id));
  }

  /** Checks the copy against the API. */
  async check(): Promise<Outcome<undefined>> {
    const outcome = await this.lists.check();
    this.dispatchEvent(new Event("change"));
    return outcome;
  }

  /** Creates a list (C2). */
  async create(name: string): Promise<WriteResult> {
    return this.#written(undefined, await this.#api.createList(name));
  }

  /** Renames a list (C2, D11). */
  async rename(id: string, name: string): Promise<WriteResult> {
    return this.#written(id, await this.#api.renameList(id, name));
  }

  /**
   * Makes a list the main one (C1): the API sends back the versions only, the copy swaps the
   * two flags itself, and checks itself if another member wrote in between.
   */
  async setMain(id: string): Promise<WriteResult> {
    const outcome = await this.#api.setMainList(id);
    if (outcome.kind !== "ok") {
      return outcome;
    }
    const write = outcome.value;
    if (write.kind !== "written") {
      return this.#refusedOrDeleted(id, write);
    }
    const version = write.versions[collection];
    if (version === undefined) {
      // Already the main list: the copy was behind.
      await this.check();
    } else {
      const swapped = (this.lists.items ?? []).filter((l) => l.main !== (l.id === id)).map((l) => ({ ...l, main: l.id === id }));
      await this.lists.applyWrite(swapped, version);
      this.dispatchEvent(new Event("change"));
    }
    return { kind: "done", list: undefined, removed: 0 };
  }

  /**
   * Deletes a list (C12). Refused for a list become not empty in between, with the list up to
   * date, or become the main one; a list already deleted is deleted too (D11).
   */
  async delete(id: string): Promise<WriteResult> {
    const outcome = await this.#api.deleteList(id);
    if (outcome.kind !== "ok") {
      return outcome;
    }
    const write = outcome.value;
    if (write.kind === "not-found") {
      this.lists.put([], [id]);
      await this.check();
      return { kind: "done", list: undefined, removed: 0 };
    }
    if (write.kind === "refused") {
      return this.#refusedOrDeleted(id, write);
    }
    await this.#apply([], write.versions, [id]);
    return { kind: "done", list: undefined, removed: 0 };
  }

  /** Adds an item by hand (C3); an ingredient created reloads the catalog of the library. */
  async addItem(id: string, body: ItemBody): Promise<WriteResult> {
    return this.#written(id, await this.#api.addItem(id, body));
  }

  /** Removes an item added by hand (C3); an item already removed is removed too (D11). */
  async removeItem(id: string, item: string): Promise<WriteResult> {
    return this.#written(id, await this.#api.removeItem(id, item));
  }

  /**
   * Checks or unchecks an item (C6, D8). The copy shows it at once, sorted again (Q40), and
   * comes back to the list sent back, or to what it was without network.
   */
  async setChecked(id: string, item: string, checked: boolean): Promise<WriteResult> {
    const before = this.find(id);
    if (before !== undefined) {
      this.lists.put([{ ...before, items: sortItems(before.items.map((i) => (i.id === item ? { ...i, checked } : i))) }]);
      this.dispatchEvent(new Event("change"));
    }
    const done = await this.#written(id, await this.#api.setChecked(id, item, checked));
    if (done.kind !== "done" && done.kind !== "deleted" && before !== undefined && this.find(id) !== undefined) {
      this.lists.put([before]);
      this.dispatchEvent(new Event("change"));
    }
    return done;
  }

  /** Removes the items seen checked that still are (C11, D9). */
  async checkOut(id: string, items: string[]): Promise<WriteResult> {
    return this.#written(id, await this.#api.checkOut(id, items));
  }

  /** Takes the answer to a write on a list: the list written, a refusal, or a list unknown. */
  async #written(id: string | undefined, outcome: Outcome<ShoppingWrite>): Promise<WriteResult> {
    if (outcome.kind !== "ok") {
      return outcome;
    }
    const write = outcome.value;
    if (write.kind !== "written") {
      return this.#refusedOrDeleted(id, write);
    }
    const list = listOf(write.list);
    if (list === undefined) {
      return { kind: "network-error" };
    }
    await this.#apply([list], write.versions);
    return { kind: "done", list, removed: write.removed ?? 0 };
  }

  /** Takes lists written and the versions after the write: the catalog reloads if it changed. */
  async #apply(lists: ShoppingList[], versions: Record<string, number>, removed: string[] = []): Promise<void> {
    const version = versions[collection];
    const writes: Promise<unknown>[] = [];
    if (version === undefined) {
      this.lists.put(lists, removed);
    } else {
      writes.push(this.lists.applyWrite(lists, version, removed));
    }
    if (versions["ingredients"] !== undefined && this.library.catalog.version !== versions["ingredients"]) {
      writes.push(this.library.catalog.check());
    }
    await Promise.all(writes);
    this.dispatchEvent(new Event("change"));
  }

  /**
   * A refusal: the list sent back is taken, and the copy checks itself, the list having no
   * version. A list or an item unknown: the copy checks itself, then tells whether the list
   * was deleted in between (D11) or only the item (D8), which needs no message.
   */
  async #refusedOrDeleted(id: string | undefined, write: Exclude<ShoppingWrite, { kind: "written" }>): Promise<WriteResult> {
    if (write.kind === "refused") {
      const list = write.list === undefined ? undefined : listOf(write.list);
      if (list !== undefined) {
        this.lists.put([list]);
        this.dispatchEvent(new Event("change"));
      }
      // A conflict means the copy is behind: it checks itself.
      if (list !== undefined || conflicts.has(write.error)) {
        await this.check();
      }
      return { kind: "refused", error: write.error };
    }
    const checked = await this.check();
    if (checked.kind !== "ok") {
      return checked;
    }
    if (id === undefined || this.find(id) === undefined) {
      return { kind: "deleted" };
    }
    return { kind: "done", list: this.find(id), removed: 0 };
  }

  /** Forgets the copy (D2). */
  clear(): void {
    this.lists.clear();
    this.dispatchEvent(new Event("change"));
  }
}
