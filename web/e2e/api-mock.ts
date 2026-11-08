// Simulated responses of the API (D15), defined once, after the contract of the notes of
// lot B in docs/plans/2026-10-03-socle-donnees-et-connexion.md.
import { type Locator, type Page, type Route, test } from "@playwright/test";

/** A simulated response, or a network failure, possibly after a delay. */
export interface Reply {
  status?: number;
  body?: unknown;
  /** No response: the request fails as without network. */
  fail?: boolean;
  delayMs?: number;
  headers?: Record<string, string>;
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
  tribe: { status: 200, body: { name: "Les Démo", size: 4, activeMembers: 4 } },
  networkError: { fail: true },
} satisfies Record<string, Reply | ((n: number) => Reply)>;

/** The replies of each route, in order; the last one repeats. */
export interface Script {
  getSession?: Reply[];
  requestCode?: Reply[];
  openSession?: Reply[];
  deleteSession?: Reply[];
  getTribe?: Reply[];
  getDishes?: Reply[];
  getIngredients?: Reply[];
  createDish?: Reply[];
  /** PUT dishes/<id>, whatever the identifier. */
  updateDish?: Reply[];
  setTribeSize?: Reply[];
  getMealPlan?: Reply[];
  serveDish?: Reply[];
  /** PUT meal-plan/<id>/servings, whatever the identifier. */
  setServings?: Reply[];
  /** DELETE meal-plan/<id>, whatever the identifier. */
  removeServedDish?: Reply[];
  getShoppingLists?: Reply[];
  createList?: Reply[];
  /** PUT shopping-lists/<id>/name, whatever the identifier. */
  renameList?: Reply[];
  setMainList?: Reply[];
  /** DELETE shopping-lists/<id>, whatever the identifier. */
  deleteList?: Reply[];
  /** POST shopping-lists/<id>/items, whatever the identifier. */
  addItem?: Reply[];
  /** DELETE shopping-lists/<id>/items/<item>, whatever the identifiers. */
  removeItem?: Reply[];
  /** PUT shopping-lists/<id>/items/<item>/checked, whatever the identifiers. */
  setChecked?: Reply[];
  /** POST shopping-lists/<id>/checkout, whatever the identifier. */
  checkOut?: Reply[];
  /** POST shopping-lists/<id>/meal-shopping, whatever the identifier. */
  addMealShopping?: Reply[];
}

export interface Received {
  method: string;
  path: string;
  body: unknown;
  /** The query and the conditions of a request of a page (ADR 0029), when it has some. */
  query?: string;
  ifNoneMatch?: string;
  ifMatch?: string;
}

/** Routes the API of the page to the script, and returns the requests it receives. */
export async function mockApi(page: Page, script: Script): Promise<Received[]> {
  const received: Received[] = [];
  const queues: Record<string, Reply[]> = {
    "GET session": [...(script.getSession ?? [replies.noSession])],
    "POST login-codes": [...(script.requestCode ?? [replies.codeSent])],
    "POST sessions": [...(script.openSession ?? [replies.sessionOpened])],
    "DELETE session": [...(script.deleteSession ?? [replies.signedOut])],
    "GET tribe": [...(script.getTribe ?? [replies.tribe])],
    "GET dishes": [...(script.getDishes ?? [pageReply("dishes", 1, [])])],
    "GET ingredients": [...(script.getIngredients ?? [pageReply("ingredients", 1, [])])],
    "POST dishes": [...(script.createDish ?? [replies.networkError])],
    "PUT dishes/{id}": [...(script.updateDish ?? [replies.networkError])],
    "PUT tribe/size": [...(script.setTribeSize ?? [replies.networkError])],
    "GET meal-plan": [...(script.getMealPlan ?? [pageReply("meal_plan", 1, [])])],
    "POST meal-plan": [...(script.serveDish ?? [replies.networkError])],
    "PUT meal-plan/{id}/servings": [...(script.setServings ?? [replies.networkError])],
    "DELETE meal-plan/{id}": [...(script.removeServedDish ?? [replies.networkError])],
    "GET shopping-lists": [...(script.getShoppingLists ?? [pageReply("shopping_lists", 1, [])])],
    "POST shopping-lists": [...(script.createList ?? [replies.networkError])],
    "PUT shopping-lists/{id}/name": [...(script.renameList ?? [replies.networkError])],
    "PUT shopping-lists/main": [...(script.setMainList ?? [replies.networkError])],
    "DELETE shopping-lists/{id}": [...(script.deleteList ?? [replies.networkError])],
    "POST shopping-lists/{id}/items": [...(script.addItem ?? [replies.networkError])],
    "DELETE shopping-lists/{id}/items/{item}": [...(script.removeItem ?? [replies.networkError])],
    "PUT shopping-lists/{id}/items/{item}/checked": [...(script.setChecked ?? [replies.networkError])],
    "POST shopping-lists/{id}/checkout": [...(script.checkOut ?? [replies.networkError])],
    "POST shopping-lists/{id}/meal-shopping": [...(script.addMealShopping ?? [replies.networkError])],
  };
  await page.route("**/api/**", async (route: Route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname.replace(/^.*\/api\//, "");
    const data = request.postData();
    const entry: Received = { method: request.method(), path, body: data === null ? undefined : JSON.parse(data) };
    const query = new URL(request.url()).search;
    const ifNoneMatch = await request.headerValue("if-none-match");
    const ifMatch = await request.headerValue("if-match");
    if (query !== "") {
      entry.query = query;
    }
    if (ifNoneMatch !== null) {
      entry.ifNoneMatch = ifNoneMatch;
    }
    if (ifMatch !== null) {
      entry.ifMatch = ifMatch;
    }
    received.push(entry);
    const routed = routedPath(request.method(), path);
    const queue = queues[`${request.method()} ${routed}`] ?? [{ status: 404, body: { error: "not_found" } }];
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
      headers: { "Cache-Control": "no-store", ...reply.headers },
      ...(reply.body === undefined ? {} : { contentType: "application/json", body: JSON.stringify(reply.body) }),
    });
  });
  return received;
}

/** The route of a path whose identifier is left out. */
function routedPath(method: string, path: string): string {
  if (method === "PUT" && path.startsWith("dishes/")) {
    return "dishes/{id}";
  }
  if (method === "PUT" && /^meal-plan\/[^/]+\/servings$/.test(path)) {
    return "meal-plan/{id}/servings";
  }
  if (method === "DELETE" && path.startsWith("meal-plan/")) {
    return "meal-plan/{id}";
  }
  const shopping = /^shopping-lists\/([^/]+)(\/.*)?$/.exec(path);
  if (shopping !== null && !(method === "PUT" && path === "shopping-lists/main")) {
    const rest = (shopping[2] ?? "").replace(/^\/items\/[^/]+/, "/items/{item}");
    return `shopping-lists/{id}${rest}`;
  }
  return path;
}

/** A page of a collection, with the ETag of its version (ADR 0029). */
export function pageReply(collection: string, version: number, items: unknown[], next?: string): Reply {
  return { status: 200, headers: { ETag: `"${collection}-${version}"` }, body: next === undefined ? { items } : { items, next } };
}

/** The version of a collection has not changed. */
export function notModified(collection: string, version: number): Reply {
  return { status: 304, headers: { ETag: `"${collection}-${version}"` } };
}

/**
 * The number of navigations « Onglets » on a screen opened from a tab, with its link back:
 * none in compact, as its mock-up; the rail stays in moyen and etendu (plan grands-ecrans, D4).
 */
export function tabsOnSubScreen(): number {
  return test.info().project.metadata["format"] === "compact" ? 0 : 1;
}

/**
 * The way back from a meal to the planning: the link « Planning » of the screen, or in etendu
 * the cross of the panel over the planning by week (plan grands-ecrans, D6).
 */
export function mealBackLink(page: Page): Locator {
  return test.info().project.metadata["format"] === "etendu"
    ? page.getByRole("link", { name: "Fermer le repas" })
    : page.getByRole("main").getByRole("link", { name: "Planning", exact: true });
}
