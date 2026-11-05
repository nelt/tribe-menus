// The application: checks the session at startup, then shows the login or the planning of
// the tribe, under /tribes/<identifier>/ (ADR 0004, 0006).
import { LitElement, html, nothing } from "lit";
import { keyed } from "lit/directives/keyed.js";
import { Api, type Session } from "./api.ts";
import "./dish-library-screen.ts";
import "./dish-screen.ts";
import type { Dish } from "./dishes.ts";
import { Library } from "./library.ts";
import "./loading-screen.ts";
import "./login-screen.ts";
import type { SignedInEvent } from "./login-screen.ts";
import type { Day } from "./meal-plan.ts";
import { Planning } from "./planning.ts";
import "./planning-screen.ts";
import { type Page, Router, pages, tabOf } from "./router.ts";
import "./stepper.ts";
import "./tab-bar.ts";
import "./tribe-screen.ts";

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
  /** The meal plan in memory, checked with the library (plan planning, D15). */
  readonly #planning: Planning;

  constructor() {
    super();
    const base = new URL(document.baseURI).pathname;
    this.#router = new Router(window, base, (page) => this.#show(page));
    this.page = this.#router.page;
    this.#api = new Api({
      fetch: (input, init) => window.fetch(input, init),
      onUnauthorized: () => {
        this.session = undefined;
        this.#clearCopies();
        this.#router.navigate(pages.login, { replace: true });
      },
      timeoutMs: requestTimeoutMs,
    });
    this.#library = new Library(this.#api);
    this.#planning = new Planning(this.#api, this.#library);
    this.addEventListener("click", (e) => this.#followLink(e));
    // Back in the foreground, the copies are checked: what another member changed in between
    // appears (D25, D29; plan planning, D15).
    document.addEventListener("visibilitychange", () => {
      if (document.visibilityState !== "visible" || this.session === undefined) {
        return;
      }
      if (this.#planning.loaded) {
        void this.#planning.check();
      } else if (this.#library.loaded) {
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
    // The screens opened from a tab, with their link back, have no tab bar, as their mock-ups.
    const subScreen = this.page.name === "new-dish" || this.page.name === "dish" || this.page.name === "tribe";
    return html`${this.#screen(this.page)}
      ${subScreen
        ? nothing
        : html`<mt-tab-bar active=${tabOf(this.page)} .planningHref=${this.#router.href(pages.planning)} .dishesHref=${this.#router.href(pages.dishes)}></mt-tab-bar>`}`;
  }

  #screen(page: Page) {
    switch (page.name) {
      case "planning":
        return html`<mt-planning-screen
          .planning=${this.#planning}
          .api=${this.#api}
          .start=${page.start}
          .tribeHref=${this.#router.href(pages.tribe)}
          .go=${(start: Day | undefined) => this.#router.navigate(start === undefined ? pages.planning : { name: "planning", start })}
        ></mt-planning-screen>`;
      case "dishes":
        return html`<mt-dish-library
          .library=${this.#library}
          .newDishHref=${this.#router.href(pages.newDish)}
          .dishHref=${(dish: Dish) => this.#router.href({ name: "dish", id: dish.id })}
        ></mt-dish-library>`;
      case "new-dish":
      case "dish": {
        const id = page.name === "dish" ? page.id : undefined;
        return keyed(
          id ?? "new",
          html`<mt-dish-screen
            .library=${this.#library}
            .api=${this.#api}
            .dishId=${id}
            .libraryHref=${this.#router.href(pages.dishes)}
            @saved=${() => this.#router.navigate(pages.dishes, { replace: true })}
          ></mt-dish-screen>`,
        );
      }
      case "tribe":
        return html`<mt-tribe-screen .api=${this.#api} .planningHref=${this.#router.href(pages.planning)} @signed-out=${this.#signedOut}></mt-tribe-screen>`;
      case "login":
        return nothing;
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
    if (this.#router.follow(link.pathname, link.search)) {
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
      this.#router.navigate(pages.planning, { replace: true });
    } else if (this.session === undefined && page.name !== "login") {
      this.#router.navigate(pages.login, { replace: true });
    }
  }

  #signedIn(e: SignedInEvent) {
    this.session = e.session;
    this.#router.navigate(pages.planning, { replace: true });
  }

  #signedOut() {
    this.session = undefined;
    this.#clearCopies();
    this.#router.navigate(pages.login, { replace: true });
  }

  /** Forgets the copies in memory: at the sign-out, or on a 401 (D25). */
  #clearCopies() {
    this.#planning.clear();
    this.#library.clear();
  }
}

customElements.define("mt-app", App);

declare global {
  interface HTMLElementTagNameMap {
    "mt-app": App;
  }
}
