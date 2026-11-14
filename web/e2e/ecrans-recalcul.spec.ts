// States of the screens of a stale list and of the taken quantity (Doux-ListeSuite,
// Doux-ChoixListesSuite; plan courses-suite, lot D), with a simulated API (plan socle, D15).
// The recomputation and the staleness are proved against the real server by godog; C8 on
// screen by its scenarios @ui (scenarios.spec.ts).
import { expect, type Page, test } from "@playwright/test";
import { type Received, type Reply, mockApi, pageReply, replies, staleReply, twoPanes } from "./api-mock.ts";

const id = (n: number) => n.toString(16).padStart(32, "0");

const chili = { id: id(100), name: "Chili con carne", color: "#c2502e", servings: 4, revision: 1, ingredients: [] };
const rata = { id: id(101), name: "Ratatouille", color: "#b36a0b", servings: 4, revision: 1, ingredients: [] };

const source = (servedDish: number, date: string, mealtime: string, servings: number, quantity: string, unit: string, dish = chili) => ({
  servedDish: id(servedDish),
  dish: dish.id,
  date,
  mealtime,
  servings,
  quantity,
  unit,
});
const boeuf = { id: id(1), ingredient: "bœuf haché", quantity: "875", unit: "g", checked: true, manual: false, sources: [source(50, "2026-10-06", "dinner", 3, "375", "g"), source(52, "2026-10-08", "lunch", 4, "500", "g")] };
const oignon = { id: id(2), ingredient: "oignon", quantity: "2", unit: "piece", checked: false, manual: false, sources: [source(50, "2026-10-06", "dinner", 3, "0.75", "piece"), source(52, "2026-10-08", "lunch", 4, "1", "piece")] };
const pain = { id: id(4), ingredient: "pain", quantity: "2", unit: "piece", checked: false, manual: true, sources: [] };
// After the recomputation: the chili of Tuesday for 6 servings, the beef taken in part (D9).
const boeufAfter = { ...boeuf, quantity: "1250", taken: "875", checked: false, sources: [source(50, "2026-10-06", "dinner", 6, "750", "g"), source(52, "2026-10-08", "lunch", 4, "500", "g")] };
const oignonAfter = { ...oignon, quantity: "3", sources: [source(50, "2026-10-06", "dinner", 6, "1.5", "piece"), source(52, "2026-10-08", "lunch", 4, "1", "piece")] };

const list = (n: number, name: string, items: unknown[], change: Record<string, unknown> = {}) => ({ id: id(n), name, main: false, rank: n, items, received: [] as string[], ...change });
const courses = list(10, "Courses", [boeuf, oignon, pain], { main: true, rank: 1, received: [id(50), id(52)] });
const recomputed = { ...courses, items: [boeufAfter, oignonAfter, pain] };
const marche = list(11, "Marché du samedi", [pain]);
const drive = list(12, "Drive", [{ ...oignon, id: id(5) }], { received: [id(50), id(52)] });

function mock(page: Page, extra: Parameters<typeof mockApi>[1] = {}): Promise<Received[]> {
  return mockApi(page, {
    getSession: [replies.session],
    getShoppingLists: [pageReply("shopping_lists", 7, [courses, marche, drive])],
    getDishes: [pageReply("dishes", 3, [chili, rata])],
    ...extra,
  });
}

const writes = (received: Received[]) => received.filter((r) => r.method !== "GET").map((r) => `${r.method} ${r.path} ${JSON.stringify(r.body ?? null)}`);
const itemTexts = (page: Page) => page.locator(".item").evaluateAll((all) => all.map((e) => (e.textContent ?? "").replace(/\s+/g, " ").trim()));
const staleText = "Des repas ont changé depuis l’ajout de leurs courses à cette liste.";

test("a stale list: the banner, « Recalculer » in progress, « Liste recalculée. », what remains to take (C8, D9, D13)", async ({ page }) => {
  const recompute: Reply = { status: 200, body: { list: recomputed, versions: { shopping_lists: 8 } }, delayMs: 300 };
  const received = await mock(page, { getStaleLists: [staleReply([courses.id])], recomputeList: [recompute] });
  await page.goto("/tribes/demo/courses");
  await expect(page.getByText(staleText)).toBeVisible();
  // Under the header, before the progress (docs/design/README.md).
  const order = await page.locator(".banner-stale, .progress").evaluateAll((all) => all.map((e) => e.className));
  expect(order).toEqual(["banner-stale", "progress"]);
  await page.screenshot({ path: "test-results/courses-perimee.png", fullPage: true });

  await page.getByRole("button", { name: "Recalculer" }).click();
  await expect(page.getByRole("button", { name: "Recalcul en cours…" })).toBeDisabled();
  await expect(page.getByRole("status").filter({ hasText: "Liste recalculée." })).toBeVisible();
  await expect(page.getByText(staleText)).toHaveCount(0);
  expect(writes(received)).toEqual([`POST shopping-lists/${courses.id}/recompute null`]);

  expect(await itemTexts(page)).toEqual(["bœuf haché encore 375 g 2 plats 1,25 kg", "oignon 2 plats 3 pièces", "pain ajouté 2 pièces"]);
  const row = page.getByRole("button", { name: "bœuf haché, 1,25 kg, encore 375 g à prendre, 2 plats. Afficher les repas" });
  await row.click();
  await expect(page.getByText("875 g déjà pris : il reste 375 g à prendre. Cochez quand tout est pris.")).toBeVisible();
  await page.screenshot({ path: "test-results/courses-part-prise.png", fullPage: true });
});

test("checking an item with a taken quantity erases it at once (D9)", async ({ page }) => {
  const checked = { ...recomputed, items: [{ ...boeufAfter, taken: undefined, checked: true }, oignonAfter, pain] };
  await mock(page, {
    getShoppingLists: [pageReply("shopping_lists", 8, [recomputed, marche, drive])],
    setChecked: [{ status: 200, body: { list: checked, versions: { shopping_lists: 9 } }, delayMs: 300 }],
  });
  await page.goto("/tribes/demo/courses");
  await expect(page.getByText("encore 375 g")).toBeVisible();
  await page.getByRole("checkbox", { name: "Cocher bœuf haché" }).click();
  await expect(page.getByText("encore 375 g")).toHaveCount(0);
  await expect(page.getByRole("checkbox", { name: "Cocher bœuf haché" })).toBeChecked();
});

test("« Recalculer » refused beyond 300 items: the message under the banner, still stale; a list deleted, named (Q38, D12)", async ({ page }) => {
  await mock(page, {
    // Read at the opening, after the refusal of 300 items, after the 404, then by the main list.
    getShoppingLists: [pageReply("shopping_lists", 7, [courses, marche, drive]), pageReply("shopping_lists", 7, [courses, marche, drive]), pageReply("shopping_lists", 9, [courses, marche])],
    getStaleLists: [staleReply([drive.id])],
    recomputeList: [{ status: 409, body: { error: "too_many_items" } }, { status: 404, body: { error: "not_found" } }],
  });
  await page.goto(`/tribes/demo/courses/${drive.id}`);
  await page.getByRole("button", { name: "Recalculer" }).click();
  await expect(page.getByRole("alert")).toHaveText("La liste ne peut pas dépasser 300 articles : rien n’a été recalculé.");
  await expect(page.getByText(staleText)).toBeVisible();
  const order = await page.locator(".banner-stale, .banner-error").evaluateAll((all) => all.map((e) => e.className));
  expect(order).toEqual(["banner-stale", "banner-error"]);

  await page.getByRole("button", { name: "Recalculer" }).click();
  await expect(page).toHaveURL(/\/tribes\/demo\/courses$/);
  await expect(page.getByRole("alert")).toHaveText("La liste « Drive » a été supprimée entre-temps. Rien n’a été recalculé.");
});

test("« Recalculer » without network: « Réseau nécessaire », the banner stays (D2)", async ({ page }) => {
  await mock(page, { getStaleLists: [staleReply([courses.id])], recomputeList: [replies.networkError] });
  await page.goto("/tribes/demo/courses");
  await page.getByRole("button", { name: "Recalculer" }).click();
  await expect(page.getByRole("alert")).toHaveText("Réseau nécessaire. Le réseau est lent ou absent : réessayez.");
  await expect(page.getByRole("button", { name: "Recalculer" })).toBeEnabled();
});

test("the list picker: « Repas modifiés » on a stale list only; gone once recomputed (C8)", async ({ page }) => {
  const twice = staleReply([courses.id, drive.id]);
  await mock(page, {
    // On a phone, read by the picker, by the list, then by the picker again; beside the list
    // in etendu, the picker is not checked again: the list recomputed leaves it at once.
    getStaleLists: twoPanes() ? [twice] : [twice, twice, staleReply([drive.id])],
    recomputeList: [{ status: 200, body: { list: recomputed, versions: { shopping_lists: 8 } } }],
  });
  await page.goto("/tribes/demo/courses/listes");
  const card = (name: string) => page.locator(".list-card").filter({ has: page.locator(".list-card-name", { hasText: new RegExp(`^${name}$`) }) });
  await expect(card("Courses")).toContainText("Repas modifiés");
  await expect(card("Drive")).toContainText("Repas modifiés");
  await expect(card("Marché du samedi")).not.toContainText("Repas modifiés");
  await page.screenshot({ path: "test-results/courses-listes-perimees.png", fullPage: true });

  await card("Courses").getByRole("link").click();
  await page.getByRole("button", { name: "Recalculer" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Liste recalculée." })).toBeVisible();
  if (!twoPanes()) {
    await page.getByRole("link", { name: /Changer de liste/ }).click();
  }
  await expect(card("Drive")).toContainText("Repas modifiés");
  await expect(card("Courses")).not.toContainText("Repas modifiés");
});

test("a source whose dish the copy of the library does not hold checks it once, then shows (plan courses-suite, D15)", async ({ page }) => {
  const received = await mock(page, { getDishes: [pageReply("dishes", 3, [rata]), pageReply("dishes", 4, [chili, rata])] });
  await page.goto("/tribes/demo/courses");
  await page.getByRole("button", { name: /^oignon, 2 pièces/ }).click();
  const meals = page.getByRole("list", { name: "Repas de oignon" });
  await expect(meals.getByRole("listitem")).toHaveCount(2);
  await expect(meals).toContainText("Chili con carne · 3 parts");
  expect(received.filter((r) => r.method === "GET" && r.path === "dishes")).toHaveLength(2);
});
