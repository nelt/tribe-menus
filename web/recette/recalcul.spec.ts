// Targeted journey of v0.6.0 (docs/recettes/v0.6.0.md, entries 3, 4 and 6 to 8): a list made stale
// by another device, « Recalculer », the taken part. A dish with 200 g of an ingredient, served
// tomorrow evening for the size of the tribe, its shopping added to « Courses » and checked; a
// second context on the same session doubles the servings of the meal; the first comes back to
// the application, sees the stale banner and the pill of the list picker, recomputes: the item is
// unchecked at 400 g, « encore 200 g ». Then checked and taken out, the tribe left as found.
import type { Browser } from "@playwright/test";
import { recetteSettings } from "./config.ts";
import { addMealShopping, check, checkout, comeBack, itemLine, listCard, openCourses, openListPicker, openTomorrow, runSuffix, serve } from "./helpers.ts";
import { expect, step, test } from "./test.ts";

const staleText = "Des repas ont changé depuis l’ajout de leurs courses à cette liste.";

/** Doubles the servings of a dish served tomorrow evening, from another device of the same member (not filmed). */
async function doubleFromAnotherDevice(browser: Browser, baseURL: string, dish: string) {
  const context = await browser.newContext({ baseURL, storageState: recetteSettings().storageState });
  try {
    const page = await context.newPage();
    await openTomorrow(page, "soir");
    const servings = page.getByRole("group", { name: `Parts de « ${dish} »` });
    const before = Number(await servings.locator("output").innerText());
    for (let n = 0; n < before; n++) {
      await servings.getByRole("button", { name: "Une part de plus" }).click();
    }
    await expect(servings.locator("output")).toHaveText(String(2 * before));
    // Kept by the server: read again after a reload.
    await expect(async () => {
      await openTomorrow(page, "soir");
      await expect(servings.locator("output")).toHaveText(String(2 * before), { timeout: 2_000 });
    }).toPass();
  } finally {
    await context.close();
  }
}

test("stale list, Recalculer and the taken part", async ({ page, browser, baseURL }, testInfo) => {
  const run = runSuffix(testInfo.project.name);
  const dish = `Recalcul ${run}`;
  const ingredient = `farine ${run}`;

  await step(page, "List up to date", async () => {
    // A run stopped halfway may have left « Courses » stale: recomputed first.
    await openCourses(page);
    const recompute = page.locator(".banner-stale").getByRole("button", { name: "Recalculer" });
    if (await recompute.isVisible()) {
      await recompute.click();
      await expect(page.getByText(staleText)).toHaveCount(0);
    }
  });

  await step(page, "Dish of 200 g created", async () => {
    await page.goto("plats/nouveau");
    await expect(page.getByRole("heading", { name: "Nouveau plat" })).toBeVisible();
    await page.getByLabel("Nom du plat").fill(dish);
    await page.getByRole("button", { name: "Ajouter un ingrédient" }).click();
    await page.getByRole("combobox", { name: "Ingrédient", exact: true }).fill(ingredient);
    await page.getByRole("option", { name: `Créer « ${ingredient} »` }).click();
    await page.getByLabel(`Quantité de ${ingredient}`, { exact: true }).fill("200");
    await page.getByLabel(`Unité de ${ingredient}`, { exact: true }).selectOption("g");
    await page.getByRole("button", { name: "Enregistrer le plat" }).click();
    await expect(page.getByRole("heading", { name: "Plats", exact: true })).toBeVisible();
  });

  await step(page, "Dish served tomorrow evening", async () => {
    await openTomorrow(page, "soir");
    await serve(page, dish);
  });

  await step(page, "Meal shopping checked", async () => {
    await addMealShopping(page);
    // Served for the size of the tribe, which the dish is defined for by default.
    await expect(itemLine(page, ingredient).locator(".item-quantity")).toHaveText("200 g");
    await check(page, ingredient);
  });

  await test.step("Servings doubled on another device", () => doubleFromAnotherDevice(browser, baseURL ?? "", dish));

  await step(page, "Stale list seen on return", async () => {
    await comeBack(page);
    await expect(page.locator(".banner-stale")).toContainText(staleText);
    await expect(page.locator(".banner-stale").getByRole("button", { name: "Recalculer" })).toBeEnabled();
  });

  await step(page, "Lists with modified meals", async () => {
    await openListPicker(page);
    await expect(listCard(page, "Courses")).toContainText("Repas modifiés");
  });

  await step(page, "List recomputed", async () => {
    await openCourses(page);
    await page.locator(".banner-stale").getByRole("button", { name: "Recalculer" }).click();
    await expect(page.getByRole("status").filter({ hasText: "Liste recalculée." })).toBeVisible();
    await expect(page.getByText(staleText)).toHaveCount(0);
  });

  await step(page, "Taken part shown", async () => {
    const line = itemLine(page, ingredient);
    await expect(line.locator(".item-quantity")).toHaveText("400 g");
    await expect(line.locator(".item-remaining")).toHaveText("encore 200 g");
    await expect(page.getByRole("checkbox", { name: `Cocher ${ingredient}` })).not.toBeChecked();
    await line.locator(".item-unfold").click();
    await expect(line.locator(".item-taken-note")).toHaveText("200 g déjà pris : il reste 200 g à prendre. Cochez quand tout est pris.");
  });

  await step(page, "Item taken out", async () => {
    await check(page, ingredient);
    await checkout(page, ingredient);
  });
});
