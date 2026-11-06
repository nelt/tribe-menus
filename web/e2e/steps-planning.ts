// Steps of the scenarios @ui of the meal plan (planning-repas.feature), against the real
// server (ADR 0028), the clock of the browser set by Playwright (plan planning, D4). The
// dishes are written through the API; what is proved is what the browser shows. The expected
// labels are written here from the dates of the scenario, not by the module under test.
import { expect } from "@playwright/test";
import { World, step } from "./steps.ts";

const weekdays = ["dimanche", "lundi", "mardi", "mercredi", "jeudi", "vendredi", "samedi"];
const months = ["janvier", "février", "mars", "avril", "mai", "juin", "juillet", "août", "septembre", "octobre", "novembre", "décembre"];
const shortMonths = ["janv.", "févr.", "mars", "avr.", "mai", "juin", "juil.", "août", "sept.", "oct.", "nov.", "déc."];

/** A day of a scenario: « lundi 5 octobre 2026 », the month and year possibly left out. */
interface Written {
  weekday: string;
  date: number;
  month?: number;
  year?: number;
}

function parseWritten(text: string): Written {
  const m = /^(\S+) (1er|\d{1,2})(?: (\S+))?(?: (\d{4}))?$/.exec(text.trim());
  if (m === null) {
    throw new Error(`unreadable day "${text}"`);
  }
  const [, weekday = "", date = "", month, year] = m;
  const written: Written = { weekday, date: date === "1er" ? 1 : Number(date) };
  if (month !== undefined) {
    const index = months.indexOf(month);
    if (index < 0) {
      throw new Error(`unknown month "${month}"`);
    }
    written.month = index;
  }
  if (year !== undefined) {
    written.year = Number(year);
  }
  return written;
}

/** The local date of a day written, checking its day of the week. */
function dateOf(written: Written, fallback?: Date): Date {
  const month = written.month ?? fallback?.getMonth();
  const year = written.year ?? fallback?.getFullYear();
  if (month === undefined || year === undefined) {
    throw new Error("a day without month or year");
  }
  const date = new Date(year, month, written.date, 12);
  if (weekdays[date.getDay()] !== written.weekday) {
    throw new Error(`${written.date}/${month + 1}/${year} is not a ${written.weekday}`);
  }
  return date;
}

/** The label of a day on the planning: « Lundi 5 oct. », « Jeudi 1er oct. ». */
function dayLabel(date: Date): string {
  const weekday = weekdays[date.getDay()] ?? "";
  const n = date.getDate() === 1 ? "1er" : String(date.getDate());
  return `${weekday.charAt(0).toUpperCase()}${weekday.slice(1)} ${n} ${shortMonths[date.getMonth()] ?? ""}`;
}

async function createDish(world: World, name: string): Promise<void> {
  const tribe = await world.tribe();
  await tribe.apiOk("je", "POST", "dishes", { name, ingredients: [] });
}

/** The screen Planning, its days shown once both copies are read (D15). */
async function planningScreen(world: World): Promise<void> {
  await expect(world.page.getByRole("heading", { name: "Menus", level: 1 })).toBeVisible();
  await expect(world.page.locator(".day")).toHaveCount(7);
}

// Contexte: the date of the device (D4), a little after ten in the morning.
step(/(?:que )?nous sommes le (.+)/, async (world, text) => {
  const date = dateOf(parseWritten(text ?? ""));
  date.setHours(10, 0, 0, 0);
  await world.page.clock.setFixedTime(date);
  world.memory.set("today", date);
});

step(/la bibliothèque contient les plats "([^"]*)" et "([^"]*)"/, async (world, a, b) => {
  await createDish(world, a ?? "");
  await createDish(world, b ?? "");
});

// R1
step(/j'ouvre le planning/, async (world) => {
  await world.open("je", "");
  await planningScreen(world);
});

step(/je vois les jours du (.+) au (.+)/, async (world, from, to) => {
  const last = dateOf(parseWritten(to ?? ""));
  const first = dateOf(parseWritten(from ?? ""), last);
  const expected: string[] = [];
  for (let d = new Date(first); d <= last; d.setDate(d.getDate() + 1)) {
    expected.push(dayLabel(d));
  }
  await expect(world.page.locator(".day-name")).toHaveText(expected);
  // « Aujourd'hui » on today only, when the range shows it.
  const today = world.memory.get("today") as Date | undefined;
  const todayLabel = today === undefined ? undefined : dayLabel(today);
  await expect(world.page.locator(".day-today")).toHaveCount(todayLabel !== undefined && expected.includes(todayLabel) ? 1 : 0);
});

step(/chaque jour comporte deux repas : midi et soir/, async (world) => {
  const days = world.page.locator(".day");
  await expect(days).toHaveCount(7);
  for (const day of await days.all()) {
    await expect(day.locator(".meal-tile-label")).toHaveText(["MIDI", "SOIR"]);
  }
});

// R5
step(/je passe à la semaine (suivante|précédente)/, async (world, which) => {
  await world.page.getByRole("button", { name: `Semaine ${which ?? ""}` }).click();
  await planningScreen(world);
});

const mealtimes: Record<string, { api: string; path: string; label: string }> = {
  "à midi": { api: "lunch", path: "midi", label: "midi" },
  "au soir": { api: "dinner", path: "soir", label: "soir" },
};

/** A day of the API: YYYY-MM-DD, from the local date. */
function apiDay(date: Date): string {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
}

/** The meal of a scenario: « mardi 6 octobre 2026 au soir ». */
function mealOf(day: string, mealtime: string): { date: Date; mealtime: { api: string; path: string; label: string } } {
  const time = mealtimes[mealtime];
  if (time === undefined) {
    throw new Error(`unknown mealtime "${mealtime}"`);
  }
  return { date: dateOf(parseWritten(day)), mealtime: time };
}

async function dishId(world: World, name: string): Promise<string> {
  const tribe = await world.tribe();
  const page = (await tribe.apiOk("je", "GET", "dishes")) as { items: { id: string; name: string }[] };
  const dish = page.items.find((d) => d.name === name);
  if (dish === undefined) {
    throw new Error(`no dish "${name}" in the library`);
  }
  return dish.id;
}

// R2
step(/le plat "([^"]*)" servi pour (\d+) parts le (.+?) (à midi|au soir)/, async (world, name, servings, day, mealtime) => {
  const meal = mealOf(day ?? "", mealtime ?? "");
  const tribe = await world.tribe();
  await tribe.apiOk("je", "POST", "meal-plan", { date: apiDay(meal.date), mealtime: meal.mealtime.api, dish: await dishId(world, name ?? ""), servings: Number(servings) });
});

step(/je choisis un plat à ajouter au repas du (.+?) (à midi|au soir)/, async (world, day, mealtime) => {
  const meal = mealOf(day ?? "", mealtime ?? "");
  // Through the planning, from today if its range holds the meal, else from the meal's day.
  const today = world.memory.get("today") as Date | undefined;
  const offset = today === undefined ? -1 : Math.round((meal.date.getTime() - today.getTime()) / 86_400_000);
  await world.open("je", offset >= 0 && offset < 7 ? "" : `?du=${apiDay(meal.date)}`);
  await planningScreen(world);
  await world.page.getByRole("link", { name: new RegExp(`^${dayLabel(meal.date)}, ${meal.mealtime.label} : `) }).click();
  await expect(world.page.getByRole("heading", { name: "Ajouter un plat", level: 2 })).toBeVisible();
});

step(/"([^"]*)" (ne )?m'est (pas )?proposé/, async (world, name, ne, pas) => {
  const suggestions = world.page.getByRole("list", { name: "Plats proposés" }).getByRole("button", { name: `Ajouter ${name ?? ""}`, exact: true });
  await expect(world.page.getByRole("heading", { name: "Plats servis", level: 2 })).toBeVisible();
  if ((ne ?? "") !== "" && (pas ?? "") !== "") {
    await expect(suggestions).toHaveCount(0);
  } else {
    await expect(suggestions).toHaveCount(1);
  }
});
