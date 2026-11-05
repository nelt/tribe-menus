import assert from "node:assert/strict";
import { test } from "node:test";
import type { Api, CollectionPage, MealPlanWrite, Outcome } from "./api.ts";
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

const vegeId = "0000000000000000000000000000000v";
const other = { ...served, id: "11111111111111111111111111111111", dish: vegeId, servings: 2, rank: 4 };
const elsewhere = { ...served, id: "22222222222222222222222222222222", date: "2026-10-07", rank: 5 };

/** An API whose meal plan is given, and whose writes answer in turn. */
function writingApi(plan: unknown[], version: number, writes: Outcome<MealPlanWrite>[], reloads: { version: number; items: unknown[] }[] = []) {
  const calls: string[] = [];
  const pages = [{ version, items: plan }, ...reloads];
  const api = {
    getPage: async (collection: string): Promise<Outcome<CollectionPage>> => {
      if (collection !== "meal-plan") {
        return { kind: "ok", value: { kind: "page", items: collection === "dishes" ? [chili] : [], next: undefined, etag: `"${collection}-1"` } };
      }
      calls.push("GET meal-plan");
      const p = pages.length > 1 ? pages.shift() : pages[0];
      return { kind: "ok", value: { kind: "page", items: p?.items ?? [], next: undefined, etag: `"meal_plan-${p?.version ?? 0}"` } };
    },
    serveDish: async (body: unknown) => {
      calls.push(`POST ${JSON.stringify(body)}`);
      return writes.shift() ?? { kind: "network-error" };
    },
    setServings: async (id: string, n: number) => {
      calls.push(`PUT ${id} ${n}`);
      return writes.shift() ?? { kind: "network-error" };
    },
    removeServedDish: async (id: string) => {
      calls.push(`DELETE ${id}`);
      return writes.shift() ?? { kind: "network-error" };
    },
  } as unknown as Api;
  return { api, calls };
}

async function loaded(api: Api): Promise<Planning> {
  const planning = new Planning(api, new Library(api));
  await planning.check();
  return planning;
}

const ids = (planning: Planning) => planning.served.items?.map((s) => s.id).sort();

test("Planning.serve: the dish served and the next version are taken, nothing reloaded", async () => {
  const { api, calls } = writingApi([served], 7, [{ kind: "ok", value: { kind: "written", servedDish: other, version: 8 } }]);
  const planning = await loaded(api);
  let changes = 0;
  planning.addEventListener("change", () => changes++);
  assert.deepEqual(await planning.serve("2026-10-06", "dinner", vegeId), { kind: "served", served: other });
  assert.deepEqual(calls, ["GET meal-plan", `POST {"date":"2026-10-06","mealtime":"dinner","dish":"${vegeId}"}`]);
  assert.deepEqual(ids(planning), [served.id, other.id].sort());
  assert.equal(planning.served.version, 8);
  assert.equal(changes, 1);
});

test("Planning.serve: another member wrote in between, the copy is checked", async () => {
  const { api, calls } = writingApi([served], 7, [{ kind: "ok", value: { kind: "written", servedDish: other, version: 9 } }], [{ version: 9, items: [served, other, elsewhere] }]);
  const planning = await loaded(api);
  await planning.serve("2026-10-06", "dinner", vegeId);
  assert.equal(calls.filter((c) => c === "GET meal-plan").length, 2);
  assert.deepEqual(ids(planning), [served.id, other.id, elsewhere.id].sort());
  assert.equal(planning.served.version, 9);
});

// Review of PR #88, point 1: the meal sent back has no version, the copy is checked.
test("Planning.serve: a dish already served, the meal as it is now taken, then the copy checked (D3)", async () => {
  const { api, calls } = writingApi([served, elsewhere], 7, [{ kind: "ok", value: { kind: "already-served", meal: [other] } }], [{ version: 9, items: [other, elsewhere] }]);
  const planning = await loaded(api);
  assert.deepEqual(await planning.serve("2026-10-06", "dinner", vegeId), { kind: "already-served" });
  assert.equal(calls.filter((c) => c === "GET meal-plan").length, 2);
  assert.deepEqual(ids(planning), [other.id, elsewhere.id].sort());
  assert.equal(planning.served.version, 9);
});

test("Planning.serve: an unreadable answer is a network error", async () => {
  const { api } = writingApi([served], 7, [{ kind: "ok", value: { kind: "written", servedDish: { id: "x" }, version: 8 } }]);
  const planning = await loaded(api);
  assert.deepEqual(await planning.serve("2026-10-06", "dinner", vegeId), { kind: "network-error" });
  assert.deepEqual(ids(planning), [served.id]);
});

test("Planning.setServings: the dish served taken; removed in between, dropped and the copy checked (D3)", async () => {
  const set = { ...served, servings: 3 };
  const { api, calls } = writingApi([served, other], 7, [{ kind: "ok", value: { kind: "written", servedDish: set, version: 8 } }, { kind: "ok", value: { kind: "not-found" } }], [{ version: 10, items: [set] }]);
  const planning = await loaded(api);
  assert.deepEqual(await planning.setServings(served.id, 3), { kind: "saved", served: set });
  assert.equal(planning.served.items?.find((s) => s.id === served.id)?.servings, 3);
  assert.deepEqual(await planning.setServings(other.id, 1), { kind: "removed" });
  assert.deepEqual(ids(planning), [served.id]);
  assert.equal(planning.served.version, 10);
  assert.deepEqual(calls.slice(1, 3), [`PUT ${served.id} 3`, `PUT ${other.id} 1`]);
});

test("Planning.remove: removed with the version; already removed, dropped without failure (D19)", async () => {
  const { api } = writingApi([served, other], 7, [{ kind: "ok", value: { kind: "written", servedDish: undefined, version: 8 } }, { kind: "ok", value: { kind: "not-found" } }], [{ version: 9, items: [] }]);
  const planning = await loaded(api);
  assert.deepEqual(await planning.remove(served.id), { kind: "ok", value: undefined });
  assert.deepEqual(ids(planning), [other.id]);
  assert.equal(planning.served.version, 8);
  assert.deepEqual(await planning.remove(other.id), { kind: "ok", value: undefined });
  assert.deepEqual(ids(planning), []);
  assert.equal(planning.served.version, 9);
});

test("Planning: a write without network changes nothing", async () => {
  const { api } = writingApi([served], 7, [{ kind: "network-error" }, { kind: "network-error" }, { kind: "network-error" }]);
  const planning = await loaded(api);
  assert.deepEqual(await planning.serve("2026-10-06", "dinner", vegeId), { kind: "network-error" });
  assert.deepEqual(await planning.setServings(served.id, 3), { kind: "network-error" });
  assert.deepEqual(await planning.remove(served.id), { kind: "network-error" });
  assert.deepEqual(planning.served.items, [served]);
  assert.equal(planning.served.version, 7);
});
