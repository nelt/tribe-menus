// States of the login, loading and home screens, with a simulated API (D15): each screen
// shows the right state for a given response. The rules themselves are proved against the
// API by the Gherkin scenarios (godog).
import { type Page, expect, test } from "@playwright/test";
import { type Script, mockApi, replies } from "./api-mock.ts";

const email = "alice@exemple.fr";

async function openLogin(page: Page, script: Script = {}) {
  const received = await mockApi(page, script);
  await page.goto("/tribes/demo/");
  await expect(page).toHaveURL(/\/tribes\/demo\/connexion$/);
  return received;
}

async function requestCode(page: Page, address = email) {
  await page.getByLabel("Votre adresse e-mail").fill(address);
  await page.getByRole("button", { name: "Recevoir un code" }).click();
}

async function openCodeScreen(page: Page, script: Script = {}) {
  const received = await openLogin(page, script);
  await requestCode(page);
  await expect(page.getByRole("heading", { name: "Vérifiez vos e-mails" })).toBeVisible();
  return received;
}

async function submitCode(page: Page, code = "123456") {
  await page.getByLabel("Code de connexion").fill(code);
  await page.getByRole("button", { name: "Se connecter" }).click();
}

test("malformed address", async ({ page }) => {
  await openLogin(page, { requestCode: [replies.invalidEmail] });
  await requestCode(page, "alice@exemple");
  await expect(page.getByRole("alert")).toHaveText("Cette adresse ne semble pas complète. Vérifiez-la.");
  await expect(page.getByLabel("Votre adresse e-mail")).toHaveAttribute("aria-invalid", "true");
  await expect(page.getByLabel("Votre adresse e-mail")).toHaveValue("alice@exemple");
});

test("too many requests: the button is inactive until the address changes", async ({ page }) => {
  await openLogin(page, { requestCode: [replies.tooManyRequests] });
  await requestCode(page);
  await expect(page.getByRole("alert")).toHaveText("Trop de demandes depuis cet appareil. Réessayez dans quelques minutes.");
  await expect(page.getByRole("button", { name: "Recevoir un code" })).toBeDisabled();
  await page.getByLabel("Votre adresse e-mail").fill("bruno@exemple.fr");
  await expect(page.getByRole("button", { name: "Recevoir un code" })).toBeEnabled();
  await expect(page.getByRole("alert")).toHaveCount(0);
});

test("network error on the email form: the form stays as it was", async ({ page }) => {
  await openLogin(page, { requestCode: [replies.networkError] });
  await requestCode(page);
  await expect(page.getByRole("alert")).toHaveText("Le réseau est lent ou absent. Réessayez.");
  await expect(page.getByLabel("Votre adresse e-mail")).toHaveValue(email);
  await expect(page.getByRole("button", { name: "Recevoir un code" })).toBeEnabled();
});

test("code screen: the message does not say whether the address is a member", async ({ page }) => {
  const received = await openCodeScreen(page);
  await expect(page.getByText(`Si ${email} fait partie de la tribu, un code à 6 chiffres vient d’y être envoyé.`)).toBeVisible();
  const field = page.getByLabel("Code de connexion");
  await expect(field).toHaveAttribute("inputmode", "numeric");
  await expect(field).toHaveAttribute("autocomplete", "one-time-code");
  await expect(field).toHaveAttribute("maxlength", "6");
  await expect(field).toBeFocused();
  expect(received).toContainEqual({ method: "POST", path: "login-codes", body: { email } });
});

test("the code screen follows the address requested, even if the field changes meanwhile", async ({ page }) => {
  const received = await openLogin(page, { requestCode: [{ ...replies.codeSent, delayMs: 1500 }], openSession: [replies.incorrectCode(2)] });
  await requestCode(page);
  await page.getByLabel("Votre adresse e-mail").fill("alice@exemple.com");
  await expect(page.getByText(`Si ${email} fait partie de la tribu, un code à 6 chiffres vient d’y être envoyé.`)).toBeVisible();
  await submitCode(page);
  await expect(page.getByRole("alert")).toBeVisible();
  expect(received).toContainEqual({ method: "POST", path: "sessions", body: { email, code: "123456", installedApp: false } });
});

async function paste(page: Page, text: string) {
  await page.getByLabel("Code de connexion").evaluate((input, text) => {
    const data = new DataTransfer();
    data.setData("text", text);
    input.dispatchEvent(new ClipboardEvent("paste", { clipboardData: data, bubbles: true, cancelable: true }));
  }, text);
}

test("pasted code: only the digits are kept, even with spaces", async ({ page }) => {
  const received = await openCodeScreen(page, { openSession: [replies.incorrectCode(2)] });
  await paste(page, "12 34-56");
  await expect(page.getByLabel("Code de connexion")).toHaveValue("123456");
  await page.getByRole("button", { name: "Se connecter" }).click();
  await expect(page.getByRole("alert")).toBeVisible();
  expect(received).toContainEqual({ method: "POST", path: "sessions", body: { email, code: "123456", installedApp: false } });
});

test("pasted code: inserted at the caret, a paste without digit changes nothing", async ({ page }) => {
  await openCodeScreen(page);
  const field = page.getByLabel("Code de connexion");
  await field.pressSequentially("12");
  await paste(page, "3456");
  await expect(field).toHaveValue("123456");
  await paste(page, "bonjour");
  await expect(field).toHaveValue("123456");
  await field.fill("12");
  await paste(page, "bonjour");
  await expect(field).toHaveValue("12");
});

test("incorrect code, with the attempts left", async ({ page }) => {
  await openCodeScreen(page, { openSession: [replies.incorrectCode(2), replies.incorrectCode(1)] });
  await submitCode(page, "111111");
  await expect(page.getByRole("alert")).toHaveText("Code incorrect. Il vous reste 2 essais.");
  await expect(page.getByLabel("Code de connexion")).toHaveAttribute("aria-invalid", "true");
  await submitCode(page, "222222");
  await expect(page.getByRole("alert")).toHaveText("Code incorrect. Il vous reste 1 essai.");
});

test("attempts exhausted: the only action is to receive a new code", async ({ page }) => {
  await openCodeScreen(page, { openSession: [replies.incorrectCode(0)] });
  await submitCode(page);
  await expect(page.getByRole("alert")).toHaveText("Trop d’essais : ce code n’est plus valable. Demandez-en un nouveau.");
  await expect(page.getByRole("button", { name: "Se connecter" })).toHaveCount(0);
  await expect(page.getByLabel("Code de connexion")).toBeDisabled();
  await page.getByRole("button", { name: "Recevoir un nouveau code" }).click();
  await expect(page.getByRole("button", { name: "Se connecter" })).toBeVisible();
  await expect(page.getByLabel("Code de connexion")).toHaveValue("");
  await expect(page.getByRole("alert")).toHaveCount(0);
});

test("code expired", async ({ page }) => {
  await openCodeScreen(page, { openSession: [replies.newCodeNeeded] });
  await submitCode(page);
  await expect(page.getByRole("alert")).toHaveText("Ce code a expiré. Demandez-en un nouveau.");
  await expect(page.getByRole("button", { name: "Recevoir un nouveau code" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Se connecter" })).toHaveCount(0);
});

test("resend refused: the message of the email screen, the code received stays usable", async ({ page }) => {
  await openCodeScreen(page, { requestCode: [replies.codeSent, replies.tooManyRequests] });
  await page.getByRole("button", { name: "Je n’ai rien reçu : renvoyer un code" }).click();
  await expect(page.getByRole("alert")).toHaveText("Trop de demandes depuis cet appareil. Réessayez dans quelques minutes.");
  await submitCode(page);
  await expect(page.getByRole("heading", { name: "Les Démo" })).toBeVisible();
});

test("network error on the code form", async ({ page }) => {
  await openCodeScreen(page, { openSession: [replies.networkError] });
  await submitCode(page, "654321");
  await expect(page.getByRole("alert")).toHaveText("Le réseau est lent ou absent. Réessayez.");
  await expect(page.getByLabel("Code de connexion")).toHaveValue("654321");
});

test("“Modifier” goes back to the email, which is kept", async ({ page }) => {
  await openCodeScreen(page);
  await page.getByRole("button", { name: "Modifier" }).click();
  await expect(page.getByLabel("Votre adresse e-mail")).toHaveValue(email);
  await expect(page.getByLabel("Votre adresse e-mail")).toBeFocused();
});

test("the address is kept in memory only: a reload goes back to the email", async ({ page }) => {
  await openCodeScreen(page);
  await page.reload();
  await expect(page.getByRole("heading", { name: "Bienvenue !" })).toBeVisible();
  await expect(page.getByLabel("Votre adresse e-mail")).toHaveValue("");
});

test("slow loading: the loading screen, then the home page", async ({ page }) => {
  await mockApi(page, { getSession: [{ ...replies.session, delayMs: 1500 }] });
  await page.goto("/tribes/demo/");
  await expect(page.getByRole("status", { name: "Chargement" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Les Démo" })).toBeVisible();
  await expect(page.getByRole("status", { name: "Chargement" })).toHaveCount(0);
});

test("fast loading: no loading screen", async ({ page }) => {
  await mockApi(page, { getSession: [replies.session] });
  let loadingShown = false;
  await page.exposeFunction("loadingShown", () => (loadingShown = true));
  await page.addInitScript(() => {
    new MutationObserver(() => {
      if (document.querySelector("mt-loading") !== null) {
        (window as unknown as { loadingShown: () => void }).loadingShown();
      }
    }).observe(document, { childList: true, subtree: true });
  });
  await page.goto("/tribes/demo/");
  await expect(page.getByRole("heading", { name: "Les Démo" })).toBeVisible();
  expect(loadingShown).toBe(false);
});

test("slow or absent network at startup: the message and “Réessayer”", async ({ page }) => {
  await mockApi(page, { getSession: [replies.networkError, replies.noSession] });
  await page.goto("/tribes/demo/");
  await expect(page.getByText("Le réseau est lent ou absent.")).toBeVisible();
  await expect(page.getByText(/planning/)).toHaveCount(0);
  await page.getByRole("button", { name: "Réessayer" }).click();
  await expect(page.getByRole("heading", { name: "Bienvenue !" })).toBeVisible();
});

test("home and sign-out", async ({ page }) => {
  const received = await mockApi(page, { getSession: [replies.session] });
  await page.goto("/tribes/demo/connexion");
  await expect(page).toHaveURL(/\/tribes\/demo\/$/);
  await expect(page.getByRole("heading", { name: "Les Démo" })).toBeVisible();
  await page.getByRole("button", { name: "Se déconnecter" }).click();
  await expect(page.getByRole("heading", { name: "Bienvenue !" })).toBeVisible();
  await expect(page).toHaveURL(/\/tribes\/demo\/connexion$/);
  expect(received).toContainEqual({ method: "DELETE", path: "session", body: undefined });
});

test("sign-out without network", async ({ page }) => {
  await mockApi(page, { getSession: [replies.session], deleteSession: [replies.networkError] });
  await page.goto("/tribes/demo/");
  await page.getByRole("button", { name: "Se déconnecter" }).click();
  await expect(page.getByRole("alert")).toHaveText("Le réseau est lent ou absent. Réessayez.");
  await expect(page.getByRole("heading", { name: "Les Démo" })).toBeVisible();
});

test("footer: the link to the source code", async ({ page }) => {
  await openLogin(page);
  await expect(page.getByRole("link", { name: "logiciel libre, code source" })).toHaveAttribute("href", /^https:\/\/github\.com\/nelt\/tribe-menus/);
});
