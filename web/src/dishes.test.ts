import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";
import { type Dish, initialOf, normalizeName, searchDishes, sortDishes, words } from "./dishes.ts";

// The table of cases shared with NormalizeName in Go (D9).
const table = JSON.parse(readFileSync(new URL("../../internal/dish/testdata/normalized-names.json", import.meta.url), "utf8")) as {
  cases: { name: string; normalized: string }[];
};

test("normalizeName: the table shared with Go", () => {
  assert.ok(table.cases.length > 0);
  for (const c of table.cases) {
    assert.equal(normalizeName(c.name), c.normalized, JSON.stringify(c.name));
  }
});

test("normalizeName: spaces out of the set are kept, as in Go", () => {
  // U+2003 (em space) and U+FEFF are spaces for \s in JavaScript, not for the rule.
  assert.equal(normalizeName("a b"), "a b");
  assert.equal(normalizeName("﻿a"), "﻿a");
});

test("words", () => {
  assert.deepEqual(words("Soupe à l'oignon"), ["soupe", "a", "l", "oignon"]);
  assert.deepEqual(words("  Bœuf haché, 2x "), ["boeuf", "hache", "2x"]);
  assert.deepEqual(words(""), []);
});

function dish(name: string, ingredients: string[], id = name): Dish {
  return { id, name, color: "#C2502E", servings: 4, revision: 1, ingredients: ingredients.map((n) => ({ name: n })) };
}

// The library of the scenario P3 (bibliotheque-plats.feature).
const library = [
  dish("Ratatouille", ["courgette", "aubergine", "tomate"]),
  dish("Gratin dauphinois", ["pomme de terre", "crème"]),
  dish("Chili con carne", ["bœuf haché", "haricots rouges"]),
  dish("Gratin de courgettes", ["courgette", "crème", "gruyère"]),
];

const searches: { query: string; results: string[] }[] = [
  { query: "chili", results: ["Chili con carne"] },
  { query: "courgette", results: ["Gratin de courgettes", "Ratatouille"] },
  { query: "GRATIN", results: ["Gratin dauphinois", "Gratin de courgettes"] },
  { query: "creme", results: ["Gratin dauphinois", "Gratin de courgettes"] },
  { query: "gratin courgette", results: ["Gratin de courgettes"] },
  { query: "courg", results: ["Gratin de courgettes", "Ratatouille"] },
  { query: "poulet", results: [] },
  { query: "boeuf", results: ["Chili con carne"] },
  { query: "ourgette", results: [] },
  { query: "  ", results: ["Chili con carne", "Gratin dauphinois", "Gratin de courgettes", "Ratatouille"] },
];

for (const { query, results } of searches) {
  test(`searchDishes: "${query}"`, () => {
    assert.deepEqual(
      searchDishes(library, query).map((r) => r.dish.name),
      results,
    );
  });
}

test("searchDishes: the ingredients found", () => {
  const [gratin] = searchDishes(library, "gratin crem");
  assert.deepEqual([...(gratin?.found ?? [])], ["crème"]);
  const [chili] = searchDishes(library, "chili");
  assert.deepEqual([...(chili?.found ?? [])], []);
});

test("sortDishes: alphabetical order in French", () => {
  const sorted = sortDishes([dish("Œufs brouillés", []), dish("éclair", []), dish("Zucchini", []), dish("Eau", []), dish("ail", [])]);
  assert.deepEqual(
    sorted.map((d) => d.name),
    ["ail", "Eau", "éclair", "Œufs brouillés", "Zucchini"],
  );
});

test("initialOf: the first character as the reader sees it (review of PR #77, point 3)", () => {
  assert.equal(initialOf("Chili con carne"), "C");
  assert.equal(initialOf("Éclair"), "É");
  assert.equal(initialOf("e\u0301clair"), "e\u0301");
  assert.equal(initialOf("🍕 Pizza"), "🍕");
  assert.equal(initialOf("👨‍👩‍👧 Repas de famille"), "👨‍👩‍👧");
  assert.equal(initialOf("🇫🇷 Cassoulet"), "🇫🇷");
  assert.equal(initialOf(""), "");
});
