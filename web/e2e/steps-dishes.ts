// Steps of the scenarios @ui of the dish library (bibliotheque-plats.feature, tribu.feature),
// against the real server (ADR 0028). The dishes are written through the API, as a member
// would with another device; what is proved is what the browser shows.
import { expect } from "@playwright/test";
import { units } from "../src/dishes.ts";
import { World, step } from "./steps.ts";

interface ApiDish {
  id: string;
  name: string;
  servings: number;
  revision: number;
  ingredients: { name: string; quantity?: string; unit?: string }[];
}

type Line = { name: string; quantity?: string; unit?: string };

async function createDish(world: World, who: string, name: string, ingredients: (string | Line)[], servings?: number): Promise<void> {
  const tribe = await world.tribe();
  const lines = ingredients.map((i) => (typeof i === "string" ? { name: i } : i));
  await tribe.apiOk(who, "POST", "dishes", { name, ...(servings === undefined ? {} : { servings }), ingredients: lines });
}

/** A dish of its own for each ingredient a scenario puts in the catalog through the API. */
async function addToCatalog(world: World, who: string, line: Line): Promise<void> {
  const n = ((world.memory.get("catalogDishes") as number | undefined) ?? 0) + 1;
  world.memory.set("catalogDishes", n);
  await createDish(world, who, `Plat ${n}`, [line]);
}

/** The code of a unit of the scenarios, by its label (« g », « pièce »). */
export function unitCode(label: string): string {
  const unit = units.find((u) => u.label === label || u.plural === label || u.name === label);
  if (unit === undefined) {
    throw new Error(`unknown unit "${label}"`);
  }
  return unit.code;
}

/** The screen of a new dish, opened by this person, its form ready. */
async function newDishScreen(world: World, who: string): Promise<void> {
  if (world.pageOwner === undefined) {
    await world.open(who, "plats/nouveau");
  } else {
    // Followed by the router, without reloading the application.
    await world.page.getByRole("link", { name: "Plats" }).click();
    await world.page.getByRole("link", { name: "Nouveau" }).click();
  }
  await expect(world.page.getByRole("heading", { name: "Nouveau plat" })).toBeVisible();
  // The size of the tribe is read: the parts are proposed (D16).
  await expect(world.page.getByRole("group", { name: "Pour combien de parts ?" })).toBeVisible();
}

function combobox(world: World) {
  return world.page.getByRole("combobox", { name: "Ingrédient", exact: true });
}

/** Types a text as the ingredient of a new line. */
async function typeIngredient(world: World, text: string): Promise<void> {
  await world.page.getByRole("button", { name: "Ajouter un ingrédient" }).click();
  await combobox(world).fill(text);
}

/** The option of an ingredient suggested, by its exact name. */
function suggestion(world: World, name: string) {
  return world.page.getByRole("listbox", { name: "Ingrédients proposés" }).getByRole("option").filter({ has: world.page.locator(".suggestion-name").getByText(name, { exact: true }) });
}

/** Kept by the page as long as it is not reloaded. */
async function markPage(world: World): Promise<void> {
  await world.page.evaluate(() => ((window as unknown as { notReloaded: boolean }).notReloaded = true));
}

async function notReloaded(world: World): Promise<void> {
  expect(await world.page.evaluate(() => (window as unknown as { notReloaded?: boolean }).notReloaded)).toBe(true);
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

// P4
step(/l'ingrédient "([^"]*)" existe dans le référentiel/, async (world, name) => {
  await addToCatalog(world, "je", { name: name ?? "" });
});

step(/l'ingrédient "([^"]*)" a été saisi pour la première fois en "([^"]*)"/, async (world, name, unit) => {
  await addToCatalog(world, "je", { name: name ?? "", quantity: "1", unit: unitCode(unit ?? "") });
});

step(/l'ingrédient "([^"]*)" n'existe pas dans le référentiel/, async (world, name) => {
  const tribe = await world.tribe();
  const page = (await tribe.apiOk("je", "GET", "ingredients")) as { items: { name: string }[] };
  expect(page.items.map((i) => i.name)).not.toContain(name);
});

step(/(je saisis|Bruno saisit) "([^"]*)" comme ingrédient d'un plat/, async (world, who, text) => {
  await newDishScreen(world, who === "je" ? "je" : "Bruno");
  await typeIngredient(world, text ?? "");
});

step(/l'ingrédient "([^"]*)" m'est proposé|"([^"]*)" fait partie des propositions/, async (world, mine, theirs) => {
  await expect(suggestion(world, mine || theirs || "")).toBeVisible();
});

// The default unit is shown with the suggestion, then taken by the line once a quantity is
// typed (docs/design/README.md: without quantity, no unit).
step(/l'unité "([^"]*)" est proposée/, async (world, unit) => {
  const option = suggestion(world, (await combobox(world).inputValue()).trim());
  await expect(option.locator(".suggestion-unit")).toHaveText(unit ?? "");
  const name = await option.locator(".suggestion-name").innerText();
  await option.click();
  await world.page.getByLabel(`Quantité de ${name}`, { exact: true }).fill("1");
  await expect(world.page.getByLabel(`Unité de ${name}`, { exact: true })).toHaveValue(unitCode(unit ?? ""));
  world.memory.set("line", name);
});

step(/je peux choisir une autre unité pour ce plat/, async (world) => {
  const name = world.memory.get("line") as string;
  const select = world.page.getByLabel(`Unité de ${name}`, { exact: true });
  await expect(select).toBeEnabled();
  await select.selectOption("kg");
  await expect(select).toHaveValue("kg");
  await expect(world.page.locator(".line-unit-label")).toHaveText(["kg"]);
});

step(/j'ai ajouté l'ingrédient "([^"]*)" au plat "([^"]*)"/, async (world, ingredient, dish) => {
  await newDishScreen(world, "je");
  await markPage(world);
  await world.page.getByLabel("Nom du plat").fill(dish ?? "");
  await typeIngredient(world, ingredient ?? "");
  await world.page.getByRole("option", { name: `Créer « ${ingredient ?? ""} »` }).click();
  await world.page.getByRole("button", { name: "Enregistrer le plat" }).click();
  await expect(world.page.getByRole("heading", { name: "Plats" })).toBeVisible();
  await expect(world.page.locator(".dish-name").getByText(dish ?? "", { exact: true })).toBeVisible();
});

step(/je saisis "([^"]*)" comme ingrédient d'un autre plat, sans avoir rechargé l'application/, async (world, text) => {
  await newDishScreen(world, "je");
  await typeIngredient(world, text ?? "");
  await notReloaded(world);
});

// T1
step(/Alice ajoute le nouvel ingrédient "([^"]*)" au plat "([^"]*)"/, async (world, ingredient, dish) => {
  await createDish(world, "Alice", dish ?? "", [ingredient ?? ""]);
});

// T3
step(/la taille de la tribu est (\d+)/, async (world, size) => {
  const tribe = await world.tribe();
  await tribe.apiOk("je", "PUT", "tribe/size", { size: Number(size) });
});

step(/je crée le plat "([^"]*)" sans modifier le nombre de parts proposé/, async (world, name) => {
  await newDishScreen(world, "je");
  await world.page.getByLabel("Nom du plat").fill(name ?? "");
  await world.page.getByRole("button", { name: "Enregistrer le plat" }).click();
  await expect(world.page.getByRole("heading", { name: "Plats" })).toBeVisible();
});

step(/le plat "([^"]*)" est (?:toujours )?défini pour (\d+) parts/, async (world, name, servings) => {
  const dish = await findDish(world, "je", name ?? "");
  expect(dish.servings).toBe(Number(servings));
});
