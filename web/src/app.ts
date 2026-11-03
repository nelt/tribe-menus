// The application: checks the session at startup, then shows the login or the home page
// of the tribe, under /tribes/<identifier>/ (ADR 0004, 0006).
import { LitElement, html } from "lit";
import { Api, type Session } from "./api.ts";
import "./dish-library-screen.ts";
import type { Dish } from "./dishes.ts";
import { Library } from "./library.ts";
import "./home-screen.ts";
import "./loading-screen.ts";
import "./login-screen.ts";
import type { SignedInEvent } from "./login-screen.ts";
import { type Page, Router, pages, tabOf } from "./router.ts";
import "./tab-bar.ts";

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
  /** The dish library and the catalog in memory, shared by the screens (D25). */
  readonly #library: Library;

  constructor() {
    super();
    const base = new URL(document.baseURI).pathname;
    this.#router = new Router(window, base, (page) => this.#show(page));
    this.page = this.#router.page;
    this.#api = new Api({
      fetch: (input, init) => window.fetch(input, init),
      onUnauthorized: () => {
        this.session = undefined;
        this.#library.clear();
        this.#router.navigate(pages.login, { replace: true });
      },
      timeoutMs: requestTimeoutMs,
    });
    this.#library = new Library(this.#api);
    this.addEventListener("click", (e) => this.#followLink(e));
    // Back in the foreground, the copy is checked: what another member changed in between
    // appears (D25, D29).
    document.addEventListener("visibilitychange", () => {
      if (document.visibilityState === "visible" && this.session !== undefined && this.#library.loaded) {
        void this.#library.check();
      }
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
    if (this.session === undefined || this.page.name === "login") {
      return html`<mt-login .api=${this.#api} @signed-in=${this.#signedIn}></mt-login>`;
    }
    return html`${this.#screen(this.page, this.session)}
      <mt-tab-bar active=${tabOf(this.page)} .planningHref=${this.#router.href(pages.home)} .dishesHref=${this.#router.href(pages.dishes)}></mt-tab-bar>`;
  }

  #screen(page: Page, session: Session) {
    switch (page.name) {
      case "home":
        return html`<mt-home .api=${this.#api} .session=${session} .tribeHref=${this.#router.href(pages.tribe)} @signed-out=${this.#signedOut}></mt-home>`;
      case "dishes":
        return html`<mt-dish-library
          .library=${this.#library}
          .newDishHref=${this.#router.href(pages.newDish)}
          .dishHref=${(dish: Dish) => this.#router.href({ name: "dish", id: dish.id })}
        ></mt-dish-library>`;
      default:
        return comingSoon(page);
    }
  }

  /** A link to a page of the tribe is shown by the router, without reloading the page. */
  #followLink(e: MouseEvent) {
    if (e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) {
      return;
    }
    const link = e.target instanceof Element ? e.target.closest("a") : null;
    if (link === null || link.target !== "" || link.origin !== location.origin) {
      return;
    }
    if (this.#router.follow(link.pathname)) {
      e.preventDefault();
    }
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
    if (this.session !== undefined && page.name === "login") {
      this.#router.navigate(pages.home, { replace: true });
    } else if (this.session === undefined && page.name !== "login") {
      this.#router.navigate(pages.login, { replace: true });
    }
  }

  #signedIn(e: SignedInEvent) {
    this.session = e.session;
    this.#router.navigate(pages.home, { replace: true });
  }

  #signedOut() {
    this.session = undefined;
    this.#library.clear();
    this.#router.navigate(pages.login, { replace: true });
  }
}

/** The screens of the next lot of the plan bibliotheque: the creation and the change of a
 * dish, the screen Tribu. */
function comingSoon(page: Page) {
  const title = page.name === "tribe" ? "Tribu" : "Plats";
  return html`<main class="screen">
    <h1 class="screen-title">${title}</h1>
    <p class="screen-text">Cet écran arrive bientôt.</p>
  </main>`;
}

customElements.define("mt-app", App);

declare global {
  interface HTMLElementTagNameMap {
    "mt-app": App;
  }
}
