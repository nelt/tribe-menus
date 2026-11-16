// What the journeys of the acceptance run share: names carrying the run and the project, and
// the screens they go through, found as the scenarios @ui find them in every format.
import { basename } from "node:path";
import type { Page } from "@playwright/test";
import { dayLabel } from "../e2e/steps-planning.ts";
import { recetteSettings } from "./config.ts";
import { expect } from "./test.ts";

/** Short names of the projects: an ingredient holds 40 characters at most. */
const projects: Record<string, string> = { "phone-chromium": "pc", "phone-webkit": "pw", "tablet-landscape-chromium": "tc", "computer-chromium": "cc" };

/** « 20261115-152550 pc »: the time of the run, then the project, to suffix what a journey writes. */
export function runSuffix(project: string): string {
  return `${basename(recetteSettings().outputDir)} ${projects[project] ?? project}`;
}

export function tabs(page: Page) {
  return page.getByRole("navigation", { name: "Onglets" });
}

/** The screen Planning, its seven days shown once the copies are read. */
export async function planningShown(page: Page) {
  await expect(page.getByRole("heading", { name: "Menus", level: 1 })).toBeVisible();
  await expect(page.locator(".day")).toHaveCount(7);
}

/** A meal of tomorrow, on the device's date, opened from the planning. */
export async function openTomorrow(page: Page, mealtime: "midi" | "soir") {
  await page.goto("./");
  await planningShown(page);
  const tomorrow = new Date(await page.evaluate(() => Date.now()));
  tomorrow.setDate(tomorrow.getDate() + 1);
  await page.getByRole("link", { name: new RegExp(`^${dayLabel(tomorrow)}, ${mealtime} : `) }).click();
  await expect(page.getByRole("heading", { name: "Plats servis", level: 2 })).toBeVisible();
}

/** Serves a dish of the library at the meal shown, for the size of the tribe. */
export async function serve(page: Page, dish: string) {
  await page.getByRole("searchbox", { name: "Rechercher un plat", exact: true }).fill(dish);
  await page.getByRole("list", { name: "Plats proposés" }).getByRole("button", { name: `Ajouter ${dish}`, exact: true }).click();
  await expect(page.getByRole("button", { name: `Retirer « ${dish} » du repas` })).toBeVisible();
}

/** The title of the list shown. */
export function listTitle(page: Page) {
  return page.getByRole("heading", { level: 1 }).locator(".shopping-title-name");
}

/** The main list « Courses », opened by the tab's path. */
export async function openCourses(page: Page) {
  await page.goto("courses");
  await expect(listTitle(page)).toHaveText("Courses");
}

/** Adds the meal shopping of the default period to « Courses », back on the list. */
export async function addMealShopping(page: Page) {
  await openCourses(page);
  await page.getByRole("link", { name: "Ajouter les courses des repas" }).click();
  await expect(page.getByRole("heading", { name: "Courses des repas", level: 1 })).toBeVisible();
  await page.getByRole("button", { name: "Ajouter à « Courses »" }).click();
  await expect(listTitle(page)).toHaveText("Courses");
}

/** Checks an item, waiting for the server to keep it. */
export async function check(page: Page, ingredient: string) {
  const saved = page.waitForResponse((r) => r.request().method() === "PUT" && r.url().endsWith("/checked"));
  await page.getByRole("checkbox", { name: `Cocher ${ingredient}` }).check();
  expect((await saved).ok()).toBe(true);
}

/** « Courses faites » on the list shown: the checked items leave it. */
export async function checkout(page: Page, ...ingredients: string[]) {
  await page.getByRole("button", { name: "Courses faites" }).click();
  await page.getByRole("dialog").getByRole("button", { name: "Retirer les articles cochés" }).click();
  for (const ingredient of ingredients) {
    await expect(page.getByRole("checkbox", { name: `Cocher ${ingredient}` })).toHaveCount(0);
  }
}

/** The line of an item in the list shown, by its ingredient. */
export function itemLine(page: Page, ingredient: string) {
  return page.locator(".item").filter({ has: page.locator(".item-name", { hasText: new RegExp(`^${ingredient}$`) }) });
}

/** The list picker: beside the list in etendu, its own screen otherwise. */
export async function openListPicker(page: Page) {
  if ((await page.locator("html").getAttribute("data-format")) === "etendu") {
    await expect(page.getByRole("heading", { name: "Listes", level: 2 })).toBeVisible();
    return;
  }
  await page.getByRole("link", { name: /Changer de liste$/ }).click();
  await expect(page.getByRole("heading", { name: "Listes", level: 1 })).toBeVisible();
}

/** The card of a list in the picker. */
export function listCard(page: Page, name: string) {
  return page.locator(".list-card").filter({ has: page.locator(".list-card-name", { hasText: new RegExp(`^${name}$`) }) });
}

/** The application leaves the screen then comes back, as when the phone is unlocked: the page is not reloaded. */
export async function comeBack(page: Page) {
  await page.evaluate(() => {
    for (const state of ["hidden", "visible"]) {
      Object.defineProperty(document, "visibilityState", { configurable: true, get: () => state });
      Object.defineProperty(document, "hidden", { configurable: true, get: () => state === "hidden" });
      document.dispatchEvent(new Event("visibilitychange"));
    }
  });
}
