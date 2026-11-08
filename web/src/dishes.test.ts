import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";
import {
  type Dish,
  type Ingredient,
  checkDish,
  dishErrorMessage,
  hasControl,
  hasVisible,
  highlight,
  inputDiffers,
  inputOfDish,
  initialOf,
  normalizeName,
  parseQuantity,
  quantityInput,
  searchDishes,
  sortDishes,
  splitSaved,
  suggestIngredients,
  trimSpaces,
  unitLabel,
  units,
  words,
} from "./dishes.ts";

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

// The table of cases shared with ParseQuantity in Go (D20).
const quantities = JSON.parse(readFileSync(new URL("../../internal/dish/testdata/quantities.json", import.meta.url), "utf8")) as {
  cases: { text: string; unit: string; amount?: number; error?: string }[];
};

test("parseQuantity: the table shared with Go", () => {
  assert.ok(quantities.cases.length > 0);
  for (const c of quantities.cases) {
    const want = c.error === undefined ? { amount: c.amount } : { error: c.error };
    assert.deepEqual(parseQuantity(c.text, c.unit), want, `${JSON.stringify(c.text)} ${c.unit}`);
  }
});

test("units: the closed list, in the singular, the spoons abridged", () => {
  assert.deepEqual(
    units.map((u) => u.label),
    ["g", "kg", "ml", "cl", "l", "pièce", "c. à soupe", "c. à café", "pincée"],
  );
  assert.equal(units.find((u) => u.code === "tablespoon")?.name, "cuillère à soupe");
});

test("unitLabel: plural from 2, spoons and metric units invariable", () => {
  const cases: [string, string, string][] = [
    ["1", "piece", "pièce"],
    ["1,5", "piece", "pièce"],
    ["2", "piece", "pièces"],
    ["3", "piece", "pièces"],
    ["2", "pinch", "pincées"],
    ["2.5", "pinch", "pincées"],
    ["500", "g", "g"],
    ["2", "kg", "kg"],
    ["2", "tablespoon", "c. à soupe"],
    ["3", "teaspoon", "c. à café"],
    ["", "piece", "pièce"],
    ["abc", "piece", "pièce"],
    ["2", "cup", ""],
  ];
  for (const [quantity, unit, want] of cases) {
    assert.equal(unitLabel(unit, quantity), want, `${quantity} ${unit}`);
  }
});

test("quantityInput: the quantity of the API with a decimal comma", () => {
  assert.equal(quantityInput("2.5"), "2,5");
  assert.equal(quantityInput("0.125"), "0,125");
  assert.equal(quantityInput("500"), "500");
  assert.equal(quantityInput(undefined), "");
});

const catalog: Ingredient[] = [
  { name: "tomate", defaultUnit: "piece" },
  { name: "tomate concassée", defaultUnit: "g" },
  { name: "crème fraîche" },
  { name: "riz", defaultUnit: "g" },
  { name: "piment d'Espelette" },
  { name: "Bœuf haché", defaultUnit: "g" },
];

const suggestions: { text: string; exclude?: string[]; names: string[]; create: boolean }[] = [
  { text: "tom", names: ["tomate", "tomate concassée"], create: true },
  { text: "CREME", names: ["crème fraîche"], create: true },
  { text: "fraiche", names: ["crème fraîche"], create: true },
  { text: "pim", names: ["piment d'Espelette"], create: true },
  { text: "boeuf", names: ["Bœuf haché"], create: true },
  { text: "tomate", names: ["tomate", "tomate concassée"], create: false },
  { text: " Tomate ", names: ["tomate", "tomate concassée"], create: false },
  { text: "tom", exclude: ["Tomate"], names: ["tomate concassée"], create: true },
  { text: "tomate", exclude: ["tomate"], names: ["tomate concassée"], create: false },
  { text: "ging", names: [], create: true },
  { text: "ate", names: [], create: true },
  { text: "  ", names: [], create: false },
  { text: "\u200b", names: [], create: false },
  { text: "\u0301", names: [], create: false },
  { text: "\u2003\ufeff", names: [], create: false },
];

for (const { text, exclude, names, create } of suggestions) {
  test(`suggestIngredients: "${text}"${exclude === undefined ? "" : ` without ${exclude.join(", ")}`}`, () => {
    const got = suggestIngredients(catalog, text, exclude ?? []);
    assert.deepEqual(
      got.ingredients.map((i) => i.name),
      names,
    );
    assert.equal(got.create, create);
  });
}

test("suggestIngredients: at most 6, by alphabetical order", () => {
  const many = ["pomme f", "pomme b", "pomme a", "pomme e", "pomme d", "pomme c", "pomme g"].map((name) => ({ name }));
  assert.deepEqual(
    suggestIngredients(many, "pom", []).ingredients.map((i) => i.name),
    ["pomme a", "pomme b", "pomme c", "pomme d", "pomme e", "pomme f"],
  );
});

test("highlight: the beginnings of words the text found", () => {
  assert.deepEqual(highlight("tomate concassée", "tom"), [
    { text: "tom", found: true },
    { text: "ate concassée", found: false },
  ]);
  assert.deepEqual(highlight("crème fraîche", "CREME"), [
    { text: "crème", found: true },
    { text: " fraîche", found: false },
  ]);
  assert.deepEqual(highlight("Bœuf haché", "boeu ha"), [
    { text: "Bœu", found: true },
    { text: "f ", found: false },
    { text: "ha", found: true },
    { text: "ché", found: false },
  ]);
  assert.deepEqual(highlight("piment d'Espelette", "esp"), [
    { text: "piment d'", found: false },
    { text: "Esp", found: true },
    { text: "elette", found: false },
  ]);
  assert.deepEqual(highlight("riz", ""), [{ text: "riz", found: false }]);
});

const line = (name: string, quantity = "", unit = "") => ({ name, quantity, unit });

const checks: { name: string; form: Parameters<typeof checkDish>[0]; want: ReturnType<typeof checkDish> }[] = [
  { name: "valid", form: { name: "Chili", lines: [line("bœuf", "500", "g"), line("sel")] }, want: undefined },
  { name: "name required", form: { name: "  ", lines: [] }, want: { error: "name_required" } },
  { name: "invisible name", form: { name: "\u200b", lines: [] }, want: { error: "name_required" } },
  { name: "mark alone as name", form: { name: "\u0301", lines: [] }, want: { error: "name_required" } },
  { name: "emoji as name", form: { name: "🍲", lines: [] }, want: undefined },
  {
    name: "quantity of spaces trim() would remove",
    form: { name: "Chili", lines: [line("riz", "\u2003", "")] },
    want: { error: "unit_required", line: 0 },
  },
  { name: "control character in the name", form: { name: "Soupe\u0007", lines: [] }, want: { error: "name_invalid" } },
  { name: "control character alone in the name", form: { name: "\u0085", lines: [] }, want: { error: "name_invalid" } },
  { name: "spaces are not control characters", form: { name: "Soupe\tà\nl'oignon\r", lines: [line("sel\t")] }, want: undefined },
  { name: "format character kept", form: { name: "Pain 👨\u200d🍳", lines: [] }, want: undefined },
  {
    name: "control character in a line",
    form: { name: "Chili", lines: [line("sel"), line("riz\u0085", "0", "g")] },
    want: { error: "ingredient_name_invalid", line: 1 },
  },
  { name: "delete character in a line", form: { name: "Chili", lines: [line("\u007f")] }, want: { error: "ingredient_name_invalid", line: 0 } },
  { name: "quantity without unit", form: { name: "Chili", lines: [line("sel"), line("riz", "1")] }, want: { error: "unit_required", line: 1 } },
  { name: "invalid quantity", form: { name: "Chili", lines: [line("riz", "0", "g")] }, want: { error: "invalid_quantity", line: 0 } },
  { name: "quantity too precise", form: { name: "Chili", lines: [line("lait", "2,5", "ml")] }, want: { error: "quantity_too_precise", line: 0 } },
  { name: "unit without quantity", form: { name: "Chili", lines: [line("sel", " ", "g")] }, want: undefined },
  {
    name: "same ingredient twice",
    form: { name: "Ratatouille", lines: [line("courgette", "3", "piece"), line("tomate"), line(" Courgette", "200", "g")] },
    want: { error: "duplicate_ingredient", line: 2, ingredient: "courgette" },
  },
];

for (const { name, form, want } of checks) {
  test(`checkDish: ${name}`, () => {
    assert.deepEqual(checkDish(form), want);
  });
}

// The table of cases of a displayed name shared with Go (D30).
const names = JSON.parse(readFileSync(new URL("../../internal/dish/testdata/names.json", import.meta.url), "utf8")) as {
  cases: { text: string; control: boolean; visible: boolean; trimmed: string }[];
};

test("hasControl, hasVisible and trimSpaces: the table shared with Go", () => {
  assert.ok(names.cases.length > 0);
  for (const c of names.cases) {
    const text = JSON.stringify(c.text);
    assert.equal(hasControl(c.text), c.control, `hasControl(${text})`);
    assert.equal(hasVisible(c.text), c.visible, `hasVisible(${text})`);
    assert.equal(trimSpaces(c.text), c.trimmed, `trimSpaces(${text})`);
  }
});

test("dishErrorMessage: the messages of docs/design/README.md", () => {
  assert.equal(dishErrorMessage({ error: "name_required" }), "Donnez un nom au plat.");
  assert.equal(dishErrorMessage({ error: "name_invalid" }), "Ce nom contient un caractère invalide.");
  assert.equal(dishErrorMessage({ error: "ingredient_name_invalid", line: 0 }), "Ce nom contient un caractère invalide.");
  assert.equal(dishErrorMessage({ error: "name_taken" }), "Un plat porte déjà ce nom.");
  assert.equal(dishErrorMessage({ error: "duplicate_ingredient", line: 1, ingredient: "courgette" }), "« courgette » figure déjà dans le plat.");
  assert.equal(dishErrorMessage({ error: "invalid_quantity", line: 0 }), "Quantité invalide.");
  assert.equal(dishErrorMessage({ error: "unit_required", line: 0 }), "Choisissez une unité.");
  assert.equal(dishErrorMessage({ error: "quantity_too_precise", line: 0 }), "Quantité trop précise pour cette unité.");
  // The codes a bounded entry cannot produce: an unforeseen response.
  assert.equal(dishErrorMessage({ error: "name_too_long" }), undefined);
  assert.equal(dishErrorMessage({ error: "too_many_lines" }), undefined);
});

test("splitSaved: the dish without the default units, and the ingredients of its lines (D27)", () => {
  const saved: Dish = {
    id: "a",
    name: "Curry de lentilles",
    color: "#3F7F25",
    servings: 5,
    revision: 1,
    ingredients: [
      { name: "lentilles", quantity: "300", unit: "g", defaultUnit: "g" },
      { name: "gingembre", quantity: "1", unit: "piece", defaultUnit: "piece" },
      { name: "sel" },
    ],
  };
  assert.deepEqual(splitSaved(saved), {
    dish: {
      ...saved,
      ingredients: [{ name: "lentilles", quantity: "300", unit: "g" }, { name: "gingembre", quantity: "1", unit: "piece" }, { name: "sel" }],
    },
    ingredients: [{ name: "lentilles", defaultUnit: "g" }, { name: "gingembre", defaultUnit: "piece" }, { name: "sel" }],
  });
});

// Plan grands-ecrans, D9: an input identical to what is saved is no input to confirm.
const chili: Dish = {
  id: "c",
  name: "Chili con carne",
  color: "#C2502E",
  servings: 4,
  revision: 2,
  ingredients: [{ name: "bœuf haché", quantity: "500", unit: "g" }, { name: "oignon", quantity: "1.5", unit: "piece" }, { name: "sel" }],
};

const differences: { name: string; change: (f: ReturnType<typeof inputOfDish>) => void; want: boolean }[] = [
  { name: "as saved", change: () => {}, want: false },
  { name: "the name changed", change: (f) => (f.name = "Chili du dimanche"), want: true },
  { name: "the name changed then back", change: (f) => (f.name = "Chili con carne"), want: false },
  { name: "spaces around the name", change: (f) => (f.name = " Chili con carne "), want: false },
  { name: "the servings changed", change: (f) => (f.servings = 6), want: true },
  { name: "the servings not read yet", change: (f) => (f.servings = undefined), want: false },
  { name: "a quantity changed", change: (f) => (f.lines[0]!.quantity = "600"), want: true },
  { name: "a quantity as saved, with its comma", change: (f) => (f.lines[1]!.quantity = "1,5"), want: false },
  { name: "a unit changed", change: (f) => (f.lines[0]!.unit = "kg"), want: true },
  { name: "the unit of a line without quantity", change: (f) => (f.lines[2]!.unit = "g"), want: false },
  { name: "a line removed", change: (f) => f.lines.pop(), want: true },
  { name: "a line added", change: (f) => f.lines.push({ name: "cumin", quantity: "", unit: "" }), want: true },
  { name: "the lines in another order", change: (f) => f.lines.reverse(), want: true },
];

for (const { name, change, want } of differences) {
  test(`inputDiffers: ${name}`, () => {
    const input = inputOfDish(chili);
    change(input);
    assert.equal(inputDiffers(inputOfDish(chili), input), want);
  });
}

test("inputOfDish", () => {
  assert.deepEqual(inputOfDish(chili), {
    name: "Chili con carne",
    servings: 4,
    lines: [
      { name: "bœuf haché", quantity: "500", unit: "g" },
      { name: "oignon", quantity: "1,5", unit: "piece" },
      { name: "sel", quantity: "", unit: "" },
    ],
  });
});
