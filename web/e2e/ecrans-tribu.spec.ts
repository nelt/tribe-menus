// States of the screen Tribu (Doux-Tribu, reduced to the size of the tribe, plan
// bibliotheque, D3, with the sign-out, plan planning, D5), with a simulated API (plan socle,
// D15). The size proposed for a new dish
// is proved by the scenario T3 against the real server (scenarios.spec.ts).
import { expect, test } from "@playwright/test";
import { mockApi, replies } from "./api-mock.ts";

const tribe = (size: number, activeMembers = 2) => ({ status: 200, body: { name: "Les Démo", size, activeMembers } });

test("the tribe, its active members, and its size set by the stepper", async ({ page }) => {
  const received = await mockApi(page, { getSession: [replies.session], getTribe: [tribe(4)], setTribeSize: [tribe(5), tribe(6)] });
  await page.goto("/tribes/demo/tribu");
  await expect(page.getByRole("heading", { name: "Les Démo" })).toBeVisible();
  await expect(page.getByText("TRIBU", { exact: true })).toBeVisible();
  await expect(page.getByText("2 membres actifs")).toBeVisible();
  await expect(page.getByRole("link", { name: "Planning" })).toHaveAttribute("href", "/tribes/demo/");
  const stepper = page.getByRole("group", { name: "Taille de la tribu" });
  await expect(stepper).toContainText("4");

  await page.getByRole("button", { name: "Une personne de plus" }).click();
  await expect(stepper).toContainText("5");
  await page.getByRole("button", { name: "Une personne de plus" }).click();
  await expect(stepper).toContainText("6");
  await expect.poll(() => received.filter((r) => r.method === "PUT").map((r) => r.body)).toEqual([{ size: 5 }, { size: 6 }]);
  await expect(page.getByRole("alert")).toHaveCount(0);
});

test("one active member", async ({ page }) => {
  await mockApi(page, { getSession: [replies.session], getTribe: [tribe(4, 1)] });
  await page.goto("/tribes/demo/tribu");
  await expect(page.getByText("1 membre actif")).toBeVisible();
});

test("the size is bounded from 1 to 50", async ({ page }) => {
  await mockApi(page, { getSession: [replies.session], getTribe: [tribe(50)] });
  await page.goto("/tribes/demo/tribu");
  await expect(page.getByRole("button", { name: "Une personne de plus" })).toBeDisabled();
  await expect(page.getByRole("button", { name: "Une personne de moins" })).toBeEnabled();
});

test("a size not saved without network comes back, with the message (D18)", async ({ page }) => {
  await mockApi(page, { getSession: [replies.session], getTribe: [tribe(4)], setTribeSize: [replies.networkError] });
  await page.goto("/tribes/demo/tribu");
  await page.getByRole("button", { name: "Une personne de moins" }).click();
  await expect(page.getByRole("alert")).toHaveText("Réseau nécessaire. Le réseau est lent ou absent : réessayez.");
  await expect(page.getByRole("group", { name: "Taille de la tribu" })).toContainText("4");
});

test("the tribe read without network: the message and “Réessayer”", async ({ page }) => {
  await mockApi(page, { getSession: [replies.session], getTribe: [replies.networkError, tribe(4)] });
  await page.goto("/tribes/demo/tribu");
  await expect(page.getByRole("alert")).toHaveText("Le réseau est lent ou absent. Réessayez.");
  await page.getByRole("button", { name: "Réessayer" }).click();
  await expect(page.getByRole("heading", { name: "Les Démo" })).toBeVisible();
  await expect(page.getByRole("alert")).toHaveCount(0);
});

test("sign-out at the bottom of the screen, without confirmation (D5)", async ({ page }) => {
  const received = await mockApi(page, { getSession: [replies.session], getTribe: [tribe(4)] });
  await page.goto("/tribes/demo/tribu");
  const signOut = page.getByRole("button", { name: "Se déconnecter" });
  await expect(signOut).toBeVisible();
  // Under the size of the tribe, in tomato red, outlined.
  const size = await page.getByRole("group", { name: "Taille de la tribu" }).boundingBox();
  const button = await signOut.boundingBox();
  expect(button!.y).toBeGreaterThan(size!.y + size!.height);
  expect(await signOut.evaluate((e) => getComputedStyle(e).borderColor)).toBe("rgb(198, 61, 36)");
  // The link to the source code of the provisional home is kept (ADR 0011, point 2).
  await expect(page.getByRole("link", { name: "logiciel libre, code source" })).toBeVisible();

  await signOut.click();
  await expect(page.getByRole("heading", { name: "Bienvenue !" })).toBeVisible();
  await expect(page).toHaveURL(/\/tribes\/demo\/connexion$/);
  expect(received).toContainEqual({ method: "DELETE", path: "session", body: undefined });
});

test("sign-out without network: the message, the screen stays", async ({ page }) => {
  await mockApi(page, { getSession: [replies.session], getTribe: [tribe(4)], deleteSession: [replies.networkError] });
  await page.goto("/tribes/demo/tribu");
  await page.getByRole("button", { name: "Se déconnecter" }).click();
  await expect(page.getByRole("alert")).toHaveText("Le réseau est lent ou absent. Réessayez.");
  await expect(page.getByRole("heading", { name: "Les Démo" })).toBeVisible();
});

test("sign-out while the tribe cannot be read", async ({ page }) => {
  await mockApi(page, { getSession: [replies.session], getTribe: [replies.networkError] });
  await page.goto("/tribes/demo/tribu");
  await page.getByRole("button", { name: "Se déconnecter" }).click();
  await expect(page.getByRole("heading", { name: "Bienvenue !" })).toBeVisible();
});
