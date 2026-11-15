// The base journeys (ADR 0032, point 5; plan recette-auto, step 13), played at every acceptance
// run in each of the four projects: each tab opens; a dish created then found by the search; served
// at tomorrow's lunch; its shopping added to « Courses », an item checked, then taken out by
// « Courses faites »; each write read again after a reload. The data carry the time of the run
// and the project, and accumulate in the tribe recette-auto, except the items, taken out.
import { basename } from "node:path";
import type { Page } from "@playwright/test";
import { dayLabel } from "../e2e/steps-planning.ts";
import { recetteSettings } from "./config.ts";
import { expect, step, test } from "./test.ts";

const settings = recetteSettings();

/** Short names of the projects: an ingredient holds 40 characters at most. */
const projects: Record<string, string> = { "phone-chromium": "pc", "phone-webkit": "pw", "tablet-landscape-chromium": "tc", "computer-chromium": "cc" };

/** The names written by this run in this project: the time of the run, then the project. */
function names(project: string) {
  const run = `${basename(settings.outputDir)} ${projects[project] ?? project}`;
  return { dish: `Recette ${run}`, ingredient: `essai ${run}` };
}

function tabs(page: Page) {
  return page.getByRole("navigation", { name: "Onglets" });
}

/** The screen Planning, its seven days shown once the copies are read. */
async function planningShown(page: Page) {
  await expect(page.getByRole("heading", { name: "Menus", level: 1 })).toBeVisible();
  await expect(page.locator(".day")).toHaveCount(7);
}

/** The meal of tomorrow at lunch, on the device's date, opened from the planning. */
async function openTomorrowLunch(page: Page) {
  const now = await page.evaluate(() => Date.now());
  const tomorrow = new Date(now);
  tomorrow.setDate(tomorrow.getDate() + 1);
  await page.getByRole("link", { name: new RegExp(`^${dayLabel(tomorrow)}, midi : `) }).click();
  await expect(page.getByRole("heading", { name: "Plats servis", level: 2 })).toBeVisible();
}

/** The main list « Courses », its title shown. */
async function mainListShown(page: Page) {
  await expect(page.getByRole("heading", { level: 1 }).locator(".shopping-title-name")).toHaveText("Courses");
}

test("base journey: each tab, a dish, a meal, its shopping", async ({ page }, testInfo) => {
  const { dish, ingredient } = names(testInfo.project.name);

  await step(page, "Planning", async () => {
    await page.goto("./");
    await planningShown(page);
  });

  await step(page, "Plats", async () => {
    await tabs(page).getByRole("link", { name: "Plats" }).click();
    await expect(page.getByRole("heading", { name: "Plats", exact: true })).toBeVisible();
  });

  await step(page, "Courses", async () => {
    await tabs(page).getByRole("link", { name: "Courses" }).click();
    await mainListShown(page);
  });

  await step(page, "Dish created", async () => {
    await page.goto("plats/nouveau");
    await expect(page.getByRole("heading", { name: "Nouveau plat" })).toBeVisible();
    await page.getByLabel("Nom du plat").fill(dish);
    await page.getByRole("button", { name: "Ajouter un ingrédient" }).click();
    await page.getByRole("combobox", { name: "Ingrédient", exact: true }).fill(ingredient);
    await page.getByRole("option", { name: `Créer « ${ingredient} »` }).click();
    await page.getByRole("button", { name: "Enregistrer le plat" }).click();
    await expect(page.getByRole("heading", { name: "Plats", exact: true })).toBeVisible();
  });

  await step(page, "Dish found by the search, after a reload", async () => {
    await page.reload();
    await page.getByRole("searchbox", { name: "Rechercher un plat ou un ingrédient" }).fill(dish);
    await expect(page.locator(".dish-list .dish-name")).toHaveText([dish]);
  });

  await step(page, "Dish served tomorrow at lunch", async () => {
    await page.goto("./");
    await planningShown(page);
    await openTomorrowLunch(page);
    await page.getByRole("searchbox", { name: "Rechercher un plat", exact: true }).fill(dish);
    await page.getByRole("list", { name: "Plats proposés" }).getByRole("button", { name: `Ajouter ${dish}`, exact: true }).click();
    await expect(page.getByRole("button", { name: `Retirer « ${dish} » du repas` })).toBeVisible();
  });

  await step(page, "Meal read again, after a reload", async () => {
    await page.goto("./");
    await planningShown(page);
    await openTomorrowLunch(page);
    await expect(page.getByRole("button", { name: `Retirer « ${dish} » du repas` })).toBeVisible();
  });

  await step(page, "Meal shopping added to Courses", async () => {
    await page.goto("courses");
    await mainListShown(page);
    await page.getByRole("link", { name: "Ajouter les courses des repas" }).click();
    await expect(page.getByRole("heading", { name: "Courses des repas", level: 1 })).toBeVisible();
    await page.getByRole("button", { name: "Ajouter à « Courses »" }).click();
    await mainListShown(page);
    await expect(page.getByRole("checkbox", { name: `Cocher ${ingredient}` })).toBeVisible();
  });

  await step(page, "Item checked", async () => {
    const saved = page.waitForResponse((r) => r.request().method() === "PUT" && r.url().endsWith("/checked"));
    await page.getByRole("checkbox", { name: `Cocher ${ingredient}` }).check();
    expect((await saved).ok()).toBe(true);
  });

  await step(page, "Item still checked, after a reload", async () => {
    await page.reload();
    await mainListShown(page);
    await expect(page.getByRole("checkbox", { name: `Cocher ${ingredient}` })).toBeChecked();
  });

  await step(page, "Checked items taken out", async () => {
    await page.getByRole("button", { name: "Courses faites" }).click();
    await page.getByRole("dialog").getByRole("button", { name: "Retirer les articles cochés" }).click();
    await expect(page.getByRole("checkbox", { name: `Cocher ${ingredient}` })).toHaveCount(0);
    await page.reload();
    await mainListShown(page);
    await expect(page.getByRole("checkbox", { name: `Cocher ${ingredient}` })).toHaveCount(0);
  });
});
