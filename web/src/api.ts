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

/** The tribe as the screen Tribu and the chip of the planning show it (T3). */
export interface TribeInfo {
  name: string;
  size: number;
  activeMembers: number;
}

/** Returns the tribe described by a body, or undefined if the body is not one. */
export function tribeInfoOf(body: unknown): TribeInfo | undefined {
  if (!isRecord(body)) {
    return undefined;
  }
  const { name, size, activeMembers } = body;
  if (typeof name !== "string" || typeof size !== "number" || typeof activeMembers !== "number") {
    return undefined;
  }
  return { name, size, activeMembers };
}

/**
 * A page of a collection (ADR 0029): its items and the cursor of the next one, with the
 * ETag of its version; or, on a conditional request, the version unchanged (304) or changed
 * during the loading (412).
 */
export type CollectionPage =
  | { kind: "page"; items: unknown[]; next: string | undefined; etag: string }
  | { kind: "not-modified" }
  | { kind: "changed" };

/** Reads the response to a request of a page; undefined if it is not one. */
export function collectionPageOf(response: RawResponse, etag: string | null): CollectionPage | undefined {
  if (response === "network-error") {
    return undefined;
  }
  if (response.status === 304) {
    return { kind: "not-modified" };
  }
  const { status, body } = response;
  if (status === 412 && isRecord(body) && body["error"] === "collection_changed") {
    return { kind: "changed" };
  }
  if (status !== 200 || etag === null || !isRecord(body) || !Array.isArray(body["items"])) {
    return undefined;
  }
  const next = body["next"];
  if (next !== undefined && typeof next !== "string") {
    return undefined;
  }
  return { kind: "page", items: body["items"] as unknown[], next, etag };
}

/** Conditions of a request of a page: If-None-Match on the first, If-Match on the next. */
export interface PageCondition {
  ifNoneMatch?: string;
  ifMatch?: string;
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

  /** The tribe: its name, its size and its number of active members. */
  async getTribe(): Promise<Outcome<TribeInfo>> {
    const outcome = await this.#call("GET", "api/tribe");
    if (outcome.kind !== "ok") {
      return outcome;
    }
    const tribe = tribeInfoOf(outcome.value);
    return tribe === undefined ? { kind: "network-error" } : { kind: "ok", value: tribe };
  }

  /** A page of a collection of the API (ADR 0029), after the cursor given. */
  async getPage(collection: string, after: string | undefined, condition: PageCondition = {}): Promise<Outcome<CollectionPage>> {
    const headers: Record<string, string> = {};
    if (condition.ifNoneMatch !== undefined) {
      headers["If-None-Match"] = condition.ifNoneMatch;
    }
    if (condition.ifMatch !== undefined) {
      headers["If-Match"] = condition.ifMatch;
    }
    const url = after === undefined ? `api/${collection}` : `api/${collection}?after=${encodeURIComponent(after)}`;
    const { raw, etag } = await this.#send("GET", url, undefined, headers);
    const page = collectionPageOf(raw, etag);
    if (page !== undefined) {
      return { kind: "ok", value: page };
    }
    const outcome = this.#outcome(raw);
    return outcome.kind === "ok" ? { kind: "network-error" } : outcome;
  }

  async #call(method: string, url: string, body?: unknown): Promise<Outcome<unknown>> {
    return this.#outcome((await this.#send(method, url, body)).raw);
  }

  #outcome(raw: RawResponse): Outcome<unknown> {
    const outcome = outcomeOf(raw);
    if (outcome.kind === "unauthorized") {
      this.#options.onUnauthorized();
    }
    return outcome;
  }

  async #send(method: string, url: string, body?: unknown, extra: Record<string, string> = {}): Promise<{ raw: RawResponse; etag: string | null }> {
    const headers: Record<string, string> = { Accept: "application/json", ...extra };
    const init: RequestInit = {
      method,
      credentials: "same-origin",
      headers,
      signal: AbortSignal.timeout(this.#options.timeoutMs),
    };
    if (body !== undefined) {
      headers["Content-Type"] = "application/json";
      init.body = JSON.stringify(body);
    }
    try {
      const response = await this.#options.fetch(url, init);
      return { raw: { status: response.status, body: await jsonBody(response) }, etag: response.headers.get("ETag") };
    } catch {
      return { raw: "network-error", etag: null };
    }
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
