import assert from "node:assert/strict";
import { test } from "node:test";
import { type Api, type CollectionPage, type Outcome, type ShoppingWrite, shoppingWriteOf } from "./api.ts";
import { Library } from "./library.ts";
import { ShoppingLists, listOf } from "./shopping-lists.ts";

const pain = { id: "a1", ingredient: "pain", quantity: "2", unit: "piece", checked: false, manual: true, sources: [] };
const boeuf = {
  id: "a2",
  ingredient: "bœuf haché",
  quantity: "875",
  unit: "g",
  checked: false,
  manual: false,
  sources: [{ servedDish: "s1", dish: "d1", date: "2026-10-06", mealtime: "dinner", servings: 3, quantity: "375", unit: "g" }],
};
const courses = { id: "c", name: "Courses", main: true, rank: 1, items: [pain, boeuf], received: ["s1"] };
const marche = { id: "m", name: "Marché", main: false, rank: 2, items: [], received: [] };

test("listOf: a list of GET shopping-lists", () => {
  assert.deepEqual(listOf(courses), courses);
  assert.deepEqual(listOf(marche), marche);
});

const refused: [string, unknown][] = [
  ["no body", undefined],
  ["no received", { ...marche, received: undefined }],
  ["a rank as text", { ...marche, rank: "2" }],
  ["an item without sources", { ...marche, items: [{ ...pain, sources: undefined }] }],
  ["an item checked as text", { ...marche, items: [{ ...pain, checked: "true" }] }],
  ["a source of an unknown mealtime", { ...marche, items: [{ ...boeuf, sources: [{ ...boeuf.sources[0], mealtime: "brunch" }] }] }],
  ["a received served dish not a text", { ...marche, received: [1] }],
];

for (const [name, raw] of refused) {
  test(`listOf: ${name}`, () => {
    assert.equal(listOf(raw), undefined);
  });
}

const writeCases: [string, Parameters<typeof shoppingWriteOf>[0], Outcome<ShoppingWrite>][] = [
  ["a list written", { status: 201, body: { list: marche, versions: { shopping_lists: 3 } } }, { kind: "ok", value: { kind: "written", list: marche, versions: { shopping_lists: 3 }, removed: undefined } }],
  ["a checkout", { status: 200, body: { list: marche, removed: 2, versions: { shopping_lists: 4 } } }, { kind: "ok", value: { kind: "written", list: marche, versions: { shopping_lists: 4 }, removed: 2 } }],
  [
    "a meal shopping",
    { status: 200, body: { list: marche, added: 4, ignored: 1, versions: { shopping_lists: 5 } } },
    { kind: "ok", value: { kind: "written", list: marche, versions: { shopping_lists: 5 }, removed: undefined, added: 4, ignored: 1 } },
  ],
  ["a count not a number", { status: 200, body: { list: marche, added: "4", ignored: 1, versions: {} } }, { kind: "network-error" }],
  ["the versions only, empty", { status: 200, body: { versions: {} } }, { kind: "ok", value: { kind: "written", list: undefined, versions: {}, removed: undefined } }],
  ["a list unknown", { status: 404, body: { error: "not_found" } }, { kind: "ok", value: { kind: "not-found" } }],
  ["a name refused", { status: 400, body: { error: "list_name_empty" } }, { kind: "ok", value: { kind: "refused", error: "list_name_empty", list: undefined } }],
  ["a list no longer empty", { status: 409, body: { error: "list_not_empty", list: marche } }, { kind: "ok", value: { kind: "refused", error: "list_not_empty", list: marche } }],
  ["a request refused for its form", { status: 400, body: { error: "bad_request" } }, { kind: "network-error" }],
  ["a server error", { status: 500, body: undefined }, { kind: "network-error" }],
  ["no session", { status: 401, body: { error: "no_session" } }, { kind: "unauthorized" }],
  ["no response", "network-error", { kind: "network-error" }],
];

for (const [name, response, want] of writeCases) {
  test(`shoppingWriteOf: ${name}`, () => {
    assert.deepEqual(shoppingWriteOf(response), want);
  });
}

type Collection = { version: number; items: unknown[] };

/** An API whose collections are read in turn (the last one stays), and whose writes answer in turn. */
function fakeApi(lists: Collection[], writes: Outcome<ShoppingWrite>[], catalog: Collection[] = [{ version: 1, items: [] }]) {
  const calls: string[] = [];
  const pages: Record<string, Collection[]> = { "shopping-lists": lists, ingredients: catalog, dishes: [{ version: 1, items: [] }] };
  const write =
    (name: string) =>
    async (...args: unknown[]) => {
      calls.push(`${name} ${JSON.stringify(args)}`);
      return writes.shift() ?? { kind: "network-error" };
    };
  const api = {
    getPage: async (collection: string): Promise<Outcome<CollectionPage>> => {
      calls.push(`GET ${collection}`);
      const turns = pages[collection] ?? [];
      const c = turns.length > 1 ? turns.shift() : turns[0];
      if (c === undefined) {
        return { kind: "network-error" };
      }
      return { kind: "ok", value: { kind: "page", items: c.items, next: undefined, etag: `"${collection.replace("-", "_")}-${c.version}"` } };
    },
    createList: write("create"),
    renameList: write("rename"),
    setMainList: write("main"),
    deleteList: write("delete"),
    addItem: write("add"),
    removeItem: write("remove"),
    setChecked: write("checked"),
    checkOut: write("checkout"),
    addMealShopping: write("meal-shopping"),
  } as unknown as Api;
  return { api, calls };
}

async function loaded(api: Api): Promise<ShoppingLists> {
  const lists = new ShoppingLists(api, new Library(api));
  assert.deepEqual(await lists.check(), { kind: "ok", value: undefined });
  return lists;
}

const written = (list: unknown, versions: Record<string, number>, removed?: number): Outcome<ShoppingWrite> => ({ kind: "ok", value: { kind: "written", list, versions, removed } });
const names = (lists: ShoppingLists) => lists.lists.items?.map((l) => `${l.name}${l.main ? "*" : ""}`).sort();

test("ShoppingLists.check: the lists and the library at the same time (C5)", async () => {
  const { api, calls } = fakeApi([{ version: 5, items: [courses] }], []);
  const lists = await loaded(api);
  assert.deepEqual(calls, ["GET shopping-lists", "GET dishes", "GET ingredients"]);
  assert.equal(lists.library.loaded, true);
});

test("ShoppingLists.check and find: the main list without identifier", async () => {
  const { api } = fakeApi([{ version: 5, items: [courses, marche] }], []);
  const lists = await loaded(api);
  assert.equal(lists.loaded, true);
  assert.equal(lists.find(undefined)?.id, "c");
  assert.equal(lists.find("m")?.name, "Marché");
  assert.equal(lists.find("x"), undefined);
});

test("ShoppingLists.create: the list and the next version taken, nothing reloaded", async () => {
  const { api, calls } = fakeApi([{ version: 5, items: [courses] }], [written(marche, { shopping_lists: 6 })]);
  const lists = await loaded(api);
  assert.deepEqual(await lists.create("Marché"), { kind: "done", list: marche, removed: 0 });
  assert.deepEqual(names(lists), ["Courses*", "Marché"]);
  assert.equal(lists.lists.version, 6);
  assert.deepEqual(calls, ["GET shopping-lists", "GET dishes", "GET ingredients", 'create ["Marché"]']);
});

test("ShoppingLists.create: another member wrote in between, the copy is checked", async () => {
  const drive = { ...marche, id: "d", name: "Drive", rank: 3 };
  const { api, calls } = fakeApi(
    [
      { version: 5, items: [courses] },
      { version: 7, items: [courses, marche, drive] },
    ],
    [written(drive, { shopping_lists: 7 })],
  );
  const lists = await loaded(api);
  await lists.create("Drive");
  assert.deepEqual(names(lists), ["Courses*", "Drive", "Marché"]);
  assert.equal(calls.filter((c) => c === "GET shopping-lists").length, 2);
});

test("ShoppingLists.create: a name refused, nothing checked", async () => {
  const { api, calls } = fakeApi([{ version: 5, items: [courses] }], [{ kind: "ok", value: { kind: "refused", error: "list_name_empty", list: undefined } }]);
  const lists = await loaded(api);
  assert.deepEqual(await lists.create(" "), { kind: "refused", error: "list_name_empty" });
  assert.deepEqual(calls.slice(3), ['create [" "]']);
});

test("ShoppingLists.setMain: the two flags swapped; already main, the copy checked (C1)", async () => {
  const { api, calls } = fakeApi(
    [
      { version: 5, items: [courses, marche] },
      { version: 6, items: [{ ...courses, main: false }, { ...marche, main: true }] },
    ],
    [written(undefined, { shopping_lists: 6 }), written(undefined, {})],
  );
  const lists = await loaded(api);
  assert.deepEqual(await lists.setMain("m"), { kind: "done", list: undefined, removed: 0 });
  assert.deepEqual(names(lists), ["Courses", "Marché*"]);
  assert.equal(lists.lists.version, 6);
  assert.equal(calls.filter((c) => c === "GET shopping-lists").length, 1);
  await lists.setMain("m");
  assert.equal(calls.filter((c) => c === "GET shopping-lists").length, 2);
});

test("ShoppingLists.delete: removed with the version; already deleted, removed too (D11)", async () => {
  const drive = { ...marche, id: "d", name: "Drive", rank: 3 };
  const { api } = fakeApi(
    [
      { version: 5, items: [courses, marche, drive] },
      { version: 8, items: [courses] },
    ],
    [written(undefined, { shopping_lists: 6 }), { kind: "ok", value: { kind: "not-found" } }],
  );
  const lists = await loaded(api);
  assert.deepEqual(await lists.delete("m"), { kind: "done", list: undefined, removed: 0 });
  assert.deepEqual(names(lists), ["Courses*", "Drive"]);
  assert.deepEqual(await lists.delete("d"), { kind: "done", list: undefined, removed: 0 });
  assert.deepEqual(names(lists), ["Courses*"]);
});

test("ShoppingLists.delete: a list no longer empty, taken up to date, then the copy checked (D11)", async () => {
  const full = { ...marche, items: [pain] };
  const { api, calls } = fakeApi(
    [
      { version: 5, items: [courses, marche] },
      { version: 6, items: [courses, full] },
    ],
    [{ kind: "ok", value: { kind: "refused", error: "list_not_empty", list: full } }],
  );
  const lists = await loaded(api);
  assert.deepEqual(await lists.delete("m"), { kind: "refused", error: "list_not_empty" });
  assert.equal(lists.find("m")?.items.length, 1);
  assert.equal(calls.filter((c) => c === "GET shopping-lists").length, 2);
});

test("ShoppingLists.addItem: an ingredient created reloads the catalog", async () => {
  const added = { ...marche, items: [pain] };
  const { api, calls } = fakeApi(
    [{ version: 5, items: [courses, marche] }],
    [written(added, { shopping_lists: 6, ingredients: 2 }), written({ ...added, items: [pain, { ...pain, id: "a3" }] }, { shopping_lists: 7 })],
    [
      { version: 1, items: [] },
      { version: 2, items: [{ name: "pain", defaultUnit: "piece" }] },
    ],
  );
  const lists = await loaded(api);
  await lists.library.catalog.check();
  assert.deepEqual(await lists.addItem("m", { ingredient: "pain", quantity: "2", unit: "piece" }), { kind: "done", list: added, removed: 0 });
  assert.deepEqual(lists.library.catalog.items, [{ name: "pain", defaultUnit: "piece" }]);
  assert.equal(lists.lists.version, 6);
  await lists.addItem("m", { ingredient: "pain" });
  assert.equal(calls.filter((c) => c === "GET ingredients").length, 2);
});

test("ShoppingLists.addItem: a list deleted in between (D11)", async () => {
  const { api } = fakeApi(
    [
      { version: 5, items: [courses, marche] },
      { version: 6, items: [courses] },
    ],
    [{ kind: "ok", value: { kind: "not-found" } }],
  );
  const lists = await loaded(api);
  assert.deepEqual(await lists.addItem("m", { ingredient: "pain" }), { kind: "deleted" });
  assert.equal(lists.find("m"), undefined);
});

test("ShoppingLists.removeItem: an item already removed, no message (D11)", async () => {
  const { api } = fakeApi(
    [
      { version: 5, items: [courses] },
      { version: 6, items: [{ ...courses, items: [boeuf] }] },
    ],
    [{ kind: "ok", value: { kind: "not-found" } }],
  );
  const lists = await loaded(api);
  const result = await lists.removeItem("c", "a1");
  assert.equal(result.kind, "done");
  assert.deepEqual(lists.find("c")?.items.map((i) => i.id), ["a2"]);
});

test("ShoppingLists.setChecked: shown at once and sorted again, then the list sent back (C6, Q40)", async () => {
  const checked = { ...courses, items: [boeuf, { ...pain, checked: true }] };
  const { api } = fakeApi([{ version: 5, items: [courses] }], [written(checked, { shopping_lists: 6 })]);
  const lists = await loaded(api);
  const order: string[][] = [];
  lists.addEventListener("change", () => order.push(lists.find("c")?.items.map((i) => `${i.ingredient}${i.checked ? "✓" : ""}`) ?? []));
  await lists.setChecked("c", "a1", true);
  assert.deepEqual(order[0], ["bœuf haché", "pain✓"]);
  assert.equal(lists.lists.version, 6);
});

test("ShoppingLists.setChecked: without network, the item comes back to its state (D2)", async () => {
  const { api } = fakeApi([{ version: 5, items: [courses] }], [{ kind: "network-error" }]);
  const lists = await loaded(api);
  assert.deepEqual(await lists.setChecked("c", "a1", true), { kind: "network-error" });
  assert.equal(lists.find("c")?.items.find((i) => i.id === "a1")?.checked, false);
  assert.equal(lists.lists.version, 5);
});

test("ShoppingLists.checkOut: the number of items removed (D9)", async () => {
  const { api, calls } = fakeApi([{ version: 5, items: [courses] }], [written({ ...courses, items: [boeuf] }, { shopping_lists: 6 }, 1)]);
  const lists = await loaded(api);
  const result = await lists.checkOut("c", ["a1"]);
  assert.equal(result.kind === "done" && result.removed, 1);
  assert.deepEqual(calls.at(-1), 'checkout ["c",["a1"]]');
});

test("ShoppingLists.clear: the copy is forgotten", async () => {
  const { api } = fakeApi([{ version: 5, items: [courses] }], []);
  const lists = await loaded(api);
  lists.clear();
  assert.equal(lists.loaded, false);
  assert.equal(lists.lists.version, undefined);
});

// Review of PR #100, point 1: a check that fails gives back its own item only.
const lait = { id: "a5", ingredient: "lait", quantity: "1", unit: "l", checked: false, manual: true, sources: [] };

/** Two checks: the first answers once released, the second at once. */
async function twoChecks(painAnswer: Outcome<ShoppingWrite>, laitList: unknown) {
  let release: (o: Outcome<ShoppingWrite>) => void = () => {};
  const first = new Promise<Outcome<ShoppingWrite>>((resolve) => (release = resolve));
  const answers = [first, Promise.resolve(written(laitList, { shopping_lists: 6 }))];
  const api = {
    getPage: async (): Promise<Outcome<CollectionPage>> => ({ kind: "ok", value: { kind: "page", items: [{ ...courses, items: [pain, lait] }], next: undefined, etag: '"shopping_lists-5"' } }),
    setChecked: () => answers.shift(),
  } as unknown as Api;
  const lists = new ShoppingLists(api, new Library(api));
  await lists.lists.check();
  const painChecked = lists.setChecked("c", "a1", true);
  assert.deepEqual(await lists.setChecked("c", "a5", true), { kind: "done", list: listOf(laitList), removed: 0 });
  release(painAnswer);
  await painChecked;
  return Object.fromEntries((lists.find("c")?.items ?? []).map((i) => [i.ingredient, i.checked]));
}

test("ShoppingLists.setChecked: a check that fails after another one done leaves it checked", async () => {
  const server = { ...courses, items: [pain, { ...lait, checked: true }] };
  assert.deepEqual(await twoChecks({ kind: "network-error" }, server), { pain: false, lait: true });
});

test("ShoppingLists.setChecked: a check that fails while the server has it already keeps the server's state", async () => {
  // The answer to « pain » was lost, but the server took it: the answer to « lait » says so.
  const server = { ...courses, items: [{ ...pain, checked: true }, { ...lait, checked: true }] };
  assert.deepEqual(await twoChecks({ kind: "network-error" }, server), { pain: true, lait: true });
});

test("ShoppingLists.addMealShopping: the list taken, the served dishes added and ignored (C4, Q22)", async () => {
  const after = { ...courses, items: [pain, boeuf, { ...boeuf, id: "a3", ingredient: "oignon" }], received: ["s1", "s2"] };
  const { api, calls } = fakeApi([{ version: 5, items: [courses] }], [{ kind: "ok", value: { kind: "written", list: after, versions: { shopping_lists: 6 }, removed: undefined, added: 1, ignored: 1 } }]);
  const lists = await loaded(api);
  assert.deepEqual(await lists.addMealShopping("c", { from: "2026-10-06", to: "2026-10-12" }), { kind: "done", added: 1, ignored: 1 });
  assert.deepEqual(lists.find("c")?.received, ["s1", "s2"]);
  assert.equal(lists.lists.version, 6);
  assert.deepEqual(calls.at(-1), 'meal-shopping ["c","2026-10-06","2026-10-12"]');
});

test("ShoppingLists.addMealShopping: nothing added, the versions empty; 300 items refused; a list deleted (C4, Q38, D11)", async () => {
  const { api } = fakeApi(
    [{ version: 5, items: [courses, marche] }, { version: 5, items: [courses, marche] }, { version: 6, items: [courses] }],
    [
      { kind: "ok", value: { kind: "written", list: courses, versions: {}, removed: undefined, added: 0, ignored: 2 } },
      { kind: "ok", value: { kind: "refused", error: "too_many_items", list: undefined } },
      { kind: "ok", value: { kind: "not-found" } },
    ],
  );
  const lists = await loaded(api);
  assert.deepEqual(await lists.addMealShopping("c", { from: "2026-10-06", to: "2026-10-12" }), { kind: "done", added: 0, ignored: 2 });
  assert.equal(lists.lists.version, 5);
  assert.deepEqual(await lists.addMealShopping("c", { from: "2026-10-06", to: "2026-10-12" }), { kind: "refused", error: "too_many_items" });
  assert.deepEqual(await lists.addMealShopping("m", { from: "2026-10-06", to: "2026-10-12" }), { kind: "deleted" });
});
