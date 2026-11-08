// Every screen at the edges of the formats (ENF-03; plan grands-ecrans, review of PR #106,
// point 6): from 320 to 2560 px wide, the format set on the root, and no horizontal scroll of
// the page. With a simulated API (plan socle, D15); the projects of the phones set their own
// window, so the widths are run there only, under both browsers.
import { expect, test } from "@playwright/test";
import { mockApi, pageReply, replies } from "./api-mock.ts";

const id = (n: number) => n.toString(16).padStart(32, "0");

const chili = { id: id(1), name: "Chili con carne avec un nom assez long pour être coupé", color: "#C2502E", servings: 4, revision: 1, ingredients: [{ name: "bœuf haché", quantity: "500", unit: "g" }, { name: "sel" }] };
const served = { id: id(2), date: "2026-10-06", mealtime: "dinner", dish: chili.id, servings: 4, rank: 1 };
const item = { id: id(3), ingredient: "pain", quantity: "2", unit: "piece", checked: false, manual: true, sources: [] };
const courses = { id: id(10), name: "Courses", main: true, rank: 1, items: [item], received: [] };
const marche = { id: id(11), name: "Marché du samedi", main: false, rank: 2, items: [], received: [] };

const widths: { width: number; format: string }[] = [
  { width: 320, format: "compact" },
  { width: 720, format: "moyen" },
  { width: 1023, format: "moyen" },
  { width: 1024, format: "etendu" },
  { width: 2560, format: "etendu" },
];

const screens = ["connexion", "", "?du=2026-10-05", "repas/2026-10-06/soir", "plats", "plats/nouveau", `plats/${chili.id}`, "courses", `courses/${marche.id}`, "courses/listes", "courses/repas", "tribu"];

test.beforeEach(({}, testInfo) => {
  test.skip(testInfo.project.metadata["format"] !== "compact", "the widths are set by the test, once per browser");
});

for (const { width, format } of widths) {
  test(`${width} px wide: the format ${format}, no horizontal scroll on any screen (ENF-03)`, async ({ page }) => {
    await page.clock.setFixedTime(new Date(2026, 9, 5, 10));
    await page.setViewportSize({ width, height: 900 });
    for (const screen of screens) {
      await mockApi(page, {
        getSession: [screen === "connexion" ? replies.noSession : replies.session],
        getDishes: [pageReply("dishes", 1, [chili])],
        getMealPlan: [pageReply("meal_plan", 1, [served])],
        getShoppingLists: [pageReply("shopping_lists", 1, [courses, marche])],
      });
      await page.goto(`/tribes/demo/${screen}`);
      await expect(page.locator("main, [role=main], .login").first()).toBeVisible();
      await expect(page.locator("html")).toHaveAttribute("data-format", format);
      const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
      expect(overflow, `horizontal scroll on /${screen}`).toBeLessThanOrEqual(0);
      await page.unrouteAll({ behavior: "ignoreErrors" });
    }
  });
}

test("the rail shows the reduced symbol, made for 48 px and less (docs/design/README.md)", async ({ page }) => {
  await page.setViewportSize({ width: 1024, height: 900 });
  await mockApi(page, { getSession: [replies.session] });
  await page.goto("/tribes/demo/");
  const symbol = page.locator("img.rail-symbol");
  await expect(symbol).toBeVisible();
  await expect(symbol).toHaveAttribute("src", /(^|\/)symbole-reduit(-[A-Z0-9]+)?\.svg$/);
  // The file served, not only its name.
  await expect.poll(() => symbol.evaluate((img: HTMLImageElement) => img.complete && img.naturalWidth > 0)).toBe(true);
});
