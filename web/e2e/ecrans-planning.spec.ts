// States of the screen Planning (Doux-Main, plan planning), with a simulated API (plan socle,
// D15) and the clock of Playwright on Monday 5 October 2026. R1 and R5 are proved by their
// scenarios against the real server (scenarios.spec.ts).
import { expect, type Page, test } from "@playwright/test";
import { type Received, mealBackLink, mockApi, pageReply, replies } from "./api-mock.ts";

const dish = (n: number, name: string, color: string, servings = 4) => ({
  id: n.toString(16).padStart(32, "0"),
  name,
  color,
  servings,
  revision: 1,
  ingredients: [],
});

const chili = dish(1, "Chili con carne", "#C2502E");
const vege = dish(2, "Chili végétarien", "#3F7F25");
const pesto = dish(3, "Pâtes pesto", "#2E8A6E");
const ratatouille = dish(4, "Ratatouille", "#B36A0B");
const library = [chili, vege, pesto, ratatouille];

let served = 0;
const serve = (date: string, mealtime: "lunch" | "dinner", d: { id: string }, servings: number, rank: number) => ({
  id: (++served).toString(16).padStart(32, "a"),
  date,
  mealtime,
  dish: d.id,
  servings,
  rank,
});

const week = [
  serve("2026-10-05", "dinner", ratatouille, 4, 1),
  serve("2026-10-06", "lunch", pesto, 4, 2),
  serve("2026-10-06", "dinner", vege, 1, 4),
  serve("2026-10-06", "dinner", chili, 3, 3),
];

/** Monday 5 October 2026, mid-morning, on the device (D4). */
async function mondayMorning(page: Page, day = 5) {
  await page.clock.setFixedTime(new Date(2026, 9, day, 10, 0));
}

function mockPlanning(page: Page, plan = week, extra: Parameters<typeof mockApi>[1] = {}): Promise<Received[]> {
  return mockApi(page, { getSession: [replies.session], getDishes: [pageReply("dishes", 3, library)], getMealPlan: [pageReply("meal_plan", 7, plan)], ...extra });
}

const title = (page: Page) => page.getByRole("heading", { level: 2 });
/** A meal tile, a link named by its day, its mealtime and its dishes. */
const tile = (page: Page, name: string) => page.getByRole("link", { name: new RegExp(`^${name} : `) });
const dayNames = (page: Page) => page.getByRole("heading", { level: 3 });

test("the 7 days from today, two meals a day, the dishes in the order they were added (R1, D11)", async ({ page }) => {
  await mondayMorning(page);
  const received = await mockPlanning(page);
  await page.goto("/tribes/demo/");
  await expect(page.getByRole("heading", { name: "Menus", level: 1 })).toBeVisible();
  await expect(title(page)).toHaveText("5 – 11 octobre");
  await expect(page.getByText("À partir d'aujourd'hui")).toBeVisible();
  await expect(page.getByRole("button", { name: "Revenir à aujourd'hui" })).toHaveCount(0);
  await expect(dayNames(page)).toHaveText(["Lundi 5 oct.", "Mardi 6 oct.", "Mercredi 7 oct.", "Jeudi 8 oct.", "Vendredi 9 oct.", "Samedi 10 oct.", "Dimanche 11 oct."]);
  await expect(page.locator(".day-today")).toHaveCount(1);
  await expect(page.locator(".day").first().locator(".day-today")).toHaveText("Aujourd'hui");
  await expect(page.locator(".meal-tile-label")).toHaveText(Array.from({ length: 7 }, () => ["MIDI", "SOIR"]).flat());

  const tuesdayDinner = tile(page, "Mardi 6 oct., soir");
  await expect(tuesdayDinner.locator(".dish-pill-name")).toHaveText(["Chili con carne", "Chili végétarien"]);
  await expect(tuesdayDinner.locator(".dish-pill-servings")).toHaveText(["3", "1"]);
  // Read aloud: the badge alone would be a bare number.
  await expect(tuesdayDinner).toHaveAccessibleName("Mardi 6 oct., soir : Chili con carne, 3 parts ; Chili végétarien, 1 part");
  await expect(tuesdayDinner).toHaveAttribute("href", "/tribes/demo/repas/2026-10-06/soir");
  // The colour of a dish on its pill: the badge in its strong hue.
  expect(await tuesdayDinner.locator(".dish-pill-servings").first().evaluate((e) => getComputedStyle(e).color)).toBe("rgb(194, 80, 46)");
  // An empty meal: dashed, « Ajouter ».
  const wednesdayLunch = tile(page, "Mercredi 7 oct., midi");
  await expect(wednesdayLunch).toContainText("Ajouter");
  await expect(wednesdayLunch).toHaveAccessibleName("Mercredi 7 oct., midi : aucun plat");
  await expect(wednesdayLunch).toHaveAttribute("href", "/tribes/demo/repas/2026-10-07/midi");
  expect(await wednesdayLunch.evaluate((e) => getComputedStyle(e).borderStyle)).toBe("dashed");
  expect(await tile(page, "Lundi 5 oct., soir").evaluate((e) => getComputedStyle(e).borderColor)).toBe("rgb(189, 186, 240)");
  expect(await tile(page, "Mardi 6 oct., midi").evaluate((e) => getComputedStyle(e).borderColor)).toBe("rgb(242, 210, 122)");

  // The meal plan and the library are read together (D15).
  expect(received.filter((r) => r.method === "GET").map((r) => r.path).sort()).toEqual(["dishes", "ingredients", "meal-plan", "session", "tribe"]);
});

test("next and previous weeks, without request (R5, D1, D14)", async ({ page }) => {
  await mondayMorning(page);
  const received = await mockPlanning(page);
  await page.goto("/tribes/demo/");
  await expect(dayNames(page)).toHaveCount(7);
  const reads = received.length;

  await page.getByRole("button", { name: "Semaine suivante" }).click();
  await expect(page).toHaveURL(/\/tribes\/demo\/\?du=2026-10-12$/);
  await expect(title(page)).toHaveText("12 – 18 octobre");
  await expect(dayNames(page).first()).toHaveText("Lundi 12 oct.");
  await expect(page.locator(".day-today")).toHaveCount(0);
  await expect(page.getByText("À partir d'aujourd'hui")).toHaveCount(0);

  await page.getByRole("button", { name: "Semaine précédente" }).click();
  // The range of today is the root of the tribe.
  await expect(page).toHaveURL(/\/tribes\/demo\/$/);
  await expect(title(page)).toHaveText("5 – 11 octobre");
  await page.getByRole("button", { name: "Semaine précédente" }).click();
  await expect(page).toHaveURL(/\/tribes\/demo\/\?du=2026-09-28$/);
  await expect(title(page)).toHaveText("28 septembre – 4 octobre");
  await expect(dayNames(page).last()).toHaveText("Dimanche 4 oct.");

  await page.getByRole("button", { name: "Revenir à aujourd'hui" }).click();
  await expect(page).toHaveURL(/\/tribes\/demo\/$/);
  await expect(title(page)).toHaveText("5 – 11 octobre");
  // Back in the history, the range before.
  await page.goBack();
  await expect(title(page)).toHaveText("28 septembre – 4 octobre");
  expect(received.length).toBe(reads);
});

test("a range of the path: across two years, in another year; an unreadable day gives today (D14)", async ({ page }) => {
  await mondayMorning(page);
  await mockPlanning(page);
  await page.goto("/tribes/demo/?du=2026-12-29");
  await expect(title(page)).toHaveText("29 décembre 2026 – 4 janvier 2027");
  await expect(page.getByRole("button", { name: "Revenir à aujourd'hui" })).toBeVisible();
  await expect(dayNames(page)).toHaveText(["Mardi 29 déc.", "Mercredi 30 déc.", "Jeudi 31 déc.", "Vendredi 1er janv.", "Samedi 2 janv.", "Dimanche 3 janv.", "Lundi 4 janv."]);

  await page.goto("/tribes/demo/?du=2025-10-06");
  await expect(title(page)).toHaveText("6 – 12 octobre 2025");

  await page.goto("/tribes/demo/?du=2026-02-30");
  await expect(title(page)).toHaveText("5 – 11 octobre");
  await expect(page.getByText("À partir d'aujourd'hui")).toBeVisible();
});

test("at the bounds of 2000 and 2099: no range beyond, no meal beyond (review of PR #87)", async ({ page }) => {
  await mondayMorning(page);
  await mockPlanning(page);
  await page.goto("/tribes/demo/?du=2099-12-28");
  await expect(page.getByRole("button", { name: "Semaine suivante" })).toBeDisabled();
  await expect(page.getByRole("button", { name: "Semaine précédente" })).toBeEnabled();
  await expect(tile(page, "Jeudi 31 déc., soir")).toContainText("Ajouter");
  // Out of the bounds, a tile is not a link.
  await expect(page.getByRole("group", { name: "Vendredi 1er janv., midi" })).not.toContainText("Ajouter");
  await expect(page.getByRole("link", { name: /^Vendredi 1er janv\./ })).toHaveCount(0);
  await expect(page.locator(".meal-tile--inactive")).toHaveCount(6);

  await page.goto("/tribes/demo/?du=2000-01-03");
  await page.getByRole("button", { name: "Semaine précédente" }).click();
  await expect(page).toHaveURL(/\?du=2000-01-01$/);
  await expect(page.getByRole("button", { name: "Semaine précédente" })).toBeDisabled();
});

test("loading: the header and the range at once, the days once both copies are read (D15)", async ({ page }) => {
  await mondayMorning(page);
  await mockPlanning(page, week, { getMealPlan: [{ ...pageReply("meal_plan", 7, week), delayMs: 1000 }] });
  await page.goto("/tribes/demo/");
  await expect(title(page)).toHaveText("5 – 11 octobre");
  await expect(page.getByRole("link", { name: "Tribu Les Démo, 4 membres" })).toBeVisible();
  await expect(dayNames(page)).toHaveCount(0);
  await expect(dayNames(page)).toHaveCount(7);
});

test("loading impossible: the message and “Réessayer”", async ({ page }) => {
  await mondayMorning(page);
  await mockPlanning(page, week, { getMealPlan: [replies.networkError, pageReply("meal_plan", 7, week)] });
  await page.goto("/tribes/demo/");
  await expect(page.getByRole("alert")).toHaveText("Le réseau est lent ou absent. Réessayez.");
  await expect(dayNames(page)).toHaveCount(0);
  await page.getByRole("button", { name: "Réessayer" }).click();
  await expect(dayNames(page)).toHaveCount(7);
  await expect(page.getByRole("alert")).toHaveCount(0);
});

// Review of PR #90, point 1: the chip is the only way to the screen Tribu, so to the sign-out.
test("the tribe not read: read again by “Réessayer”, and back in the foreground", async ({ page }) => {
  await mondayMorning(page);
  await mockPlanning(page, week, { getTribe: [replies.networkError, replies.tribe] });
  await page.goto("/tribes/demo/");
  await expect(dayNames(page)).toHaveCount(7);
  await expect(page.getByRole("alert")).toHaveText("Le réseau est lent ou absent. Réessayez.");
  await expect(page.locator(".tribe-chip")).toHaveCount(0);
  await page.getByRole("button", { name: "Réessayer" }).click();
  await expect(page.getByRole("link", { name: "Tribu Les Démo, 4 membres" })).toBeVisible();
  await expect(page.getByRole("alert")).toHaveCount(0);
});

test("the tribe not read: read again back in the foreground", async ({ page }) => {
  await mondayMorning(page);
  const received = await mockPlanning(page, week, { getTribe: [replies.networkError, replies.tribe] });
  await page.goto("/tribes/demo/");
  await expect(page.getByRole("alert")).toBeVisible();
  await backToForeground(page);
  await expect(page.getByRole("link", { name: "Tribu Les Démo, 4 membres" })).toBeVisible();
  // Nothing missing any more: the message goes, as with « Réessayer » (review of PR #91, point 1).
  await expect(page.getByRole("alert")).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Réessayer" })).toHaveCount(0);
  // Once read, the tribe is not read again.
  await backToForeground(page);
  await expect.poll(() => received.filter((r) => r.path === "meal-plan").length).toBe(3);
  expect(received.filter((r) => r.path === "tribe").length).toBe(2);
});

test("the tribe read again back in the foreground: the message stays if the copies could not be read", async ({ page }) => {
  await mondayMorning(page);
  await mockPlanning(page, week, { getTribe: [replies.networkError, replies.tribe], getMealPlan: [replies.networkError] });
  await page.goto("/tribes/demo/");
  await expect(page.getByRole("alert")).toBeVisible();
  await backToForeground(page);
  await expect(page.getByRole("link", { name: "Tribu Les Démo, 4 membres" })).toBeVisible();
  await expect(page.getByRole("alert")).toHaveText("Le réseau est lent ou absent. Réessayez.");
  await expect(page.getByRole("button", { name: "Réessayer" })).toBeVisible();
});

test("a meal tile opens its meal; « Planning » goes back to the same range (D14)", async ({ page }) => {
  await mondayMorning(page);
  await mockPlanning(page);
  await page.goto("/tribes/demo/?du=2026-10-12");
  await tile(page, "Mardi 13 oct., soir").click();
  await expect(page).toHaveURL(/\/tribes\/demo\/repas\/2026-10-13\/soir$/);
  await expect(page.getByRole("heading", { level: 1 })).toHaveText(/Mardi 13 octobre\s*Soir/);
  await mealBackLink(page).click();
  await expect(page).toHaveURL(/\?du=2026-10-12$/);
  await expect(title(page)).toHaveText("12 – 18 octobre");
});

async function backToForeground(page: Page) {
  await page.evaluate(() => {
    for (const state of ["hidden", "visible"]) {
      Object.defineProperty(document, "visibilityState", { configurable: true, get: () => state });
      document.dispatchEvent(new Event("visibilitychange"));
    }
  });
}

test("back in the foreground: the copies are checked, a dish served by another member appears (T1, D15)", async ({ page }) => {
  await mondayMorning(page);
  const thursday = serve("2026-10-08", "lunch", chili, 4, 5);
  const received = await mockPlanning(page, week, {
    getMealPlan: [pageReply("meal_plan", 7, week), pageReply("meal_plan", 8, [...week, thursday])],
    // WebKit cannot be given back a 304 by Playwright (tribe.ts): the library is sent again.
    getDishes: [pageReply("dishes", 3, library)],
  });
  await page.goto("/tribes/demo/");
  await expect(dayNames(page)).toHaveCount(7);
  await expect(tile(page, "Jeudi 8 oct., midi")).toContainText("Ajouter");
  await backToForeground(page);
  await expect(tile(page, "Jeudi 8 oct., midi")).toContainText("Chili con carne");
  const checks = received.filter((r) => r.ifNoneMatch !== undefined).map((r) => [r.path, r.ifNoneMatch]);
  expect(checks).toContainEqual(["meal-plan", '"meal_plan-7"']);
  expect(checks).toContainEqual(["dishes", '"dishes-3"']);
});

test("a new day on return to the foreground: the range of today follows it, another range stays (D17)", async ({ page }) => {
  await mondayMorning(page);
  await mockPlanning(page);
  await page.goto("/tribes/demo/");
  await expect(title(page)).toHaveText("5 – 11 octobre");
  await mondayMorning(page, 6);
  await backToForeground(page);
  await expect(title(page)).toHaveText("6 – 12 octobre");
  await expect(page.locator(".day").first().locator(".day-today")).toHaveText("Aujourd'hui");

  await page.getByRole("button", { name: "Semaine suivante" }).click();
  await expect(page).toHaveURL(/\?du=2026-10-13$/);
  await mondayMorning(page, 7);
  await backToForeground(page);
  await expect(title(page)).toHaveText("13 – 19 octobre");
  await page.getByRole("button", { name: "Revenir à aujourd'hui" }).click();
  await expect(title(page)).toHaveText("7 – 13 octobre");
});
