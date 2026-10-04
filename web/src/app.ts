// The application: checks the session at startup, then shows the login or the home page
// of the tribe, under /tribes/<identifier>/ (ADR 0004, 0006).
import { LitElement, html } from "lit";
import { Api, type Session } from "./api.ts";
import "./home-screen.ts";
import "./loading-screen.ts";
import "./login-screen.ts";
import type { SignedInEvent } from "./login-screen.ts";
import { type Page, Router } from "./router.ts";

/** Beyond this delay, a call to the API is a network error. */
const requestTimeoutMs = 10_000;
/** The loading screen appears only beyond this delay, to avoid a flash (docs/design). */
const loadingDelayMs = 300;

/** Where the startup is: checking the session (blank, then loading), slow network, or done. */
type Startup = "checking" | "loading" | "slow-network" | "done";

export class App extends LitElement {
  static override properties = {
    startup: { state: true },
    session: { state: true },
    page: { state: true },
  };

  startup: Startup = "checking";
  session: Session | undefined = undefined;
  declare page: Page;
  readonly #router: Router;
  readonly #api: Api;

  constructor() {
    super();
    const base = new URL(document.baseURI).pathname;
    this.#router = new Router(window, base, (page) => this.#show(page));
    this.page = this.#router.page;
    this.#api = new Api({
      fetch: (input, init) => window.fetch(input, init),
      onUnauthorized: () => {
        this.session = undefined;
        this.#router.navigate("login", { replace: true });
      },
      timeoutMs: requestTimeoutMs,
    });
  }

  protected override createRenderRoot() {
    return this;
  }

  override connectedCallback() {
    super.connectedCallback();
    void this.#checkSession();
  }

  override render() {
    switch (this.startup) {
      case "checking":
        return html``;
      case "loading":
        return html`<mt-loading></mt-loading>`;
      case "slow-network":
        return html`<mt-loading slow @retry=${this.#checkSession}></mt-loading>`;
    }
    if (this.session === undefined || this.page === "login") {
      return html`<mt-login .api=${this.#api} @signed-in=${this.#signedIn}></mt-login>`;
    }
    return html`<mt-home .api=${this.#api} .session=${this.session} @signed-out=${this.#signedOut}></mt-home>`;
  }

  async #checkSession() {
    this.startup = "checking";
    const timer = setTimeout(() => {
      if (this.startup === "checking") {
        this.startup = "loading";
      }
    }, loadingDelayMs);
    const outcome = await this.#api.getSession();
    clearTimeout(timer);
    if (outcome.kind === "network-error") {
      this.startup = "slow-network";
      return;
    }
    // On a 401, the API client has already gone to the login.
    this.session = outcome.kind === "ok" ? outcome.value : undefined;
    this.startup = "done";
    this.#show(this.#router.page);
  }

  /** Shows a page, or redirects: the login once signed in, the login without a session. */
  #show(page: Page) {
    this.page = page;
    if (this.startup !== "done") {
      return;
    }
    if (this.session !== undefined && page === "login") {
      this.#router.navigate("home", { replace: true });
    } else if (this.session === undefined && page !== "login") {
      this.#router.navigate("login", { replace: true });
    }
  }

  #signedIn(e: SignedInEvent) {
    this.session = e.session;
    this.#router.navigate("home", { replace: true });
  }

  #signedOut() {
    this.session = undefined;
    this.#router.navigate("login", { replace: true });
  }
}

customElements.define("mt-app", App);

declare global {
  interface HTMLElementTagNameMap {
    "mt-app": App;
  }
}
