// States of the screens of the selection of items and its move (Doux-ListeSuite,
// GE-CoursesSuite-*; plan courses-suite, lot E), with a simulated API (plan socle, D15). The
// move and its refusals are proved against the real server by godog; C9 and C10 on screen by
// their scenarios @ui (scenarios.spec.ts).
import { expect, type Page, test } from "@playwright/test";
import { type Received, mockApi, pageReply, replies, staleReply, tabsOnSubScreen, twoPanes } from "./api-mock.ts";

const id = (n: number) => n.toString(16).padStart(32, "0");

const chili = { id: id(100), name: "Chili con carne", color: "#c2502e", servings: 4, revision: 1, ingredients: [] };
const source = { servedDish: id(50), dish: chili.id, date: "2026-10-06", mealtime: "dinner", servings: 4, quantity: "500", unit: "g" };
const boeuf = { id: id(1), ingredient: "bœuf haché", quantity: "500", unit: "g", checked: true, manual: false, sources: [source] };
const pain = { id: id(4), ingredient: "pain", quantity: "2", unit: "piece", checked: false, manual: true, sources: [] };
const lessive = { id: id(5), ingredient: "lessive", checked: false, manual: true, sources: [] };
const pommes = { id: id(6), ingredient: "pommes", quantity: "1.5", unit: "kg", checked: false, manual: true, sources: [] };

const list = (n: number, name: string, items: unknown[], change: Record<string, unknown> = {}) => ({ id: id(n), name, main: false, rank: n, items, received: [] as string[], ...change });
const courses = list(10, "Courses", [boeuf, pain, lessive], { main: true, rank: 1, received: [id(50)] });
const marche = list(11, "Marché du samedi", [pommes, { ...pommes, id: id(7), ingredient: "poires" }]);
const drive = list(12, "Drive", []);

function mock(page: Page, extra: Parameters<typeof mockApi>[1] = {}): Promise<Received[]> {
  return mockApi(page, {
    getSession: [replies.session],
    getShoppingLists: [pageReply("shopping_lists", 7, [courses, marche, drive])],
    getDishes: [pageReply("dishes", 3, [chili])],
    ...extra,
  });
}

const writes = (received: Received[]) => received.filter((r) => r.method !== "GET").map((r) => `${r.method} ${r.path} ${JSON.stringify(r.body ?? null)}`);
const actions = (page: Page) => page.getByRole("group", { name: "Actions de la sélection" });
const pick = (page: Page, name: string) => page.getByRole("button", { name: new RegExp(`^Sélectionner ${name}(,|$)`) });

test("the selection mode: header, items selected, « Tout » and « Aucun », the actions inactive without selection (C9, C10, D14)", async ({ page }) => {
  await mock(page, { getStaleLists: [staleReply([courses.id])] });
  await page.goto("/tribes/demo/courses");
  await page.getByRole("button", { name: "Sélectionner", exact: true }).click();
  await expect(page.getByRole("heading", { level: 1 })).toHaveText("Aucun article");
  await expect(page.getByRole("button", { name: "Annuler" })).toBeFocused();
  await expect(page.getByText("Choisissez les articles à sortir de « Courses ». Ils gardent leur coche et leurs plats.")).toBeVisible();
  // The stale banner, the progress, the meal shopping, the item added by hand and « Courses faites » are gone.
  for (const gone of [page.locator(".banner-stale"), page.locator(".progress"), page.getByRole("link", { name: "Ajouter les courses des repas" }), page.getByRole("button", { name: "Courses faites" })]) {
    await expect(gone).toHaveCount(0);
  }
  await expect(actions(page).getByRole("button", { name: "Déplacer vers…" })).toBeDisabled();
  await expect(actions(page).getByRole("button", { name: "Nouvelle liste" })).toBeDisabled();
  // In compact, the actions take the place of the tab bar; the rail stays in moyen and etendu.
  await expect(page.getByRole("navigation", { name: "Onglets" })).toBeVisible({ visible: tabsOnSubScreen() === 1 });
  await page.screenshot({ path: "test-results/selection-vide.png", fullPage: true });

  // A checked item is selected too, without unfolding.
  await pick(page, "bœuf haché").click();
  await pick(page, "pain").click();
  await expect(page.getByRole("heading", { level: 1 })).toHaveText("2 sélectionnés");
  await expect(pick(page, "bœuf haché")).toHaveAttribute("aria-pressed", "true");
  await expect(page.locator(".item-sources")).toHaveCount(0);
  await expect(actions(page).getByRole("button", { name: "Déplacer vers…" })).toBeEnabled();
  await page.screenshot({ path: "test-results/selection.png", fullPage: true });

  await page.getByRole("button", { name: "Tout" }).click();
  await expect(page.getByRole("heading", { level: 1 })).toHaveText("3 sélectionnés");
  await page.getByRole("button", { name: "Aucun" }).click();
  await expect(page.getByRole("heading", { level: 1 })).toHaveText("Aucun article");
  await pick(page, "pain").click();
  await expect(page.getByRole("heading", { level: 1 })).toHaveText("1 sélectionné");

  await page.getByRole("button", { name: "Annuler" }).click();
  await expect(page.getByRole("button", { name: "Sélectionner", exact: true })).toBeFocused();
  await expect(page.locator(".banner-stale")).toBeVisible();
  await expect(page.getByRole("navigation", { name: "Onglets" })).toBeVisible();
});

test("an empty list offers no selection", async ({ page }) => {
  await mock(page);
  await page.goto(`/tribes/demo/courses/${drive.id}`);
  await expect(page.getByText("La liste est vide.", { exact: false })).toBeVisible();
  await expect(page.getByRole("button", { name: "Sélectionner", exact: true })).toHaveCount(0);
});

test("« Déplacer vers… »: the other lists, then the banner whose link opens the target list (C9, D14)", async ({ page }) => {
  const moved = { list: { ...courses, items: [boeuf, lessive] }, target: { ...marche, items: [...marche.items, pain] }, moved: 1, missing: 0, versions: { shopping_lists: 8 } };
  const received = await mock(page, {
    // Read at the opening, then by the target list once opened.
    getShoppingLists: [pageReply("shopping_lists", 7, [courses, marche, drive]), pageReply("shopping_lists", 8, [moved.list, moved.target, drive])],
    moveItems: [{ status: 200, body: moved, delayMs: 200 }],
  });
  await page.goto("/tribes/demo/courses");
  await page.getByRole("button", { name: "Sélectionner", exact: true }).click();
  await pick(page, "pain").click();
  await actions(page).getByRole("button", { name: "Déplacer vers…" }).click();
  const sheet = page.getByRole("dialog", { name: "Déplacer 1 article vers…" });
  await expect(sheet).toContainText("Un article calculé rejoint la ligne du même ingrédient s’il y en a une ; un article ajouté à la main reste à part.");
  const targets = sheet.locator(".sheet-target");
  await expect(targets).toHaveText([/^\s*Marché du samedi\s*2 articles\s*$/, /^\s*Drive\s*Vide\s*$/]);
  await page.screenshot({ path: "test-results/selection-deplacer.png" });

  await targets.first().click();
  const banner = page.getByRole("status").filter({ hasText: "déplacé" });
  await expect(banner).toHaveText("1 article déplacé vers « Marché du samedi ».");
  await expect(sheet).toHaveCount(0);
  expect(writes(received)).toEqual([`POST shopping-lists/${courses.id}/move ${JSON.stringify({ items: [pain.id], to: marche.id })}`]);
  // Out of the selection mode, on the source list.
  await expect(page.getByRole("button", { name: "Sélectionner", exact: true })).toBeVisible();
  await expect(page.locator(".item-name")).toHaveText(["lessive", "bœuf haché"]);
  await page.screenshot({ path: "test-results/selection-apres-deplacement.png", fullPage: true });

  await banner.getByRole("link", { name: "« Marché du samedi »" }).click();
  await expect(page).toHaveURL(new RegExp(`/tribes/demo/courses/${marche.id}$`));
  await expect(page.locator(".item-name")).toHaveText(["pain", "poires", "pommes"]);
});

test("« Nouvelle liste »: its name, 20 lists in the sheet, then a list created empty when every item has gone (C10, Q38, Q42)", async ({ page }) => {
  const created = list(13, "Boucherie", [], { rank: 4 });
  const received = await mock(page, {
    moveItems: [
      { status: 409, body: { error: "too_many_lists" } },
      { status: 200, body: { list: { ...courses, items: [boeuf, pain] }, target: created, moved: 0, missing: 1, versions: { shopping_lists: 9 } } },
    ],
  });
  await page.goto("/tribes/demo/courses");
  await page.getByRole("button", { name: "Sélectionner", exact: true }).click();
  await pick(page, "lessive").click();
  await actions(page).getByRole("button", { name: "Nouvelle liste" }).click();
  const sheet = page.getByRole("dialog", { name: "Nouvelle liste" });
  await expect(sheet).toContainText("L’article sélectionné quittera « Courses » pour cette nouvelle liste.");
  const name = sheet.getByRole("textbox", { name: "Nom de la liste" });
  await expect(name).toBeFocused();
  await expect(name).toHaveAttribute("maxlength", "40");
  await expect(sheet.getByRole("button", { name: "Créer la liste" })).toBeDisabled();
  await name.fill("Boucherie");
  await page.screenshot({ path: "test-results/selection-nouvelle-liste.png" });
  await sheet.getByRole("button", { name: "Créer la liste" }).click();
  await expect(sheet.getByRole("alert")).toHaveText("20 listes au plus : supprimez une liste vide pour en créer une autre.");
  await expect(page.getByRole("heading", { level: 1 })).toHaveText("1 sélectionné");

  // Another member has removed « lessive » in between.
  await sheet.getByRole("button", { name: "Créer la liste" }).click();
  await expect(page.getByRole("status").filter({ hasText: "nouvelle liste" })).toHaveText("La nouvelle liste « Boucherie » a été créée, vide. 1 article n’était plus dans la liste.");
  expect(writes(received).at(-1)).toBe(`POST shopping-lists/${courses.id}/move ${JSON.stringify({ items: [lessive.id], name: "Boucherie" })}`);
});

test("refusals of a move: the target deleted in between, 300 items; the selection kept, an item gone left out (Q38, Q42, D14)", async ({ page }) => {
  await mock(page, {
    // Read at the opening, then after the refusal, which checks the copy: « Marché du samedi »
    // and « lessive » are gone.
    getShoppingLists: [pageReply("shopping_lists", 7, [courses, marche, drive]), pageReply("shopping_lists", 8, [{ ...courses, items: [boeuf, pain] }, drive])],
    moveItems: [
      { status: 404, body: { error: "target_not_found" } },
      { status: 409, body: { error: "too_many_items" } },
    ],
  });
  await page.goto("/tribes/demo/courses");
  await page.getByRole("button", { name: "Sélectionner", exact: true }).click();
  await pick(page, "pain").click();
  await pick(page, "lessive").click();
  await actions(page).getByRole("button", { name: "Déplacer vers…" }).click();
  await page.getByRole("dialog").locator(".sheet-target").filter({ hasText: "Marché du samedi" }).click();
  await expect(page.getByRole("alert")).toHaveText("La liste « Marché du samedi » a été supprimée entre-temps. Rien n’a été déplacé.");
  await expect(page.getByRole("heading", { level: 1 })).toHaveText("1 sélectionné");
  await page.screenshot({ path: "test-results/selection-arrivee-supprimee.png", fullPage: true });

  // The list deleted is no longer offered.
  await actions(page).getByRole("button", { name: "Déplacer vers…" }).click();
  await expect(page.getByRole("dialog").locator(".sheet-target-name")).toHaveText(["Drive"]);
  await page.getByRole("dialog").locator(".sheet-target").click();
  await expect(page.getByRole("alert")).toHaveText("La liste « Drive » ne peut pas dépasser 300 articles : rien n’a été déplacé.");
  await expect(pick(page, "pain")).toHaveAttribute("aria-pressed", "true");
});

test("a move without network: « Réseau nécessaire », the selection kept (D2)", async ({ page }) => {
  await mock(page, { moveItems: [replies.networkError] });
  await page.goto("/tribes/demo/courses");
  await page.getByRole("button", { name: "Sélectionner", exact: true }).click();
  await pick(page, "pain").click();
  await actions(page).getByRole("button", { name: "Déplacer vers…" }).click();
  await page.getByRole("dialog").locator(".sheet-target").first().click();
  await expect(page.getByRole("alert")).toHaveText("Réseau nécessaire. Le réseau est lent ou absent : réessayez.");
  await expect(page.getByRole("heading", { level: 1 })).toHaveText("1 sélectionné");
});

test("the actions at the bottom of the pane of the list, beside the picker in etendu (D14)", async ({ page }) => {
  await mock(page);
  await page.goto("/tribes/demo/courses");
  await page.getByRole("button", { name: "Sélectionner", exact: true }).click();
  const bar = await actions(page).boundingBox();
  const main = await page.getByRole("main").boundingBox();
  const viewport = page.viewportSize();
  expect(bar !== null && main !== null && viewport !== null).toBe(true);
  if (bar === null || main === null || viewport === null) {
    return;
  }
  expect(Math.abs(bar.y + bar.height - viewport.height)).toBeLessThanOrEqual(1);
  if (twoPanes()) {
    expect(bar.x).toBeGreaterThanOrEqual(main.x - 1);
    expect(bar.x + bar.width).toBeLessThanOrEqual(main.x + main.width + 1);
    // Touching another list leaves the selection, without confirmation.
    await page.locator("mt-list-picker .list-card-link").filter({ hasText: "Drive" }).click();
    await expect(page).toHaveURL(new RegExp(`/tribes/demo/courses/${drive.id}$`));
    await expect(actions(page)).toHaveCount(0);
  }
});

async function backToForeground(page: Page) {
  await page.evaluate(() => {
    for (const state of ["hidden", "visible"]) {
      Object.defineProperty(document, "visibilityState", { configurable: true, get: () => state });
      document.dispatchEvent(new Event("visibilitychange"));
    }
  });
}

test("a list emptied in between ends the selection mode; an item added afterwards does not bring it back (D14; review of PR #121, point 1)", async ({ page }) => {
  const emptied = { ...courses, items: [] };
  const added = { ...courses, items: [lessive] };
  await mock(page, {
    // Read at the opening, then at the return to the foreground: another member has emptied the list.
    getShoppingLists: [pageReply("shopping_lists", 7, [courses, marche, drive]), pageReply("shopping_lists", 8, [emptied, marche, drive])],
    addItem: [{ status: 201, body: { list: added, versions: { shopping_lists: 9 } } }],
  });
  await page.goto("/tribes/demo/courses");
  await page.getByRole("button", { name: "Sélectionner", exact: true }).click();
  await pick(page, "lessive").click();
  await actions(page).getByRole("button", { name: "Déplacer vers…" }).click();
  await expect(page.getByRole("dialog")).toBeVisible();

  await backToForeground(page);
  await expect(page.getByText("La liste est vide.", { exact: false })).toBeVisible();
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await expect(actions(page)).toHaveCount(0);

  await page.getByRole("button", { name: "Ajouter un article à la main" }).click();
  await page.getByRole("combobox", { name: "Article", exact: true }).fill("lessive");
  await page.getByRole("button", { name: "Ajouter à la liste" }).click();
  await expect(page.locator(".item-name")).toHaveText(["lessive"]);
  await expect(page.getByRole("button", { name: "Sélectionner", exact: true })).toBeVisible();
  await expect(actions(page)).toHaveCount(0);
});

test("an item gone from the copy leaves the selection, and comes back unselected (D14; review of PR #121, point 1)", async ({ page }) => {
  await mock(page, {
    // « pain » leaves the list, then comes back with its identifier (moved there and back, D10).
    getShoppingLists: [pageReply("shopping_lists", 7, [courses, marche, drive]), pageReply("shopping_lists", 8, [{ ...courses, items: [boeuf, lessive] }, marche, drive]), pageReply("shopping_lists", 9, [courses, marche, drive])],
  });
  await page.goto("/tribes/demo/courses");
  await page.getByRole("button", { name: "Sélectionner", exact: true }).click();
  await pick(page, "pain").click();
  await backToForeground(page);
  await expect(page.getByRole("heading", { level: 1 })).toHaveText("Aucun article");
  await backToForeground(page);
  await expect(pick(page, "pain")).toHaveAttribute("aria-pressed", "false");
  await expect(page.getByRole("heading", { level: 1 })).toHaveText("Aucun article");
});
