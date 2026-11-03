import assert from "node:assert/strict";
import { test } from "node:test";
import type { Api, CollectionPage, Outcome } from "./api.ts";
import { Library, dishOf, ingredientOf } from "./library.ts";

const chili = {
  id: "0123456789abcdef0123456789abcdef",
  name: "Chili con carne",
  color: "#C2502E",
  servings: 4,
  revision: 2,
  ingredients: [{ name: "bœuf haché", quantity: "500", unit: "g" }, { name: "sel" }],
};

test("dishOf: a dish of GET dishes", () => {
  assert.deepEqual(dishOf(chili), chili);
});

test("dishOf: a dish written, with the default units (D27)", () => {
  const written = { ...chili, ingredients: [{ name: "sel", defaultUnit: "pinch" }] };
  assert.deepEqual(dishOf(written), written);
});

const refused: [string, unknown][] = [
  ["no body", undefined],
  ["no ingredients", { ...chili, ingredients: undefined }],
  ["servings as text", { ...chili, servings: "4" }],
  ["a line without name", { ...chili, ingredients: [{ quantity: "1", unit: "g" }] }],
  ["a quantity as a number", { ...chili, ingredients: [{ name: "sel", quantity: 1, unit: "g" }] }],
];

for (const [name, raw] of refused) {
  test(`dishOf: ${name}`, () => {
    assert.equal(dishOf(raw), undefined);
  });
}

test("ingredientOf", () => {
  assert.deepEqual(ingredientOf({ name: "sel" }), { name: "sel" });
  assert.deepEqual(ingredientOf({ name: "riz", defaultUnit: "g" }), { name: "riz", defaultUnit: "g" });
  assert.equal(ingredientOf({ name: 1 }), undefined);
  assert.equal(ingredientOf({ name: "riz", defaultUnit: 1 }), undefined);
});

/** An API whose collections are given, at the version given. */
function fakeApi(collections: Record<string, { version: number; items: unknown[] }>): Api {
  return {
    getPage: async (collection: string): Promise<Outcome<CollectionPage>> => {
      const c = collections[collection];
      if (c === undefined) {
        return { kind: "network-error" };
      }
      return { kind: "ok", value: { kind: "page", items: c.items, next: undefined, etag: `"${collection}-${c.version}"` } };
    },
  } as unknown as Api;
}

test("Library.applySaved: the dish and the ingredients of its lines enter the copy (D26, D27)", async () => {
  const library = new Library(fakeApi({ dishes: { version: 3, items: [chili] }, ingredients: { version: 2, items: [{ name: "bœuf haché", defaultUnit: "g" }, { name: "sel" }] } }));
  await library.check();
  let changes = 0;
  library.addEventListener("change", () => changes++);
  const curry = {
    ...chili,
    id: "fedcba9876543210fedcba9876543210",
    name: "Curry de lentilles",
    ingredients: [{ name: "gingembre", quantity: "1", unit: "piece", defaultUnit: "piece" }, { name: "sel", defaultUnit: "pinch" }],
  };
  await library.applySaved(curry, { dishes: 4, ingredients: 3 });
  assert.deepEqual(
    library.dishes.items?.map((d) => d.name),
    ["Chili con carne", "Curry de lentilles"],
  );
  assert.deepEqual(library.dishes.items?.[1]?.ingredients, [{ name: "gingembre", quantity: "1", unit: "piece" }, { name: "sel" }]);
  assert.deepEqual(library.catalog.items, [{ name: "bœuf haché", defaultUnit: "g" }, { name: "sel", defaultUnit: "pinch" }, { name: "gingembre", defaultUnit: "piece" }]);
  assert.equal(library.dishes.version, 4);
  assert.equal(library.catalog.version, 3);
  assert.equal(changes, 1);
});

test("Library.applyCurrent: the current dish of a refusal enters the copy, its version stays (D5, D26)", async () => {
  const library = new Library(fakeApi({ dishes: { version: 3, items: [chili] }, ingredients: { version: 2, items: [{ name: "bœuf haché" }] } }));
  await library.check();
  const current = { ...chili, revision: 3, servings: 6, ingredients: [{ name: "bœuf haché", quantity: "600", unit: "g", defaultUnit: "g" }, { name: "cumin", defaultUnit: "teaspoon" }] };
  library.applyCurrent(current);
  assert.deepEqual(library.dishes.items, [{ ...current, ingredients: [{ name: "bœuf haché", quantity: "600", unit: "g" }, { name: "cumin" }] }]);
  assert.deepEqual(library.catalog.items, [{ name: "bœuf haché", defaultUnit: "g" }, { name: "cumin", defaultUnit: "teaspoon" }]);
  assert.equal(library.dishes.version, 3);
});
