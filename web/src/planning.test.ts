import assert from "node:assert/strict";
import { test } from "node:test";
import type { Api, CollectionPage, Outcome } from "./api.ts";
import { Library } from "./library.ts";
import { Planning, servedDishOf } from "./planning.ts";

const served = { id: "0123456789abcdef0123456789abcdef", date: "2026-10-06", mealtime: "dinner", dish: "fedcba9876543210fedcba9876543210", servings: 4, rank: 3 };

test("servedDishOf: a dish served of GET meal-plan", () => {
  assert.deepEqual(servedDishOf(served), served);
  assert.deepEqual(servedDishOf({ ...served, mealtime: "lunch" }), { ...served, mealtime: "lunch" });
});

const refused: [string, unknown][] = [
  ["no body", undefined],
  ["an unknown mealtime", { ...served, mealtime: "breakfast" }],
  ["servings as text", { ...served, servings: "4" }],
  ["no rank", { ...served, rank: undefined }],
  ["a date as a number", { ...served, date: 20261006 }],
];

for (const [name, raw] of refused) {
  test(`servedDishOf: ${name}`, () => {
    assert.equal(servedDishOf(raw), undefined);
  });
}

/** An API whose collections are given; the collections absent fail as without network. */
function fakeApi(collections: Record<string, { version: number; items: unknown[] }>, read: string[] = []): Api {
  return {
    getPage: async (collection: string): Promise<Outcome<CollectionPage>> => {
      read.push(collection);
      const c = collections[collection];
      if (c === undefined) {
        return { kind: "network-error" };
      }
      return { kind: "ok", value: { kind: "page", items: c.items, next: undefined, etag: `"${collection.replace("-", "_")}-${c.version}"` } };
    },
  } as unknown as Api;
}

const chili = { id: served.dish, name: "Chili con carne", color: "#C2502E", servings: 4, revision: 1, ingredients: [] };

test("Planning.check: the meal plan and the library together (D15)", async () => {
  const read: string[] = [];
  const api = fakeApi({ "meal-plan": { version: 7, items: [served] }, dishes: { version: 3, items: [chili] }, ingredients: { version: 1, items: [] } }, read);
  const planning = new Planning(api, new Library(api));
  assert.equal(planning.loaded, false);
  let changes = 0;
  planning.addEventListener("change", () => changes++);
  assert.deepEqual(await planning.check(), { kind: "ok", value: undefined });
  assert.deepEqual(read.sort(), ["dishes", "ingredients", "meal-plan"]);
  assert.equal(planning.loaded, true);
  assert.deepEqual(planning.served.items, [served]);
  assert.equal(planning.served.version, 7);
  assert.deepEqual(planning.library.dishes.items, [chili]);
  assert.equal(changes, 1);
});

test("Planning.check: the meal plan or the library not read is a failure", async () => {
  const library = fakeApi({ dishes: { version: 3, items: [chili] }, ingredients: { version: 1, items: [] } });
  const withoutPlan = new Planning(library, new Library(library));
  assert.deepEqual(await withoutPlan.check(), { kind: "network-error" });
  assert.equal(withoutPlan.loaded, false);

  const plan = fakeApi({ "meal-plan": { version: 7, items: [served] } });
  const withoutLibrary = new Planning(plan, new Library(plan));
  assert.deepEqual(await withoutLibrary.check(), { kind: "network-error" });
  assert.equal(withoutLibrary.loaded, false);
});

test("Planning.check: an unreadable dish served is a failure", async () => {
  const api = fakeApi({ "meal-plan": { version: 7, items: [{ ...served, mealtime: "brunch" }] }, dishes: { version: 3, items: [] }, ingredients: { version: 1, items: [] } });
  const planning = new Planning(api, new Library(api));
  assert.deepEqual(await planning.check(), { kind: "network-error" });
  assert.equal(planning.served.items, undefined);
});

test("Planning.clear: the copy of the meal plan is forgotten", async () => {
  const api = fakeApi({ "meal-plan": { version: 7, items: [served] }, dishes: { version: 3, items: [] }, ingredients: { version: 1, items: [] } });
  const planning = new Planning(api, new Library(api));
  await planning.check();
  planning.clear();
  assert.equal(planning.served.items, undefined);
  assert.equal(planning.served.version, undefined);
  assert.equal(planning.loaded, false);
});
