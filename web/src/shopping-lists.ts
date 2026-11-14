// The shopping lists of the tribe, in memory (plan courses, D4, D18): a copy of the collection
// shopping_lists (ADR 0029), checked together with the copy of the library, whose dishes give
// the names and colours of the unfolded lines (C5), at each visit of the tab Courses and each
// return of the application to the foreground. A write of the member takes the list sent back and the
// version it gives, or checks the copy if another member wrote in between; a list or an item
// unknown (404) makes it check itself, then tells whether the list is still there (D8, D11),
// named after the copy before the check (plan courses-suite, D12). After each check, the stale
// lists, a state derived from the meal plan and the dishes, are read apart (C8, ADR 0031), and
// again after a move or a checkout, which may change them (review of PR #119, point 2).
// Nothing is written on the device (D2): the copy goes with a reload, the sign-out or a 401.
import type { Api, Failure, ItemBody, Outcome, ShoppingWrite } from "./api.ts";
import { CachedCollection } from "./collections.ts";
import type { Library } from "./library.ts";
import { mealtimes } from "./meal-plan.ts";
import { type Item, type ItemSource, type Period, type ShoppingList, sortItems } from "./shopping.ts";

/** The name of the collection, and of its version in the answers to the writes. */
const collection = "shopping_lists";

/** The refusals of a write that another member's write may cause (409). */
const conflicts = new Set(["main_list", "list_not_empty", "too_many_lists", "too_many_items", "target_not_found"]);

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
  const { id, ingredient, quantity, unit, taken, checked, manual } = raw;
  if (typeof id !== "string" || typeof ingredient !== "string" || !optionalString(quantity) || !optionalString(unit) || !optionalString(taken) || typeof checked !== "boolean" || typeof manual !== "boolean") {
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
  return { id, ingredient, ...(quantity === undefined ? {} : { quantity }), ...(unit === undefined ? {} : { unit }), ...(taken === undefined ? {} : { taken }), checked, manual, sources };
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
 * deleted in between (D11), with its name in the copy before the check (D12), undefined if
 * the copy did not hold it; or a failure.
 */
export type WriteResult =
  | { kind: "done"; list: ShoppingList | undefined; removed: number }
  | { kind: "refused"; error: string }
  | { kind: "deleted"; name: string | undefined }
  | Failure;

/** The result of the meal shopping (C4): the served dishes added and ignored (Q22), or as a write. */
export type MealShoppingResult = { kind: "done"; added: number; ignored: number } | Exclude<WriteResult, { kind: "done" }>;

/**
 * The result of a move (C9, C10): the target list up to date, the items moved and those gone
 * in between (Q42), or as a write.
 */
export type MoveResult = { kind: "done"; target: ShoppingList; moved: number; missing: number } | Exclude<WriteResult, { kind: "done" }>;

/** ShoppingLists holds the copy of the lists; it dispatches "change" after each check and write. */
export class ShoppingLists extends EventTarget {
  readonly lists: CachedCollection<ShoppingList>;
  readonly library: Library;
  readonly #api: Api;
  /**
   * The checks shown before their answer, by item: the mark of the last one. A list taken
   * from the API replaces them; a check that fails gives back its item only if its mark is
   * still there (review of PR #100, point 1).
   */
  readonly #shownChecks = new Map<string, symbol>();
  /** The stale lists (C8), read after each check of the copy; kept as they were when the reading fails. */
  #stale: ReadonlySet<string> = new Set();

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

  /** Whether a list is stale (C8): the recomputation would change its items (D6). */
  isStale(id: string): boolean {
    return this.#stale.has(id);
  }

  /** Checks the copy of the lists and that of the library against the API, at the same time. */
  async check(): Promise<Outcome<undefined>> {
    const [lists, library] = await Promise.all([this.#checkLists(false), this.library.check()]);
    this.dispatchEvent(new Event("change"));
    return lists.kind !== "ok" ? lists : library;
  }

  /**
   * Checks the copy of the lists alone, behind after a refusal or a list unknown, then reads
   * the stale lists, just after it (D7): they depend on the meal plan and the dishes, not on
   * the version of the lists. Without answer, the stale lists stay as they were.
   */
  async #checkLists(tell = true): Promise<Outcome<undefined>> {
    const outcome = await this.lists.check();
    if (outcome.kind === "ok") {
      await this.#readStale();
    }
    if (tell) {
      this.dispatchEvent(new Event("change"));
    }
    return outcome;
  }

  /** Reads the stale lists (D7); without answer, they stay as they were. */
  async #readStale(): Promise<void> {
    const stale = await this.#api.getStaleLists();
    if (stale.kind === "ok") {
      this.#stale = new Set(stale.value);
    }
  }

  /** Checks the copy of the library alone: a source whose dish it does not hold (plan courses-suite, D15). */
  async checkLibrary(): Promise<void> {
    await this.library.check();
    this.dispatchEvent(new Event("change"));
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
      await this.#checkLists();
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
      await this.#checkLists();
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
   * comes back to the list sent back. Without network, the item alone comes back to what it
   * was, in the list as the copy holds it now, unless an answer of the API replaced it in
   * between: the other writes taken meanwhile stay (review of PR #100, point 1).
   */
  async setChecked(id: string, item: string, checked: boolean): Promise<WriteResult> {
    const mark = Symbol(item);
    if (this.#setItemChecked(id, item, checked)) {
      this.#shownChecks.set(item, mark);
    }
    const done = await this.#written(id, await this.#api.setChecked(id, item, checked));
    if (this.#shownChecks.get(item) === mark) {
      this.#shownChecks.delete(item);
      if (done.kind !== "done" && done.kind !== "deleted") {
        this.#setItemChecked(id, item, !checked);
      }
    }
    return done;
  }

  /** Shows an item of the copy checked or not, without version; false if the copy has no such item. */
  #setItemChecked(id: string, item: string, checked: boolean): boolean {
    const list = this.find(id);
    if (list === undefined || !list.items.some((i) => i.id === item)) {
      return false;
    }
    // Checked or unchecked, an item has no taken quantity (D9).
    const shown = (i: Item): Item => {
      const { taken: _, ...rest } = i;
      return { ...rest, checked };
    };
    this.lists.put([{ ...list, items: sortItems(list.items.map((i) => (i.id === item ? shown(i) : i))) }]);
    this.dispatchEvent(new Event("change"));
    return true;
  }

  /**
   * Removes the items seen checked that still are (C11, D9). The last items of a served dish
   * changed may leave: the stale lists are read again.
   */
  async checkOut(id: string, items: string[]): Promise<WriteResult> {
    const result = await this.#written(id, await this.#api.checkOut(id, items));
    if (result.kind === "done") {
      await this.#readStale();
      this.dispatchEvent(new Event("change"));
    }
    return result;
  }

  /**
   * Adds the meal shopping of a period to a list (C4, D6): computed by the server, which sends
   * back the list and how many served dishes it added and ignored (Q22).
   */
  async addMealShopping(id: string, period: Period): Promise<MealShoppingResult> {
    const outcome = await this.#api.addMealShopping(id, period.from, period.to);
    const result = await this.#written(id, outcome);
    if (result.kind !== "done") {
      return result;
    }
    const write = outcome.kind === "ok" && outcome.value.kind === "written" ? outcome.value : undefined;
    return { kind: "done", added: write?.added ?? 0, ignored: write?.ignored ?? 0 };
  }

  /**
   * Brings a list back in line with the meals (C8, D5): the list sent back is no longer stale,
   * even when nothing changed. Refused beyond 300 items, the list stays stale (Q38).
   */
  async recompute(id: string): Promise<WriteResult> {
    const result = await this.#written(id, await this.#api.recomputeList(id));
    if (result.kind === "done") {
      this.#stale = new Set([...this.#stale].filter((l) => l !== id));
      this.dispatchEvent(new Event("change"));
    }
    return result;
  }

  /**
   * Moves the items selected to another list, or to a new list of a name (C9, C10, D10): the
   * two lists sent back are taken. The served dishes received go with the items: the stale
   * lists are read again, the target may become stale and the source cease to be. A target
   * list unknown was deleted in between (Q42): refused, the copy checks itself.
   */
  async move(id: string, items: string[], to: { list: string } | { name: string }): Promise<MoveResult> {
    const outcome = await this.#api.moveItems(id, items, to);
    if (outcome.kind !== "ok") {
      return outcome;
    }
    const write = outcome.value;
    if (write.kind !== "written") {
      const result = await this.#refusedOrDeleted(id, write);
      // The items gone are counted, never unknown: a 404 on a list still there is unforeseen.
      return result.kind === "done" ? { kind: "network-error" } : result;
    }
    const source = listOf(write.list);
    const target = listOf(write.target);
    if (source === undefined || target === undefined) {
      return { kind: "network-error" };
    }
    await this.#apply([source, target], write.versions);
    await this.#readStale();
    this.dispatchEvent(new Event("change"));
    return { kind: "done", target, moved: write.moved ?? 0, missing: write.missing ?? 0 };
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
    for (const list of lists) {
      for (const item of list.items) {
        this.#shownChecks.delete(item.id);
      }
    }
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
        await this.#checkLists();
      }
      return { kind: "refused", error: write.error };
    }
    // Named after the copy before the check, which forgets it (D12).
    const name = id === undefined ? undefined : this.find(id)?.name;
    const checked = await this.#checkLists();
    if (checked.kind !== "ok") {
      return checked;
    }
    if (id === undefined || this.find(id) === undefined) {
      return { kind: "deleted", name };
    }
    return { kind: "done", list: this.find(id), removed: 0 };
  }

  /** Forgets the copy (D2). */
  clear(): void {
    this.#shownChecks.clear();
    this.#stale = new Set();
    this.lists.clear();
    this.dispatchEvent(new Event("change"));
  }
}
