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
