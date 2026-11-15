// What the server shows to the browser (plan recette-auto, step 14), read in the responses that
// Playwright records, nothing added to the application: the page of the tribe is revalidated
// (no-cache), the files named after their content are kept (immutable, ADR 0012), the session
// cookie is prefixed __Host- (ADR 0024). Only Caddy and the real server give these.
import type { Response } from "@playwright/test";
import { expect, step, test } from "./test.ts";

/** A file of the front end named after its content, as esbuild writes it. */
const hashed = /-[A-Z0-9]{8}\.[a-z0-9]+$/;

test("headers of the pages and files, session cookie", async ({ page }) => {
  const responses: Response[] = [];
  page.on("response", (r) => responses.push(r));

  await step(page, "Tribe page loaded", async () => {
    await page.goto("./");
    await expect(page.getByRole("heading", { name: "Menus", level: 1 })).toBeVisible();
    await page.evaluate(() => document.fonts.ready);
  });

  await test.step("Tribe page revalidated", async () => {
    const document = responses.find((r) => r.request().resourceType() === "document");
    expect(document?.status()).toBe(200);
    expect(document?.headers()["cache-control"]).toMatch(/\bno-cache\b/);
  });

  await test.step("Hashed files immutable", async () => {
    const files = responses.filter((r) => hashed.test(new URL(r.url()).pathname));
    expect(files.map((r) => new URL(r.url()).pathname.split(".").at(-1))).toEqual(expect.arrayContaining(["js", "css"]));
    for (const r of files) {
      expect(r.status(), r.url()).toBe(200);
      expect(r.headers()["cache-control"], r.url()).toMatch(/\bimmutable\b/);
    }
  });

  await test.step("Session cookie prefixed __Host-", async () => {
    const cookies = await page.context().cookies();
    expect(cookies.map((c) => ({ name: c.name, path: c.path, domain: c.domain.startsWith("."), httpOnly: c.httpOnly, secure: c.secure, sameSite: c.sameSite }))).toEqual([
      { name: expect.stringMatching(/^__Host-session-[0-9a-f]{16}$/), path: "/", domain: false, httpOnly: true, secure: true, sameSite: "Lax" },
    ]);
  });
});
