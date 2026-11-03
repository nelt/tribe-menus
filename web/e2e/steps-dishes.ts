// Steps of the scenarios @ui of the dish library (bibliotheque-plats.feature, tribu.feature),
// against the real server (ADR 0028). The dishes are written through the API, as a member
// would with another device; what is proved is what the browser shows.
import { expect } from "@playwright/test";
import { World, step } from "./steps.ts";

interface ApiDish {
  id: string;
  name: string;
  servings: number;
  revision: number;
  ingredients: { name: string; quantity?: string; unit?: string }[];
}

async function createDish(world: World, who: string, name: string, ingredients: string[], servings?: number): Promise<void> {
  const tribe = await world.tribe();
  await tribe.apiOk(who, "POST", "dishes", { name, ...(servings === undefined ? {} : { servings }), ingredients: ingredients.map((i) => ({ name: i })) });
}

async function findDish(world: World, who: string, name: string): Promise<ApiDish> {
  const tribe = await world.tribe();
  const page = (await tribe.apiOk(who, "GET", "dishes")) as { items: ApiDish[]; next?: string };
  const dish = page.items.find((d) => d.name === name);
  if (dish === undefined || page.next !== undefined) {
    throw new Error(`no dish "${name}" on the only page of the library`);
  }
  return dish;
}

/** The screen Plats of the browser, opened by "je" if nobody did. */
async function libraryScreen(world: World): Promise<void> {
  if (world.pageOwner === undefined) {
    await world.open("je", "plats");
  }
  await expect(world.page.getByRole("heading", { name: "Plats" })).toBeVisible();
}

// The members of a tribe of the scenario (Contexte of tribu.feature): every tribe of a
// scenario has them.
step(/Alice et Bruno sont membres de la même tribu/, async (world) => {
  await world.tribe();
});

// P3
step(/la bibliothèque contient les plats :/, async (world) => {
  for (const [name, ingredients] of world.table.slice(1)) {
    if (name === undefined) {
      continue;
    }
    const names = (ingredients ?? "").split(",").map((i) => i.trim()).filter((i) => i !== "");
    await createDish(world, "je", name, names);
  }
});

step(/je recherche "([^"]*)"/, async (world, query) => {
  await libraryScreen(world);
  await world.page.getByRole("searchbox", { name: "Rechercher un plat ou un ingrédient" }).fill(query ?? "");
});

step(/les résultats sont "([^"]*)"/, async (world, expected) => {
  const names = (expected ?? "").split(",").map((n) => n.trim()).filter((n) => n !== "");
  const results = world.page.locator(".dish-list .dish-name");
  await expect(results).toHaveText(names);
  if (names.length === 0) {
    await expect(world.page.getByText("Aucun plat ne correspond")).toBeVisible();
  }
});

// T1, D29
step(/le plat "([^"]*)" défini pour (\d+) parts/, async (world, name, servings) => {
  await createDish(world, "Alice", name ?? "", [], Number(servings));
});

step(/Alice a affiché la bibliothèque de plats sur son téléphone/, async (world) => {
  await world.open("Alice", "plats");
  await expect(world.page.getByRole("heading", { name: "Plats" })).toBeVisible();
  await expect(world.page.locator(".dish-name").first()).toBeVisible();
  // Kept by the page as long as it is not reloaded.
  await world.page.evaluate(() => ((window as unknown as { notReloaded: boolean }).notReloaded = true));
});

step(/Bruno modifie le plat "([^"]*)" pour (\d+) parts/, async (world, name, servings) => {
  const dish = await findDish(world, "Bruno", name ?? "");
  const tribe = await world.tribe();
  await tribe.apiOk("Bruno", "PUT", `dishes/${dish.id}`, { ...dish, servings: Number(servings) });
});

// The application goes to the background, then comes back, as when the phone is unlocked:
// the page is not reloaded.
step(/Alice revient sur l'application, sans l'avoir rechargée/, async (world) => {
  await world.page.evaluate(() => {
    for (const state of ["hidden", "visible"]) {
      Object.defineProperty(document, "visibilityState", { configurable: true, get: () => state });
      Object.defineProperty(document, "hidden", { configurable: true, get: () => state === "hidden" });
      document.dispatchEvent(new Event("visibilitychange"));
    }
  });
});

step(/Alice voit le plat "([^"]*)" défini pour (\d+) parts/, async (world, name, servings) => {
  await expect(world.page.getByRole("link", { name: new RegExp(`^${name ?? ""}`) })).toContainText(`pour ${servings ?? ""}`);
  expect(await world.page.evaluate(() => (window as unknown as { notReloaded?: boolean }).notReloaded)).toBe(true);
});
