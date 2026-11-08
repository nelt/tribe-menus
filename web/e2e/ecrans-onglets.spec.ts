// Tab bar and chip of the tribe (plan bibliotheque, D11; plan courses, C1), with a simulated
// API (plan socle, D15).
import { expect, test } from "@playwright/test";
import { mockApi, replies } from "./api-mock.ts";

// The bar at the bottom, and the chip in the planning: the format compact only (ADR 0030,
// point 6); the rail is proved by formats-ecran.feature.
test.beforeEach(({}, testInfo) => {
  test.skip(testInfo.project.metadata["format"] !== "compact", "the tab bar of the format compact");
});

test("the tab bar: Planning, Plats and Courses", async ({ page }) => {
  await mockApi(page, { getSession: [replies.session] });
  await page.goto("/tribes/demo/");
  const tabs = page.getByRole("navigation", { name: "Onglets" });
  await expect(tabs.getByRole("link", { name: "Planning" })).toHaveAttribute("aria-current", "page");
  await expect(tabs.getByRole("link", { name: "Plats" })).toHaveAttribute("aria-current", "false");
  await expect(tabs.getByRole("link", { name: "Courses" })).toHaveAttribute("aria-current", "false");
  await expect(tabs.getByText("À venir")).toHaveCount(0);

  // A tab is followed by the router, without reloading the page.
  await page.evaluate(() => ((window as unknown as { marker: boolean }).marker = true));
  await tabs.getByRole("link", { name: "Plats" }).click();
  await expect(page).toHaveURL(/\/tribes\/demo\/plats$/);
  await expect(tabs.getByRole("link", { name: "Plats" })).toHaveAttribute("aria-current", "page");
  expect(await page.evaluate(() => (window as unknown as { marker?: boolean }).marker)).toBe(true);

  await page.goBack();
  await expect(page).toHaveURL(/\/tribes\/demo\/$/);
  await expect(tabs.getByRole("link", { name: "Planning" })).toHaveAttribute("aria-current", "page");
});

test("wide but low, the tabs keep to the column (recette du plan bibliotheque)", async ({ page }) => {
  await mockApi(page, { getSession: [replies.session] });
  // Wide, but too low for the rail (plan grands-ecrans, D3).
  await page.setViewportSize({ width: 1600, height: 520 });
  await page.goto("/tribes/demo/");
  const tabs = page.getByRole("navigation", { name: "Onglets" });
  const planning = await tabs.getByRole("link", { name: "Planning" }).boundingBox();
  const courses = await tabs.getByRole("link", { name: "Courses" }).boundingBox();
  expect(planning).not.toBeNull();
  expect(courses).not.toBeNull();
  // From the left edge of the first tab to the right edge of the last: the 480 px column, less its margins.
  expect(courses!.x + courses!.width - planning!.x).toBeLessThanOrEqual(480);
  expect(Math.abs(planning!.x + (courses!.x + courses!.width - planning!.x) / 2 - 800)).toBeLessThan(2);
});

test("the chip of the tribe opens the screen Tribu", async ({ page }) => {
  await mockApi(page, { getSession: [replies.session] });
  await page.goto("/tribes/demo/");
  const chip = page.getByRole("link", { name: "Tribu Les Démo, 4 membres" });
  await expect(chip).toHaveText("Les Démo · 4");
  await expect(chip).toBeVisible();
  await chip.click();
  await expect(page).toHaveURL(/\/tribes\/demo\/tribu$/);
  await expect(page.getByRole("heading", { name: "Les Démo" })).toBeVisible();
  // A screen opened from a tab has its link back, and no tab bar, as its mock-up.
  await expect(page.getByRole("navigation", { name: "Onglets" })).toHaveCount(0);
  await page.getByRole("link", { name: "Planning" }).click();
  await expect(page).toHaveURL(/\/tribes\/demo\/$/);
});

test("without the tribe, no chip", async ({ page }) => {
  await mockApi(page, { getSession: [replies.session], getTribe: [replies.networkError] });
  await page.goto("/tribes/demo/");
  await expect(page.getByRole("heading", { name: "Menus" })).toBeVisible();
  await expect(page.getByRole("link", { name: /membre/ })).toHaveCount(0);
});

test("a page of the tribe opened directly, without session, goes to the login", async ({ page }) => {
  await mockApi(page, {});
  await page.goto("/tribes/demo/plats");
  await expect(page).toHaveURL(/\/tribes\/demo\/connexion$/);
});
