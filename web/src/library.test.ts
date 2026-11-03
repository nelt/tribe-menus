import assert from "node:assert/strict";
import { test } from "node:test";
import { dishOf, ingredientOf } from "./library.ts";

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
