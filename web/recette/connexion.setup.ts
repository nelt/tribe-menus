// The project setup (ADR 0032, point 9; D8): the only code request of a run. It signs in
// with the code received in the box of the tribe recette-auto, fetched by POP3
// (internal/tools/mailbox), and writes the state of the session, which the four projects
// take up. Filmed like the journeys.
import { execFile } from "node:child_process";
import { chmod } from "node:fs/promises";
import { promisify } from "node:util";
import { recetteSettings } from "./config.ts";
import { expect, step, test } from "./test.ts";

const settings = recetteSettings();

/** The code sent after since, written alone on its standard output by the tool. */
async function fetchCode(since: Date): Promise<string> {
  const { stdout } = await promisify(execFile)(
    "go",
    ["run", "../internal/tools/mailbox", "-config", settings.configDir, "-since", since.toISOString()],
    { timeout: 3 * 60_000 },
  );
  const code = stdout.trim();
  // Neither the code nor the output in the error: the report keeps it.
  if (!/^\d{8}$/.test(code)) {
    throw new Error("mailbox: no code on the standard output");
  }
  return code;
}

test("sign in with the code received by email", async ({ page }) => {
  // The tool waits for 2 minutes at most, after its compilation.
  test.setTimeout(4 * 60_000);

  await step(page, "Login screen", async () => {
    await page.goto("./");
    await expect(page).toHaveURL(/\/connexion$/);
    await expect(page.getByRole("heading", { name: "Bienvenue !" })).toBeVisible();
  });

  let since = new Date();
  await step(page, "Code request", async () => {
    await page.getByLabel("Votre adresse e-mail").fill(settings.address);
    since = new Date();
    await page.getByRole("button", { name: "Recevoir un code" }).click();
    await expect(page.getByRole("heading", { name: "Vérifiez vos e-mails" })).toBeVisible();
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
