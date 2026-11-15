// The project setup (ADR 0032, point 9; D8): the only code request of a run. It signs in
// with the code received in the box of the tribe recette-auto, fetched by POP3
// (internal/tools/mailbox), and writes the state of the session, which the four projects
// take up. Filmed like the journeys.
import { execFile } from "node:child_process";
import { chmod, readFile } from "node:fs/promises";
import type { BrowserContext, Page } from "@playwright/test";
import { promisify } from "node:util";
import { recetteSettings } from "./config.ts";
import { expect, step, test } from "./test.ts";

const settings = recetteSettings();

/** Runs internal/tools/mailbox with these arguments, returning its standard output. */
async function mailbox(...args: string[]): Promise<string> {
  const { stdout } = await promisify(execFile)("go", ["run", "../internal/tools/mailbox", "-config", settings.configDir, ...args], { timeout: 3 * 60_000 });
  return stdout;
}

/** The code sent after since, written alone on its standard output by the tool. */
async function fetchCode(since: Date): Promise<string> {
  const stdout = await mailbox("-since", since.toISOString());
  const code = stdout.trim();
  // Neither the code nor the output in the error: the report keeps it.
  if (!/^\d{8}$/.test(code)) {
    throw new Error("mailbox: no code on the standard output");
  }
  return code;
}

/**
 * With make recette-auto REUSE=1, while journeys are written: takes up the session of the last
 * run if it still opens the tribe, without a code request. The acceptance run of a release
 * always signs in by email.
 */
async function reusedSession(page: Page): Promise<boolean> {
  const state = await readFile(settings.storageState, "utf8").catch(() => undefined);
  if (!settings.reuseSession || state === undefined) {
    return false;
  }
  await page.context().addCookies((JSON.parse(state) as { cookies: Parameters<BrowserContext["addCookies"]>[0] }).cookies);
  await page.goto("./");
  const menus = page.getByRole("heading", { name: "Menus" });
  const login = page.getByRole("heading", { name: "Bienvenue !" });
  await expect(menus.or(login)).toBeVisible();
  return menus.isVisible();
}

test("sign in with the code received by email", async ({ page }) => {
  // The tool waits for 2 minutes at most, after its compilation.
  test.setTimeout(4 * 60_000);

  if (await test.step("Session of the last run reused", () => reusedSession(page))) {
    test.info().annotations.push({ type: "session", description: "reused, no code request" });
    return;
  }

  await step(page, "Login screen", async () => {
    await page.goto("./");
    await expect(page).toHaveURL(/\/connexion$/);
    await expect(page.getByRole("heading", { name: "Bienvenue !" })).toBeVisible();
  });

  // A code of an earlier request, arriving late, would be taken for this one (review of #123,
  // point 2).
  await test.step("Box emptied", () => mailbox("-drain"));

  let since = new Date();
  await step(page, "Code request", async () => {
    await page.getByLabel("Votre adresse e-mail").fill(settings.address);
    since = new Date();
    await page.getByRole("button", { name: "Recevoir un code" }).click();
    // A refusal, by the limits of the code requests above all (ADR 0021), is said as such.
    const sent = page.getByRole("heading", { name: "Vérifiez vos e-mails" });
    const refused = page.getByRole("alert");
    await expect(sent.or(refused)).toBeVisible();
    if (await refused.isVisible()) {
      throw new Error(`code request refused: ${await refused.innerText()}`);
    }
  });

  const code = await test.step("Code fetched from the box", () => fetchCode(since));

  await step(page, "Signed in", async () => {
    await page.getByLabel("Code de connexion").fill(code);
    await page.getByRole("button", { name: "Se connecter" }).click();
    await expect(page.getByRole("heading", { name: "Menus" })).toBeVisible();
  });

  await page.context().storageState({ path: settings.storageState });
  await chmod(settings.storageState, 0o600);
});
