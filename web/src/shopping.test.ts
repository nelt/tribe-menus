import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";
import type { Dish } from "./dishes.ts";
import type { ServedDish } from "./meal-plan.ts";
import {
  type Item,
  type ItemSource,
  type ShoppingList,
  addMonths,
  alreadyReceived,
  checkListName,
  checkoutOf,
  contributedQuantity,
  defaultPeriod,
  deleteBlocked,
  dishCount,
  ignoredMessage,
  itemQuantity,
  itemRowLabel,
  listSummary,
  mealDays,
  mealShoppingErrorMessage,
  mealShoppingMessage,
  monthGrid,
  monthOf,
  monthTitle,
  periodDays,
  periodError,
  periodSummary,
  servedInPeriod,
  shortDay,
  sourceRows,
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
  ["beyond a million grams", { quantity: "49999950000", unit: "g", manual: false }, "49999950 kg"],
];

for (const [name, change, want] of quantities) {
  test(`itemQuantity: ${name}`, () => {
    assert.equal(itemQuantity(item("x", change)), want);
  });
}

// The table of cases shared with Contribution and RoundItem in Go (Q37, D7; plan courses, lot C).
const mealShoppingTable = JSON.parse(readFileSync(new URL("../../internal/shopping/testdata/meal-shopping.json", import.meta.url), "utf8")) as {
  contributions: { contributed: string; contributedUnit: string; shown: string }[];
  items: { quantity: string; unit: string; shown: string }[];
};

test("contributedQuantity: the table shared with Go (C5, D7)", () => {
  assert.ok(mealShoppingTable.contributions.length > 0);
  for (const c of mealShoppingTable.contributions) {
    assert.equal(contributedQuantity({ quantity: c.contributed, unit: c.contributedUnit }), c.shown, `${c.contributed} ${c.contributedUnit}`);
  }
  assert.equal(contributedQuantity({}), "");
});

test("itemQuantity: the rounded items of the table shared with Go (Q37)", () => {
  assert.ok(mealShoppingTable.items.length > 0);
  for (const c of mealShoppingTable.items) {
    assert.equal(itemQuantity({ quantity: c.quantity, unit: c.unit, manual: false }), c.shown, `${c.quantity} ${c.unit}`);
  }
});

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

const served = (id: string, date: string, mealtime: "lunch" | "dinner" = "dinner", dish = "chili"): ServedDish => ({ id, date, mealtime, dish, servings: 4, rank: 1 });

test("defaultPeriod: from tomorrow to 7 days after today (D24)", () => {
  assert.deepEqual(defaultPeriod("2026-10-05"), { from: "2026-10-06", to: "2026-10-12" });
  assert.deepEqual(defaultPeriod("2026-12-28"), { from: "2026-12-29", to: "2027-01-04" });
});

test("periodDays and periodError (C4, Q38)", () => {
  assert.equal(periodDays({ from: "2026-10-06", to: "2026-10-06" }), 1);
  assert.equal(periodDays({ from: "2026-10-06", to: "2026-10-12" }), 7);
  assert.equal(periodDays({ from: "2026-10-25", to: "2026-10-26" }), 2, "across the change of time");
  assert.equal(periodError({ from: "2026-10-06", to: "2026-10-06" }), undefined);
  assert.equal(periodError({ from: "2026-10-06", to: "2026-11-05" }), undefined, "31 days");
  assert.equal(periodError({ from: "2026-10-06", to: "2026-11-06" }), "period_too_long");
  assert.equal(periodError({ from: "2026-10-08", to: "2026-10-06" }), "invalid_period");
});

test("servedInPeriod, periodSummary, alreadyReceived and ignoredMessage (C4, Q22)", () => {
  const all = [served("a", "2026-10-05"), served("b", "2026-10-06"), served("c", "2026-10-06", "lunch"), served("d", "2026-10-12"), served("e", "2026-10-13")];
  const period = { from: "2026-10-06", to: "2026-10-12" };
  const inPeriod = servedInPeriod(all, period);
  assert.deepEqual(
    inPeriod.map((s) => s.id),
    ["b", "c", "d"],
  );
  assert.deepEqual(periodSummary(period, inPeriod), { days: "7 jours", dishes: "3 plats servis" });
  assert.deepEqual(periodSummary({ from: "2026-10-06", to: "2026-10-06" }, inPeriod.slice(0, 1)), { days: "1 jour", dishes: "1 plat servi" });
  assert.deepEqual(periodSummary(period, []).dishes, "aucun plat servi");
  assert.equal(alreadyReceived({ received: ["a", "b", "d"] }, inPeriod), 2);
  assert.equal(ignoredMessage(0), undefined);
  assert.equal(ignoredMessage(1), "1 plat servi est déjà passé par cette liste : il sera ignoré.");
  assert.equal(ignoredMessage(3), "3 plats servis sont déjà passés par cette liste : ils seront ignorés.");
});

test("mealShoppingMessage and mealShoppingErrorMessage", () => {
  assert.equal(mealShoppingMessage(4, 0), "4 plats servis ajoutés");
  assert.equal(mealShoppingMessage(1, 1), "1 plat servi ajouté, 1 déjà reçu ignoré");
  assert.equal(mealShoppingMessage(3, 2), "3 plats servis ajoutés, 2 déjà reçus ignorés");
  assert.equal(mealShoppingMessage(0, 0), "Aucun plat servi ajouté");
  assert.equal(mealShoppingMessage(0, 2), "Aucun plat servi ajouté, 2 déjà reçus ignorés");
  assert.equal(mealShoppingErrorMessage("invalid_period"), "La date de fin doit suivre la date de début.");
  assert.equal(mealShoppingErrorMessage("period_too_long"), "La période ne peut pas dépasser 31 jours.");
  assert.equal(mealShoppingErrorMessage("too_many_items"), "La liste ne peut pas dépasser 300 articles : rien n’a été ajouté.");
  assert.equal(mealShoppingErrorMessage("invalid_date"), undefined);
});

test("months: of a day, shifted within 2000 to 2099, titled (Q35)", () => {
  assert.deepEqual(monthOf("2026-10-05"), { year: 2026, month: 9 });
  assert.deepEqual(addMonths({ year: 2026, month: 11 }, 1), { year: 2027, month: 0 });
  assert.deepEqual(addMonths({ year: 2026, month: 0 }, -1), { year: 2025, month: 11 });
  assert.equal(addMonths({ year: 2000, month: 0 }, -1), undefined);
  assert.equal(addMonths({ year: 2099, month: 11 }, 1), undefined);
  assert.equal(monthTitle({ year: 2026, month: 9 }), "Octobre 2026");
  assert.equal(shortDay("2026-10-06"), "mar. 6 oct.");
  assert.equal(shortDay("2026-10-01"), "jeu. 1er oct.");
});

test("monthGrid: Monday first, bounds, days between, today, meals (C4)", () => {
  const grid = monthGrid({ year: 2026, month: 9 }, { from: "2026-10-06", to: "2026-10-12" }, "2026-10-05", mealDays([served("a", "2026-10-06"), served("b", "2026-10-07")]));
  // 1 October 2026 is a Thursday: three blanks.
  assert.equal(grid.length, 3 + 31);
  assert.deepEqual(grid.slice(0, 3), [undefined, undefined, undefined]);
  const day = (n: number) => grid[2 + n];
  assert.equal(day(1)?.date, "2026-10-01");
  assert.deepEqual(
    [5, 6, 7, 12, 13].map((n) => [day(n)?.bound, day(n)?.between, day(n)?.today, day(n)?.meals, day(n)?.past]),
    [
      [false, false, true, false, false],
      [true, false, false, true, false],
      [false, true, false, true, false],
      [true, false, false, false, false],
      [false, false, false, false, false],
    ],
  );
  assert.equal(day(4)?.past, true);
  assert.equal(day(5)?.label, "lundi 5 octobre, aujourd’hui");
  assert.equal(day(6)?.label, "mardi 6 octobre, repas planifiés");
  assert.equal(day(1)?.label, "jeudi 1er octobre");
  // The end before the start: the bounds, nothing between.
  const backwards = monthGrid({ year: 2026, month: 9 }, { from: "2026-10-08", to: "2026-10-06" }, "2026-10-05", new Set());
  assert.deepEqual(
    backwards.filter((d) => d?.bound).map((d) => d?.date),
    ["2026-10-06", "2026-10-08"],
  );
  assert.equal(backwards.filter((d) => d?.between).length, 0);
  // A month starting on a Monday: no blank.
  assert.notEqual(monthGrid({ year: 2026, month: 5 }, { from: "2026-06-01", to: "2026-06-01" }, "2026-10-05", new Set())[0], undefined);
});

const dish = (id: string, name: string): Dish => ({ id, name, color: "#c2502e", servings: 4, revision: 1, ingredients: [] });

test("sourceRows: the meals in order, the quantity brought, a dish unknown left out (C5)", () => {
  const source = (servedDish: string, date: string, mealtime: "lunch" | "dinner", dishId: string, servings: number, quantity?: string): ItemSource => ({
    servedDish,
    dish: dishId,
    date,
    mealtime,
    servings,
    ...(quantity === undefined ? {} : { quantity, unit: "piece" }),
  });
  const item = {
    sources: [source("s3", "2026-10-08", "lunch", "chili", 4, "1"), source("s2", "2026-10-06", "dinner", "rata", 1), source("s1", "2026-10-06", "dinner", "chili", 3, "0.75"), source("s4", "2026-10-06", "lunch", "inconnu", 2, "1")],
  };
  const rows = sourceRows(item, [dish("chili", "Chili con carne"), dish("rata", "Ratatouille")]);
  assert.deepEqual(
    rows.map((r) => [r.day, r.mealtime, r.dish.name, r.servings, r.quantity]),
    [
      ["mar. 6 oct.", "dinner", "Chili con carne", "3 parts", "0,75 pièce"],
      ["mar. 6 oct.", "dinner", "Ratatouille", "1 part", ""],
      ["jeu. 8 oct.", "lunch", "Chili con carne", "4 parts", "1 pièce"],
    ],
  );
});

test("itemRowLabel (C5)", () => {
  const boeuf = item("b", { ingredient: "bœuf haché", quantity: "875", unit: "g", manual: false, sources: [{ servedDish: "s", dish: "d", date: "2026-10-06", mealtime: "dinner", servings: 3 }, { servedDish: "t", dish: "d", date: "2026-10-08", mealtime: "lunch", servings: 4 }] });
  assert.equal(itemRowLabel(boeuf, false), "bœuf haché, 875 g, 2 plats. Afficher les repas");
  assert.equal(itemRowLabel({ ingredient: "sel", manual: false, sources: boeuf.sources }, true), "sel, sans quantité, 2 plats. Masquer les repas");
});
