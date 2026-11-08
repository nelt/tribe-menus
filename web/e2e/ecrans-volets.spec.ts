// Two panes for the dishes and the shopping lists, the confirmation before losing an input
// and the rotation (plan grands-ecrans, D5, D8, D9), with a simulated API (plan socle, D15).
// The window of a tablet is set by the test, so it runs with every pull request, under both
// browsers, in the projects of the phones; the scenarios of formats-ecran.feature prove the
// rest against the real server.
import { expect, type Page, test } from "@playwright/test";
import { type Received, mockApi, notModified, pageReply, replies } from "./api-mock.ts";

const id = (n: number) => n.toString(16).padStart(32, "0");
const lines = Array.from({ length: 24 }, (_, i) => ({ name: `ingrédient ${i + 1}`, quantity: "1", unit: "piece" }));
const chili = { id: id(1), name: "Chili con carne", color: "#C2502E", servings: 4, revision: 1, ingredients: lines };
const vege = { id: id(2), name: "Chili végétarien", color: "#3F7F25", servings: 4, revision: 1, ingredients: [{ name: "maïs" }] };
const list = (n: number, name: string, main = false) => ({ id: id(n), name, main, rank: n, items: [], received: [] });
const courses = list(10, "Courses", true);
const marche = list(11, "Marché");

const landscape = { width: 1180, height: 820 };
const portrait = { width: 820, height: 1180 };
const phone = { width: 412, height: 915 };

test.beforeEach(async ({ page }, testInfo) => {
  test.skip(testInfo.project.metadata["format"] !== "compact", "the window is set by the test, once per browser");
  await page.setViewportSize(landscape);
});

function mock(page: Page): Promise<Received[]> {
  return mockApi(page, {
    getSession: [replies.session],
    getTribe: [replies.tribe],
    getDishes: [pageReply("dishes", 3, [chili, vege])],
    getShoppingLists: [pageReply("shopping_lists", 7, [courses, marche])],
  });
}

const library = (page: Page) => page.locator("mt-dish-library section");
const picker = (page: Page) => page.locator("mt-list-picker section");
const dishName = (page: Page) => page.getByRole("textbox", { name: "Nom du plat" });
const abandon = (page: Page) => page.getByRole("dialog", { name: "Abandonner la modification ?" });
const card = (page: Page, name: string) => library(page).getByRole("link", { name: new RegExp(`^${name} pour`) });

test("Plats in etendu: the invitation, then the dish beside the library, its card marked, without link back (D5)", async ({ page }) => {
  await mock(page);
  await page.goto("/tribes/demo/plats");
  await expect(page.getByRole("heading", { name: "Plats", level: 2 })).toBeVisible();
  const main = page.getByRole("main");
  await expect(main.getByRole("heading", { level: 1 })).toHaveText("Choisissez un plat ou créez-en un");
  await expect(main.getByRole("link", { name: "Nouveau plat" })).toHaveAttribute("href", "/tribes/demo/plats/nouveau");
  await page.screenshot({ path: "test-results/volets-plats.png" });

  await card(page, "Chili végétarien").click();
  await expect(page).toHaveURL(`/tribes/demo/plats/${vege.id}`);
  await expect(main.getByRole("heading", { level: 1 })).toHaveText("Chili végétarien");
  await expect(card(page, "Chili végétarien")).toHaveAttribute("aria-current", "page");
  await expect(card(page, "Chili con carne")).not.toHaveAttribute("aria-current", "page");
  await expect(main.getByRole("link", { name: "Plats" })).toHaveCount(0);
  // « Enregistrer le plat » at the bottom of the pane of the dish (D11).
  const save = await page.getByRole("button", { name: "Enregistrer le plat" }).boundingBox();
  expect(Math.round((save?.y ?? 0) + (save?.height ?? 0))).toBeGreaterThan(landscape.height - 40);
  await page.screenshot({ path: "test-results/volets-plat.png" });
});

test("a dish changed, then another dish, the rail, « Nouveau »: « Abandonner la modification ? » (D9)", async ({ page }) => {
  await mock(page);
  await page.goto(`/tribes/demo/plats/${chili.id}`);
  await dishName(page).fill("Chili du dimanche");

  await card(page, "Chili végétarien").click();
  await expect(abandon(page)).toBeVisible();
  await expect(abandon(page).getByRole("button", { name: "Continuer la modification" })).toBeFocused();
  await page.screenshot({ path: "test-results/volets-abandon.png" });
  await page.keyboard.press("Escape");
  await expect(abandon(page)).toHaveCount(0);
  await expect(page).toHaveURL(`/tribes/demo/plats/${chili.id}`);
  await expect(dishName(page)).toHaveValue("Chili du dimanche");

  await page.getByRole("navigation", { name: "Onglets" }).getByRole("link", { name: "Courses" }).click();
  await abandon(page).getByRole("button", { name: "Continuer la modification" }).click();
  await expect(dishName(page)).toHaveValue("Chili du dimanche");

  await library(page).getByRole("link", { name: "Nouveau", exact: true }).click();
  await abandon(page).getByRole("button", { name: "Abandonner", exact: true }).click();
  await expect(page).toHaveURL("/tribes/demo/plats/nouveau");
  await expect(page.getByRole("main").getByRole("heading", { level: 1 })).toHaveText("Nouveau plat");

  // A creation not changed is no input to lose; once typed, it is.
  await card(page, "Chili con carne").click();
  await expect(page).toHaveURL(`/tribes/demo/plats/${chili.id}`);
  await library(page).getByRole("link", { name: "Nouveau", exact: true }).click();
  await page.getByRole("button", { name: "Ajouter un ingrédient" }).click();
  await page.getByRole("combobox", { name: "Ingrédient", exact: true }).fill("sel");
  await card(page, "Chili con carne").click();
  await expect(abandon(page)).toBeVisible();
});

test("the main list beside the list picker; an item typed, then another list: « Abandonner la modification ? » (D5, D9)", async ({ page }) => {
  await mock(page);
  await page.goto("/tribes/demo/courses");
  const main = page.getByRole("main");
  await expect(main.getByRole("heading", { level: 1 })).toHaveText(/Courses\s*principale/);
  await expect(main.getByRole("link", { name: /Changer de liste/ })).toHaveCount(0);
  await expect(picker(page).getByRole("heading", { name: "Listes", level: 2 })).toBeVisible();
  await expect(picker(page).getByRole("link", { name: "Retour à la liste" })).toHaveCount(0);
  await expect(picker(page).locator('[aria-current="page"] .list-card-name')).toHaveText("Courses");

  // The name of a new list stays while another list opens: the picker is not left.
  await picker(page).getByRole("button", { name: "Nouvelle liste" }).click();
  await picker(page).getByRole("textbox", { name: "Nom de la liste" }).fill("Drive");
  await picker(page).getByRole("link", { name: /^Marché/ }).click();
  await expect(abandon(page)).toHaveCount(0);
  await expect(page).toHaveURL(`/tribes/demo/courses/${marche.id}`);
  await expect(picker(page).locator('[aria-current="page"] .list-card-name')).toHaveText("Marché");
  await expect(picker(page).getByRole("textbox", { name: "Nom de la liste" })).toHaveValue("Drive");

  await main.getByRole("button", { name: "Ajouter un article à la main" }).click();
  await page.getByRole("combobox", { name: "Article", exact: true }).fill("pain");
  await picker(page).getByRole("link", { name: /^Courses/ }).click();
  await expect(abandon(page)).toBeVisible();
  await abandon(page).getByRole("button", { name: "Abandonner", exact: true }).click();
  await expect(page).toHaveURL("/tribes/demo/courses");

  // The name of a new list is lost by leaving the tab.
  await page.getByRole("navigation", { name: "Onglets" }).getByRole("link", { name: "Planning" }).click();
  await expect(abandon(page)).toBeVisible();
});

test("an item typed, then a new list created: « Abandonner la modification ? » before opening it (D9; review of PR #109, point 1)", async ({ page }) => {
  const drive = list(12, "Drive");
  await mockApi(page, {
    getSession: [replies.session],
    getTribe: [replies.tribe],
    // Then unchanged: the copy keeps the list created.
    getShoppingLists: [pageReply("shopping_lists", 7, [courses, marche]), notModified("shopping_lists", 8)],
    createList: [{ status: 201, body: { list: drive, versions: { shopping_lists: 8 } } }],
  });
  await page.goto("/tribes/demo/courses");
  const main = page.getByRole("main");
  await main.getByRole("button", { name: "Ajouter un article à la main" }).click();
  await page.getByRole("combobox", { name: "Article", exact: true }).fill("pain");
  await picker(page).getByRole("button", { name: "Nouvelle liste" }).click();
  await picker(page).getByRole("textbox", { name: "Nom de la liste" }).fill("Drive");
  await picker(page).getByRole("button", { name: "Créer la liste" }).click();
  await expect(abandon(page)).toBeVisible();

  // Kept: the list is created, not opened.
  await abandon(page).getByRole("button", { name: "Continuer la modification" }).click();
  await expect(page).toHaveURL("/tribes/demo/courses");
  await expect(page.getByRole("combobox", { name: "Article", exact: true })).toHaveValue("pain");
  await expect(picker(page).locator(".list-card-name")).toHaveText(["Courses", "Marché", "Drive"]);
  await expect(picker(page).locator('[aria-current="page"] .list-card-name')).toHaveText("Courses");
});

test("on a phone, an input is left without confirmation (D9)", async ({ page }) => {
  await page.setViewportSize(phone);
  await mock(page);
  await page.goto(`/tribes/demo/plats/${chili.id}`);
  await dishName(page).fill("Chili du dimanche");
  await page.getByRole("main").getByRole("link", { name: "Plats" }).click();
  await expect(page).toHaveURL("/tribes/demo/plats");
  await expect(abandon(page)).toHaveCount(0);
});

test("turning the tablet keeps the dish, its input and its scroll; the library comes beside (D8)", async ({ page }) => {
  await page.setViewportSize(portrait);
  await mock(page);
  await page.goto(`/tribes/demo/plats/${chili.id}`);
  await dishName(page).fill("Chili du dimanche");
  await page.evaluate(() => ((document.querySelector("mt-dish-screen") as unknown as { kept: boolean }).kept = true));
  // Out of the field: a browser may bring a field with the focus back into view.
  await dishName(page).blur();
  // A line at the top of what is seen: the top stays where it was, as a browser does when turned.
  const line = page.getByRole("button", { name: "Retirer l’ingrédient ingrédient 16" });
  await line.evaluate((e) => e.scrollIntoView({ block: "start" }));
  await expect(dishName(page)).not.toBeInViewport();

  await page.setViewportSize(landscape);
  await expect(library(page)).toBeVisible();
  // The same element: never created again by a change of format.
  expect(await page.evaluate(() => (document.querySelector("mt-dish-screen") as unknown as { kept?: boolean }).kept)).toBe(true);
  await expect(dishName(page)).toHaveValue("Chili du dimanche");
  // Scrolled with the window in moyen, on its own beside the library: still at the same line.
  await expect(line).toBeInViewport();
  await expect(dishName(page)).not.toBeInViewport();

  await page.setViewportSize(portrait);
  await expect(library(page)).toHaveCount(0);
  await expect(dishName(page)).toHaveValue("Chili du dimanche");
  await expect(line).toBeInViewport();
});

test("a dish saved in etendu stays open, with « Plat enregistré. », until it changes again (GE-Plats-*)", async ({ page }) => {
  const created = { id: id(3), name: "Curry", color: "#B0306A", servings: 4, revision: 1, ingredients: [] };
  await mockApi(page, {
    getSession: [replies.session],
    getTribe: [replies.tribe],
    // Then unchanged: the copy keeps what was saved.
    getDishes: [pageReply("dishes", 3, [chili, vege]), notModified("dishes", 5)],
    updateDish: [{ status: 200, body: { dish: { ...vege, name: "Chili vert", revision: 2 }, versions: { dishes: 4 } } }],
    createDish: [{ status: 201, body: { dish: created, versions: { dishes: 5 } } }],
  });
  const saved = page.getByRole("status").filter({ hasText: "Plat enregistré." });
  await page.goto(`/tribes/demo/plats/${vege.id}`);
  await dishName(page).fill("Chili vert");
  await page.getByRole("button", { name: "Enregistrer le plat" }).click();
  await expect(saved).toBeVisible();
  await expect(page).toHaveURL(`/tribes/demo/plats/${vege.id}`);
  await expect(page.getByRole("main").getByRole("heading", { level: 1 })).toHaveText("Chili vert");
  await expect(card(page, "Chili vert")).toHaveAttribute("aria-current", "page");
  await page.screenshot({ path: "test-results/volets-plat-enregistre.png" });
  // Saved, the dish holds no input to lose.
  await card(page, "Chili con carne").click();
  await expect(abandon(page)).toHaveCount(0);
  await card(page, "Chili vert").click();
  await expect(saved).toHaveCount(0);

  // A dish created opens at its path; the message goes at the first change.
  await library(page).getByRole("link", { name: "Nouveau", exact: true }).click();
  await dishName(page).fill("Curry");
  await page.getByRole("button", { name: "Enregistrer le plat" }).click();
  await expect(page).toHaveURL(`/tribes/demo/plats/${created.id}`);
  await expect(saved).toBeVisible();
  await expect(page.getByRole("main").getByRole("heading", { level: 1 })).toHaveText("Curry");
  await expect(card(page, "Curry")).toHaveAttribute("aria-current", "page");
  await dishName(page).fill("Curry vert");
  await expect(saved).toHaveCount(0);
});

test("a dish saved on a phone: back to the library", async ({ page }) => {
  await page.setViewportSize(phone);
  await mockApi(page, {
    getSession: [replies.session],
    getDishes: [pageReply("dishes", 3, [chili, vege])],
    updateDish: [{ status: 200, body: { dish: { ...vege, name: "Chili vert", revision: 2 }, versions: { dishes: 4 } } }],
  });
  await page.goto(`/tribes/demo/plats/${vege.id}`);
  await dishName(page).fill("Chili vert");
  await page.getByRole("button", { name: "Enregistrer le plat" }).click();
  await expect(page).toHaveURL("/tribes/demo/plats");
  await expect(page.getByRole("heading", { name: "Plats", level: 1 })).toBeVisible();
});
