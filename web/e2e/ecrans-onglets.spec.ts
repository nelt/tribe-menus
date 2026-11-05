// Tab bar and chip of the tribe (plan bibliotheque, D11), with a simulated API (plan socle,
// D15).
import { expect, test } from "@playwright/test";
import { mockApi, replies } from "./api-mock.ts";

test("the tab bar: Planning, Plats, and Courses to come", async ({ page }) => {
  await mockApi(page, { getSession: [replies.session] });
  await page.goto("/tribes/demo/");
  const tabs = page.getByRole("navigation", { name: "Onglets" });
  await expect(tabs.getByRole("link", { name: "Planning" })).toHaveAttribute("aria-current", "page");
  await expect(tabs.getByRole("link", { name: "Plats" })).toHaveAttribute("aria-current", "false");
  await expect(tabs.getByRole("link", { name: /Courses/ })).toHaveCount(0);
  await expect(tabs.getByText("Courses")).toHaveAttribute("aria-disabled", "true");
  await expect(tabs.getByText("À venir")).toBeVisible();

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

test("Courses inactive is paler than the other tabs; wide, the tabs keep to the column (recette du plan bibliotheque)", async ({ page }) => {
  await mockApi(page, { getSession: [replies.session] });
  await page.setViewportSize({ width: 1600, height: 900 });
  await page.goto("/tribes/demo/");
  const tabs = page.getByRole("navigation", { name: "Onglets" });
  const color = (name: string) => tabs.getByText(name, { exact: true }).evaluate((e) => getComputedStyle(e).color);
  expect(await tabs.locator(".tab--inactive").evaluate((e) => getComputedStyle(e).color)).toBe("rgb(207, 198, 222)");
  expect(await color("À venir")).toBe("rgb(110, 102, 128)");

  const planning = await tabs.getByRole("link", { name: "Planning" }).boundingBox();
  const courses = await tabs.locator(".tab--inactive").boundingBox();
  expect(planning).not.toBeNull();
  expect(courses).not.toBeNull();
  // From the left edge of the first tab to the right edge of the last: the 480 px column, less its margins.
  expect(courses!.x + courses!.width - planning!.x).toBeLessThanOrEqual(480);
  expect(Math.abs(planning!.x + (courses!.x + courses!.width - planning!.x) / 2 - 800)).toBeLessThan(2);
});

test("the chip of the tribe opens the screen Tribu", async ({ page }) => {
  await mockApi(page, { getSession: [replies.session] });
  await page.goto("/tribes/demo/");
  const chip = page.getByRole("link", { name: "Les Démo 4 membres" });
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
  await expect(page.getByRole("heading", { name: "Les Démo" })).toBeVisible();
  await expect(page.getByRole("link", { name: /membre/ })).toHaveCount(0);
});

test("a page of the tribe opened directly, without session, goes to the login", async ({ page }) => {
  await mockApi(page, {});
  await page.goto("/tribes/demo/plats");
  await expect(page).toHaveURL(/\/tribes\/demo\/connexion$/);
});
