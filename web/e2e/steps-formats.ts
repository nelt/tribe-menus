// Steps of the scenarios @ui of the formats of the screen (formats-ecran.feature, ENF-03),
// against the real server (ADR 0028, 0030): what is proved is where the browser places the
// navigation, the dialogs, the days of the planning by week and the panel of a meal, measured
// on the page.
import { type Locator, type Page, expect } from "@playwright/test";
import { scrollEnded } from "./scroll.ts";
import { apiDay, dateOf, dayLabel, mealOf, parseWritten } from "./steps-planning.ts";
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

// The planning by week (D6).

/** The day of the planning written in a scenario. */
function day(page: Page, written: string): Locator {
  return page.locator(".day").filter({ has: page.locator(".day-name", { hasText: dayLabel(dateOf(parseWritten(written))) }) });
}

/** Where a day is in the room of the days, scrolled or not: its edges, in pixels from the left of the room. */
async function placeInWeek(locator: Locator): Promise<{ left: number; right: number; width: number }> {
  return locator.evaluate((element) => {
    const room = element.closest(".days")?.getBoundingClientRect();
    const box = element.getBoundingClientRect();
    if (room === undefined) {
      throw new Error("a day outside the days");
    }
    return { left: box.left - room.left, right: box.right - room.left, width: room.width };
  });
}

async function entirelyVisible(locator: Locator): Promise<boolean> {
  const { left, right, width } = await placeInWeek(locator);
  return left >= -1 && right <= width + 1;
}

step(/les jours du (.+) au (.+) sont côte à côte/, async (world, from, to) => {
  const last = dateOf(parseWritten(to ?? ""));
  const first = dateOf(parseWritten(from ?? ""), last);
  const expected: string[] = [];
  for (let d = new Date(first); d <= last; d.setDate(d.getDate() + 1)) {
    expected.push(dayLabel(d));
  }
  await expect(world.page.locator(".day-name")).toHaveText(expected);
  const boxes = await Promise.all((await world.page.locator(".day").all()).map((d) => d.boundingBox()));
  for (const [i, box] of boxes.entries()) {
    const previous = boxes[i - 1];
    if (box === null || previous === null) {
      throw new Error("a day not shown");
    }
    if (previous !== undefined) {
      expect(box.y).toBe(previous.y);
      expect(box.x).toBeGreaterThanOrEqual(previous.x + previous.width);
    }
  }
});

step(/chaque jour montre le repas du midi au-dessus de celui du soir/, async (world) => {
  for (const d of await world.page.locator(".day").all()) {
    const lunch = await d.locator(".meal-tile--lunch").boundingBox();
    const dinner = await d.locator(".meal-tile--dinner").boundingBox();
    if (lunch === null || dinner === null) {
      throw new Error("a meal not shown");
    }
    expect(lunch.x).toBe(dinner.x);
    expect(lunch.y + lunch.height).toBeLessThanOrEqual(dinner.y);
  }
});

step(/le (.+) n'est pas entièrement visible/, async (world, written) => {
  expect(await entirelyVisible(day(world.page, written ?? ""))).toBe(false);
});

step(/le (.+) est entièrement visible/, async (world, written) => {
  await expect.poll(() => entirelyVisible(day(world.page, written ?? ""))).toBe(true);
});

step(/le (.+) est le premier jour visible, entièrement/, async (world, written) => {
  // The days settle on a day once scrolled (scroll-snap).
  await expect.poll(async () => Math.round((await placeInWeek(day(world.page, written ?? ""))).left)).toBe(0);
});

step(/je fais défiler le semainier d'un jour vers les jours suivants/, async (world) => {
  await world.page.getByRole("button", { name: "Jours suivants" }).click();
  await scrollEnded(world.page.locator(".days"));
});

step(/je fais défiler le semainier jusqu'au dernier jour/, async (world) => {
  const next = world.page.getByRole("button", { name: "Jours suivants" });
  for (let i = 0; i < 7 && (await next.count()) > 0; i++) {
    await next.click();
    await scrollEnded(world.page.locator(".days"));
  }
  await expect(next).toHaveCount(0);
  await expect(world.page.getByRole("button", { name: "Jours précédents" })).toBeVisible();
});

step(/les 7 jours sont visibles sans défilement/, async (world) => {
  const days = await world.page.locator(".day").all();
  expect(days).toHaveLength(7);
  for (const d of days) {
    expect(await entirelyVisible(d)).toBe(true);
  }
  expect(await world.page.locator(".days").evaluate((e) => e.scrollWidth <= e.clientWidth)).toBe(true);
  await expect(world.page.getByRole("button", { name: /^Jours (précédents|suivants)$/ })).toHaveCount(0);
});

// The panel of a meal (D6).

step(/j'ouvre le repas du (.+?) (à midi|au soir)/, async (world, written, mealtime) => {
  const meal = mealOf(written ?? "", mealtime ?? "");
  await world.page.getByRole("link", { name: new RegExp(`^${dayLabel(meal.date)}, ${meal.mealtime.label} : `) }).click();
  await expect(world.page.getByRole("heading", { name: "Plats servis", level: 2 })).toBeVisible();
});

step(/le repas du (.+?) (à midi|au soir) s'affiche en panneau par-dessus le semainier/, async (world, written, mealtime) => {
  const meal = mealOf(written ?? "", mealtime ?? "");
  await expect(world.page).toHaveURL(new RegExp(`/repas/${apiDay(meal.date)}/${meal.mealtime.path}$`));
  // The meal is the only main and the only title: the days behind are out of reach.
  await expect(world.page.getByRole("heading", { level: 1 })).toHaveText(new RegExp(`^\\s*${dayLabel(meal.date).replace(/ \S+$/, "")}\\s[\\s\\S]*\\s${meal.mealtime.label}\\s*$`, "i"));
  // The panel on the right, full height, over the days, which stay behind it.
  const { box, viewport } = await measure(world.page, world.page.locator(".meal-panel"));
  expect(Math.abs(box.x + box.width - viewport.width)).toBeLessThanOrEqual(1);
  expect(box.y).toBe(0);
  expect(box.height).toBe(viewport.height);
  expect(box.width).toBeLessThanOrEqual(440);
  await expect(world.page.locator(".day")).toHaveCount(7);
  const days = await world.page.locator(".days").boundingBox();
  expect(days?.x ?? 0).toBeLessThan(box.x);
  expect((days?.x ?? 0) + (days?.width ?? 0)).toBeGreaterThan(box.x);
});

step(/je ferme le panneau du repas/, async (world) => {
  await world.page.getByRole("link", { name: "Fermer le repas" }).click();
});

step(/j'appuie sur Échap/, async (world) => {
  await world.page.keyboard.press("Escape");
});

step(/aucun repas n'est ouvert/, async (world) => {
  await expect(world.page).not.toHaveURL(/\/repas\//);
  await expect(world.page.locator("mt-meal-screen")).toHaveCount(0);
});
