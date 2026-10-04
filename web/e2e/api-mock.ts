// Simulated responses of the API (D15), defined once, after the contract of the notes of
// lot B in docs/plans/2026-10-03-socle-donnees-et-connexion.md.
import type { Page, Route } from "@playwright/test";

/** A simulated response, or a network failure, possibly after a delay. */
export interface Reply {
  status?: number;
  body?: unknown;
  /** No response: the request fails as without network. */
  fail?: boolean;
  delayMs?: number;
}

export const session = { tribe: { name: "Les Démo" }, member: { email: "alice@exemple.fr", displayName: "Alice" } };

export const replies = {
  noSession: { status: 401, body: { error: "no_session" } },
  session: { status: 200, body: session },
  sessionOpened: { status: 201, body: session },
  signedOut: { status: 204 },
  codeSent: { status: 202 },
  invalidEmail: { status: 400, body: { error: "invalid_email" } },
  tooManyRequests: { status: 429, body: { error: "too_many_requests" } },
  incorrectCode: (attemptsLeft: number): Reply => ({ status: 400, body: { error: "incorrect_code", attemptsLeft } }),
  newCodeNeeded: { status: 400, body: { error: "new_code_needed" } },
  networkError: { fail: true },
} satisfies Record<string, Reply | ((n: number) => Reply)>;

/** The replies of each route, in order; the last one repeats. */
export interface Script {
  getSession?: Reply[];
  requestCode?: Reply[];
  openSession?: Reply[];
  deleteSession?: Reply[];
}

export interface Received {
  method: string;
  path: string;
  body: unknown;
}

/** Routes the API of the page to the script, and returns the requests it receives. */
export async function mockApi(page: Page, script: Script): Promise<Received[]> {
  const received: Received[] = [];
  const queues: Record<string, Reply[]> = {
    "GET session": [...(script.getSession ?? [replies.noSession])],
    "POST login-codes": [...(script.requestCode ?? [replies.codeSent])],
    "POST sessions": [...(script.openSession ?? [replies.sessionOpened])],
    "DELETE session": [...(script.deleteSession ?? [replies.signedOut])],
  };
  await page.route("**/api/**", async (route: Route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname.replace(/^.*\/api\//, "");
    const data = request.postData();
    received.push({ method: request.method(), path, body: data === null ? undefined : JSON.parse(data) });
    const queue = queues[`${request.method()} ${path}`] ?? [{ status: 404, body: { error: "not_found" } }];
    const reply = (queue.length > 1 ? queue.shift() : queue[0]) ?? {};
    if (reply.delayMs !== undefined) {
      await new Promise((resolve) => setTimeout(resolve, reply.delayMs));
    }
    if (reply.fail) {
      await route.abort("failed");
      return;
    }
    await route.fulfill({
      status: reply.status ?? 200,
      headers: { "Cache-Control": "no-store" },
      ...(reply.body === undefined ? {} : { contentType: "application/json", body: JSON.stringify(reply.body) }),
    });
  });
  return received;
}
