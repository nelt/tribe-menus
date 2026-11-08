// Steps of the scenarios @ui of the formats of the screen (formats-ecran.feature, ENF-03),
// against the real server (ADR 0028, 0030): what is proved is where the browser places the
// navigation and the dialogs, measured on the page.
import { type Locator, type Page, expect } from "@playwright/test";
import { step } from "./steps.ts";

/** The box of an element shown, and the size of the window. */
async function measure(page: Page, locator: Locator) {
  await expect(locator).toBeVisible();
  const box = await locator.boundingBox();
  const viewport = page.viewportSize();
  if (box === null || viewport === null) {
    throw new Error("nothing to measure");
  }
  return { box, viewport };
}

function tabs(page: Page): Locator {
  return page.getByRole("navigation", { name: "Onglets" });
}

step(/une fenêtre de (\d+) px de large sur (\d+) px de haut/, async (world, width, height) => {
  await world.page.setViewportSize({ width: Number(width), height: Number(height) });
});

step(/les onglets sont en bas de l'écran/, async (world) => {
  const { box, viewport } = await measure(world.page, tabs(world.page));
  expect(Math.abs(box.y + box.height - viewport.height)).toBeLessThanOrEqual(1);
  // Side by side, on one line.
  const ys = await Promise.all(["Planning", "Plats", "Courses"].map(async (name) => (await tabs(world.page).getByRole("link", { name }).boundingBox())?.y));
  expect(new Set(ys).size).toBe(1);
});

step(/les onglets sont sur le côté gauche de l'écran/, async (world) => {
  const { box, viewport } = await measure(world.page, tabs(world.page));
  expect(box.x).toBeLessThanOrEqual(16);
  expect(box.x + box.width).toBeLessThan(viewport.width / 4);
  // One under the other.
  const boxes = await Promise.all(["Planning", "Plats", "Courses"].map((name) => tabs(world.page).getByRole("link", { name }).boundingBox()));
  const [planning, dishes, shopping] = boxes;
  expect(planning?.x).toBe(dishes?.x);
  expect(dishes?.x).toBe(shopping?.x);
  expect(planning!.y).toBeLessThan(dishes!.y);
  expect(dishes!.y).toBeLessThan(shopping!.y);
});

step(/la pastille de la tribu, à côté des onglets, ouvre la tribu/, async (world) => {
  const chip = world.page.getByRole("link", { name: /^Tribu .+, \d+ membres?$/ });
  const { box } = await measure(world.page, chip);
  const rail = await tabs(world.page).boundingBox();
  expect(box.x + box.width).toBeLessThanOrEqual((rail?.x ?? 0) + (rail?.width ?? 0) + 1);
  await chip.click();
  await expect(world.page).toHaveURL(/\/tribu$/);
  await expect(world.page.getByRole("button", { name: "Se déconnecter" })).toBeVisible();
});

function confirmation(page: Page): Locator {
  return page.getByRole("dialog");
}

step(/la confirmation s'affiche en bas de l'écran/, async (world) => {
  const { box, viewport } = await measure(world.page, confirmation(world.page));
  expect(Math.abs(box.y + box.height - viewport.height)).toBeLessThanOrEqual(1);
});

step(/la confirmation s'affiche au milieu de l'écran/, async (world) => {
  const { box, viewport } = await measure(world.page, confirmation(world.page));
  expect(Math.abs(box.x + box.width / 2 - viewport.width / 2)).toBeLessThanOrEqual(2);
  expect(Math.abs(box.y + box.height / 2 - viewport.height / 2)).toBeLessThanOrEqual(2);
  expect(box.width).toBeLessThanOrEqual(440);
});
