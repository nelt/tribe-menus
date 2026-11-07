// States of the screen of the meal shopping (Doux-CoursesRepas) and of the unfolded items
// (Doux-Liste; plan courses, lot E), with a simulated API (plan socle, D15) and the clock of
// Playwright on Monday 5 October 2026. The computation and its refusals are proved against
// the real server by godog; C4 and C5 on screen by their scenarios @ui (scenarios.spec.ts).
import { expect, type Page, test } from "@playwright/test";
import { type Received, mockApi, pageReply, replies } from "./api-mock.ts";

const id = (n: number) => n.toString(16).padStart(32, "0");

const chili = { id: id(100), name: "Chili con carne", color: "#c2502e", servings: 4, revision: 1, ingredients: [] };
const rata = { id: id(101), name: "Ratatouille", color: "#b36a0b", servings: 4, revision: 1, ingredients: [] };
const served = (n: number, date: string, mealtime: string, dish = chili) => ({ id: id(n), date, mealtime, dish: dish.id, servings: 4, rank: n });
const mealPlan = [served(50, "2026-10-06", "dinner"), served(51, "2026-10-07", "dinner", rata), served(52, "2026-10-08", "lunch"), served(53, "2026-10-08", "dinner", rata), served(54, "2026-10-20", "lunch")];

const source = (servedDish: number, date: string, mealtime: string, servings: number, dish = chili, quantity?: string, unit?: string) => ({
  servedDish: id(servedDish),
  dish: dish.id,
  date,
  mealtime,
  servings,
  ...(quantity === undefined ? {} : { quantity, unit }),
});
const boeuf = { id: id(1), ingredient: "bœuf haché", quantity: "875", unit: "g", checked: false, manual: false, sources: [source(52, "2026-10-08", "lunch", 4, chili, "500", "g"), source(50, "2026-10-06", "dinner", 3, chili, "375", "g")] };
const oignon = { id: id(2), ingredient: "oignon", quantity: "2", unit: "piece", checked: false, manual: false, sources: [source(50, "2026-10-06", "dinner", 3, chili, "0.75", "piece"), source(52, "2026-10-08", "lunch", 4, chili, "1", "piece")] };
const sel = { id: id(3), ingredient: "sel", checked: false, manual: false, sources: [source(51, "2026-10-07", "dinner", 4, rata)] };
const pain = { id: id(4), ingredient: "pain", quantity: "2", unit: "piece", checked: false, manual: true, sources: [] };

const list = (n: number, name: string, items: unknown[], change: Record<string, unknown> = {}) => ({ id: id(n), name, main: false, rank: n, items, received: [] as string[], ...change });
const courses = list(10, "Courses", [boeuf, oignon, sel, pain], { main: true, rank: 1, received: [id(50), id(52)] });
const marche = list(11, "Marché du samedi", [pain], { received: [id(51)] });
const drive = list(12, "Drive", []);

function mock(page: Page, extra: Parameters<typeof mockApi>[1] = {}): Promise<Received[]> {
  return mockApi(page, {
    getSession: [replies.session],
    getShoppingLists: [pageReply("shopping_lists", 7, [courses, marche, drive])],
    getDishes: [pageReply("dishes", 3, [chili, rata])],
    getMealPlan: [pageReply("meal_plan", 4, mealPlan)],
    ...extra,
  });
}

test.beforeEach(async ({ page }) => {
  await page.clock.setFixedTime(new Date(2026, 9, 5, 10, 0));
});

const writes = (received: Received[]) => received.filter((r) => r.method !== "GET").map((r) => `${r.method} ${r.path} ${JSON.stringify(r.body ?? null)}`);
const field = (page: Page, name: "Du" | "Au") => page.getByRole("button", { name: new RegExp(`^${name} `) });

test("from a list: the default period, the points, the summary, the served dishes already received (C4, D24, Q22)", async ({ page }) => {
  await mock(page);
  await page.goto("/tribes/demo/courses");
  await page.getByRole("link", { name: "Ajouter les courses des repas" }).click();
  await expect(page).toHaveURL(`/tribes/demo/courses/repas?liste=${courses.id}`);
  await expect(page.getByRole("heading", { name: "Courses des repas", level: 1 })).toBeVisible();
  await expect(page.getByRole("navigation", { name: "Onglets" })).toHaveCount(0);
  await expect(field(page, "Du")).toHaveText(/mar\. 6 oct\./);
  await expect(field(page, "Au")).toHaveText(/lun\. 12 oct\./);
  await expect(field(page, "Du")).toHaveAttribute("aria-pressed", "true");
  await expect(page.getByText("Octobre 2026")).toBeVisible();
  await expect(page.getByRole("button", { name: "mardi 6 octobre, repas planifiés" })).toHaveAttribute("aria-pressed", "true");
  await expect(page.getByRole("button", { name: "lundi 5 octobre, aujourd’hui" })).toHaveAttribute("aria-pressed", "false");
  await expect(page.getByRole("button", { name: "mardi 20 octobre, repas planifiés" })).toBeVisible();
  await expect(page.getByText("7 jours")).toBeVisible();
  await expect(page.getByText("4 plats servis")).toBeVisible();
  await expect(page.getByRole("button", { name: /Ajouter à la liste Courses/ })).toBeVisible();
  await expect(page.getByText("2 plats servis sont déjà passés par cette liste : ils seront ignorés.")).toBeVisible();
  await expect(page.getByRole("button", { name: "Ajouter à « Courses »" })).toBeEnabled();
  await page.screenshot({ path: "test-results/courses-repas.png", fullPage: true });

  await page.getByRole("link", { name: "Retour à la liste" }).click();
  await expect(page).toHaveURL(/\/tribes\/demo\/courses$/);
});

test("a day touched sets « Du », then « Au »; the end before the start, more than 31 days (C4, Q38)", async ({ page }) => {
  await mock(page);
  await page.goto("/tribes/demo/courses/repas");
  await page.getByRole("button", { name: "jeudi 8 octobre, repas planifiés" }).click();
  await expect(field(page, "Du")).toHaveText(/jeu\. 8 oct\./);
  await expect(field(page, "Au")).toHaveAttribute("aria-pressed", "true");
  await page.getByRole("button", { name: "mardi 6 octobre, repas planifiés" }).click();
  await expect(field(page, "Au")).toHaveText(/mar\. 6 oct\./);
  await expect(page.getByRole("alert")).toHaveText("La date de fin doit suivre la date de début.");
  await expect(field(page, "Au")).toHaveAttribute("aria-invalid", "true");
  await expect(page.getByRole("button", { name: "Ajouter à « Courses »" })).toBeDisabled();
  await expect(page.getByText(/plats? servis? sont déjà passés/)).toHaveCount(0);

  await page.getByRole("button", { name: "Mois suivant" }).click();
  await expect(page.getByText("Novembre 2026")).toBeVisible();
  await page.getByRole("button", { name: "dimanche 8 novembre" }).click();
  await expect(page.getByRole("alert")).toHaveText("La période ne peut pas dépasser 31 jours.");
  await page.getByRole("button", { name: "samedi 7 novembre" }).click();
  await expect(page.getByRole("alert")).toHaveCount(0);
  await expect(page.getByText("31 jours")).toBeVisible();

  await field(page, "Du").click();
  await expect(page.getByText("Octobre 2026")).toBeVisible();
});

test("the months stop at January 2000 and December 2099 (Q35)", async ({ page }) => {
  await mock(page);
  await page.clock.setFixedTime(new Date(2000, 0, 1, 10, 0));
  await page.goto("/tribes/demo/courses/repas");
  await expect(page.getByText("Janvier 2000")).toBeVisible();
  await expect(page.getByRole("button", { name: "Mois précédent" })).toBeDisabled();
});

test("the target list chosen in the sheet; a period without meals (C4, Q22)", async ({ page }) => {
  await mock(page);
  await page.goto(`/tribes/demo/courses/repas?liste=${drive.id}`);
  await expect(page.getByRole("button", { name: "Ajouter à « Drive »" })).toBeVisible();
  await page.getByRole("button", { name: /Ajouter à la liste Drive/ }).click();
  const sheet = page.getByRole("dialog", { name: "Ajouter à quelle liste ?" });
  await expect(sheet.locator(".target-option-name")).toHaveText(["Courses", "Marché du samedi", "Drive"]);
  await expect(sheet.getByRole("button", { name: /^Drive/ })).toHaveAttribute("aria-pressed", "true");
  await expect(sheet.getByRole("button", { name: /^Courses/ })).toContainText("principale");
  await expect(sheet.getByRole("button", { name: /^Courses/ })).toContainText("4 articles");
  await expect(sheet.getByRole("button", { name: /^Drive/ })).toContainText("vide");
  await sheet.getByRole("button", { name: /^Marché du samedi/ }).click();
  await expect(sheet).toHaveCount(0);
  await expect(page.getByText("1 plat servi est déjà passé par cette liste : il sera ignoré.")).toBeVisible();

  await page.getByRole("button", { name: "Mois suivant" }).click();
  await page.getByRole("button", { name: "lundi 9 novembre" }).click();
  await page.getByRole("button", { name: "dimanche 15 novembre" }).click();
  await expect(page.getByText("aucun plat servi")).toBeVisible();
  await expect(page.getByText("Aucun repas planifié sur cette période : l’ajout ne changera rien à la liste.")).toBeVisible();
  await expect(page.getByRole("button", { name: "Ajouter à « Marché du samedi »" })).toBeEnabled();
});

test("added: back to the target list with the green banner (C4, Q22)", async ({ page }) => {
  const after = { ...marche, items: [pain, boeuf], received: [id(51), id(50), id(52)] };
  const received = await mock(page, {
    addMealShopping: [{ status: 200, body: { list: after, added: 2, ignored: 1, versions: { shopping_lists: 8 } }, delayMs: 300 }],
    // Read by the list, by the meal shopping, then by the list again, after the write.
    getShoppingLists: [pageReply("shopping_lists", 7, [courses, marche, drive]), pageReply("shopping_lists", 7, [courses, marche, drive]), pageReply("shopping_lists", 8, [courses, after, drive])],
  });
  await page.goto(`/tribes/demo/courses/${marche.id}`);
  await page.getByRole("link", { name: "Ajouter les courses des repas" }).click();
  await page.getByRole("button", { name: "Ajouter à « Marché du samedi »" }).click();
  await expect(page.getByRole("button", { name: "Ajout en cours…" })).toBeDisabled();
  await expect(page).toHaveURL(`/tribes/demo/courses/${marche.id}`);
  await expect(page.getByRole("status").filter({ hasText: "2 plats servis ajoutés, 1 déjà reçu ignoré" })).toBeVisible();
  await expect(page.locator(".item")).toHaveCount(2);
  expect(writes(received)).toEqual([`POST shopping-lists/${marche.id}/meal-shopping {"from":"2026-10-06","to":"2026-10-12"}`]);
  // The meal shopping replaced its page in the history: back goes to the list we came from.
  await page.goBack();
  await expect(page).toHaveURL(`/tribes/demo/courses/${marche.id}`);
});

test("300 items, without network: the message, nothing added; a list deleted in between (Q38, D2, D11)", async ({ page }) => {
  await mock(page, {
    addMealShopping: [{ status: 409, body: { error: "too_many_items" } }, replies.networkError, { status: 404, body: { error: "not_found" } }],
    // Read at the opening, after the refusal of 300 items, then after the 404.
    getShoppingLists: [pageReply("shopping_lists", 7, [courses, marche, drive]), pageReply("shopping_lists", 7, [courses, marche, drive]), pageReply("shopping_lists", 9, [courses, marche])],
  });
  await page.goto(`/tribes/demo/courses/repas?liste=${drive.id}`);
  const add = page.getByRole("button", { name: "Ajouter à « Drive »" });
  await add.click();
  await expect(page.getByRole("alert")).toHaveText("La liste ne peut pas dépasser 300 articles : rien n’a été ajouté.");
  await add.click();
  await expect(page.getByRole("alert")).toHaveText("Réseau nécessaire. Le réseau est lent ou absent : réessayez.");
  await add.click();
  await expect(page).toHaveURL(/\/tribes\/demo\/courses$/);
  await expect(page.getByRole("alert")).toHaveText("Cette liste a été supprimée entre-temps.");
});

test("loading impossible: the message and « Réessayer »", async ({ page }) => {
  await mock(page, { getMealPlan: [replies.networkError, pageReply("meal_plan", 4, mealPlan)] });
  await page.goto("/tribes/demo/courses/repas");
  await expect(page.getByRole("alert")).toHaveText("Le réseau est lent ou absent. Réessayez.");
  await page.getByRole("button", { name: "Réessayer" }).click();
  await expect(page.getByText("4 plats servis")).toBeVisible();
});

test("a computed item unfolds its meals, folds back; an item added does not (C5, Q37)", async ({ page }) => {
  await mock(page);
  await page.goto("/tribes/demo/courses");
  const row = page.getByRole("button", { name: "bœuf haché, 875 g, 2 plats. Afficher les repas" });
  await expect(row).toHaveAttribute("aria-expanded", "false");
  await row.click();
  const meals = page.getByRole("list", { name: "Repas de bœuf haché" });
  const lines = await meals.getByRole("listitem").evaluateAll((all) => all.map((e) => (e.textContent ?? "").replace(/\s+/g, " ").trim()));
  expect(lines).toEqual(["C Chili con carne · 3 parts mar. 6 oct. · soir 375 g", "C Chili con carne · 4 parts jeu. 8 oct. · midi 500 g"]);
  await expect(page.getByRole("button", { name: "bœuf haché, 875 g, 2 plats. Masquer les repas" })).toHaveAttribute("aria-expanded", "true");

  await page.getByRole("button", { name: /^oignon, 2 pièces/ }).click();
  await expect(page.getByRole("list", { name: "Repas de oignon" })).toContainText("0,75 pièce");
  await page.getByRole("button", { name: /^sel, sans quantité, 1 plat/ }).click();
  const salt = page.getByRole("list", { name: "Repas de sel" });
  await expect(salt).toContainText("Ratatouille · 4 parts");
  await expect(salt.locator(".item-source-quantity")).toHaveCount(0);
  await page.screenshot({ path: "test-results/courses-depliage.png", fullPage: true });

  await page.getByRole("button", { name: /^bœuf haché.*Masquer les repas$/ }).click();
  await expect(meals).toHaveCount(0);
  // An item added: no unfolding.
  await expect(page.getByRole("button", { name: /^pain/ })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Retirer pain" })).toBeVisible();
});

test("« Retour à la liste » toward the main list opened by its identifier: the path of the tab (D15; review of PR #101, point 2)", async ({ page }) => {
  await mock(page, { getShoppingLists: [{ ...pageReply("shopping_lists", 7, [courses, marche, drive]), delayMs: 300 }] });
  await page.goto(`/tribes/demo/courses/repas?liste=${courses.id}`);
  await expect(page.getByRole("button", { name: "Ajouter à « Courses »" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Retour à la liste" })).toHaveAttribute("href", "/tribes/demo/courses");
  await page.goto(`/tribes/demo/courses/repas?liste=${marche.id}`);
  await expect(page.getByRole("link", { name: "Retour à la liste" })).toHaveAttribute("href", `/tribes/demo/courses/${marche.id}`);
  await page.goto(`/tribes/demo/courses/repas?liste=${id(99)}`);
  await expect(page.getByRole("button", { name: "Ajouter à « Courses »" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Retour à la liste" })).toHaveAttribute("href", "/tribes/demo/courses");
});
