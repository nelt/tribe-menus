// States of the screen of a dish (Doux-Plat, Doux-PlatModif, Doux-PlatConflit), with a
// simulated API (plan socle, D15). The suggestions, the default unit and the size of the
// tribe proposed are proved by the scenarios P4, T1 and T3 against the real server
// (scenarios.spec.ts).
import { expect, test } from "@playwright/test";
import { mockApi, pageReply, replies, tabsOnSubScreen, twoPanes } from "./api-mock.ts";

const id = (n: number) => n.toString(16).padStart(32, "0");

const chili = {
  id: id(1),
  name: "Chili con carne",
  color: "#C2502E",
  servings: 4,
  revision: 2,
  ingredients: [
    { name: "bœuf haché", quantity: "500", unit: "g" },
    { name: "oignon", quantity: "2", unit: "piece" },
    { name: "sel" },
  ],
};

const catalog = [
  { name: "tomate", defaultUnit: "piece" },
  { name: "tomate concassée", defaultUnit: "g" },
  { name: "bœuf haché", defaultUnit: "g" },
  { name: "oignon", defaultUnit: "piece" },
  { name: "sel" },
];

const tribeOf5 = { status: 200, body: { name: "Les Démo", size: 5, activeMembers: 2 } };

test("a new dish: the size of the tribe, an ingredient chosen, another created, saved", async ({ page }) => {
  const curry = {
    id: id(2),
    name: "Curry",
    color: "#3F7F25",
    servings: 5,
    revision: 1,
    ingredients: [
      { name: "tomate", quantity: "4", unit: "piece", defaultUnit: "piece" },
      { name: "gingembre", defaultUnit: undefined },
    ],
  };
  const received = await mockApi(page, {
    getSession: [replies.session],
    getTribe: [tribeOf5],
    getDishes: [pageReply("dishes", 1, [chili]), pageReply("dishes", 2, [chili, curry])],
    getIngredients: [pageReply("ingredients", 1, catalog), pageReply("ingredients", 2, [...catalog, { name: "gingembre" }])],
    createDish: [{ status: 201, body: { dish: curry, versions: { dishes: 2, ingredients: 2 } } }],
  });
  await page.goto("/tribes/demo/plats");
  await page.getByRole("link", { name: "Nouveau", exact: true }).click();
  await expect(page).toHaveURL(/\/tribes\/demo\/plats\/nouveau$/);
  await expect(page.getByRole("heading", { name: "Nouveau plat" })).toBeVisible();
  await expect(page.getByRole("navigation", { name: "Onglets" })).toHaveCount(tabsOnSubScreen());
  const stepper = page.getByRole("group", { name: "Pour combien de parts ?" });
  await expect(stepper).toContainText("5");

  await page.getByLabel("Nom du plat").fill("Curry");
  await page.getByRole("button", { name: "Ajouter un ingrédient" }).click();
  const combobox = page.getByRole("combobox", { name: "Ingrédient", exact: true });
  await expect(combobox).toBeFocused();
  await combobox.fill("tom");
  const options = page.getByRole("listbox", { name: "Ingrédients proposés" }).getByRole("option");
  await expect(options).toHaveText(["tomate pièce", "tomate concassée g", "Créer « tom »"]);
  await expect(combobox).toHaveAttribute("aria-expanded", "true");
  await options.first().click();
  await expect(page.locator(".line-name")).toHaveText(["tomate"]);
  await expect(page.getByRole("combobox", { name: "Ingrédient", exact: true })).toHaveCount(0);

  // Without quantity, the unit is empty and inactive; a quantity takes the default unit.
  const unit = page.getByLabel("Unité de tomate");
  await expect(unit).toBeDisabled();
  await page.getByLabel("Quantité de tomate").fill("4");
  await expect(unit).toHaveValue("piece");
  await expect(page.locator(".line-unit-label").first()).toHaveText("pièces");

  // An ingredient already in the dish is not proposed (Q27).
  await page.getByRole("button", { name: "Ajouter un ingrédient" }).click();
  await combobox.fill("tomate");
  await expect(options).toHaveText(["tomate concassée g"]);
  await combobox.fill("gingembre");
  await expect(options).toHaveText(["Créer « gingembre »"]);
  await combobox.press("ArrowDown");
  await combobox.press("Enter");
  await expect(page.locator(".line-name")).toHaveText(["tomate", "gingembre"]);

  await page.getByRole("button", { name: "Enregistrer le plat" }).click();
  await expect(page).toHaveURL(/\/tribes\/demo\/plats$/);
  await expect(page.locator(".dish-name")).toHaveText(["Chili con carne", "Curry"]);
  const created = received.filter((r) => r.method === "POST" && r.path === "dishes");
  expect(created.map((r) => r.body)).toEqual([
    { name: "Curry", servings: 5, ingredients: [{ name: "tomate", quantity: "4", unit: "piece" }, { name: "gingembre" }] },
  ]);
});

test("a new dish: the messages, before sending and from the API, and without network", async ({ page }) => {
  const received = await mockApi(page, {
    getSession: [replies.session],
    getIngredients: [pageReply("ingredients", 1, catalog)],
    createDish: [{ status: 400, body: { error: "name_taken" } }, replies.networkError],
  });
  await page.goto("/tribes/demo/plats/nouveau");
  await page.getByRole("button", { name: "Enregistrer le plat" }).click();
  await expect(page.getByRole("alert")).toHaveText("Donnez un nom au plat.");
  await expect(page.getByLabel("Nom du plat")).toHaveAttribute("aria-invalid", "true");

  await page.getByLabel("Nom du plat").fill("Chili con carne");
  await page.getByRole("button", { name: "Ajouter un ingrédient" }).click();
  await page.getByLabel("Quantité", { exact: true }).fill("2");
  await page.getByRole("combobox", { name: "Ingrédient", exact: true }).fill("cumin");
  await page.getByRole("option", { name: "Créer « cumin »" }).click();
  // A new ingredient has no default unit: none is chosen (docs/design/README.md).
  await expect(page.getByLabel("Unité de cumin")).toHaveValue("");
  await page.getByRole("button", { name: "Enregistrer le plat" }).click();
  await expect(page.getByRole("alert")).toHaveText("Choisissez une unité.");

  await page.getByLabel("Unité de cumin").selectOption("teaspoon");
  await expect(page.locator(".line-unit-label")).toHaveText("c. à café");
  await page.getByLabel("Quantité de cumin").fill("2,5555");
  await page.getByRole("button", { name: "Enregistrer le plat" }).click();
  await expect(page.getByRole("alert")).toHaveText("Quantité trop précise pour cette unité.");
  expect(received.filter((r) => r.method === "POST")).toHaveLength(0);

  await page.getByLabel("Quantité de cumin").fill("2");
  await page.getByRole("button", { name: "Enregistrer le plat" }).click();
  await expect(page.getByRole("alert")).toHaveText("Un plat porte déjà ce nom.");

  await page.getByRole("button", { name: "Enregistrer le plat" }).click();
  await expect(page.getByRole("alert")).toHaveText("Réseau nécessaire. Le réseau est lent ou absent : réessayez.");
  await expect(page).toHaveURL(/\/plats\/nouveau$/);
  await expect(page.getByLabel("Nom du plat")).toHaveValue("Chili con carne");
});

test("a dish refused on a line: the message under the line", async ({ page }) => {
  await mockApi(page, {
    getSession: [replies.session],
    getDishes: [pageReply("dishes", 1, [chili])],
    updateDish: [{ status: 400, body: { error: "duplicate_ingredient", line: 1, ingredient: "bœuf haché" } }],
  });
  await page.goto(`/tribes/demo/plats/${chili.id}`);
  await page.getByRole("button", { name: "Enregistrer le plat" }).click();
  const line = page.locator(".line").nth(1);
  await expect(line.getByRole("alert")).toHaveText("« bœuf haché » figure déjà dans le plat.");
});

test("a draft refused as a duplicate can be removed, an empty one has nothing to remove (recette du plan bibliotheque)", async ({ page }) => {
  const received = await mockApi(page, {
    getSession: [replies.session],
    getDishes: [pageReply("dishes", 1, [chili]), pageReply("dishes", 2, [chili])],
    getIngredients: [pageReply("ingredients", 1, catalog)],
    updateDish: [{ status: 200, body: { dish: { ...chili, revision: 3 }, versions: { dishes: 2 } } }],
  });
  await page.goto(`/tribes/demo/plats/${chili.id}`);
  await page.getByRole("button", { name: "Ajouter un ingrédient" }).click();
  const remove = page.getByRole("button", { name: "Retirer l’ingrédient en cours de saisie" });
  await expect(remove).toHaveCount(0);

  await page.getByRole("combobox", { name: "Ingrédient", exact: true }).fill("Oignon");
  await page.getByLabel("Quantité", { exact: true }).fill("1");
  await page.getByLabel("Unité", { exact: true }).selectOption("piece");
  await page.getByRole("button", { name: "Enregistrer le plat" }).click();
  const draft = page.locator(".line").nth(3);
  await expect(draft.getByRole("alert")).toHaveText("« oignon » figure déjà dans le plat.");

  // Refused before sending, as during the recette.
  expect(received.filter((r) => r.method === "PUT")).toHaveLength(0);
  await remove.click();
  await expect(page.locator(".line")).toHaveCount(3);
  await expect(page.getByRole("alert")).toHaveCount(0);
  await page.getByRole("button", { name: "Enregistrer le plat" }).click();
  await expect(page).toHaveURL(/\/tribes\/demo\/plats$/);
  const updates = received.filter((r) => r.method === "PUT");
  expect(updates.map((r) => (r.body as { ingredients: unknown[] }).ingredients.length)).toEqual([3]);
});

test("without quantity, the default unit of the ingredient chosen is announced, not chosen (D4)", async ({ page }) => {
  await mockApi(page, {
    getSession: [replies.session],
    getDishes: [pageReply("dishes", 1, [chili])],
    getIngredients: [pageReply("ingredients", 1, catalog)],
  });
  await page.goto("/tribes/demo/plats/nouveau");
  await page.getByRole("button", { name: "Ajouter un ingrédient" }).click();
  const combobox = page.getByRole("combobox", { name: "Ingrédient", exact: true });
  await combobox.fill("tomate");
  // Typed, not chosen: nothing announced yet.
  const line = page.locator(".line").nth(0);
  await expect(line.locator(".line-unit-label")).toHaveText("");
  await page.getByRole("option", { name: "tomate pièce", exact: true }).click();

  const hint = line.locator(".line-unit-label");
  await expect(hint).toHaveText("pièce");
  await expect(hint).toHaveClass(/line-unit-label--hint/);
  const unit = page.getByLabel("Unité de tomate");
  await expect(unit).toBeDisabled();
  await expect(unit).toHaveValue("");

  await page.getByLabel("Quantité de tomate").fill("2");
  await expect(unit).toBeEnabled();
  await expect(unit).toHaveValue("piece");
  await expect(hint).not.toHaveClass(/line-unit-label--hint/);

  // An ingredient without default unit announces nothing.
  await page.getByRole("button", { name: "Ajouter un ingrédient" }).click();
  await combobox.fill("sel");
  await page.getByRole("option", { name: "sel", exact: true }).click();
  await expect(page.locator(".line").nth(1).locator(".line-unit-label")).toHaveText("");
});

test("a control character pasted in a name: refused before sending, and from the API (D30)", async ({ page }) => {
  const received = await mockApi(page, {
    getSession: [replies.session],
    getDishes: [pageReply("dishes", 1, [chili])],
    updateDish: [{ status: 400, body: { error: "ingredient_name_invalid", line: 0 } }],
  });
  await page.goto("/tribes/demo/plats/nouveau");
  await page.getByLabel("Nom du plat").fill("Soupe\u0007");
  await page.getByRole("button", { name: "Enregistrer le plat" }).click();
  await expect(page.getByRole("alert")).toHaveText("Ce nom contient un caractère invalide.");
  await expect(page.getByLabel("Nom du plat")).toHaveAttribute("aria-invalid", "true");
  expect(received.filter((r) => r.method === "POST")).toHaveLength(0);

  // A refusal the screen did not foresee is shown with its message, not as a network error.
  await page.goto(`/tribes/demo/plats/${chili.id}`);
  await page.getByRole("button", { name: "Enregistrer le plat" }).click();
  const line = page.locator(".line").nth(0);
  await expect(line.getByRole("alert")).toHaveText("Ce nom contient un caractère invalide.");
});

test("a draft without visible character: nothing to create, and dropped as an empty one (D30)", async ({ page }) => {
  const soup = { id: "s", name: "Soupe", color: "#C2502E", servings: 4, revision: 1, ingredients: [] };
  const received = await mockApi(page, {
    getSession: [replies.session],
    getDishes: [pageReply("dishes", 1, [chili]), pageReply("dishes", 2, [chili, soup])],
    getIngredients: [pageReply("ingredients", 1, catalog)],
    createDish: [{ status: 201, body: { dish: soup, versions: { dishes: 2, ingredients: 1 } } }],
  });
  await page.goto("/tribes/demo/plats/nouveau");
  await page.getByLabel("Nom du plat").fill("Soupe");
  await page.getByRole("button", { name: "Ajouter un ingrédient" }).click();
  // The quantity field is bounded: zeros pasted by thousands would not fit in the body read by the server.
  await expect(page.getByLabel("Quantité", { exact: true })).toHaveAttribute("maxlength", "10");
  const combobox = page.getByRole("combobox", { name: "Ingrédient", exact: true });
  await combobox.fill("\u200b");
  await expect(page.getByRole("listbox", { name: "Ingrédients proposés" }).getByRole("option")).toHaveCount(0);
  await page.getByRole("button", { name: "Enregistrer le plat" }).click();
  await expect(page).toHaveURL(/\/tribes\/demo\/plats$/);
  const created = received.filter((r) => r.method === "POST" && r.path === "dishes");
  // The tribe size of the mock (4) is read: the servings are sent (D16).
  expect(created.map((r) => r.body)).toEqual([{ name: "Soupe", servings: 4, ingredients: [] }]);
});

test("changing a dish: its form, then the change sent on its revision", async ({ page }) => {
  const saved = { ...chili, revision: 3, servings: 5 };
  const received = await mockApi(page, {
    getSession: [replies.session],
    getDishes: [pageReply("dishes", 1, [chili]), pageReply("dishes", 2, [saved])],
    getIngredients: [pageReply("ingredients", 1, catalog)],
    updateDish: [{ status: 200, body: { dish: saved, versions: { dishes: 2 } } }],
  });
  await page.goto("/tribes/demo/plats");
  await page.getByRole("link", { name: /Chili con carne/ }).click();
  await expect(page).toHaveURL(new RegExp(`/plats/${chili.id}$`));
  await expect(page.getByRole("heading", { name: "Chili con carne" })).toBeVisible();
  await expect(page.locator(".dish-title .dish-initial")).toHaveText("C");
  await expect(page.getByLabel("Nom du plat")).toHaveValue("Chili con carne");
  await expect(page.getByRole("group", { name: "Pour combien de parts ?" })).toContainText("4");
  await expect(page.locator(".line-name")).toHaveText(["bœuf haché", "oignon", "sel"]);
  await expect(page.getByLabel("Quantité de bœuf haché")).toHaveValue("500");
  await expect(page.locator(".line-unit-label")).toHaveText(["g", "pièces", ""]);
  await expect(page.getByLabel("Unité de sel")).toBeDisabled();

  await page.getByRole("button", { name: "Une part de plus" }).click();
  await page.getByRole("button", { name: "Retirer l’ingrédient oignon" }).click();
  await page.getByLabel("Quantité de sel").fill("1");
  await page.getByLabel("Unité de sel").selectOption("pinch");
  await page.getByRole("button", { name: "Enregistrer le plat" }).click();
  await expect(page).toHaveURL(/\/tribes\/demo\/plats$/);
  await expect(page.getByRole("link", { name: /Chili con carne/ })).toContainText("pour 5");
  const updates = received.filter((r) => r.method === "PUT");
  expect(updates).toEqual([
    {
      method: "PUT",
      path: `dishes/${chili.id}`,
      body: {
        name: "Chili con carne",
        servings: 5,
        ingredients: [{ name: "bœuf haché", quantity: "500", unit: "g" }, { name: "sel", quantity: "1", unit: "pinch" }],
        revision: 2,
      },
    },
  ]);
});

test("a crossed change: the banner, the current dish, then a save on its revision (D5)", async ({ page }) => {
  const current = {
    ...chili,
    revision: 3,
    servings: 6,
    ingredients: [{ name: "bœuf haché", quantity: "600", unit: "g", defaultUnit: "g" }, { name: "ail", quantity: "2", unit: "piece", defaultUnit: "piece" }],
  };
  const received = await mockApi(page, {
    getSession: [replies.session],
    getDishes: [pageReply("dishes", 1, [chili])],
    updateDish: [{ status: 409, body: { error: "dish_changed", dish: current } }, { status: 400, body: { error: "name_taken" } }],
  });
  await page.goto(`/tribes/demo/plats/${chili.id}`);
  await page.getByLabel("Quantité de bœuf haché").fill("700");
  await page.getByRole("button", { name: "Enregistrer le plat" }).click();
  const banner = page.getByRole("alert");
  await expect(banner).toHaveText("Ce plat a été modifié entre-temps. Voici sa version à jour : refaites votre modification.");
  await expect(page).toHaveURL(new RegExp(`/plats/${chili.id}$`));
  await expect(page.getByRole("group", { name: "Pour combien de parts ?" })).toContainText("6");
  await expect(page.locator(".line-name")).toHaveText(["bœuf haché", "ail"]);
  await expect(page.getByLabel("Quantité de bœuf haché")).toHaveValue("600");

  // The banner goes at the next save.
  await page.getByRole("button", { name: "Enregistrer le plat" }).click();
  await expect(page.getByRole("alert")).toHaveText("Un plat porte déjà ce nom.");
  expect(received.filter((r) => r.method === "PUT").map((r) => (r.body as { revision: number }).revision)).toEqual([2, 3]);
});

test("the stepper is bounded, and 40 lines leave “Ajouter un ingrédient” inactive", async ({ page }) => {
  const full = { ...chili, servings: 1, ingredients: Array.from({ length: 40 }, (_, i) => ({ name: `ingrédient ${i}` })) };
  await mockApi(page, { getSession: [replies.session], getDishes: [pageReply("dishes", 1, [full])] });
  await page.goto(`/tribes/demo/plats/${chili.id}`);
  await expect(page.getByRole("button", { name: "Une part de moins" })).toBeDisabled();
  await expect(page.getByRole("button", { name: "Une part de plus" })).toBeEnabled();
  await expect(page.getByRole("button", { name: "Ajouter un ingrédient" })).toBeDisabled();
});

test("a dish unknown to the library", async ({ page }) => {
  await mockApi(page, { getSession: [replies.session], getDishes: [pageReply("dishes", 1, [chili])] });
  await page.goto(`/tribes/demo/plats/${id(9)}`);
  await expect(page.getByText("Ce plat est introuvable.")).toBeVisible();
  // In etendu, the library is beside the dish, without link back (plan grands-ecrans, D5).
  await expect(page.getByRole("main").getByRole("link", { name: "Plats" })).toHaveCount(twoPanes() ? 0 : 1);
  if (!twoPanes()) {
    await expect(page.getByRole("main").getByRole("link", { name: "Plats" })).toHaveAttribute("href", "/tribes/demo/plats");
  }
});

test("a dish opened without network: the message and “Réessayer”", async ({ page }) => {
  await mockApi(page, { getSession: [replies.session], getDishes: [replies.networkError, pageReply("dishes", 1, [chili])] });
  await page.goto(`/tribes/demo/plats/${chili.id}`);
  // In etendu, the library beside tells it too.
  await expect(page.getByRole("main").getByRole("alert")).toHaveText("Le réseau est lent ou absent. Réessayez.");
  await page.getByRole("main").getByRole("button", { name: "Réessayer" }).click();
  await expect(page.getByLabel("Nom du plat")).toHaveValue("Chili con carne");
});

test("the name of an ingredient is bounded to 40 characters, as the server (review of PR #81, point 1)", async ({ page }) => {
  await mockApi(page, { getSession: [replies.session] });
  await page.goto("/tribes/demo/plats/nouveau");
  await page.getByRole("button", { name: "Ajouter un ingrédient" }).click();
  const combobox = page.getByRole("combobox", { name: "Ingrédient", exact: true });
  await combobox.pressSequentially("a".repeat(41));
  await expect(combobox).toHaveValue("a".repeat(40));
});

test("“Ajouter un ingrédient” with a line already being typed does not take the focus later (review of PR #81, point 2)", async ({ page }) => {
  await mockApi(page, { getSession: [replies.session] });
  await page.goto("/tribes/demo/plats/nouveau");
  await page.getByRole("button", { name: "Ajouter un ingrédient" }).click();
  const combobox = page.getByRole("combobox", { name: "Ingrédient", exact: true });
  await combobox.fill("riz");
  await page.getByRole("button", { name: "Ajouter un ingrédient" }).click();
  await expect(combobox).toBeFocused();
  const quantity = page.getByLabel("Quantité", { exact: true });
  await quantity.click();
  await quantity.pressSequentially("25");
  await expect(quantity).toHaveValue("25");
  await expect(combobox).toHaveValue("riz");
});

test("a click on a button below the open suggestions reaches it", async ({ page }) => {
  await mockApi(page, { getSession: [replies.session], getIngredients: [pageReply("ingredients", 1, catalog)] });
  await page.goto("/tribes/demo/plats/nouveau");
  await page.getByRole("button", { name: "Ajouter un ingrédient" }).click();
  const combobox = page.getByRole("combobox", { name: "Ingrédient", exact: true });
  await combobox.fill("tom");
  const suggestions = page.getByRole("listbox", { name: "Ingrédients proposés" }).getByRole("option");
  await expect(suggestions).toHaveCount(3);
  // The suggestions close when the combobox loses the focus, but only after the click, so
  // that the button does not move under the pointer.
  await page.getByRole("button", { name: "Ajouter un ingrédient" }).click();
  await expect(combobox).toBeFocused();
  await page.getByRole("button", { name: "Une part de plus" }).click();
  await expect(page.getByRole("group", { name: "Pour combien de parts ?" })).toContainText("5");
  await expect(suggestions).toHaveCount(0);
});

// A tap reached the button even before the suggestions closed after the gesture: the test
// keeps it so.
test("a tap on a button below the open suggestions reaches it", async ({ page, hasTouch }) => {
  test.skip(!hasTouch, "a tap needs a touch screen");
  await mockApi(page, { getSession: [replies.session], getIngredients: [pageReply("ingredients", 1, catalog)] });
  await page.goto("/tribes/demo/plats/nouveau");
  await page.getByRole("button", { name: "Ajouter un ingrédient" }).tap();
  const combobox = page.getByRole("combobox", { name: "Ingrédient", exact: true });
  await combobox.fill("tom");
  const suggestions = page.getByRole("listbox", { name: "Ingrédients proposés" }).getByRole("option");
  await expect(suggestions).toHaveCount(3);
  await page.getByRole("button", { name: "Une part de plus" }).tap();
  await expect(page.getByRole("group", { name: "Pour combien de parts ?" })).toContainText("5");
  await expect(suggestions).toHaveCount(0);
});
