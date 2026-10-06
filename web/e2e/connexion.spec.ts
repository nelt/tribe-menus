// The real login journey, against the server with the demonstration tribe (D14, D15):
// Chromium only, WebKit refusing the Secure cookie on http://localhost. Two code requests
// per run, under the limit of 10 per hour for an IP address.
import { readFile } from "node:fs/promises";
import { type Page, expect, test } from "@playwright/test";

test.skip(({ browserName }) => browserName !== "chromium", "WebKit refuses the Secure cookie on http://localhost (D14)");

// Written by the server of the tests (playwright.config.ts), one JSON object per line (D6).
const mailFile = ".e2e-data/mails.jsonl";

/** Waits for the login code sent to the address, read in the file of the emails. */
async function codeSentTo(address: string): Promise<string> {
  let code: string | undefined;
  await expect
    .poll(async () => {
      const content = await readFile(mailFile, "utf8").catch(() => "");
      const messages = content
        .split("\n")
        .filter((line) => line !== "")
        .map((line) => JSON.parse(line) as { to: string; body: string });
      code = messages.filter((m) => m.to === address).at(-1)?.body.match(/\b\d{8}\b/)?.[0];
      return code;
    })
    .toBeDefined();
  return code ?? "";
}

async function requestCode(page: Page, address: string) {
  await page.getByLabel("Votre adresse e-mail").fill(address);
  await page.getByRole("button", { name: "Recevoir un code" }).click();
  await expect(page.getByRole("heading", { name: "Vérifiez vos e-mails" })).toBeVisible();
}

test("sign in with the code received by email, then sign out", async ({ page }) => {
  await page.goto("/tribes/demo/");
  await expect(page).toHaveURL(/\/tribes\/demo\/connexion$/);
  await requestCode(page, "alice@exemple.fr");

  await page.getByLabel("Code de connexion").fill(await codeSentTo("alice@exemple.fr"));
  await page.getByRole("button", { name: "Se connecter" }).click();
  await expect(page.getByRole("heading", { name: "Les Démo" })).toBeVisible();
  await expect(page).toHaveURL(/\/tribes\/demo\/$/);

  // The session survives a reload.
  await page.reload();
  await expect(page.getByRole("heading", { name: "Les Démo" })).toBeVisible();

  await page.getByRole("button", { name: "Se déconnecter" }).click();
  await expect(page.getByRole("heading", { name: "Bienvenue !" })).toBeVisible();
  await page.reload();
  await expect(page.getByRole("heading", { name: "Bienvenue !" })).toBeVisible();
});

test("an unknown tribe shows the same screens as an existing one (ENF-02)", async ({ page }) => {
  await page.goto("/tribes/demo/");
  await expect(page.getByRole("heading", { name: "Bienvenue !" })).toBeVisible();
  const existing = await page.locator("body").innerText();

  await page.goto("/tribes/inconnue/");
  await expect(page).toHaveURL(/\/tribes\/inconnue\/connexion$/);
  await expect(page.getByRole("heading", { name: "Bienvenue !" })).toBeVisible();
  expect(await page.locator("body").innerText()).toBe(existing);

  await requestCode(page, "alice@exemple.fr");
  await expect(page.getByText("Si alice@exemple.fr fait partie de la tribu, un code à 8 chiffres vient d’y être envoyé.")).toBeVisible();
  await expect(page.getByRole("alert")).toHaveCount(0);
});

test("the page loads its files under the names given by the build, without error (ADR 0012)", async ({ page }) => {
  // A refusal of the CSP is written to the console. Failed loads are not: the API answers 401
  // without a session, and the status of each file is checked below.
  const errors: string[] = [];
  page.on("console", (message) => {
    if (message.type() === "error" && !message.text().startsWith("Failed to load resource")) {
      errors.push(message.text());
    }
  });
  page.on("pageerror", (error) => errors.push(error.message));
  const loaded: string[] = [];
  page.on("response", (response) => {
    const type = response.request().resourceType();
    if (["script", "stylesheet", "font", "image"].includes(type)) {
      expect(response.status(), response.url()).toBe(200);
      loaded.push(decodeURIComponent(new URL(response.url()).pathname));
    }
  });

  await page.goto("/tribes/demo/");
  await expect(page.getByRole("heading", { name: "Bienvenue !" })).toBeVisible();
  await page.evaluate(() => document.fonts.ready);

  // A name ends with the hash of its content, as esbuild writes it.
  for (const extension of ["js", "css", "svg", "woff2"]) {
    expect(loaded, extension).toContainEqual(expect.stringMatching(new RegExp(`^/tribes/demo/[^/]+-[A-Z0-9]{8}\\.${extension}$`)));
  }
  expect(loaded.filter((path) => !/-[A-Z0-9]{8}\.[a-z0-9]+$/.test(path))).toEqual([]);
  expect(errors).toEqual([]);
});
