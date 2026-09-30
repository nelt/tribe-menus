// Smoke test of the skeleton: the public site and the application are served.
import { expect, test } from "@playwright/test";

test("the root shows the public site", async ({ page }) => {
  await page.goto("/");
  await expect(page).toHaveTitle("Melting Tribe");
  await expect(page.getByRole("heading", { name: "Melting Tribe" })).toBeVisible();
});

test("a tribe URL shows the application", async ({ page }) => {
  await page.goto("/tribes/demo/planning");
  await expect(page.getByRole("heading", { name: "Melting Tribe" })).toBeVisible();
  await expect(page.getByText("tribu demo")).toBeVisible();
});
