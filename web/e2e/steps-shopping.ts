// Steps of the scenarios @ui of the shopping lists (liste-courses.feature), against the real
// server (ADR 0028; plan courses, D16). The lists and the dishes are written through the API,
// as a member would with another device; what is proved is what the browser shows.
import { type Page, expect } from "@playwright/test";
import { unitCode } from "./steps-dishes.ts";
import { apiDay, mealOf } from "./steps-planning.ts";
import { World, step } from "./steps.ts";

interface ApiList {
  id: string;
  name: string;
  main: boolean;
  items: { id: string; ingredient: string }[];
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
  // In etendu, the list picker is beside the list (plan grands-ecrans, D5).
  if ((await world.page.locator("html").getAttribute("data-format")) === "etendu") {
    await expect(world.page.getByRole("heading", { name: "Listes", level: 2 })).toBeVisible();
    return;
  }
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

// C6, C11
step(/"([^"]*)" est coché dans la liste "([^"]*)"/, async (world, ingredient, name) => {
  const list = (await lists(world)).find((l) => l.name === name);
  const item = list?.items.find((i) => i.ingredient === ingredient);
  if (list === undefined || item === undefined) {
    throw new Error(`no item "${ingredient}" in the list "${name}"`);
  }
  const tribe = await world.tribe();
  await tribe.apiOk("je", "PUT", `shopping-lists/${list.id}/items/${item.id}/checked`, { checked: true });
});

step(/je demande à terminer les courses/, async (world) => {
  await world.page.getByRole("button", { name: "Courses faites" }).click();
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

// C8
step(/la liste "([^"]*)"/, async (world, name) => {
  await ensureList(world, name ?? "");
});

/** Sets the servings of a dish served, through the API, as a member would with another device. */
async function setServings(world: World, who: string, name: string, day: string, mealtime: string, servings: string): Promise<void> {
  const meal = mealOf(day, mealtime);
  const tribe = await world.tribe();
  const dishes = (await tribe.apiOk(who, "GET", "dishes")) as { items: { id: string; name: string }[] };
  const dish = dishes.items.find((d) => d.name === name);
  const plan = (await tribe.apiOk(who, "GET", "meal-plan")) as { items: { id: string; date: string; mealtime: string; dish: string }[] };
  const served = plan.items.find((s) => s.date === apiDay(meal.date) && s.mealtime === meal.mealtime.api && s.dish === dish?.id);
  if (served === undefined) {
    throw new Error(`no "${name}" served ${day} ${mealtime}`);
  }
  await tribe.apiOk(who, "PUT", `meal-plan/${served.id}/servings`, { servings: Number(servings) });
}

step(/un membre règle à (\d+) parts le "([^"]*)" du (.+?) (à midi|au soir)/, async (world, servings, name, day, mealtime) => {
  await setServings(world, "Bruno", name ?? "", day ?? "", mealtime ?? "", servings ?? "");
});

step(/le "([^"]*)" du (.+?) (à midi|au soir) réglé à (\d+) parts/, async (world, name, day, mealtime, servings) => {
  await setServings(world, "je", name ?? "", day ?? "", mealtime ?? "", servings ?? "");
});

/**
 * The name of the list shown: its title, or in selection mode the hint under the header,
 * « Choisissez les articles à sortir de « Courses ». … » (plan courses-suite, D14).
 */
export async function shownListName(page: Page): Promise<string | undefined> {
  const main = page.getByRole("main");
  const title = main.getByRole("heading", { level: 1 }).locator(".shopping-title-name");
  if ((await title.count()) > 0) {
    return (await title.textContent()) ?? undefined;
  }
  const hint = main.locator(".selection-hint");
  return (await hint.count()) > 0 ? /« (.*) »/.exec((await hint.textContent()) ?? "")?.[1] : undefined;
}

/** Shows a list in the browser, unless it already is, a selection kept: the main list at the path of the tab. */
async function showList(world: World, name: string): Promise<void> {
  if (world.pageOwner !== undefined && (await shownListName(world.page)) === name) {
    return;
  }
  const title = world.page.getByRole("heading", { level: 1 }).locator(".shopping-title-name");
  const list = (await lists(world)).find((l) => l.name === name);
  if (list === undefined) {
    throw new Error(`no list "${name}"`);
  }
  await world.open("je", list.main ? "courses" : `courses/${list.id}`);
  await expect(title).toHaveText(name);
}

const staleText = "Des repas ont changé depuis l’ajout de leurs courses à cette liste.";

step(/un bandeau en haut de la liste "([^"]*)" indique que des repas ont changé/, async (world, name) => {
  await showList(world, name ?? "");
  await expect(world.page.locator(".banner-stale")).toContainText(staleText);
});

step(/le bandeau propose l'action "([^"]*)"/, async (world, action) => {
  await expect(world.page.locator(".banner-stale").getByRole("button", { name: action ?? "" })).toBeEnabled();
});

step(/la liste "([^"]*)" contient (?:toujours )?(\S+) (\S+) de "([^"]*)"/, async (world, name, quantity, unit, ingredient) => {
  await showList(world, name ?? "");
  const item = world.page.locator(".item").filter({ has: world.page.locator(".item-name", { hasText: new RegExp(`^${ingredient ?? ""}$`) }) });
  await expect(item.locator(".item-quantity")).toHaveText(`${quantity ?? ""} ${unit ?? ""}`);
});

step(/je touche "Recalculer"/, async (world) => {
  // The scenario names no list: the main one, that of the tab.
  const main = (await lists(world)).find((l) => l.main);
  await showList(world, main?.name ?? "");
  await world.page.getByRole("button", { name: "Recalculer" }).click();
});

step(/le bandeau n'est plus affiché/, async (world) => {
  await expect(world.page.getByText(staleText)).toHaveCount(0);
});

step(/un message m'indique que la liste a été recalculée/, async (world) => {
  await expect(world.page.getByRole("status").filter({ hasText: "Liste recalculée." })).toBeVisible();
});

step(/la liste "([^"]*)" (ne )?porte (?:pas )?la mention "([^"]*)"/, async (world, name, ne, mention) => {
  const card = world.page.locator(".list-card").filter({ has: world.page.locator(".list-card-name", { hasText: new RegExp(`^${name ?? ""}$`) }) });
  await expect(card).toBeVisible();
  if (ne === "") {
    await expect(card).toContainText(mention ?? "");
  } else {
    await expect(card).not.toContainText(mention ?? "");
  }
});

// C9, C10

step(/la liste "([^"]*)" contenant les articles ajoutés "([^"]*)" et "([^"]*)"/, async (world, name, a, b) => {
  const list = await ensureList(world, name ?? "");
  const tribe = await world.tribe();
  for (const ingredient of [a, b]) {
    await tribe.apiOk("je", "POST", `shopping-lists/${list.id}/items`, { ingredient });
  }
});

/** The list shown in selection mode, entered unless it already is. */
async function selecting(world: World, name: string): Promise<void> {
  await showList(world, name);
  const start = world.page.getByRole("button", { name: "Sélectionner", exact: true });
  if ((await start.count()) > 0) {
    await start.click();
  }
  await expect(world.page.getByRole("main").locator(".selection-hint")).toBeVisible();
}

function pickButton(page: Page, ingredient: string) {
  return page.getByRole("button", { name: new RegExp(`^Sélectionner ${ingredient}(,|$)`) });
}

function selectionActions(page: Page) {
  return page.getByRole("group", { name: "Actions de la sélection" });
}

step(/je sélectionne "([^"]*)" dans la liste "([^"]*)"/, async (world, ingredient, name) => {
  await selecting(world, name ?? "");
  await pickButton(world.page, ingredient ?? "").click();
  await expect(pickButton(world.page, ingredient ?? "")).toHaveAttribute("aria-pressed", "true");
});

step(/"([^"]*)" est toujours sélectionné dans la liste "([^"]*)"/, async (world, ingredient, name) => {
  await expect.poll(() => shownListName(world.page)).toBe(name);
  await expect(pickButton(world.page, ingredient ?? "")).toHaveAttribute("aria-pressed", "true");
});

step(/aucun article n'est sélectionné/, async (world) => {
  // The scenario names no list: the main one, that of the tab.
  const main = (await lists(world)).find((l) => l.main);
  await selecting(world, main?.name ?? "");
  await expect(world.page.getByRole("heading", { level: 1 })).toHaveText("Aucun article");
});

step(/les actions de déplacement et de création à partir de la sélection sont indisponibles/, async (world) => {
  await expect(selectionActions(world.page).getByRole("button", { name: "Déplacer vers…" })).toBeDisabled();
  await expect(selectionActions(world.page).getByRole("button", { name: "Nouvelle liste" })).toBeDisabled();
});

step(/les actions de déplacement et de création à partir de la sélection s'affichent sous la liste "([^"]*)"/, async (world, name) => {
  const main = world.page.getByRole("main");
  await expect.poll(() => shownListName(world.page)).toBe(name);
  const actions = main.getByRole("group", { name: "Actions de la sélection" });
  await expect(actions.getByRole("button", { name: "Déplacer vers…" })).toBeEnabled();
  await expect(actions.getByRole("button", { name: "Nouvelle liste" })).toBeEnabled();
  // At the bottom of the pane of the list, under its items.
  const bar = await actions.boundingBox();
  const pane = await main.boundingBox();
  const last = await main.locator(".item").last().boundingBox();
  const viewport = world.page.viewportSize();
  if (bar === null || pane === null || last === null || viewport === null) {
    throw new Error("the actions of the selection are not laid out");
  }
  expect(bar.y).toBeGreaterThanOrEqual(last.y + last.height);
  expect(Math.abs(bar.y + bar.height - viewport.height)).toBeLessThanOrEqual(1);
  expect(bar.x).toBeGreaterThanOrEqual(pane.x - 1);
  expect(bar.x + bar.width).toBeLessThanOrEqual(pane.x + pane.width + 1);
});

/** Waits for the green banner after a move, and keeps its text for the outcome (the next steps open other lists). */
async function movedBanner(world: World): Promise<void> {
  const banner = world.page.locator(".banner-done").filter({ has: world.page.locator(".banner-link") });
  await expect(banner).toBeVisible();
  world.memory.set("moved", ((await banner.textContent()) ?? "").replace(/\s+/g, " ").trim());
}

step(/je déplace la sélection vers la liste "([^"]*)"/, async (world, target) => {
  await selectionActions(world.page).getByRole("button", { name: "Déplacer vers…" }).click();
  await world.page
    .getByRole("dialog")
    .locator(".sheet-target")
    .filter({ has: world.page.locator(".sheet-target-name").getByText(target ?? "", { exact: true }) })
    .click();
  await movedBanner(world);
});

step(/je crée la liste "([^"]*)" à partir de la sélection/, async (world, name) => {
  await selectionActions(world.page).getByRole("button", { name: "Nouvelle liste" }).click();
  const sheet = world.page.getByRole("dialog", { name: "Nouvelle liste" });
  await sheet.getByRole("textbox", { name: "Nom de la liste" }).fill(name ?? "");
  await sheet.getByRole("button", { name: "Créer la liste" }).click();
  await movedBanner(world);
});

step(/un message m'indique que (\d+) articles? (?:a|ont) été déplacés? vers la liste "([^"]*)"/, async (world, n, target) => {
  const moved = Number(n);
  expect(world.memory.get("moved")).toBe(`${moved} ${moved > 1 ? "articles déplacés" : "article déplacé"} vers « ${target ?? ""} ».`);
});

/** The lines of an ingredient in the list shown. */
function itemLines(page: Page, ingredient: string) {
  return page.locator(".item").filter({ has: page.locator(".item-name", { hasText: new RegExp(`^${ingredient}$`) }) });
}

step(/la liste "([^"]*)" contient (?:toujours )?l'article ajouté "([^"]*)"/, async (world, name, ingredient) => {
  await showList(world, name ?? "");
  await expect(itemLines(world.page, ingredient ?? "").locator(".pill")).toHaveText("ajouté");
});

step(/la liste "([^"]*)" ne contient plus "([^"]*)"/, async (world, name, ingredient) => {
  await showList(world, name ?? "");
  await expect(world.page.locator(".items")).toBeVisible();
  await expect(itemLines(world.page, ingredient ?? "")).toHaveCount(0);
});

step(/la liste principale de la tribu est toujours la liste "([^"]*)"/, async (world, name) => {
  // The tab Courses opens the main list (C1).
  await world.open("je", "courses");
  const title = world.page.getByRole("heading", { level: 1 });
  await expect(title.locator(".shopping-title-name")).toHaveText(name ?? "");
  await expect(title.locator(".pill--main")).toHaveText("principale");
});
