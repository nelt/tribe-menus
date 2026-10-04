import assert from "node:assert/strict";
import { test } from "node:test";
import { Api, type Outcome, type RawResponse, outcomeOf, sessionOf } from "./api.ts";

const sessionBody = { tribe: { name: "Les Démo" }, member: { email: "alice@exemple.fr", displayName: "Alice" } };

const outcomeCases: { name: string; response: RawResponse; want: Outcome<unknown> }[] = [
  { name: "accepted", response: { status: 202, body: undefined }, want: { kind: "ok", value: undefined } },
  { name: "created", response: { status: 201, body: sessionBody }, want: { kind: "ok", value: sessionBody } },
  { name: "malformed address", response: { status: 400, body: { error: "invalid_email" } }, want: { kind: "invalid-email" } },
  { name: "too many requests", response: { status: 429, body: { error: "too_many_requests" } }, want: { kind: "too-many-requests" } },
  { name: "incorrect code, attempts left", response: { status: 400, body: { error: "incorrect_code", attemptsLeft: 2 } }, want: { kind: "incorrect-code", attemptsLeft: 2 } },
  { name: "incorrect code, last attempt", response: { status: 400, body: { error: "incorrect_code", attemptsLeft: 1 } }, want: { kind: "incorrect-code", attemptsLeft: 1 } },
  { name: "incorrect code, no attempt left", response: { status: 400, body: { error: "incorrect_code", attemptsLeft: 0 } }, want: { kind: "attempts-exhausted" } },
  { name: "incorrect code, attempts missing", response: { status: 400, body: { error: "incorrect_code" } }, want: { kind: "attempts-exhausted" } },
  { name: "new code needed", response: { status: 400, body: { error: "new_code_needed" } }, want: { kind: "code-expired" } },
  { name: "no session", response: { status: 401, body: { error: "no_session" } }, want: { kind: "unauthorized" } },
  { name: "unreadable body", response: { status: 400, body: { error: "bad_request" } }, want: { kind: "network-error" } },
  { name: "not found", response: { status: 404, body: { error: "not_found" } }, want: { kind: "network-error" } },
  { name: "server error", response: { status: 500, body: undefined }, want: { kind: "network-error" } },
  { name: "bad gateway", response: { status: 502, body: "<html>" }, want: { kind: "network-error" } },
  { name: "no response", response: "network-error", want: { kind: "network-error" } },
];

for (const { name, response, want } of outcomeCases) {
  test(`outcomeOf: ${name}`, () => {
    assert.deepEqual(outcomeOf(response), want);
  });
}

const sessionCases: { name: string; body: unknown; ok: boolean }[] = [
  { name: "session", body: sessionBody, ok: true },
  { name: "empty display name", body: { ...sessionBody, member: { email: "a@b.fr", displayName: "" } }, ok: true },
  { name: "no body", body: undefined, ok: false },
  { name: "no tribe", body: { member: sessionBody.member }, ok: false },
  { name: "name not a string", body: { ...sessionBody, tribe: { name: 1 } }, ok: false },
];

for (const { name, body, ok } of sessionCases) {
  test(`sessionOf: ${name}`, () => {
    assert.equal(sessionOf(body) !== undefined, ok);
  });
}

function jsonResponse(status: number, body?: unknown): Response {
  return body === undefined
    ? new Response(null, { status })
    : new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

function apiWith(answer: (url: string, init: RequestInit) => Promise<Response>) {
  const requests: { url: string; method: string; body: unknown }[] = [];
  let unauthorized = 0;
  const api = new Api({
    fetch: async (input, init = {}) => {
      const url = String(input);
      requests.push({ url, method: init.method ?? "GET", body: init.body === undefined ? undefined : JSON.parse(String(init.body)) });
      return answer(url, init);
    },
    onUnauthorized: () => unauthorized++,
    timeoutMs: 1000,
  });
  return { api, requests, unauthorized: () => unauthorized };
}

test("Api.requestCode sends the address", async () => {
  const { api, requests } = apiWith(async () => jsonResponse(202));
  assert.deepEqual(await api.requestCode("alice@exemple.fr"), { kind: "ok", value: undefined });
  assert.deepEqual(requests, [{ url: "api/login-codes", method: "POST", body: { email: "alice@exemple.fr" } }]);
});

test("Api.openSession sends the code and whether the app is installed", async () => {
  const { api, requests } = apiWith(async () => jsonResponse(201, sessionBody));
  assert.deepEqual(await api.openSession("alice@exemple.fr", "123456", true), { kind: "ok", value: sessionBody });
  assert.deepEqual(requests, [{ url: "api/sessions", method: "POST", body: { email: "alice@exemple.fr", code: "123456", installedApp: true } }]);
});

test("Api.getSession: a body that is not a session is a network error", async () => {
  const { api } = apiWith(async () => jsonResponse(200, { tribe: {} }));
  assert.deepEqual(await api.getSession(), { kind: "network-error" });
});

test("Api: a 401 brings back to the login", async () => {
  const { api, unauthorized } = apiWith(async () => jsonResponse(401, { error: "no_session" }));
  assert.deepEqual(await api.getSession(), { kind: "unauthorized" });
  assert.deepEqual(await api.deleteSession(), { kind: "unauthorized" });
  assert.equal(unauthorized(), 2);
});

test("Api: a failed fetch is a network error", async () => {
  const { api, unauthorized } = apiWith(async () => {
    throw new TypeError("Failed to fetch");
  });
  assert.deepEqual(await api.deleteSession(), { kind: "network-error" });
  assert.equal(unauthorized(), 0);
});

test("Api: beyond the timeout, a call is a network error", async () => {
  const api = new Api({
    fetch: (_input, init) =>
      new Promise((_resolve, reject) => {
        init?.signal?.addEventListener("abort", () => reject(init.signal?.reason));
      }),
    onUnauthorized: () => {},
    timeoutMs: 10,
  });
  assert.deepEqual(await api.getSession(), { kind: "network-error" });
});
