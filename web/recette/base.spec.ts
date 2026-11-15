// The base journeys (ADR 0032, point 5; plan recette-auto, step 13), played at every acceptance
// run in each of the four projects: each tab opens; a dish created then found by the search; served
// at tomorrow's lunch; its shopping added to « Courses », an item checked, then taken out by
// « Courses faites »; each write read again after a reload. The data carry the time of the run
// and the project, and accumulate in the tribe recette-auto, except the items, taken out.
import { addMealShopping, check, checkout, listTitle, openTomorrow, planningShown, runSuffix, serve, tabs } from "./helpers.ts";
import { expect, step, test } from "./test.ts";

test("base journey: each tab, a dish, a meal, its shopping", async ({ page }, testInfo) => {
  const run = runSuffix(testInfo.project.name);
  const dish = `Recette ${run}`;
  const ingredient = `essai ${run}`;

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
    await expect(listTitle(page)).toHaveText("Courses");
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
    await openTomorrow(page, "midi");
    await serve(page, dish);
  });

  await step(page, "Meal read again, after a reload", async () => {
    await openTomorrow(page, "midi");
    await expect(page.getByRole("button", { name: `Retirer « ${dish} » du repas` })).toBeVisible();
  });

  await step(page, "Meal shopping added to Courses", async () => {
    await addMealShopping(page);
    await expect(page.getByRole("checkbox", { name: `Cocher ${ingredient}` })).toBeVisible();
  });

  await step(page, "Item checked", () => check(page, ingredient));

  await step(page, "Item still checked, after a reload", async () => {
    await page.reload();
    await expect(listTitle(page)).toHaveText("Courses");
    await expect(page.getByRole("checkbox", { name: `Cocher ${ingredient}` })).toBeChecked();
  });

  await step(page, "Checked items taken out", async () => {
    await checkout(page, ingredient);
    await page.reload();
    await expect(listTitle(page)).toHaveText("Courses");
    await expect(page.getByRole("checkbox", { name: `Cocher ${ingredient}` })).toHaveCount(0);
  });
});
