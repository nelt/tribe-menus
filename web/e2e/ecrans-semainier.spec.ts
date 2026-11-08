// The planning by week and the panel of a meal (plan grands-ecrans, D6, D8, D10), with a
// simulated API (plan socle, D15) and the clock of Playwright on Monday 5 October 2026. The
// window of a tablet in landscape is set by the test, so it runs with every pull request,
// under both browsers, in the projects of the phones; the scenarios of formats-ecran.feature
// prove the rest against the real server.
import { expect, type Page, test } from "@playwright/test";
import { type Received, mockApi, pageReply, replies } from "./api-mock.ts";
import { scrollEnded } from "./scroll.ts";

const id = (n: number) => n.toString(16).padStart(32, "0");
const chili = { id: id(1), name: "Chili con carne", color: "#C2502E", servings: 4, revision: 1, ingredients: [] };
const vege = { id: id(2), name: "Chili végétarien", color: "#3F7F25", servings: 4, revision: 1, ingredients: [] };
const served = { id: id(3), date: "2026-10-06", mealtime: "dinner", dish: chili.id, servings: 4, rank: 1 };

const landscape = { width: 1180, height: 820 };
const portrait = { width: 820, height: 1180 };

test.beforeEach(async ({ page }, testInfo) => {
  test.skip(testInfo.project.metadata["format"] !== "compact", "the window is set by the test, once per browser");
  await page.clock.setFixedTime(new Date(2026, 9, 5, 10));
  await page.setViewportSize(landscape);
});

function mockWeek(page: Page): Promise<Received[]> {
  return mockApi(page, {
    getSession: [replies.session],
    getDishes: [pageReply("dishes", 3, [chili, vege])],
    getMealPlan: [pageReply("meal_plan", 7, [served])],
  });
}

const days = (page: Page) => page.locator(".days");
const scrollOf = (page: Page) => days(page).evaluate((e) => e.scrollLeft);
const sunday = (page: Page) => page.getByRole("link", { name: /^Dimanche 11 oct\., soir : / });

test("the panel closed by its cross: the days kept where they were scrolled, the focus back on the meal (D6, D10)", async ({ page }) => {
  await mockWeek(page);
  await page.goto("/tribes/demo/");
  await page.getByRole("button", { name: "Jours suivants" }).click();
  await expect(page.getByRole("button", { name: "Jours suivants" })).toHaveCount(0);
  const scrolled = await scrollOf(page);
  expect(scrolled).toBeGreaterThan(0);

  await sunday(page).click();
  await expect(page).toHaveURL(/\/repas\/2026-10-11\/soir$/);
  // The focus in the panel; the days behind out of reach, the rail not.
  await expect(page.getByRole("link", { name: "Fermer le repas" })).toBeFocused();
  await expect(page.getByRole("heading", { name: "Menus" })).toHaveCount(0);
  await expect(page.getByRole("navigation", { name: "Onglets" })).toBeVisible();
  await expect(page.locator("mt-planning-screen")).toHaveAttribute("inert", "");

  await page.getByRole("link", { name: "Fermer le repas" }).click();
  await expect(page).toHaveURL(/\/tribes\/demo\/$/);
  await expect(sunday(page)).toBeFocused();
  expect(await scrollOf(page)).toBe(scrolled);
});

test("the veil and Échap close the panel; uncovered, the planning checks the copies (D6, D15)", async ({ page }) => {
  const received = await mockWeek(page);
  await page.goto("/tribes/demo/repas/2026-10-06/soir");
  await expect(page.getByRole("heading", { name: "Plats servis" })).toBeVisible();
  await page.locator(".meal-overlay").click({ position: { x: 20, y: 20 } });
  await expect(page).toHaveURL(/\/tribes\/demo\/$/);
  await expect(page.locator("mt-meal-screen")).toHaveCount(0);

  await page.getByRole("link", { name: /^Mardi 6 oct\., soir : / }).click();
  await expect(page.getByRole("heading", { name: "Plats servis" })).toBeVisible();
  const checks = received.filter((r) => r.method === "GET" && r.path === "meal-plan").length;
  await page.keyboard.press("Escape");
  await expect(page).toHaveURL(/\/tribes\/demo\/$/);
  await expect.poll(() => received.filter((r) => r.method === "GET" && r.path === "meal-plan").length).toBeGreaterThan(checks);
});

test("the button that disappears with the last day hands the focus to the other one (D10, review of PR #108, point 1)", async ({ page }) => {
  await mockWeek(page);
  await page.goto("/tribes/demo/");
  const next = page.getByRole("button", { name: "Jours suivants" });
  const previous = page.getByRole("button", { name: "Jours précédents" });
  await next.focus();
  for (let i = 0; i < 7 && (await next.count()) > 0; i++) {
    await page.keyboard.press("Enter");
    await scrollEnded(days(page));
  }
  await expect(next).toHaveCount(0);
  await expect(previous).toBeFocused();
  for (let i = 0; i < 7 && (await previous.count()) > 0; i++) {
    await page.keyboard.press("Enter");
    await scrollEnded(days(page));
  }
  await expect(previous).toHaveCount(0);
  await expect(next).toBeFocused();
});

test("another week starts at its first day", async ({ page }) => {
  await mockWeek(page);
  await page.goto("/tribes/demo/");
  await page.getByRole("button", { name: "Jours suivants" }).click();
  // One day is all that is left on a tablet in landscape.
  await expect(page.getByRole("button", { name: "Jours suivants" })).toHaveCount(0);
  await page.getByRole("button", { name: "Semaine suivante" }).click();
  await expect(page.getByRole("heading", { level: 2 })).toHaveText("12 – 18 octobre");
  await expect.poll(() => scrollOf(page)).toBe(0);
  await expect(page.getByRole("button", { name: "Jours précédents" })).toHaveCount(0);
});

test("turning the tablet keeps the meal and its search: a panel in etendu, a screen in moyen (D8)", async ({ page }) => {
  await mockWeek(page);
  await page.goto("/tribes/demo/repas/2026-10-06/soir");
  const search = page.getByRole("searchbox", { name: "Rechercher un plat" });
  await search.fill("végé");
  await page.locator("mt-meal-screen").evaluate((e) => e.setAttribute("data-kept", "yes"));

  await page.setViewportSize(portrait);
  await expect(page.locator("html")).toHaveAttribute("data-format", "moyen");
  await expect(page.locator("mt-meal-screen")).toHaveAttribute("data-kept", "yes");
  await expect(search).toHaveValue("végé");
  await expect(page.getByRole("main").getByRole("link", { name: "Planning", exact: true })).toBeVisible();
  await expect(page.getByRole("link", { name: "Fermer le repas" })).toHaveCount(0);
  await expect(page.locator("mt-planning-screen")).toHaveCount(0);

  await page.setViewportSize(landscape);
  await expect(page.locator("html")).toHaveAttribute("data-format", "etendu");
  await expect(page.locator("mt-meal-screen")).toHaveAttribute("data-kept", "yes");
  await expect(search).toHaveValue("végé");
  await expect(page.getByRole("link", { name: "Fermer le repas" })).toBeVisible();
  await expect(page.locator(".day")).toHaveCount(7);
});
