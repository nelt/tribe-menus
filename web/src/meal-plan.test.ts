import assert from "node:assert/strict";
import { test } from "node:test";
import type { Dish } from "./dishes.ts";
import {
  type ServedDish,
  addDays,
  dayRange,
  mealTiles,
  nextStart,
  parseDay,
  previousStart,
  rangeTitle,
  servingsLabel,
  shortDate,
  today,
  totalServings,
  weekdayName,
} from "./meal-plan.ts";

test("today: the local date of the device (D4)", () => {
  assert.equal(today(new Date(2026, 9, 5, 0, 1)), "2026-10-05");
  assert.equal(today(new Date(2026, 9, 5, 23, 59)), "2026-10-05");
  assert.equal(today(new Date(2027, 0, 1)), "2027-01-01");
});

const dayCases: { text: string; want: string | undefined }[] = [
  { text: "2026-10-05", want: "2026-10-05" },
  { text: "2000-01-01", want: "2000-01-01" },
  { text: "2099-12-31", want: "2099-12-31" },
  { text: "2024-02-29", want: "2024-02-29" },
  { text: "1999-12-31", want: undefined },
  { text: "2100-01-01", want: undefined },
  { text: "2026-02-29", want: undefined },
  { text: "2026-02-30", want: undefined },
  { text: "2026-13-01", want: undefined },
  { text: "2026-10-5", want: undefined },
  { text: "2026-10-05T00:00", want: undefined },
  { text: " 2026-10-05", want: undefined },
  { text: "", want: undefined },
];

for (const { text, want } of dayCases) {
  test(`parseDay: "${text}"`, () => {
    assert.equal(parseDay(text), want);
  });
}

test("addDays: across months, years and a leap day", () => {
  assert.equal(addDays("2026-10-05", 7), "2026-10-12");
  assert.equal(addDays("2026-10-05", -7), "2026-09-28");
  assert.equal(addDays("2026-12-29", 6), "2027-01-04");
  assert.equal(addDays("2024-02-28", 1), "2024-02-29");
  assert.equal(addDays("2026-03-28", 2), "2026-03-30");
});

test("dayRange: the 7 days from the one given (R1)", () => {
  assert.deepEqual(dayRange("2026-09-28"), ["2026-09-28", "2026-09-29", "2026-09-30", "2026-10-01", "2026-10-02", "2026-10-03", "2026-10-04"]);
});

test("previousStart and nextStart: 7 days before and after (R5)", () => {
  assert.equal(previousStart("2026-10-05"), "2026-09-28");
  assert.equal(nextStart("2026-10-05"), "2026-10-12");
});

// Review of PR #87, point 2: every day of the bounds can be reached, none beyond.
test("previousStart: at the first day of the bounds at most, none from it", () => {
  assert.equal(previousStart("2000-01-08"), "2000-01-01");
  assert.equal(previousStart("2000-01-03"), "2000-01-01");
  assert.equal(previousStart("2000-01-01"), undefined);
});

test("nextStart: none beyond the last day of the bounds", () => {
  assert.equal(nextStart("2099-12-24"), "2099-12-31");
  assert.equal(nextStart("2099-12-25"), undefined);
  assert.equal(nextStart("2099-12-31"), undefined);
});

test("weekdayName and shortDate", () => {
  assert.deepEqual(dayRange("2026-10-05").map((d) => `${weekdayName(d)} ${shortDate(d)}`), [
    "Lundi 5 oct.",
    "Mardi 6 oct.",
    "Mercredi 7 oct.",
    "Jeudi 8 oct.",
    "Vendredi 9 oct.",
    "Samedi 10 oct.",
    "Dimanche 11 oct.",
  ]);
  assert.equal(shortDate("2026-01-01"), "1er janv.");
  assert.equal(shortDate("2026-02-14"), "14 févr.");
  assert.equal(shortDate("2026-08-15"), "15 août");
  assert.equal(shortDate("2026-12-25"), "25 déc.");
});

const titleCases: { name: string; start: string; today: string; want: string }[] = [
  { name: "one month", start: "2026-10-05", today: "2026-10-05", want: "5 – 11 octobre" },
  { name: "two months", start: "2026-09-28", today: "2026-10-05", want: "28 septembre – 4 octobre" },
  { name: "two years", start: "2026-12-29", today: "2026-10-05", want: "29 décembre 2026 – 4 janvier 2027" },
  { name: "two years, seen from the next one", start: "2026-12-29", today: "2027-01-02", want: "29 décembre 2026 – 4 janvier 2027" },
  { name: "another year", start: "2025-10-06", today: "2026-10-05", want: "6 – 12 octobre 2025" },
  { name: "another year, two months", start: "2027-03-29", today: "2026-10-05", want: "29 mars – 4 avril 2027" },
  { name: "the first of a month", start: "2026-06-01", today: "2026-06-01", want: "1er – 7 juin" },
  { name: "ending on the first of a month", start: "2026-06-25", today: "2026-06-01", want: "25 juin – 1er juillet" },
];

for (const { name, start, today: t, want } of titleCases) {
  test(`rangeTitle: ${name}`, () => {
    assert.equal(rangeTitle(dayRange(start), t), want);
  });
}

function dish(id: string, name: string): Dish {
  return { id, name, color: "#C2502E", servings: 4, revision: 1, ingredients: [] };
}

function served(id: string, date: string, mealtime: "lunch" | "dinner", dishId: string, servings: number, rank: number): ServedDish {
  return { id, date, mealtime, dish: dishId, servings, rank };
}

const chili = dish("c", "Chili con carne");
const vege = dish("v", "Chili végétarien");
const pesto = dish("p", "Pâtes pesto");

test("mealTiles: two meals a day, lunch then dinner, dishes by rank (D11)", () => {
  const days = dayRange("2026-10-05");
  const plan = [
    served("s3", "2026-10-06", "dinner", "v", 1, 7),
    served("s1", "2026-10-06", "dinner", "c", 3, 2),
    served("s2", "2026-10-06", "lunch", "p", 4, 5),
    served("s4", "2026-10-12", "lunch", "p", 4, 9),
  ];
  const tiles = mealTiles(days, plan, [chili, vege, pesto], "2026-10-05");
  assert.equal(tiles.length, 7);
  assert.deepEqual(
    tiles.map((d) => [d.date, d.today, d.meals.map((m) => m.mealtime)]),
    days.map((d) => [d, d === "2026-10-05", ["lunch", "dinner"]]),
  );
  const tuesday = tiles[1];
  assert.deepEqual(tuesday?.meals.map((m) => m.dishes.map((d) => [d.dish.name, d.served.servings])), [
    [["Pâtes pesto", 4]],
    [
      ["Chili con carne", 3],
      ["Chili végétarien", 1],
    ],
  ]);
  assert.deepEqual(tiles.flatMap((d) => d.meals.filter((m) => m.dishes.length > 0).map((m) => m.date)), ["2026-10-06", "2026-10-06"]);
  assert.equal(tiles.some((d) => d.meals.some((m) => m.inactive)), false);
});

test("mealTiles: a dish served unknown to the copy of the library is left out (D15)", () => {
  const tiles = mealTiles(["2026-10-06"], [served("s1", "2026-10-06", "dinner", "unknown", 4, 1), served("s2", "2026-10-06", "dinner", "c", 4, 2)], [chili], "2026-10-05");
  assert.deepEqual(tiles[0]?.meals[1]?.dishes.map((d) => d.served.id), ["s2"]);
});

test("mealTiles: the days out of the bounds are inactive (review of PR #87)", () => {
  const tiles = mealTiles(dayRange("2099-12-28"), [], [], "2026-10-05");
  assert.deepEqual(
    tiles.map((d) => d.meals.every((m) => m.inactive)),
    [false, false, false, false, true, true, true],
  );
});

test("totalServings and servingsLabel", () => {
  assert.equal(totalServings([]), 0);
  assert.equal(totalServings([{ served: served("s1", "2026-10-06", "dinner", "c", 3, 1), dish: chili }, { served: served("s2", "2026-10-06", "dinner", "v", 1, 2), dish: vege }]), 4);
  assert.equal(servingsLabel(1), "1 part");
  assert.equal(servingsLabel(7), "7 parts");
});
