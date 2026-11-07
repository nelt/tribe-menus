// States of the screens of the shopping lists (Doux-Liste, Doux-ArticleMain, Doux-ChoixListes;
// plan courses, lot D), with a simulated API (plan socle, D15). The writes and their refusals
// are proved against the real server by godog; C1 and C2 on screen by their scenarios @ui
// (scenarios.spec.ts).
import { expect, type Page, test } from "@playwright/test";
import { type Received, type Reply, mockApi, pageReply, replies } from "./api-mock.ts";

const id = (n: number) => n.toString(16).padStart(32, "0");

const source = (servedDish: string, servings: number, quantity?: string, unit?: string) => ({
  servedDish,
  dish: id(100),
  date: "2026-10-06",
  mealtime: "dinner",
  servings,
  ...(quantity === undefined ? {} : { quantity, unit }),
});

const boeuf = { id: id(1), ingredient: "bœuf haché", quantity: "875", unit: "g", checked: false, manual: false, sources: [source(id(50), 3, "375", "g"), source(id(51), 4, "500", "g")] };
const creme = { id: id(2), ingredient: "crème", quantity: "500", unit: "ml", checked: true, manual: false, sources: [source(id(52), 4, "500", "ml")] };
const sel = { id: id(3), ingredient: "sel", checked: false, manual: false, sources: [source(id(52), 4)] };
const pain = { id: id(4), ingredient: "pain", quantity: "2", unit: "piece", checked: true, manual: true, sources: [] };
const lessive = { id: id(5), ingredient: "lessive", checked: false, manual: true, sources: [] };
const pommes = { id: id(6), ingredient: "pommes", quantity: "1.5", unit: "kg", checked: false, manual: true, sources: [] };

const list = (n: number, name: string, items: unknown[], change: Record<string, unknown> = {}) => ({ id: id(n), name, main: false, rank: n, items, received: [], ...change });
const courses = list(10, "Courses", [boeuf, sel, lessive, creme, pain], { main: true, rank: 1 });
const marche = list(11, "Marché du samedi", [pommes]);
const drive = list(12, "Drive", []);

const written = (body: Record<string, unknown>, status = 200): Reply => ({ status, body });
const notFound: Reply = { status: 404, body: { error: "not_found" } };

function mockLists(page: Page, lists: unknown[], extra: Parameters<typeof mockApi>[1] = {}): Promise<Received[]> {
  return mockApi(page, {
    getSession: [replies.session],
    getShoppingLists: [pageReply("shopping_lists", 7, lists)],
    getIngredients: [pageReply("ingredients", 3, [{ name: "pain", defaultUnit: "piece" }, { name: "pain de mie", defaultUnit: "piece" }, { name: "lait", defaultUnit: "l" }, { name: "lessive" }])],
    ...extra,
  });
}

const writes = (received: Received[]) => received.filter((r) => r.method !== "GET").map((r) => `${r.method} ${r.path} ${JSON.stringify(r.body ?? null)}`);
const items = (page: Page) => page.locator(".item");
const itemTexts = (page: Page) => items(page).evaluateAll((all) => all.map((e) => (e.textContent ?? "").replace(/\s+/g, " ").trim()));

test("the tab Courses opens the main list: name, progress, items in the order of Q40 (C1, C6)", async ({ page }) => {
  await mockLists(page, [marche, courses, drive]);
  await page.goto("/tribes/demo/");
  const tabs = page.getByRole("navigation", { name: "Onglets" });
  await tabs.getByRole("link", { name: "Courses" }).click();
  await expect(page).toHaveURL(/\/tribes\/demo\/courses$/);
  await expect(tabs.getByRole("link", { name: "Courses" })).toHaveAttribute("aria-current", "page");
  const title = page.getByRole("heading", { level: 1 });
  await expect(title).toHaveText(/Courses\s*principale/);
  await expect(title.getByRole("link", { name: "Courses, liste principale. Changer de liste" })).toHaveAttribute("href", "/tribes/demo/courses/listes");
  await expect(page.getByText("2 sur 5 articles")).toBeVisible();
  await expect(page.getByText("3 à prendre")).toBeVisible();
  expect(await itemTexts(page)).toEqual(["bœuf haché 2 plats 875 g", "lessive ajouté", "sel 1 plat", "crème 1 plat 50 cl", "pain ajouté 2 pièces"]);
  await expect(page.getByRole("checkbox", { name: "Cocher crème" })).toBeChecked();
  await expect(page.getByRole("checkbox", { name: "Cocher sel" })).not.toBeChecked();
  await expect(page.getByRole("button", { name: "Courses faites" })).toBeEnabled();
  await expect(page.getByText("Retire les 2 articles cochés de la liste ; les autres y restent.")).toBeVisible();
  await page.screenshot({ path: "test-results/courses-liste.png", fullPage: true });
});

test("no inline style: the CSP of the application refuses them", async ({ page }) => {
  const refused: string[] = [];
  page.on("console", (m) => {
    if (m.text().includes("Content Security Policy")) refused.push(m.text());
  });
  await mockLists(page, [courses, marche], { getShoppingLists: [{ ...pageReply("shopping_lists", 7, [courses, marche]), delayMs: 300 }] });
  await page.goto("/tribes/demo/courses");
  await expect(page.getByRole("status", { name: "Chargement de la liste" })).toBeVisible();
  await expect(page.getByText("2 sur 5 articles")).toBeVisible();
  expect(await page.locator("progress").evaluate((e: HTMLProgressElement) => e.position)).toBeCloseTo(0.4);
  await page.getByRole("link", { name: /Changer de liste/ }).click();
  await expect(page.getByRole("heading", { name: "Listes" })).toBeVisible();
  expect(refused).toEqual([]);
});

test("an empty list, another list, a list unknown brings back to the main list (C1, D15)", async ({ page }) => {
  await mockLists(page, [courses, drive, marche]);
  await page.goto(`/tribes/demo/courses/${drive.id}`);
  await expect(page.getByRole("heading", { level: 1 })).toHaveText("Drive");
  await expect(page.getByRole("link", { name: "Drive. Changer de liste" })).toBeVisible();
  await expect(page.getByText("La liste est vide. Ajoutez les courses des repas ou un article à la main.")).toBeVisible();
  await expect(page.locator("progress")).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Courses faites" })).toBeDisabled();
  await expect(page.getByText("Cochez les articles achetés pour les retirer de la liste.")).toBeVisible();

  await page.goto(`/tribes/demo/courses/${id(99)}`);
  await expect(page).toHaveURL(/\/tribes\/demo\/courses$/);
  await expect(page.getByRole("heading", { level: 1 })).toHaveText(/Courses/);
});

test("loading impossible: the message and « Réessayer »", async ({ page }) => {
  await mockLists(page, [], { getShoppingLists: [replies.networkError, pageReply("shopping_lists", 7, [courses])] });
  await page.goto("/tribes/demo/courses");
  await expect(page.getByRole("alert")).toHaveText("Le réseau est lent ou absent. Réessayez.");
  await page.getByRole("button", { name: "Réessayer" }).click();
  await expect(page.getByText("2 sur 5 articles")).toBeVisible();
});

test("an item checked goes down at once; without network it comes back, with « Réseau nécessaire » (C6, D2)", async ({ page }) => {
  const checked = { ...courses, items: [{ ...boeuf, checked: true }, sel, lessive, creme, pain] };
  const received = await mockLists(page, [courses], {
    setChecked: [{ ...written({ list: checked, versions: { shopping_lists: 8 } }), delayMs: 300 }, replies.networkError],
  });
  await page.goto("/tribes/demo/courses");
  await page.getByRole("checkbox", { name: "Cocher bœuf haché" }).check();
  expect((await itemTexts(page))[0]).toBe("lessive ajouté");
  await expect(page.getByText("3 sur 5 articles")).toBeVisible();
  expect(writes(received)).toEqual([`PUT shopping-lists/${courses.id}/items/${boeuf.id}/checked {"checked":true}`]);

  // A click, not check(): the box comes back at once, before check() could see it change.
  await page.getByRole("checkbox", { name: "Cocher sel" }).click();
  await expect(page.getByRole("alert")).toHaveText("Réseau nécessaire. Le réseau est lent ou absent : réessayez.");
  await expect(page.getByRole("checkbox", { name: "Cocher sel" })).not.toBeChecked();
  await expect(page.getByText("3 sur 5 articles")).toBeVisible();
});

test("add an item by hand: suggestions, default unit, refusals, the form stays open (C3, D23)", async ({ page }) => {
  const withPain = { ...courses, items: [...courses.items, { ...pain, id: id(7), checked: false, quantity: "3" }] };
  const received = await mockLists(page, [courses], {
    addItem: [written({ list: withPain, versions: { shopping_lists: 8 } }, 201)],
  });
  await page.goto("/tribes/demo/courses");
  await page.getByRole("button", { name: "Ajouter un article à la main" }).click();
  const article = page.getByRole("combobox", { name: "Article" });
  await expect(article).toBeFocused();
  await expect(page.getByText("Quantité facultative.")).toBeVisible();
  await page.getByRole("button", { name: "Ajouter à la liste" }).click();
  await expect(page.getByRole("alert")).toHaveText("Indiquez un article.");

  await article.fill("pai");
  const options = page.getByRole("listbox", { name: "Ingrédients proposés" }).getByRole("option");
  await expect(options).toHaveText([/pain\s*pièce/, /pain de mie\s*pièce/, "Créer « pai »"]);
  await options.first().click();
  await expect(article).toHaveValue("pain");
  await expect(page.getByText("Sans quantité, l’article s’affiche seul. Une quantité prend l’unité « pièce ».")).toBeVisible();
  await page.getByRole("textbox", { name: "Quantité" }).fill("3,1234");
  await expect(page.getByRole("combobox", { name: "Unité" })).toHaveValue("piece");
  await page.getByRole("button", { name: "Ajouter à la liste" }).click();
  await expect(page.getByRole("alert")).toHaveText("Quantité trop précise pour cette unité.");
  await page.getByRole("textbox", { name: "Quantité" }).fill("3");
  await page.screenshot({ path: "test-results/courses-article.png", fullPage: true });
  await page.getByRole("button", { name: "Ajouter à la liste" }).click();
  await expect(items(page)).toHaveCount(6);
  await expect(article).toHaveValue("");
  await expect(article).toBeFocused();
  expect(writes(received)).toEqual([`POST shopping-lists/${courses.id}/items {"ingredient":"pain","quantity":"3","unit":"piece"}`]);

  await page.getByRole("button", { name: "Annuler" }).click();
  await expect(page.getByRole("button", { name: "Ajouter un article à la main" })).toBeVisible();
});

test("300 items: no more item by hand (Q38)", async ({ page }) => {
  const full = { ...drive, items: [...Array(300).keys()].map((k) => ({ ...lessive, id: id(1000 + k), ingredient: `article ${k}` })) };
  await mockLists(page, [courses, full]);
  await page.goto(`/tribes/demo/courses/${drive.id}`);
  await expect(page.getByRole("button", { name: "Ajouter un article à la main" })).toBeDisabled();
  await expect(page.getByText("300 articles au plus.")).toBeVisible();
});

test("remove an item added; already removed, no message (C3, D11)", async ({ page }) => {
  const without = { ...courses, items: courses.items.filter((i) => i !== lessive) };
  await mockLists(page, [courses], {
    removeItem: [written({ list: without, versions: { shopping_lists: 8 } })],
  });
  await page.goto("/tribes/demo/courses");
  await page.getByRole("button", { name: "Retirer lessive" }).click();
  await expect(items(page)).toHaveCount(4);
  await expect(page.getByRole("alert")).toHaveCount(0);
});

test("« Courses faites »: the sheet counts, the items seen checked are sent, the green banner (C11, D9)", async ({ page }) => {
  const after = { ...courses, items: [boeuf, sel, lessive] };
  const received = await mockLists(page, [courses], {
    checkOut: [written({ list: after, removed: 2, versions: { shopping_lists: 8 } })],
  });
  await page.goto("/tribes/demo/courses");
  await page.getByRole("button", { name: "Courses faites" }).click();
  const sheet = page.getByRole("dialog", { name: "Courses faites ?" });
  await expect(sheet).toContainText("Les 2 articles cochés seront retirés de la liste. Les 3 articles non cochés y restent.");
  await page.screenshot({ path: "test-results/courses-faites.png" });
  await sheet.getByRole("button", { name: "Annuler" }).click();
  await expect(sheet).toHaveCount(0);
  await page.getByRole("button", { name: "Courses faites" }).click();
  await page.getByRole("button", { name: "Retirer les articles cochés" }).click();
  await expect(page.getByRole("status")).toHaveText("2 articles retirés");
  await expect(items(page)).toHaveCount(3);
  expect(writes(received)).toEqual([`POST shopping-lists/${courses.id}/checkout {"items":["${creme.id}","${pain.id}"]}`]);
  await page.getByRole("button", { name: "Fermer le message" }).click();
  await expect(page.getByText("2 articles retirés")).toHaveCount(0);
});

test("a write on a list deleted in between opens the main list, which tells it (D11)", async ({ page }) => {
  await mockLists(page, [courses, drive], {
    getShoppingLists: [pageReply("shopping_lists", 7, [courses, drive]), pageReply("shopping_lists", 9, [courses])],
    addItem: [notFound],
  });
  await page.goto(`/tribes/demo/courses/${drive.id}`);
  await page.getByRole("button", { name: "Ajouter un article à la main" }).click();
  await page.getByRole("combobox", { name: "Article" }).fill("lait");
  await page.getByRole("button", { name: "Ajouter à la liste" }).click();
  await expect(page).toHaveURL(/\/tribes\/demo\/courses$/);
  await expect(page.getByRole("alert")).toHaveText("Cette liste a été supprimée entre-temps.");
  await page.getByRole("checkbox", { name: "Cocher sel" }).click();
  await expect(page.getByText("Cette liste a été supprimée entre-temps.")).toHaveCount(0);
});

test("the list picker: order of C2, summaries, the sheet of each list (C1, C2, C12)", async ({ page }) => {
  await mockLists(page, [drive, marche, courses]);
  await page.goto(`/tribes/demo/courses/${marche.id}`);
  await page.getByRole("link", { name: "Marché du samedi. Changer de liste" }).click();
  await expect(page).toHaveURL(/\/tribes\/demo\/courses\/listes$/);
  await expect(page.getByRole("navigation", { name: "Onglets" }).getByRole("link", { name: "Courses" })).toHaveAttribute("aria-current", "page");
  const cards = page.locator(".list-card-link");
  expect(await cards.evaluateAll((all) => all.map((e) => (e.textContent ?? "").replace(/\s+/g, " ").trim()))).toEqual([
    "Courses principale 5 articles · 3 à prendre 2 sur 5",
    "Marché du samedi 1 article · 1 à prendre 0 sur 1",
    "Drive Vide",
  ]);
  await expect(page.getByRole("link", { name: "Retour à la liste" })).toHaveAttribute("href", `/tribes/demo/courses/${marche.id}`);
  await page.screenshot({ path: "test-results/courses-listes.png", fullPage: true });

  await page.getByRole("button", { name: "Actions pour la liste Courses" }).click();
  let sheet = page.getByRole("dialog", { name: "Courses" });
  await expect(sheet.getByRole("button", { name: "Définir comme liste principale" })).toHaveCount(0);
  await expect(sheet.getByRole("button", { name: /Supprimer la liste/ })).toBeDisabled();
  await expect(sheet).toContainText("Désignez d’abord une autre liste comme principale.");
  await sheet.getByRole("button", { name: "Fermer" }).click();

  await page.getByRole("button", { name: "Actions pour la liste Marché du samedi" }).click();
  sheet = page.getByRole("dialog", { name: "Marché du samedi" });
  await expect(sheet).toContainText("Une liste doit être vide pour être supprimée.");
  await expect(sheet.getByRole("button", { name: "Définir comme liste principale" })).toBeVisible();
  await page.screenshot({ path: "test-results/courses-feuille.png" });
  await page.keyboard.press("Escape");
  await expect(sheet).toHaveCount(0);

  await cards.nth(2).click();
  await expect(page).toHaveURL(new RegExp(`/courses/${drive.id}$`));
});

test("create a list: refusals, then it opens (C2, Q38)", async ({ page }) => {
  const boucherie = list(13, "Boucherie", []);
  const received = await mockLists(page, [courses], {
    getShoppingLists: [pageReply("shopping_lists", 7, [courses]), pageReply("shopping_lists", 8, [courses, boucherie])],
    createList: [{ status: 400, body: { error: "list_name_invalid" } }, written({ list: boucherie, versions: { shopping_lists: 8 } }, 201)],
  });
  await page.goto("/tribes/demo/courses/listes");
  await page.getByRole("button", { name: "Nouvelle liste" }).click();
  const name = page.getByRole("textbox", { name: "Nom de la liste" });
  await expect(name).toBeFocused();
  await expect(name).toHaveAttribute("maxlength", "40");
  await page.getByRole("button", { name: "Créer la liste" }).click();
  await expect(page.getByRole("alert")).toHaveText("Donnez un nom à la liste.");
  await name.fill("Boucherie");
  await page.getByRole("button", { name: "Créer la liste" }).click();
  await expect(page.getByRole("alert")).toHaveText("Ce nom contient un caractère invalide.");
  await page.getByRole("button", { name: "Créer la liste" }).click();
  await expect(page).toHaveURL(new RegExp(`/courses/${boucherie.id}$`));
  await expect(page.getByRole("heading", { level: 1 })).toHaveText("Boucherie");
  expect(writes(received)).toEqual(['POST shopping-lists {"name":"Boucherie"}', 'POST shopping-lists {"name":"Boucherie"}']);
});

test("20 lists: no new list (Q38)", async ({ page }) => {
  await mockLists(page, [courses, ...[...Array(19).keys()].map((k) => list(20 + k, `Liste ${k}`, []))]);
  await page.goto("/tribes/demo/courses/listes");
  await expect(page.getByRole("button", { name: "Nouvelle liste" })).toBeDisabled();
  await expect(page.getByText("20 listes au plus : supprimez une liste vide pour en créer une autre.")).toBeVisible();
});

test("rename, make main, delete from the sheet (C1, C2, C12)", async ({ page }) => {
  const after = [{ ...courses, main: false }, { ...marche, name: "Marché", main: true }];
  const received = await mockLists(page, [courses, marche, drive], {
    getShoppingLists: [pageReply("shopping_lists", 7, [courses, marche, drive]), pageReply("shopping_lists", 10, after)],
    renameList: [written({ list: { ...marche, name: "Marché" }, versions: { shopping_lists: 8 } })],
    setMainList: [written({ versions: { shopping_lists: 9 } })],
    deleteList: [written({ versions: { shopping_lists: 10 } })],
  });
  await page.goto("/tribes/demo/courses/listes");
  await page.getByRole("button", { name: "Actions pour la liste Marché du samedi" }).click();
  await page.getByRole("button", { name: "Renommer" }).click();
  const name = page.getByRole("dialog").getByRole("textbox", { name: "Nom de la liste" });
  await expect(name).toHaveValue("Marché du samedi");
  await name.fill("Marché");
  await page.getByRole("button", { name: "Enregistrer" }).click();
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await page.getByRole("button", { name: "Actions pour la liste Marché" }).click();
  await page.getByRole("button", { name: "Définir comme liste principale" }).click();
  await expect(page.locator(".list-card-name")).toHaveText(["Marché", "Courses", "Drive"]);
  await page.getByRole("button", { name: "Actions pour la liste Drive" }).click();
  await page.getByRole("button", { name: "Supprimer la liste" }).click();
  await expect(page.locator(".list-card-name")).toHaveText(["Marché", "Courses"]);
  expect(writes(received)).toEqual([
    `PUT shopping-lists/${marche.id}/name {"name":"Marché"}`,
    `PUT shopping-lists/main {"list":"${marche.id}"}`,
    `DELETE shopping-lists/${drive.id} null`,
  ]);
  // The tab opens the new main list.
  await page.getByRole("navigation", { name: "Onglets" }).getByRole("link", { name: "Courses" }).click();
  await expect(page.getByRole("heading", { level: 1 })).toHaveText(/Marché\s*principale/);
});

test("delete a list no longer empty: the sheet tells it, with the list up to date (D11); without network (D2)", async ({ page }) => {
  const full = { ...drive, items: [pain] };
  await mockLists(page, [courses, drive], {
    getShoppingLists: [pageReply("shopping_lists", 7, [courses, drive]), pageReply("shopping_lists", 8, [courses, full])],
    deleteList: [{ status: 409, body: { error: "list_not_empty", list: full } }],
    setMainList: [replies.networkError],
  });
  await page.goto("/tribes/demo/courses/listes");
  await page.getByRole("button", { name: "Actions pour la liste Drive" }).click();
  await page.getByRole("button", { name: "Supprimer la liste" }).click();
  const sheet = page.getByRole("dialog", { name: "Drive" });
  await expect(sheet.getByRole("alert")).toHaveText("Cette liste n’est plus vide.");
  await expect(sheet).toContainText("Une liste doit être vide pour être supprimée.");
  await expect(page.locator(".list-card-summary").nth(1)).toHaveText("1 article · 0 à prendre");
  await sheet.getByRole("button", { name: "Définir comme liste principale" }).click();
  await expect(sheet).toHaveCount(0);
  await expect(page.getByRole("alert")).toHaveText("Réseau nécessaire. Le réseau est lent ou absent : réessayez.");
});
