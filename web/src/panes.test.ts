import assert from "node:assert/strict";
import { test } from "node:test";
import type { Format } from "./layout.ts";
import { type Detail, type ListPane, detailKey, detailOf, holdsUnsaved, listPaneOf, lostBy } from "./panes.ts";
import { type Page, pages } from "./router.ts";

const id = "0123456789abcdef0123456789abcdef";
const other = "fedcba9876543210fedcba9876543210";

const paneCases: { name: string; page: Page; format: Format; list: ListPane | undefined; detail: Detail | undefined }[] = [
  { name: "planning, compact", page: pages.planning, format: "compact", list: undefined, detail: undefined },
  { name: "planning, etendu", page: pages.planning, format: "etendu", list: undefined, detail: undefined },
  { name: "a meal, etendu", page: { name: "meal", date: "2026-10-06", mealtime: "dinner" }, format: "etendu", list: undefined, detail: undefined },
  { name: "tribe, etendu", page: pages.tribe, format: "etendu", list: undefined, detail: undefined },
  { name: "library, compact", page: pages.dishes, format: "compact", list: "library", detail: undefined },
  { name: "library, moyen", page: pages.dishes, format: "moyen", list: "library", detail: undefined },
  { name: "library, etendu: an invitation to choose a dish (D5)", page: pages.dishes, format: "etendu", list: "library", detail: { kind: "choose-dish" } },
  { name: "a dish, compact", page: { name: "dish", id }, format: "compact", list: undefined, detail: { kind: "dish", id } },
  { name: "a dish, moyen", page: { name: "dish", id }, format: "moyen", list: undefined, detail: { kind: "dish", id } },
  { name: "a dish, etendu: beside the library", page: { name: "dish", id }, format: "etendu", list: "library", detail: { kind: "dish", id } },
  { name: "a new dish, etendu", page: pages.newDish, format: "etendu", list: "library", detail: { kind: "dish", id: undefined } },
  { name: "main list, compact", page: pages.shopping, format: "compact", list: undefined, detail: { kind: "list", list: undefined } },
  { name: "main list, etendu: beside the picker", page: pages.shopping, format: "etendu", list: "lists", detail: { kind: "list", list: undefined } },
  { name: "a list, etendu", page: { name: "shopping", list: id }, format: "etendu", list: "lists", detail: { kind: "list", list: id } },
  { name: "list picker, moyen", page: pages.listPicker, format: "moyen", list: "lists", detail: undefined },
  { name: "list picker, etendu: the main list (D5)", page: pages.listPicker, format: "etendu", list: "lists", detail: { kind: "list", list: undefined } },
  { name: "meal shopping, compact", page: { name: "meal-shopping", list: id }, format: "compact", list: undefined, detail: { kind: "meal-shopping", list: id } },
  { name: "meal shopping, etendu", page: pages.mealShopping, format: "etendu", list: "lists", detail: { kind: "meal-shopping", list: undefined } },
];

for (const c of paneCases) {
  test(`panes: ${c.name}`, () => {
    assert.equal(listPaneOf(c.page, c.format), c.list);
    assert.deepEqual(detailOf(c.page, c.format), c.detail);
  });
}

test("detailKey: one key per element", () => {
  const keys = [
    detailKey({ kind: "choose-dish" }),
    detailKey({ kind: "dish", id: undefined }),
    detailKey({ kind: "dish", id }),
    detailKey({ kind: "dish", id: other }),
    detailKey({ kind: "list", list: undefined }),
    detailKey({ kind: "list", list: id }),
    detailKey({ kind: "meal-shopping", list: undefined }),
    detailKey({ kind: "meal-shopping", list: id }),
  ];
  assert.equal(new Set(keys).size, keys.length);
});

const lostCases: { name: string; from: Page; to: Page; format: Format; list: boolean; detail: boolean }[] = [
  { name: "a dish to another, etendu: the library stays", from: { name: "dish", id }, to: { name: "dish", id: other }, format: "etendu", list: false, detail: true },
  { name: "a dish to the library, etendu", from: { name: "dish", id }, to: pages.dishes, format: "etendu", list: false, detail: true },
  { name: "a dish to the library, moyen", from: { name: "dish", id }, to: pages.dishes, format: "moyen", list: true, detail: true },
  { name: "a dish to the planning, etendu", from: { name: "dish", id }, to: pages.planning, format: "etendu", list: true, detail: true },
  { name: "a new dish to a dish, etendu", from: pages.newDish, to: { name: "dish", id }, format: "etendu", list: false, detail: true },
  { name: "the library to a dish, etendu", from: pages.dishes, to: { name: "dish", id }, format: "etendu", list: false, detail: true },
  { name: "the library to a dish, moyen: the library goes", from: pages.dishes, to: { name: "dish", id }, format: "moyen", list: true, detail: false },
  { name: "the main list to another, etendu: the picker stays", from: pages.shopping, to: { name: "shopping", list: id }, format: "etendu", list: false, detail: true },
  { name: "the main list to the picker, etendu: the same list", from: pages.shopping, to: pages.listPicker, format: "etendu", list: false, detail: false },
  { name: "the main list to the picker, moyen", from: pages.shopping, to: pages.listPicker, format: "moyen", list: true, detail: true },
  { name: "the picker to a list, moyen", from: pages.listPicker, to: { name: "shopping", list: id }, format: "moyen", list: true, detail: false },
  { name: "a list to its meal shopping, etendu", from: { name: "shopping", list: id }, to: { name: "meal-shopping", list: id }, format: "etendu", list: false, detail: true },
  { name: "a list to the planning, etendu", from: { name: "shopping", list: id }, to: pages.planning, format: "etendu", list: true, detail: true },
  { name: "the planning to a meal: nothing of a pane", from: pages.planning, to: { name: "meal", date: "2026-10-06", mealtime: "lunch" }, format: "etendu", list: false, detail: false },
];

for (const c of lostCases) {
  test(`lostBy: ${c.name}`, () => {
    assert.deepEqual(lostBy(c.from, c.to, c.format), { list: c.list, detail: c.detail });
  });
}

test("holdsUnsaved", () => {
  const element = (props: object) => Object.assign(Object.create(null) as Element, props);
  assert.equal(holdsUnsaved(null), false);
  assert.equal(holdsUnsaved(element({})), false);
  assert.equal(holdsUnsaved(element({ unsaved: false })), false);
  assert.equal(holdsUnsaved(element({ unsaved: true })), true);
});
