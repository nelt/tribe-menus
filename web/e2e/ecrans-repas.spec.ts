// States of the screen of a meal (Doux-Repas, plan planning, lot D), with a simulated API
// (plan socle, D15) and the clock of Playwright on Monday 5 October 2026. The writes and
// their refusals are proved against the real server by godog; R2 « ne m'est pas proposé » by
// its scenario @ui (scenarios.spec.ts).
import { expect, type Page, test } from "@playwright/test";
import { type Received, mealBackLink, mockApi, pageReply, replies } from "./api-mock.ts";

const dish = (n: number, name: string, color: string, servings = 4) => ({
  id: n.toString(16).padStart(32, "0"),
  name,
  color,
  servings,
  revision: 1,
  ingredients: [] as { name: string }[],
});

const chili = dish(1, "Chili con carne", "#C2502E");
const vege = dish(2, "Chili végétarien", "#3F7F25");
const pesto = dish(3, "Pâtes pesto", "#2E8A6E");
const ratatouille = { ...dish(4, "Ratatouille", "#B36A0B", 6), ingredients: [{ name: "Courgette" }] };
const library = [ratatouille, pesto, vege, chili];

let count = 0;
const serve = (d: { id: string }, servings: number, rank: number, date = "2026-10-06", mealtime: "lunch" | "dinner" = "dinner") => ({
  id: (++count).toString(16).padStart(32, "a"),
  date,
  mealtime,
  dish: d.id,
  servings,
  rank,
});

const chiliServed = serve(chili, 3, 3);
const vegeServed = serve(vege, 1, 4);
const tuesday = [chiliServed, vegeServed, serve(pesto, 4, 2, "2026-10-06", "lunch")];
const pestoServed = serve(pesto, 2, 5);
const threeDishes = [chiliServed, vegeServed, pestoServed];
const removed = (version: number) => ({ status: 200, body: { versions: { meal_plan: version } } });

const written = (servedDish: unknown, version: number, status = 200) => ({ status, body: { servedDish, versions: { meal_plan: version } } });
const notFound = { status: 404, body: { error: "not_found" } };

async function mondayMorning(page: Page) {
  await page.clock.setFixedTime(new Date(2026, 9, 5, 10, 0));
}

function mockMeal(page: Page, extra: Parameters<typeof mockApi>[1] = {}): Promise<Received[]> {
  return mockApi(page, { getSession: [replies.session], getDishes: [pageReply("dishes", 3, library)], getMealPlan: [pageReply("meal_plan", 7, tuesday)], ...extra });
}

const tuesdayDinner = "/tribes/demo/repas/2026-10-06/soir";
const servedNames = (page: Page) => page.locator(".served-dish-name");
const suggestions = (page: Page) => page.getByRole("list", { name: "Plats proposés" }).locator(".dish-suggestion-name");
const stepper = (page: Page, name: string) => page.getByRole("group", { name: `Parts de « ${name} »` });
const writes = (received: Received[]) => received.filter((r) => r.method !== "GET").map((r) => `${r.method} ${r.path} ${JSON.stringify(r.body ?? null)}`);

test("a meal: its day and mealtime, its total, its dishes in the order they were added, the suggestions (R2, R3, D11, D12)", async ({ page }) => {
  await mondayMorning(page);
  await mockMeal(page);
  await page.goto(tuesdayDinner);
  await expect(page.getByRole("heading", { level: 1 })).toHaveText(/^\s*Mardi 6 octobre\s*Soir\s*$/);
  await expect(page.locator(".meal-head-total")).toHaveText("4 parts");
  // The colours of the dinner, as the tiles.
  expect(await page.locator(".meal-head").evaluate((e) => getComputedStyle(e).borderColor)).toBe("rgb(189, 186, 240)");
  await expect(servedNames(page)).toHaveText(["Chili con carne", "Chili végétarien"]);
  await expect(page.locator(".served-dish-reference")).toHaveText(["Défini pour 4 parts", "Défini pour 4 parts"]);
  await expect(stepper(page, "Chili con carne")).toContainText("3");
  await expect(stepper(page, "Chili végétarien")).toContainText("1");
  await expect(stepper(page, "Chili végétarien").getByRole("button", { name: "Une part de moins" })).toBeDisabled();
  await expect(page.getByRole("button", { name: "Retirer « Chili con carne » du repas" })).toBeVisible();

  await expect(page.getByText("Les plats déjà servis dans ce repas ne sont pas proposés : réglez leurs parts ci-dessus.")).toBeVisible();
  await expect(suggestions(page)).toHaveText(["Ajouter Pâtes pesto", "Ajouter Ratatouille"]);
  // The tab bar, Planning active, as Doux-Repas.
  await expect(page.getByRole("navigation").getByRole("link", { name: "Planning" })).toHaveAttribute("aria-current", "page");
});

test("an empty meal: « Aucun plat pour ce repas », no total, the whole library by alphabetical order (D12)", async ({ page }) => {
  await mondayMorning(page);
  await mockMeal(page);
  await page.goto("/tribes/demo/repas/2026-10-07/midi");
  await expect(page.getByRole("heading", { level: 1 })).toHaveText(/Mercredi 7 octobre\s*Midi/);
  await expect(page.getByText("Aucun plat pour ce repas")).toBeVisible();
  await expect(page.locator(".meal-head-total")).toHaveCount(0);
  await expect(page.getByText(/Les plats déjà servis/)).toHaveCount(0);
  await expect(suggestions(page)).toHaveText(["Ajouter Chili con carne", "Ajouter Chili végétarien", "Ajouter Pâtes pesto", "Ajouter Ratatouille"]);
  expect(await page.locator(".meal-head").evaluate((e) => getComputedStyle(e).borderColor)).toBe("rgb(242, 210, 122)");
});

test("the search of the suggestions, as in the tab Plats (P3, D12)", async ({ page }) => {
  await mondayMorning(page);
  await mockMeal(page);
  await page.goto(tuesdayDinner);
  const search = page.getByRole("searchbox", { name: "Rechercher un plat" });
  await search.fill("chili");
  await expect(page.getByText("Tous les plats de la bibliothèque sont déjà dans ce repas.")).toHaveCount(0);
  await expect(page.getByText("Aucun plat ne correspond")).toBeVisible();
  await search.fill("courg");
  await expect(suggestions(page)).toHaveText(["Ajouter Ratatouille"]);
  await search.fill("tajine");
  await expect(page.getByText("Aucun plat ne correspond")).toBeVisible();
});

test("every dish of the library already served; an empty library", async ({ page }) => {
  await mondayMorning(page);
  await mockMeal(page, { getDishes: [pageReply("dishes", 3, [chili, vege])] });
  await page.goto(tuesdayDinner);
  await expect(page.getByText("Tous les plats de la bibliothèque sont déjà dans ce repas.")).toBeVisible();
  await expect(suggestions(page)).toHaveCount(0);

  await mockMeal(page, { getDishes: [pageReply("dishes", 3, [])], getMealPlan: [pageReply("meal_plan", 1, [])] });
  await page.goto(tuesdayDinner);
  await expect(page.getByText("Aucun plat dans la bibliothèque")).toBeVisible();
  await expect(page.getByRole("searchbox")).toHaveCount(0);
  await page.getByRole("link", { name: "Créer un plat" }).click();
  await expect(page).toHaveURL(/\/tribes\/demo\/plats\/nouveau$/);
});

test("a suggestion touched is served at once, for the size of the tribe, and no longer suggested (R2, D9)", async ({ page }) => {
  await mondayMorning(page);
  const ratatouilleServed = serve(ratatouille, 4, 9);
  const received = await mockMeal(page, {
    // WebKit cannot be given back a 304 by Playwright (tribe.ts): the next check reads it all.
    getMealPlan: [pageReply("meal_plan", 7, tuesday), pageReply("meal_plan", 8, [...tuesday, ratatouilleServed])],
    serveDish: [written(ratatouilleServed, 8, 201)],
  });
  await page.goto(tuesdayDinner);
  await suggestions(page).filter({ hasText: "Ratatouille" }).click();
  await expect(servedNames(page)).toHaveText(["Chili con carne", "Chili végétarien", "Ratatouille"]);
  await expect(page.locator(".served-dish-reference").last()).toHaveText("Défini pour 6 parts");
  await expect(stepper(page, "Ratatouille")).toContainText("4");
  await expect(page.locator(".meal-head-total")).toHaveText("8 parts");
  await expect(suggestions(page)).toHaveText(["Ajouter Pâtes pesto"]);
  // No servings sent: the server takes the size of the tribe (D9).
  expect(writes(received)).toEqual([`POST meal-plan {"date":"2026-10-06","mealtime":"dinner","dish":"${ratatouille.id}"}`]);
  await expect(page.getByRole("alert")).toHaveCount(0);

  // Back on the planning: the tile shows it; the copy took the version written.
  await mealBackLink(page).click();
  await expect(page.getByRole("link", { name: /^Mardi 6 oct\., soir : / }).locator(".dish-pill-name")).toHaveText(["Chili con carne", "Chili végétarien", "Ratatouille"]);
  await expect.poll(() => received.filter((r) => r.method === "GET" && r.path === "meal-plan").map((r) => r.ifNoneMatch ?? "")).toEqual(["", '"meal_plan-8"']);
});

// Review of PR #88, point 1: the meal sent back has no version, the copy checks itself.
test("a dish served in between by another member: the banner, the meal as it is now, the copy checked (Q33, D3)", async ({ page }) => {
  await mondayMorning(page);
  const ratatouilleServed = serve(ratatouille, 6, 10);
  const now = [chiliServed, vegeServed, ratatouilleServed];
  const received = await mockMeal(page, {
    getMealPlan: [pageReply("meal_plan", 7, tuesday), pageReply("meal_plan", 8, [...now, tuesday[2]])],
    serveDish: [{ status: 409, body: { error: "dish_already_served", meal: now } }],
    setServings: [written({ ...ratatouilleServed, servings: 5 }, 9)],
  });
  await page.goto(tuesdayDinner);
  await expect(servedNames(page)).toHaveCount(2);
  await suggestions(page).filter({ hasText: "Ratatouille" }).click();
  await expect(page.locator(".banner-warning")).toHaveText("« Ratatouille » est déjà servi dans ce repas.");
  await expect(servedNames(page)).toHaveText(["Chili con carne", "Chili végétarien", "Ratatouille"]);
  await expect(stepper(page, "Ratatouille")).toContainText("6");
  await expect.poll(() => received.filter((r) => r.path === "meal-plan" && r.ifNoneMatch === '"meal_plan-7"').length).toBe(1);
  // The banner goes at the next action on the meal.
  await page.getByRole("searchbox", { name: "Rechercher un plat" }).fill("p");
  await expect(page.locator(".banner-warning")).toBeVisible();
  await stepper(page, "Ratatouille").getByRole("button", { name: "Une part de moins" }).click();
  await expect(page.locator(".banner-warning")).toHaveCount(0);
});

test("the servings saved at each step, the last one only after a save running (R3)", async ({ page }) => {
  await mondayMorning(page);
  const received = await mockMeal(page, {
    setServings: [{ ...written({ ...chiliServed, servings: 4 }, 8), delayMs: 500 }, written({ ...chiliServed, servings: 6 }, 9)],
  });
  await page.goto(tuesdayDinner);
  const more = stepper(page, "Chili con carne").getByRole("button", { name: "Une part de plus" });
  await more.click();
  await more.click();
  await more.click();
  await expect(stepper(page, "Chili con carne")).toContainText("6");
  await expect(page.locator(".meal-head-total")).toHaveText("7 parts");
  await expect.poll(() => writes(received)).toEqual([`PUT meal-plan/${chiliServed.id}/servings {"servings":4}`, `PUT meal-plan/${chiliServed.id}/servings {"servings":6}`]);
  await expect(stepper(page, "Chili con carne")).toContainText("6");
  await expect(page.getByRole("alert")).toHaveCount(0);
});

test("the servings of a dish removed in between by another member: the banner, the dish gone (Q33, D3)", async ({ page }) => {
  await mondayMorning(page);
  const received = await mockMeal(page, {
    getMealPlan: [pageReply("meal_plan", 7, tuesday), pageReply("meal_plan", 9, [chiliServed, tuesday[2]])],
    setServings: [notFound],
  });
  await page.goto(tuesdayDinner);
  await stepper(page, "Chili végétarien").getByRole("button", { name: "Une part de plus" }).click();
  await expect(page.locator(".banner-warning")).toHaveText("Ce plat a été retiré du repas entre-temps.");
  await expect(servedNames(page)).toHaveText(["Chili con carne"]);
  await expect(page.locator(".meal-head-total")).toHaveText("3 parts");
  await expect.poll(() => received.filter((r) => r.path === "meal-plan" && r.ifNoneMatch === '"meal_plan-7"').length).toBe(1);
});

test("a dish removed, without confirmation; one already removed goes without message (R4, D19)", async ({ page }) => {
  await mondayMorning(page);
  const received = await mockMeal(page, {
    getMealPlan: [pageReply("meal_plan", 7, tuesday), pageReply("meal_plan", 10, [tuesday[2]])],
    removeServedDish: [{ status: 200, body: { versions: { meal_plan: 8 } } }, notFound],
  });
  await page.goto(tuesdayDinner);
  await page.getByRole("button", { name: "Retirer « Chili con carne » du repas" }).click();
  await expect(servedNames(page)).toHaveText(["Chili végétarien"]);
  await expect(suggestions(page)).toHaveText(["Ajouter Chili con carne", "Ajouter Pâtes pesto", "Ajouter Ratatouille"]);
  await page.getByRole("button", { name: "Retirer « Chili végétarien » du repas" }).click();
  await expect(page.getByText("Aucun plat pour ce repas")).toBeVisible();
  await expect(page.getByRole("alert")).toHaveCount(0);
  expect(writes(received)).toEqual([`DELETE meal-plan/${chiliServed.id} null`, `DELETE meal-plan/${vegeServed.id} null`]);
});

test("a slow double touch on a removal button removes one dish only, the next one coming under the finger (R4, recette de v0.3.0)", async ({ page }) => {
  await mondayMorning(page);
  const received = await mockMeal(page, { getMealPlan: [pageReply("meal_plan", 7, threeDishes)], removeServedDish: [removed(8), removed(9)] });
  await page.goto(tuesdayDinner);
  const middle = page.getByRole("button", { name: "Retirer « Chili végétarien » du repas" });
  const box = await middle.boundingBox();
  if (box === null) {
    throw new Error("no box for the removal button");
  }
  const x = box.x + box.width / 2;
  const y = box.y + box.height / 2;
  await page.mouse.click(x, y);
  await expect(servedNames(page)).toHaveText(["Chili con carne", "Pâtes pesto"]);
  await expect(page.getByRole("button", { name: "Retirer « Pâtes pesto » du repas" })).toBeDisabled();
  await page.mouse.click(x, y);
  await expect(page.getByRole("button", { name: "Retirer « Pâtes pesto » du repas" })).toBeEnabled();
  await expect(servedNames(page)).toHaveText(["Chili con carne", "Pâtes pesto"]);
  expect(writes(received)).toEqual([`DELETE meal-plan/${vegeServed.id} null`]);
  await page.getByRole("button", { name: "Retirer « Pâtes pesto » du repas" }).click();
  await expect(servedNames(page)).toHaveText(["Chili con carne"]);
  expect(writes(received)).toEqual([`DELETE meal-plan/${vegeServed.id} null`, `DELETE meal-plan/${pestoServed.id} null`]);
});

test("a dish removed from the keyboard: the focus does not go to the removal button of the next one (R4, recette de v0.3.0)", async ({ page }) => {
  await mondayMorning(page);
  const received = await mockMeal(page, { getMealPlan: [pageReply("meal_plan", 7, threeDishes)], removeServedDish: [removed(8), removed(9)] });
  await page.goto(tuesdayDinner);
  await page.getByRole("button", { name: "Retirer « Chili con carne » du repas" }).focus();
  await page.keyboard.press("Enter");
  await expect(servedNames(page)).toHaveText(["Chili végétarien", "Pâtes pesto"]);
  await expect(page.locator(".served-dish-remove:focus")).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Retirer « Chili végétarien » du repas" })).toBeEnabled();
  await page.keyboard.press("Enter");
  await page.getByRole("button", { name: "Retirer « Pâtes pesto » du repas" }).click();
  await expect(servedNames(page)).toHaveText(["Chili végétarien"]);
  expect(writes(received)).toEqual([`DELETE meal-plan/${chiliServed.id} null`, `DELETE meal-plan/${pestoServed.id} null`]);
});

test("without network, nothing changes: « Réseau nécessaire », the servings come back (D13)", async ({ page }) => {
  await mondayMorning(page);
  await mockMeal(page);
  await page.goto(tuesdayDinner);
  const message = "Réseau nécessaire. Le réseau est lent ou absent : réessayez.";
  await suggestions(page).filter({ hasText: "Pâtes pesto" }).click();
  await expect(page.getByRole("alert")).toHaveText(message);
  await expect(servedNames(page)).toHaveCount(2);

  await stepper(page, "Chili con carne").getByRole("button", { name: "Une part de plus" }).click();
  await expect(page.getByRole("alert")).toHaveText(message);
  await expect(stepper(page, "Chili con carne")).toContainText("3");
  await expect(page.locator(".meal-head-total")).toHaveText("4 parts");

  await page.getByRole("button", { name: "Retirer « Chili con carne » du repas" }).click();
  await expect(page.getByRole("alert")).toHaveText(message);
  await expect(servedNames(page)).toHaveCount(2);
});

test("loading impossible: the message and “Réessayer”", async ({ page }) => {
  await mondayMorning(page);
  await mockMeal(page, { getMealPlan: [replies.networkError, pageReply("meal_plan", 7, tuesday)] });
  await page.goto(tuesdayDinner);
  await expect(page.getByRole("alert")).toHaveText("Le réseau est lent ou absent. Réessayez.");
  await expect(page.getByRole("heading", { name: "Plats servis" })).toHaveCount(0);
  await page.getByRole("button", { name: "Réessayer" }).click();
  await expect(servedNames(page)).toHaveCount(2);
  await expect(page.getByRole("alert")).toHaveCount(0);
});

test("a meal opened by its path: « Planning » goes to the range that holds it (D14)", async ({ page }) => {
  await mondayMorning(page);
  await mockMeal(page);
  await page.goto(tuesdayDinner);
  await expect(mealBackLink(page)).toHaveAttribute("href", "/tribes/demo/");
  await page.goto("/tribes/demo/repas/2026-10-20/midi");
  await expect(mealBackLink(page)).toHaveAttribute("href", "/tribes/demo/?du=2026-10-20");
  // A day out of the bounds gives the planning.
  await page.goto("/tribes/demo/repas/2100-01-01/midi");
  await expect(page).toHaveURL(/\/tribes\/demo\/repas\/2100-01-01\/midi$/);
  await expect(page.getByRole("heading", { name: "Menus", level: 1 })).toBeVisible();
});
