// Client of the API under /tribes/<identifier>/api/ (contract: plan of 2026-10-03, notes of
// lot B). The URLs are relative to the <base> of the page. A 401 brings back to the login.

/** The tribe and the member of a session. */
export interface Session {
  tribe: { name: string };
  member: { email: string; displayName: string };
}

/** Why a call failed, as the screens show it. */
export type Failure =
  | { kind: "invalid-email" }
  | { kind: "too-many-requests" }
  | { kind: "incorrect-code"; attemptsLeft: number }
  | { kind: "attempts-exhausted" }
  | { kind: "code-expired" }
  | { kind: "unauthorized" }
  | { kind: "network-error" };

/** The result of a call: its value, or a failure. */
export type Outcome<T> = { kind: "ok"; value: T } | Failure;

/** A response of the API, its body decoded (undefined without JSON body), or no response. */
export type RawResponse = { status: number; body: unknown } | "network-error";

/**
 * Translates a response of the API into an outcome. The body of a success is returned as is.
 * A response the contract does not foresee (5xx, unknown error) is a network error.
 */
export function outcomeOf(response: RawResponse): Outcome<unknown> {
  if (response === "network-error") {
    return { kind: "network-error" };
  }
  const { status, body } = response;
  if (status >= 200 && status < 300) {
    return { kind: "ok", value: body };
  }
  if (status === 401) {
    return { kind: "unauthorized" };
  }
  const error = isRecord(body) ? body["error"] : undefined;
  if (status === 400 && error === "invalid_email") {
    return { kind: "invalid-email" };
  }
  if (status === 429 && error === "too_many_requests") {
    return { kind: "too-many-requests" };
  }
  if (status === 400 && error === "incorrect_code") {
    const attemptsLeft = isRecord(body) ? body["attemptsLeft"] : undefined;
    if (typeof attemptsLeft === "number" && attemptsLeft > 0) {
      return { kind: "incorrect-code", attemptsLeft };
    }
    return { kind: "attempts-exhausted" };
  }
  if (status === 400 && error === "new_code_needed") {
    return { kind: "code-expired" };
  }
  return { kind: "network-error" };
}

/** Returns the session described by a body, or undefined if the body is not one. */
export function sessionOf(body: unknown): Session | undefined {
  if (!isRecord(body) || !isRecord(body["tribe"]) || !isRecord(body["member"])) {
    return undefined;
  }
  const name = body["tribe"]["name"];
  const email = body["member"]["email"];
  const displayName = body["member"]["displayName"];
  if (typeof name !== "string" || typeof email !== "string" || typeof displayName !== "string") {
    return undefined;
  }
  return { tribe: { name }, member: { email, displayName } };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

/** A session outcome: a body that is not a session is a network error. */
function toSession(outcome: Outcome<unknown>): Outcome<Session> {
  if (outcome.kind !== "ok") {
    return outcome;
  }
  const session = sessionOf(outcome.value);
  return session === undefined ? { kind: "network-error" } : { kind: "ok", value: session };
}

function toVoid(outcome: Outcome<unknown>): Outcome<undefined> {
  return outcome.kind === "ok" ? { kind: "ok", value: undefined } : outcome;
}

export interface ApiOptions {
  fetch: typeof fetch;
  /** Called on every 401, to bring back to the login. */
  onUnauthorized: () => void;
  /** Beyond this delay, a call is a network error. */
  timeoutMs: number;
}

/** Api calls the API of the tribe of the page. */
export class Api {
  readonly #options: ApiOptions;

  constructor(options: ApiOptions) {
    this.#options = options;
  }

  /** Asks for a login code: the same answer whether the address is a member or not. */
  async requestCode(email: string): Promise<Outcome<undefined>> {
    return toVoid(await this.#call("POST", "api/login-codes", { email }));
  }

  /** Opens a session with the code received. */
  async openSession(email: string, code: string, installedApp: boolean): Promise<Outcome<Session>> {
    return toSession(await this.#call("POST", "api/sessions", { email, code, installedApp }));
  }

  /** The session of this device. */
  async getSession(): Promise<Outcome<Session>> {
    return toSession(await this.#call("GET", "api/session"));
  }

  /** Signs out. */
  async deleteSession(): Promise<Outcome<undefined>> {
    return toVoid(await this.#call("DELETE", "api/session"));
  }

  async #call(method: string, url: string, body?: unknown): Promise<Outcome<unknown>> {
    const init: RequestInit = {
      method,
      credentials: "same-origin",
      headers: { Accept: "application/json" },
      signal: AbortSignal.timeout(this.#options.timeoutMs),
    };
    if (body !== undefined) {
      init.headers = { Accept: "application/json", "Content-Type": "application/json" };
      init.body = JSON.stringify(body);
    }
    let raw: RawResponse;
    try {
      const response = await this.#options.fetch(url, init);
      raw = { status: response.status, body: await jsonBody(response) };
    } catch {
      raw = "network-error";
    }
    const outcome = outcomeOf(raw);
    if (outcome.kind === "unauthorized") {
      this.#options.onUnauthorized();
    }
    return outcome;
  }
}

async function jsonBody(response: Response): Promise<unknown> {
  if (!(response.headers.get("Content-Type") ?? "").startsWith("application/json")) {
    return undefined;
  }
  try {
    return await response.json();
  } catch {
    return undefined;
  }
}
