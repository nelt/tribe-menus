// Targeted journey of v0.6.0 (docs/recettes/v0.6.0.md, entries 3, 4 and 6 to 8): a list made stale
// by another device, « Recalculer », the taken part. A dish with 200 g of an ingredient, served
// tomorrow evening for the size of the tribe, its shopping added to « Courses » and checked; a
// second context on the same session doubles the servings of the meal; the first comes back to
// the application, sees the stale banner and the pill of the list picker, recomputes: the item is
// unchecked at 400 g, « encore 200 g ». Then checked and taken out, the tribe left as found.
import type { Browser } from "@playwright/test";
import { recetteSettings } from "./config.ts";
import { addMealShopping, check, checkout, comeBack, itemLine, listCard, openCourses, openListPicker, openTomorrow, runSuffix, serve } from "./helpers.ts";
import { expect, scenario, step, test } from "./test.ts";

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

test("stale list, Recalculer and the taken part", scenario("Liste rendue périmée par un autre appareil, « Recalculer » et part prise"), async ({ page, browser, baseURL }, testInfo) => {
  const run = runSuffix(testInfo.project.name);
  const dish = `Recalcul ${run}`;
  const ingredient = `farine ${run}`;

  // Tooling, not the application: no Gherkin, left out of the scenario on the control page.
  await test.step("List up to date", async () => {
    // A run stopped halfway may have left « Courses » stale: recomputed first.
    await openCourses(page);
    const recompute = page.locator(".banner-stale").getByRole("button", { name: "Recalculer" });
    if (await recompute.isVisible()) {
      await recompute.click();
      await expect(page.getByText(staleText)).toHaveCount(0);
    }
  });

  await step(page, "Dish of 200 g created", [`Étant donné le plat "${dish}" défini pour la taille de la tribu, avec 200 g de "${ingredient}", créé depuis l'écran Nouveau plat`], async () => {
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

  await step(page, "Dish served tomorrow evening", [`Et le plat "${dish}" servi demain soir pour la taille de la tribu, ajouté depuis le planning`], async () => {
    await openTomorrow(page, "soir");
    await serve(page, dish);
  });

  await step(page, "Meal shopping checked", [`Et les courses des repas ajoutées à la liste "Courses", sans modifier la période proposée`, `Et la liste "Courses" contient 200 g de "${ingredient}"`, `Et je coche "${ingredient}" dans la liste "Courses"`], async () => {
    await addMealShopping(page);
    // Served for the size of the tribe, which the dish is defined for by default.
    await expect(itemLine(page, ingredient).locator(".item-quantity")).toHaveText("200 g");
    await check(page, ingredient);
  });

  await step(undefined, "Servings doubled on another device", [`Quand, sur un autre appareil connecté à la même tribu, je double les parts du plat "${dish}" de demain soir`, "Alors le serveur garde le nouveau nombre de parts (relu après un rechargement)"], () => doubleFromAnotherDevice(browser, baseURL ?? "", dish));

  await step(page, "Stale list seen on return", ["Quand je reviens sur l'application du premier appareil, sans la recharger", `Alors un bandeau en haut de la liste "Courses" indique que des repas ont changé`, "Et le bandeau propose l'action « Recalculer »"], async () => {
    await comeBack(page);
    await expect(page.locator(".banner-stale")).toContainText(staleText);
    await expect(page.locator(".banner-stale").getByRole("button", { name: "Recalculer" })).toBeEnabled();
  });

  await step(page, "Lists with modified meals", ["Quand j'ouvre le choix des listes", `Alors la liste "Courses" porte la mention « Repas modifiés »`], async () => {
    await openListPicker(page);
    await expect(listCard(page, "Courses")).toContainText("Repas modifiés");
  });

  await step(page, "List recomputed", [`Quand je touche « Recalculer » dans la liste "Courses"`, "Alors un message m'indique « Liste recalculée. »", "Et le bandeau n'est plus affiché"], async () => {
    await openCourses(page);
    await page.locator(".banner-stale").getByRole("button", { name: "Recalculer" }).click();
    await expect(page.getByRole("status").filter({ hasText: "Liste recalculée." })).toBeVisible();
    await expect(page.getByText(staleText)).toHaveCount(0);
  });

  await step(page, "Taken part shown", [`Alors la liste "Courses" contient 400 g de "${ingredient}"`, `Et "${ingredient}" n'est plus coché, avec « encore 200 g » sous son nom`, "Et son dépliage dit « 200 g déjà pris : il reste 200 g à prendre. Cochez quand tout est pris. »"], async () => {
    const line = itemLine(page, ingredient);
    await expect(line.locator(".item-quantity")).toHaveText("400 g");
    await expect(line.locator(".item-remaining")).toHaveText("encore 200 g");
    await expect(page.getByRole("checkbox", { name: `Cocher ${ingredient}` })).not.toBeChecked();
    await line.locator(".item-unfold").click();
    await expect(line.locator(".item-taken-note")).toHaveText("200 g déjà pris : il reste 200 g à prendre. Cochez quand tout est pris.");
  });

  await step(page, "Item taken out", [`Quand je coche "${ingredient}" puis termine les courses`, `Alors "${ingredient}" ne figure plus dans la liste "Courses"`], async () => {
    await check(page, ingredient);
    await checkout(page, ingredient);
  });
});
