import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";
import {
  type Item,
  type ShoppingList,
  checkListName,
  checkoutOf,
  deleteBlocked,
  dishCount,
  itemQuantity,
  listSummary,
  progressOf,
  removedMessage,
  shoppingErrorMessage,
  sortItems,
  sortLists,
} from "./shopping.ts";

function item(id: string, change: Partial<Item> = {}): Item {
  return { id, ingredient: id, checked: false, manual: true, sources: [], ...change };
}

function list(id: string, change: Partial<ShoppingList> = {}): ShoppingList {
  return { id, name: id, main: false, rank: 1, items: [], received: [], ...change };
}

// The table of cases shared with SortItems in Go (Q40; review of PR #98, point 3).
const sortTable = JSON.parse(readFileSync(new URL("../../internal/shopping/testdata/sort-items.json", import.meta.url), "utf8")) as {
  cases: { name: string; items: { id: string; ingredient: string; quantity?: string; unit?: string; checked?: boolean; manual?: boolean }[]; order: string[] }[];
};

test("sortItems: the table shared with Go", () => {
  assert.ok(sortTable.cases.length > 0);
  for (const c of sortTable.cases) {
    const items = c.items.map((i) => item(i.id, { ...i, checked: i.checked ?? false, manual: i.manual ?? false }));
    assert.deepEqual(
      sortItems(items).map((i) => i.id),
      c.order,
      c.name,
    );
  }
});

test("sortLists: the main list, then by order of creation (C2)", () => {
  const lists = [list("marche", { rank: 2 }), list("drive", { rank: 3, main: true }), list("courses", { rank: 1 })];
  assert.deepEqual(
    sortLists(lists).map((l) => l.id),
    ["drive", "courses", "marche"],
  );
});

const quantities: [string, Partial<Item>, string][] = [
  ["a mass of grams", { quantity: "875", unit: "g", manual: false }, "875 g"],
  ["a mass of a kilogram and more", { quantity: "1500", unit: "g", manual: false }, "1,5 kg"],
  ["a kilogram exactly", { quantity: "1000", unit: "g", manual: false }, "1 kg"],
  ["grams with decimals", { quantity: "1234.5", unit: "g", manual: false }, "1,2345 kg"],
  ["a volume in centilitres", { quantity: "750", unit: "ml", manual: false }, "75 cl"],
  ["a volume in litres", { quantity: "1500", unit: "ml", manual: false }, "1,5 l"],
  ["a volume under a centilitre", { quantity: "5", unit: "ml", manual: false }, "5 ml"],
  ["a volume of a centilitre and a half", { quantity: "15", unit: "ml", manual: false }, "1,5 cl"],
  ["pieces, plural from 2", { quantity: "2", unit: "piece", manual: false }, "2 pièces"],
  ["a piece", { quantity: "1", unit: "piece", manual: false }, "1 pièce"],
  ["spoons, invariable", { quantity: "3", unit: "tablespoon", manual: false }, "3 c. à soupe"],
  ["pinches", { quantity: "2", unit: "pinch", manual: false }, "2 pincées"],
  ["a manual item keeps its unit", { quantity: "1500", unit: "g", manual: true }, "1500 g"],
  ["a manual item with decimals", { quantity: "1.5", unit: "kg", manual: true }, "1,5 kg"],
  ["a manual item of a piece and a half", { quantity: "1.5", unit: "piece", manual: true }, "1,5 pièce"],
  ["without quantity", { manual: false }, ""],
];

for (const [name, change, want] of quantities) {
  test(`itemQuantity: ${name}`, () => {
    assert.equal(itemQuantity(item("x", change)), want);
  });
}

test("dishCount (C5)", () => {
  const source = { servedDish: "s", dish: "d", date: "2026-10-06", mealtime: "dinner" as const, servings: 3 };
  assert.equal(dishCount(item("x", { sources: [source] })), "1 plat");
  assert.equal(dishCount(item("x", { sources: [source, { ...source, servedDish: "t" }] })), "2 plats");
});

test("progressOf and listSummary", () => {
  const eight = list("l", { items: [...Array(8).keys()].map((k) => item(`i${k}`, { checked: k < 3 })) });
  assert.deepEqual(progressOf(eight), { checked: 3, total: 8, label: "3 sur 8 articles", remaining: "5 à prendre", percent: 38 });
  assert.equal(listSummary(eight), "8 articles · 5 à prendre");
  assert.equal(progressOf(list("l", { items: [item("a")] })).label, "0 sur 1 article");
  assert.equal(listSummary(list("l", { items: [item("a")] })), "1 article · 1 à prendre");
  assert.equal(listSummary(list("l")), "Vide");
  assert.equal(progressOf(list("l")).percent, 0);
});

test("checkoutOf: what « Courses faites » would do (C11, D9)", () => {
  assert.deepEqual(checkoutOf(list("l", { items: [item("a")] })), {
    leaving: [],
    staying: 1,
    hint: "Cochez les articles achetés pour les retirer de la liste.",
    message: "L’article coché sera retiré de la liste. L’article non coché y reste.",
  });
  const one = checkoutOf(list("l", { items: [item("a", { checked: true }), item("b"), item("c")] }));
  assert.deepEqual(one.leaving, ["a"]);
  assert.equal(one.hint, "Retire l’article coché de la liste ; les autres y restent.");
  assert.equal(one.message, "L’article coché sera retiré de la liste. Les 2 articles non cochés y restent.");
  const all = checkoutOf(list("l", { items: [item("a", { checked: true }), item("b", { checked: true }), item("c", { checked: true })] }));
  assert.equal(all.hint, "Retire les 3 articles cochés de la liste ; les autres y restent.");
  assert.equal(all.message, "Les 3 articles cochés seront retirés de la liste. La liste sera vide.");
});

test("removedMessage", () => {
  assert.equal(removedMessage(1), "1 article retiré");
  assert.equal(removedMessage(3), "3 articles retirés");
});

test("deleteBlocked (C12)", () => {
  assert.equal(deleteBlocked(list("l")), undefined);
  assert.equal(deleteBlocked(list("l", { main: true })), "Désignez d’abord une autre liste comme principale.");
  assert.equal(deleteBlocked(list("l", { items: [item("a")] })), "Une liste doit être vide pour être supprimée.");
  assert.equal(deleteBlocked(list("l", { main: true, items: [item("a")] })), "Désignez d’abord une autre liste comme principale.");
});

test("checkListName (Q38)", () => {
  assert.equal(checkListName("Marché"), undefined);
  assert.equal(checkListName(""), "list_name_empty");
  assert.equal(checkListName("   "), "list_name_empty");
  assert.equal(checkListName("​"), "list_name_empty");
  assert.equal(checkListName("Marché\u0007"), "list_name_invalid");
  assert.equal(checkListName("🧑‍🍳 Drive"), undefined);
});

test("shoppingErrorMessage", () => {
  assert.equal(shoppingErrorMessage("list_name_empty"), "Donnez un nom à la liste.");
  assert.equal(shoppingErrorMessage("ingredient_name_required"), "Indiquez un article.");
  assert.equal(shoppingErrorMessage("too_many_items"), "La liste ne peut pas dépasser 300 articles.");
  assert.equal(shoppingErrorMessage("list_name_too_long"), undefined);
});
