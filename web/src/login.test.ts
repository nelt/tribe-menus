import assert from "node:assert/strict";
import { test } from "node:test";
import type { Failure, Outcome } from "./api.ts";
import {
  type LoginState,
  afterCodeRequest,
  afterResend,
  afterSessionFailure,
  changeEmail,
  emailEdited,
  errorMessage,
  isInstalledApp,
  normalizeCode,
  pasteIntoCode,
} from "./login.ts";

const email = "alice@exemple.fr";
const emailStep: LoginState = { step: "email", email, blocked: false };
const codeStep: LoginState = { step: "code", email, dead: false };
const deadCode: LoginState = { step: "code", email, error: "code-expired", dead: true };
const ok: Outcome<undefined> = { kind: "ok", value: undefined };

const requestCases: { name: string; outcome: Outcome<undefined>; want: LoginState }[] = [
  { name: "sent", outcome: ok, want: codeStep },
  { name: "malformed address", outcome: { kind: "invalid-email" }, want: { step: "email", email, error: "invalid-email", blocked: false } },
  { name: "too many requests", outcome: { kind: "too-many-requests" }, want: { step: "email", email, error: "too-many-requests", blocked: true } },
  { name: "network error", outcome: { kind: "network-error" }, want: { step: "email", email, error: "network-error", blocked: false } },
];

for (const { name, outcome, want } of requestCases) {
  test(`afterCodeRequest: ${name}`, () => {
    assert.deepEqual(afterCodeRequest(emailStep, outcome), want);
  });
}

const resendCases: { name: string; from: LoginState; outcome: Outcome<undefined>; want: LoginState }[] = [
  { name: "sent", from: { ...codeStep, error: "incorrect-code", attemptsLeft: 2 }, outcome: ok, want: codeStep },
  { name: "sent after a dead code", from: deadCode, outcome: ok, want: codeStep },
  { name: "refused: the code received stays usable", from: codeStep, outcome: { kind: "too-many-requests" }, want: { step: "code", email, error: "too-many-requests", dead: false } },
  { name: "refused after a dead code", from: deadCode, outcome: { kind: "too-many-requests" }, want: { step: "code", email, error: "too-many-requests", dead: true } },
  { name: "network error", from: codeStep, outcome: { kind: "network-error" }, want: { step: "code", email, error: "network-error", dead: false } },
  { name: "ignored on the email screen", from: emailStep, outcome: ok, want: emailStep },
];

for (const { name, from, outcome, want } of resendCases) {
  test(`afterResend: ${name}`, () => {
    assert.deepEqual(afterResend(from, outcome), want);
  });
}

const sessionCases: { name: string; from: LoginState; failure: Failure; want: LoginState }[] = [
  { name: "incorrect code", from: codeStep, failure: { kind: "incorrect-code", attemptsLeft: 2 }, want: { step: "code", email, error: "incorrect-code", attemptsLeft: 2, dead: false } },
  { name: "attempts exhausted", from: codeStep, failure: { kind: "attempts-exhausted" }, want: { step: "code", email, error: "attempts-exhausted", dead: true } },
  { name: "code expired", from: codeStep, failure: { kind: "code-expired" }, want: deadCode },
  { name: "network error", from: codeStep, failure: { kind: "network-error" }, want: { step: "code", email, error: "network-error", dead: false } },
  { name: "unexpected 401", from: codeStep, failure: { kind: "unauthorized" }, want: { step: "code", email, error: "network-error", dead: false } },
  { name: "ignored on the email screen", from: emailStep, failure: { kind: "code-expired" }, want: emailStep },
];

for (const { name, from, failure, want } of sessionCases) {
  test(`afterSessionFailure: ${name}`, () => {
    assert.deepEqual(afterSessionFailure(from, failure), want);
  });
}

test("changeEmail keeps the address and goes back to the email screen", () => {
  assert.deepEqual(changeEmail(deadCode), emailStep);
});

test("emailEdited clears the error and unblocks the button", () => {
  assert.deepEqual(emailEdited("bruno@exemple.fr"), { step: "email", email: "bruno@exemple.fr", blocked: false });
});

const messageCases: { name: string; got: string; want: string }[] = [
  { name: "malformed address", got: errorMessage("invalid-email"), want: "Cette adresse ne semble pas complète. Vérifiez-la." },
  { name: "too many requests", got: errorMessage("too-many-requests"), want: "Trop de demandes depuis cet appareil. Réessayez dans quelques minutes." },
  { name: "two attempts left", got: errorMessage("incorrect-code", 2), want: "Code incorrect. Il vous reste 2 essais." },
  { name: "one attempt left", got: errorMessage("incorrect-code", 1), want: "Code incorrect. Il vous reste 1 essai." },
  { name: "attempts exhausted", got: errorMessage("attempts-exhausted"), want: "Trop d’essais : ce code n’est plus valable. Demandez-en un nouveau." },
  { name: "code expired", got: errorMessage("code-expired"), want: "Ce code a expiré. Demandez-en un nouveau." },
  { name: "network error", got: errorMessage("network-error"), want: "Le réseau est lent ou absent. Réessayez." },
];

for (const { name, got, want } of messageCases) {
  test(`errorMessage: ${name}`, () => {
    assert.equal(got, want);
  });
}

const codeCases: { input: string; want: string }[] = [
  { input: "48219037", want: "48219037" },
  { input: "4821 9037", want: "48219037" },
  { input: "48-21-90-37", want: "48219037" },
  { input: "482190375", want: "48219037" },
  { input: "48a", want: "48" },
  { input: "", want: "" },
];

for (const { input, want } of codeCases) {
  test(`normalizeCode: ${JSON.stringify(input)}`, () => {
    assert.equal(normalizeCode(input), want);
  });
}

const pasteCases: { name: string; value: string; start: number; end: number; text: string; want: { code: string; caret: number } | undefined }[] = [
  { name: "empty field", value: "", start: 0, end: 0, text: "4821 9037", want: { code: "48219037", caret: 8 } },
  { name: "after the digits typed", value: "12", start: 2, end: 2, text: "345678", want: { code: "12345678", caret: 8 } },
  { name: "in the middle", value: "125678", start: 2, end: 2, text: "34", want: { code: "12345678", caret: 4 } },
  { name: "over a selection", value: "99999999", start: 0, end: 8, text: "1234-5678", want: { code: "12345678", caret: 8 } },
  { name: "over a partial selection", value: "12995678", start: 2, end: 4, text: "34", want: { code: "12345678", caret: 4 } },
  { name: "beyond eight digits", value: "1234", start: 4, end: 4, text: "567890", want: { code: "12345678", caret: 8 } },
  { name: "no digit", value: "12", start: 2, end: 2, text: "bonjour", want: undefined },
  { name: "empty text", value: "12", start: 2, end: 2, text: "", want: undefined },
];

for (const { name, value, start, end, text, want } of pasteCases) {
  test(`pasteIntoCode: ${name}`, () => {
    assert.deepEqual(pasteIntoCode(value, start, end, text), want);
  });
}

const installedCases: { name: string; standaloneMedia: boolean; standalone?: boolean; want: boolean }[] = [
  { name: "browser tab", standaloneMedia: false, want: false },
  { name: "installed app", standaloneMedia: true, want: true },
  { name: "iOS home screen", standaloneMedia: false, standalone: true, want: true },
  { name: "iOS Safari", standaloneMedia: false, standalone: false, want: false },
];

for (const { name, standaloneMedia, standalone, want } of installedCases) {
  test(`isInstalledApp: ${name}`, () => {
    const navigator = standalone === undefined ? {} : { standalone };
    const env = { matchMedia: (query: string) => ({ matches: standaloneMedia && query === "(display-mode: standalone)" }), navigator };
    assert.equal(isInstalledApp(env), want);
  });
}
