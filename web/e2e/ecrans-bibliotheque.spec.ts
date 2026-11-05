// States of the screen Plats (Doux-Bibliotheque), with a simulated API (plan socle, D15). The
// search itself is proved by the scenario P3 against the real server (scenarios.spec.ts).
import { expect, test } from "@playwright/test";
import { mockApi, notModified, pageReply, replies } from "./api-mock.ts";

const dish = (n: number, name: string, ingredients: string[], color = "#C2502E", servings = 4) => ({
  id: n.toString(16).padStart(32, "0"),
  name,
  color,
  servings,
  revision: 1,
  ingredients: ingredients.map((i) => ({ name: i })),
});

const library = [
  dish(1, "Ratatouille", ["courgette", "aubergine"], "#B36A0B"),
  dish(2, "Gratin de courgettes", ["courgette", "crème"], "#2F6FB0", 6),
  dish(3, "Chili con carne", ["bœuf haché", "haricots rouges"]),
];

test("the library, by alphabetical order, and the search", async ({ page }) => {
  await mockApi(page, { getSession: [replies.session], getDishes: [pageReply("dishes", 3, library)] });
  await page.goto("/tribes/demo/plats");
  await expect(page.getByRole("heading", { name: "Plats" })).toBeVisible();
  await expect(page.getByText("3 plats")).toBeVisible();
  const names = page.locator(".dish-name");
  await expect(names).toHaveText(["Chili con carne", "Gratin de courgettes", "Ratatouille"]);
  await expect(page.getByRole("link", { name: /Gratin de courgettes/ })).toContainText("pour 6");
  await expect(page.getByRole("link", { name: /Gratin de courgettes/ })).toHaveAttribute("href", `/tribes/demo/plats/${"2".padStart(32, "0")}`);
  await expect(page.getByRole("link", { name: "Nouveau" })).toHaveAttribute("href", "/tribes/demo/plats/nouveau");

  const search = page.getByRole("searchbox", { name: "Rechercher un plat ou un ingrédient" });
  await search.fill("courg");
  await expect(page.getByText("2 résultats pour « courg »")).toBeVisible();
  await expect(names).toHaveText(["Gratin de courgettes", "Ratatouille"]);
  // The ingredients found stand out.
  await expect(page.locator(".chip--found")).toHaveText(["courgette", "courgette"]);

  await search.fill("chili");
  await expect(page.getByText("1 résultat pour « chili »")).toBeVisible();
  await search.fill("poulet");
  await expect(page.getByText("0 résultat pour « poulet »")).toBeVisible();
  await expect(page.getByText("Aucun plat ne correspond")).toBeVisible();
  await expect(names).toHaveCount(0);
});

test("an empty library", async ({ page }) => {
  await mockApi(page, { getSession: [replies.session] });
  await page.goto("/tribes/demo/plats");
  await expect(page.getByText("Aucun plat pour l'instant")).toBeVisible();
  await expect(page.getByRole("link", { name: "Nouveau" })).toBeVisible();
});

test("the library comes by pages, the next ones with If-Match (ADR 0029)", async ({ page }) => {
  const received = await mockApi(page, {
    getSession: [replies.session],
    getDishes: [pageReply("dishes", 7, library.slice(0, 2), "b"), pageReply("dishes", 7, library.slice(2))],
  });
  await page.goto("/tribes/demo/plats");
  await expect(page.getByText("3 plats")).toBeVisible();
  const dishes = received.filter((r) => r.path === "dishes");
  expect(dishes).toEqual([
    { method: "GET", path: "dishes", body: undefined },
    { method: "GET", path: "dishes", body: undefined, query: "?after=b", ifMatch: '"dishes-7"' },
  ]);
});

test("each visit checks the copy: unchanged, one request and a 304 (D25)", async ({ page, browserName }) => {
  test.skip(browserName === "webkit", "Playwright cannot give a 304 back to WebKit");
  const received = await mockApi(page, {
    getSession: [replies.session],
    getDishes: [pageReply("dishes", 3, library), notModified("dishes", 3)],
    getIngredients: [pageReply("ingredients", 2, [{ name: "courgette" }]), notModified("ingredients", 2)],
  });
  await page.goto("/tribes/demo/plats");
  await expect(page.getByText("3 plats")).toBeVisible();
  // The Planning checks the library too, with the meal plan (plan planning, D15).
  await page.getByRole("link", { name: "Planning" }).click();
  await expect(page.locator(".day")).toHaveCount(7);
  await page.getByRole("link", { name: "Plats" }).click();
  await expect(page.getByText("3 plats")).toBeVisible();
  const checks = () => received.filter((r) => r.path === "dishes" || r.path === "ingredients");
  const unchanged = [
    { method: "GET", path: "dishes", body: undefined, ifNoneMatch: '"dishes-3"' },
    { method: "GET", path: "ingredients", body: undefined, ifNoneMatch: '"ingredients-2"' },
  ];
  await expect.poll(() => checks().slice(2)).toEqual([...unchanged, ...unchanged]);
});

test("without network: the message and “Réessayer”", async ({ page }) => {
  await mockApi(page, { getSession: [replies.session], getDishes: [replies.networkError, pageReply("dishes", 1, library)] });
  await page.goto("/tribes/demo/plats");
  await expect(page.getByRole("alert")).toHaveText("Le réseau est lent ou absent. Réessayez.");
  await page.getByRole("button", { name: "Réessayer" }).click();
  await expect(page.getByText("3 plats")).toBeVisible();
  await expect(page.getByRole("alert")).toHaveCount(0);
});

test("a 401 during the loading goes back to the login", async ({ page }) => {
  await mockApi(page, { getSession: [replies.session], getDishes: [replies.noSession] });
  await page.goto("/tribes/demo/plats");
  await expect(page).toHaveURL(/\/tribes\/demo\/connexion$/);
});
