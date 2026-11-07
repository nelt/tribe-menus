// Steps of the scenarios @ui of the shopping lists (liste-courses.feature), against the real
// server (ADR 0028; plan courses, D16). The lists and the dishes are written through the API,
// as a member would with another device; what is proved is what the browser shows.
import { expect } from "@playwright/test";
import { unitCode } from "./steps-dishes.ts";
import { World, step } from "./steps.ts";

interface ApiList {
  id: string;
  name: string;
  main: boolean;
}

async function lists(world: World): Promise<ApiList[]> {
  const tribe = await world.tribe();
  const page = (await tribe.apiOk("je", "GET", "shopping-lists")) as { items: ApiList[]; next?: string };
  if (page.next !== undefined) {
    throw new Error("the lists of a scenario hold on one page");
  }
  return page.items;
}

async function createList(world: World, name: string): Promise<ApiList> {
  const tribe = await world.tribe();
  const created = (await tribe.apiOk("je", "POST", "shopping-lists", { name })) as { list: ApiList };
  return created.list;
}

/** The list of a name, created if the tribe has none. */
async function ensureList(world: World, name: string): Promise<ApiList> {
  return (await lists(world)).find((l) => l.name === name) ?? (await createList(world, name));
}

// Contexte of liste-courses.feature; C2.
step(/la liste principale de la tribu est la liste "([^"]*)"/, async (world, name) => {
  const list = await ensureList(world, name ?? "");
  const tribe = await world.tribe();
  await tribe.apiOk("je", "PUT", "shopping-lists/main", { list: list.id });
});

step(/la bibliothèque contient le plat "([^"]*)" défini pour (\d+) parts :/, async (world, name, servings) => {
  const ingredients = world.table.slice(1).map(([ingredient = "", quantity = "", unit = ""]) => (quantity === "" ? { name: ingredient } : { name: ingredient, quantity, unit: unitCode(unit) }));
  const tribe = await world.tribe();
  await tribe.apiOk("je", "POST", "dishes", { name, servings: Number(servings), ingredients });
});

// C1
step(/les listes "([^"]*)" et "([^"]*)"/, async (world, a, b) => {
  await ensureList(world, a ?? "");
  await ensureList(world, b ?? "");
});

step(/j'ouvre l'onglet Courses/, async (world) => {
  await world.open("je", "");
  await expect(world.page.getByRole("heading", { name: "Menus", level: 1 })).toBeVisible();
  await world.page.getByRole("navigation", { name: "Onglets" }).getByRole("link", { name: "Courses" }).click();
});

step(/la liste "([^"]*)" est affichée/, async (world, name) => {
  const title = world.page.getByRole("heading", { level: 1 });
  await expect(title.locator(".shopping-title-name")).toHaveText(name ?? "");
});

// C2
step(/(?:que )?j'ai créé la liste "([^"]*)" puis la liste "([^"]*)"/, async (world, a, b) => {
  await createList(world, a ?? "");
  await createList(world, b ?? "");
});

step(/j'ouvre le choix des listes/, async (world) => {
  await world.open("je", "courses");
  await world.page.getByRole("link", { name: /Changer de liste$/ }).click();
  await expect(world.page.getByRole("heading", { name: "Listes", level: 1 })).toBeVisible();
});

step(/les listes sont présentées dans l'ordre : (.+)/, async (world, names) => {
  const expected = [...(names ?? "").matchAll(/"([^"]*)"/g)].map((m) => m[1] ?? "");
  await expect(world.page.locator(".list-card-name")).toHaveText(expected);
});

const months = ["janvier", "février", "mars", "avril", "mai", "juin", "juillet", "août", "septembre", "octobre", "novembre", "décembre"];

function pad(n: number): string {
  return String(n).padStart(2, "0");
}

/** A period of a scenario: « du 6 au 12 octobre 2026 », « du 6 octobre au 5 novembre 2026 »; days of the API. */
function periodOf(text: string): { from: string; to: string } {
  const m = /^du (1er|\d{1,2})(?: (\S+))?(?: (\d{4}))? au (1er|\d{1,2}) (\S+) (\d{4})$/.exec(text.trim());
  if (m === null) {
    throw new Error(`unreadable period "${text}"`);
  }
  const [, fromDay = "", fromMonth, fromYear, toDay = "", toMonth = "", toYear = ""] = m;
  const day = (d: string, month: string, year: string) => {
    const index = months.indexOf(month);
    if (index < 0) {
      throw new Error(`unknown month "${month}"`);
    }
    return `${year}-${pad(index + 1)}-${pad(d === "1er" ? 1 : Number(d))}`;
  };
  return { from: day(fromDay, fromMonth ?? toMonth, fromYear ?? toYear), to: day(toDay, toMonth, toYear) };
}

/** The screen of the meal shopping, once the copies are read. */
async function mealShoppingScreen(world: World): Promise<void> {
  await expect(world.page.getByRole("heading", { name: "Courses des repas", level: 1 })).toBeVisible();
  await expect(world.page.locator(".calendar")).toBeVisible();
}

// C3, C5
step(/la liste "([^"]*)" contenant l'article ajouté "([^"]*)"/, async (world, name, ingredient) => {
  const list = await ensureList(world, name ?? "");
  const tribe = await world.tribe();
  await tribe.apiOk("je", "POST", `shopping-lists/${list.id}/items`, { ingredient });
});

// C4
step(/j'ajoute les courses des repas sans modifier la période ni la liste proposées/, async (world) => {
  await world.open("je", "courses");
  await world.page.getByRole("link", { name: "Ajouter les courses des repas" }).click();
  await mealShoppingScreen(world);
  const request = world.page.waitForRequest((r) => r.method() === "POST" && r.url().endsWith("/meal-shopping"));
  await world.page.getByRole("button", { name: /^Ajouter à « / }).click();
  const sent = await request;
  world.memory.set("meal-shopping", { url: sent.url(), body: sent.postDataJSON() as unknown });
});

step(/la période retenue va (du .+) inclus/, async (world, text) => {
  const sent = world.memory.get("meal-shopping") as { body: unknown } | undefined;
  expect(sent?.body).toEqual(periodOf(text ?? ""));
});

step(/les courses sont ajoutées à la liste "([^"]*)"/, async (world, name) => {
  const list = (await lists(world)).find((l) => l.name === name);
  const sent = world.memory.get("meal-shopping") as { url: string } | undefined;
  expect(sent?.url).toContain(`/shopping-lists/${list?.id ?? "?"}/meal-shopping`);
  // Back on the list, which tells what was added.
  await expect(world.page.getByRole("heading", { level: 1 }).locator(".shopping-title-name")).toHaveText(name ?? "");
  await expect(world.page.locator(".banner-done")).toBeVisible();
});

/** Touches a day of the calendar for the field given, going through the months. */
async function pickDay(world: World, field: "Du" | "Au", written: string): Promise<void> {
  const m = /^(\S+) (1er|\d{1,2}) (\S+) (\d{4})$/.exec(written.trim());
  if (m === null) {
    throw new Error(`unreadable day "${written}"`);
  }
  const [, weekday = "", day = "", month = "", year = ""] = m;
  const page = world.page;
  await page.getByRole("button", { name: new RegExp(`^${field} `) }).click();
  const title = `${month.charAt(0).toUpperCase()}${month.slice(1)} ${year}`;
  for (let i = 0; i < 24 && (await page.locator(".calendar-title").textContent()) !== title; i++) {
    const shown = (await page.locator(".calendar-title").textContent()) ?? "";
    const [shownMonth = "", shownYear = ""] = shown.toLowerCase().split(" ");
    const later = Number(year) * 12 + months.indexOf(month) > Number(shownYear) * 12 + months.indexOf(shownMonth);
    await page.getByRole("button", { name: later ? "Mois suivant" : "Mois précédent" }).click();
  }
  await expect(page.locator(".calendar-title")).toHaveText(title);
  await page.getByRole("button", { name: new RegExp(`^${weekday} ${day} ${month}(,|$)`) }).click();
}

step(/je choisis une date de début au (.+) et une date de fin au (.+)/, async (world, from, to) => {
  await world.open("je", "courses/repas");
  await mealShoppingScreen(world);
  await pickDay(world, "Du", from ?? "");
  await pickDay(world, "Au", to ?? "");
});

step(/le bouton d'ajout des courses est inactif/, async (world) => {
  await expect(world.page.getByRole("button", { name: /^Ajouter à « / })).toBeDisabled();
});

step(/un message m'indique que la date de fin doit suivre la date de début/, async (world) => {
  await expect(world.page.getByRole("alert")).toHaveText("La date de fin doit suivre la date de début.");
});

// C5
step(/j'ai ajouté les courses des repas (du .+) à la liste "([^"]*)"/, async (world, period, name) => {
  const list = await ensureList(world, name ?? "");
  const tribe = await world.tribe();
  await tribe.apiOk("je", "POST", `shopping-lists/${list.id}/meal-shopping`, periodOf(period ?? ""));
});

step(/je touche la ligne "([^"]*)"/, async (world, ingredient) => {
  await world.open("je", "courses");
  await world.page.getByRole("button", { name: new RegExp(`^${ingredient ?? ""}, .*Afficher les repas$`) }).click();
  world.memory.set("line", ingredient);
});

step(/elle se déplie et montre :/, async (world) => {
  const ingredient = (world.memory.get("line") as string | undefined) ?? "";
  const meals = world.page.getByRole("list", { name: `Repas de ${ingredient}` });
  // « mardi 6 oct., soir » as the line shows it: « mar. 6 oct. · soir ».
  const expected = world.table.slice(1).map(([meal = "", dish = "", servings = "", quantity = ""]) => {
    const [day = "", mealtime = ""] = meal.split(", ");
    const shortDay = day.replace(/^(\S{3})\S*/, "$1.");
    return `${dish} · ${servings} parts ${shortDay} · ${mealtime} ${quantity}`;
  });
  await expect(meals.locator(".item-source-text, .item-source-quantity")).toHaveCount(expected.length * 2);
  const lines = await meals.getByRole("listitem").evaluateAll((all) =>
    all.map((e) => [...e.querySelectorAll(".item-source-dish, .item-source-meal, .item-source-quantity")].map((p) => (p.textContent ?? "").replace(/\s+/g, " ").trim()).join(" ")),
  );
  expect(lines).toEqual(expected);
});

step(/je la replie en la touchant de nouveau/, async (world) => {
  const ingredient = (world.memory.get("line") as string | undefined) ?? "";
  await world.page.getByRole("button", { name: new RegExp(`^${ingredient}, .*Masquer les repas$`) }).click();
  await expect(world.page.getByRole("list", { name: `Repas de ${ingredient}` })).toHaveCount(0);
});

step(/la ligne "([^"]*)" est marquée comme ajoutée à la main/, async (world, ingredient) => {
  await world.open("je", "courses");
  const item = world.page.locator(".item").filter({ has: world.page.locator(".item-name", { hasText: ingredient ?? "" }) });
  await expect(item.locator(".pill")).toHaveText("ajouté");
  world.memory.set("line", ingredient);
});

step(/elle ne propose pas de détail de repas/, async (world) => {
  const ingredient = (world.memory.get("line") as string | undefined) ?? "";
  const item = world.page.locator(".item").filter({ has: world.page.locator(".item-name", { hasText: ingredient }) });
  await expect(item.locator("[aria-expanded]")).toHaveCount(0);
  await expect(item.locator(".item-sources")).toHaveCount(0);
});
