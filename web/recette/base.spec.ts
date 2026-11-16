// The base journeys (ADR 0032, point 5; plan recette-auto, step 13), played at every acceptance
// run in each of the four projects: each tab opens; a dish created then found by the search; served
// at tomorrow's lunch; its shopping added to « Courses », an item checked, then taken out by
// « Courses faites »; each write read again after a reload. The data carry the time of the run
// and the project, and accumulate in the tribe recette-auto, except the items, taken out.
import { addMealShopping, check, checkout, listTitle, openTomorrow, planningShown, runSuffix, serve, tabs } from "./helpers.ts";
import { expect, scenario, step, test } from "./test.ts";

test("base journey: each tab, a dish, a meal, its shopping", scenario("Parcours de base : chaque onglet, un plat, un repas, ses courses"), async ({ page }, testInfo) => {
  const run = runSuffix(testInfo.project.name);
  const dish = `Recette ${run}`;
  const ingredient = `essai ${run}`;

  await step(page, "Planning", ["Étant donné que je suis connecté à la tribu", "Quand j'ouvre la tribu", "Alors le planning « Menus » s'affiche avec ses 7 jours"], async () => {
    await page.goto("./");
    await planningShown(page);
  });

  await step(page, "Plats", ["Quand je touche l'onglet Plats", "Alors la bibliothèque « Plats » s'affiche"], async () => {
    await tabs(page).getByRole("link", { name: "Plats" }).click();
    await expect(page.getByRole("heading", { name: "Plats", exact: true })).toBeVisible();
  });

  await step(page, "Courses", ["Quand je touche l'onglet Courses", `Alors la liste "Courses" s'affiche`], async () => {
    await tabs(page).getByRole("link", { name: "Courses" }).click();
    await expect(listTitle(page)).toHaveText("Courses");
  });

  await step(page, "Dish created", [`Quand je crée le plat "${dish}" avec le nouvel ingrédient "${ingredient}", sans quantité`, "Alors la bibliothèque s'affiche de nouveau"], async () => {
    await page.goto("plats/nouveau");
    await expect(page.getByRole("heading", { name: "Nouveau plat" })).toBeVisible();
    await page.getByLabel("Nom du plat").fill(dish);
    await page.getByRole("button", { name: "Ajouter un ingrédient" }).click();
    await page.getByRole("combobox", { name: "Ingrédient", exact: true }).fill(ingredient);
    await page.getByRole("option", { name: `Créer « ${ingredient} »` }).click();
    await page.getByRole("button", { name: "Enregistrer le plat" }).click();
    await expect(page.getByRole("heading", { name: "Plats", exact: true })).toBeVisible();
  });

  await step(page, "Dish found by the search, after a reload", ["Quand je recharge la page", `Et je recherche "${dish}"`, `Alors le seul résultat est le plat "${dish}"`], async () => {
    await page.reload();
    await page.getByRole("searchbox", { name: "Rechercher un plat ou un ingrédient" }).fill(dish);
    await expect(page.locator(".dish-list .dish-name")).toHaveText([dish]);
  });

  await step(page, "Dish served tomorrow at lunch", [`Quand j'ajoute le plat "${dish}" au repas de demain à midi, depuis le planning`, "Alors il figure parmi les plats servis de ce repas"], async () => {
    await openTomorrow(page, "midi");
    await serve(page, dish);
  });

  await step(page, "Meal read again, after a reload", ["Quand je rouvre le planning puis le repas de demain à midi", `Alors le plat "${dish}" y est toujours servi`], async () => {
    await openTomorrow(page, "midi");
    await expect(page.getByRole("button", { name: `Retirer « ${dish} » du repas` })).toBeVisible();
  });

  await step(page, "Meal shopping added to Courses", [`Quand j'ajoute les courses des repas à la liste "Courses", sans modifier la période proposée`, `Alors "${ingredient}" figure dans la liste "Courses"`], async () => {
    await addMealShopping(page);
    await expect(page.getByRole("checkbox", { name: `Cocher ${ingredient}` })).toBeVisible();
  });

  await step(page, "Item checked", [`Quand je coche "${ingredient}" dans la liste "Courses"`, "Alors le serveur enregistre la coche"], () => check(page, ingredient));

  await step(page, "Item still checked, after a reload", ["Quand je recharge la page", `Alors "${ingredient}" apparaît comme coché dans la liste "Courses"`], async () => {
    await page.reload();
    await expect(listTitle(page)).toHaveText("Courses");
    await expect(page.getByRole("checkbox", { name: `Cocher ${ingredient}` })).toBeChecked();
  });

  await step(page, "Checked items taken out", ["Quand je termine les courses : « Courses faites », puis « Retirer les articles cochés »", `Alors "${ingredient}" ne figure plus dans la liste "Courses", même après un rechargement`], async () => {
    await checkout(page, ingredient);
    await page.reload();
    await expect(listTitle(page)).toHaveText("Courses");
    await expect(page.getByRole("checkbox", { name: `Cocher ${ingredient}` })).toHaveCount(0);
  });
});
