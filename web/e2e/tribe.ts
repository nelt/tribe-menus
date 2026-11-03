// The tribe of a scenario @ui: created by e2efixture in the data directory of the server of
// the tests, with a session for each person (ADR 0028). The requests of the page to the API
// are relayed with the cookie of the session added, WebKit refusing the Secure cookie on
// http://localhost (plan socle, D14); route.continue cannot set a cookie, hence route.fetch.
// WebKit cannot be given back a 304 by Playwright either: under WebKit, the relay drops
// If-None-Match, and a check always reloads the copy. The 304 is proved under Chromium and by
// the unit tests of collections.ts.
import { execFile } from "node:child_process";
import { createHash, randomBytes } from "node:crypto";
import { promisify } from "node:util";
import type { APIRequestContext, APIResponse, Page } from "@playwright/test";

// Written by npm run e2e (package.json) and by the server of the tests (playwright.config.ts).
const fixtureTool = ".e2e-tools/e2efixture";
const dataDir = ".e2e-data";

/** The people of the scenarios, by the name the steps give them. */
export const people: Record<string, string> = {
  je: "moi@exemple.fr",
  Alice: "alice@exemple.fr",
  Bruno: "bruno@exemple.fr",
};

/** The name of a cookie of the tribe (ADR 0024), as internal/server computes it. */
export function cookieName(kind: string, slug: string): string {
  return `__Host-${kind}-${createHash("sha256").update(slug).digest("hex").slice(0, 16)}`;
}

export class Tribe {
  readonly slug: string;
  readonly #tokens: Record<string, string>;
  readonly #request: APIRequestContext;
  readonly #origin: string;

  private constructor(slug: string, tokens: Record<string, string>, request: APIRequestContext, origin: string) {
    this.slug = slug;
    this.#tokens = tokens;
    this.#request = request;
    this.#origin = origin;
  }

  /** Creates a tribe of its own for the scenario, with the people of the scenarios. */
  static async create(request: APIRequestContext, baseURL: string): Promise<Tribe> {
    const slug = `e2e-${randomBytes(6).toString("hex")}`;
    const { stdout } = await promisify(execFile)(fixtureTool, ["-data", dataDir, slug, ...Object.values(people)]);
    return new Tribe(slug, JSON.parse(stdout) as Record<string, string>, request, new URL(baseURL).origin);
  }

  /** The path of a page of the tribe. */
  path(rest: string): string {
    return `/tribes/${this.slug}/${rest}`;
  }

  #cookie(who: string): string {
    const email = people[who];
    const token = email === undefined ? undefined : this.#tokens[email];
    if (token === undefined) {
      throw new Error(`no session for ${who}`);
    }
    return `${cookieName("session", this.slug)}=${token}`;
  }

  /** Makes the requests of the page to the API those of this person. */
  async signInPage(page: Page, who: string): Promise<void> {
    const cookie = this.#cookie(who);
    const webkit = page.context().browser()?.browserType().name() === "webkit";
    await page.route(`**${this.path("api/")}**`, async (route) => {
      const headers: Record<string, string> = { ...route.request().headers(), cookie };
      if (webkit) {
        delete headers["if-none-match"];
      }
      const response = await route.fetch({ headers });
      await route.fulfill({ response });
    });
  }

  /** A request of this person to the API, from outside the browser, as the front end. */
  async api(who: string, method: string, path: string, body?: unknown): Promise<APIResponse> {
    const response = await this.#request.fetch(this.path(`api/${path}`), {
      method,
      headers: { cookie: this.#cookie(who), origin: this.#origin },
      ...(body === undefined ? {} : { data: body }),
    });
    return response;
  }

  /** Like api, for a request that must succeed: its JSON body. */
  async apiOk(who: string, method: string, path: string, body?: unknown): Promise<unknown> {
    const response = await this.api(who, method, path, body);
    if (!response.ok()) {
      throw new Error(`${method} ${path}: ${response.status()} ${await response.text()}`);
    }
    return response.json();
  }
}
