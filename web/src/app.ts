// The application: checks the session at startup, then shows the login or the planning of
// the tribe, under /tribes/<identifier>/ (ADR 0004, 0006).
import { LitElement, html, nothing } from "lit";
import { keyed } from "lit/directives/keyed.js";
import { Api, type Session } from "./api.ts";
import "./dish-library-screen.ts";
import "./dish-screen.ts";
import type { Dish } from "./dishes.ts";
import { Library } from "./library.ts";
import "./list-picker-screen.ts";
import "./loading-screen.ts";
import "./login-screen.ts";
import type { SignedInEvent } from "./login-screen.ts";
import "./meal-screen.ts";
import "./meal-shopping-screen.ts";
import type { MealShoppingAddedEvent } from "./meal-shopping-screen.ts";
import { type Day, type Mealtime, today } from "./meal-plan.ts";
import { Planning } from "./planning.ts";
import "./planning-screen.ts";
import { type Page, Router, pages, planningAround, tabOf } from "./router.ts";
import type { ShoppingList } from "./shopping.ts";
import "./shopping-list-screen.ts";
import type { Notice } from "./shopping-list-screen.ts";
import { ShoppingLists } from "./shopping-lists.ts";
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
  /** The shopping lists in memory (plan courses, D4, D18). */
  readonly #shopping: ShoppingLists;
  /** The range of the planning last shown, that the link back of a meal goes to (D14). */
  #lastPlanning: Page | undefined;
  /** The shopping list last shown, that « Retour à la liste » goes back to (D15). */
  #lastList: Page = pages.shopping;
  /** What the shopping list shown tells: a list deleted in between (D11); cleared at each page. */
  #notice: Notice | undefined;

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
    this.#shopping = new ShoppingLists(this.#api, this.#library);
    this.addEventListener("click", (e) => this.#followLink(e));
    // Back in the foreground, the copies are checked: what another member changed in between
    // appears (D25, D29; plan planning, D15).
    document.addEventListener("visibilitychange", () => {
      if (document.visibilityState !== "visible" || this.session === undefined) {
        return;
      }
      if (this.page.name === "meal-shopping") {
        void Promise.all([this.#shopping.check(), this.#planning.check()]);
      } else if (tabOf(this.page) === "shopping") {
        void this.#shopping.check();
      } else if (this.#planning.loaded) {
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
    // The screens opened from a tab, with their link back, have no tab bar in compact, as their
    // mock-ups; the rail stays in moyen and etendu (plan grands-ecrans, D4). The navigation
    // comes first, for the order of tabulation (D10), and the screen always keeps the same
    // place, never created again by a change of format (D8; ADR 0030, point 4).
    const subScreen = this.page.name === "new-dish" || this.page.name === "dish" || this.page.name === "tribe" || this.page.name === "meal-shopping";
    return html`<mt-tab-bar
        .active=${this.page.name === "tribe" ? undefined : tabOf(this.page)}
        ?sub=${subScreen}
        .api=${this.#api}
        .planningHref=${this.#router.href(pages.planning)}
        .dishesHref=${this.#router.href(pages.dishes)}
        .shoppingHref=${this.#router.href(pages.shopping)}
        .tribeHref=${this.#router.href(pages.tribe)}
      ></mt-tab-bar>
      <div class="app-content">${this.#screen(this.page)}</div>`;
  }

  #screen(page: Page) {
    switch (page.name) {
      case "planning":
        return html`<mt-planning-screen
          .planning=${this.#planning}
          .api=${this.#api}
          .start=${page.start}
          .tribeHref=${this.#router.href(pages.tribe)}
          .mealHref=${(date: Day, mealtime: Mealtime) => this.#router.href({ name: "meal", date, mealtime })}
          .go=${(start: Day | undefined) => this.#router.navigate(start === undefined ? pages.planning : { name: "planning", start })}
        ></mt-planning-screen>`;
      case "meal":
        return keyed(
          `${page.date} ${page.mealtime}`,
          html`<mt-meal-screen
            .planning=${this.#planning}
            .date=${page.date}
            .mealtime=${page.mealtime}
            .planningHref=${this.#router.href(this.#lastPlanning ?? planningAround(page.date, today()))}
            .newDishHref=${this.#router.href(pages.newDish)}
          ></mt-meal-screen>`,
        );
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
      case "shopping":
        return keyed(
          page.list ?? "main",
          html`<mt-shopping-list-screen
            .lists=${this.#shopping}
            .listId=${page.list}
            .pickerHref=${this.#router.href(pages.listPicker)}
            .mealShoppingHref=${(list: ShoppingList) => this.#router.href({ name: "meal-shopping", list: list.id })}
            .notice=${this.#notice}
            @unknown-list=${() => this.#router.navigate(pages.shopping, { replace: true })}
            @list-deleted=${this.#listDeleted}
          ></mt-shopping-list-screen>`,
        );
      case "list-picker":
        return html`<mt-list-picker
          .lists=${this.#shopping}
          .backHref=${this.#router.href(this.#lastList)}
          .listHref=${(list: ShoppingList) => this.#router.href(this.#listPage(list))}
          @open-list=${(e: CustomEvent<ShoppingList>) => this.#router.navigate(this.#listPage(e.detail))}
          @list-deleted=${this.#listDeleted}
        ></mt-list-picker>`;
      case "meal-shopping":
        return html`<mt-meal-shopping-screen
          .lists=${this.#shopping}
          .planning=${this.#planning}
          .listId=${page.list}
          .listHref=${(id: string | undefined) => this.#router.href(this.#listPageOf(id))}
          @meal-shopping-added=${this.#mealShoppingAdded}
          @list-deleted=${this.#listDeleted}
        ></mt-meal-shopping-screen>`;
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
    this.#notice = undefined;
    if (page.name === "planning") {
      this.#lastPlanning = page;
    }
    if (page.name === "shopping") {
      this.#lastList = page;
    }
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

  /** The page of a list: the main list at the path of the tab (D15). */
  #listPage(list: ShoppingList): Page {
    return list.main ? pages.shopping : { name: "shopping", list: list.id };
  }

  /**
   * The page of a list known by its identifier only: the main list at the path of the tab
   * (D15), as a list unknown once the copy is loaded. Asked again by the screen at each
   * render, the copy loaded or changed (review of PR #101, point 2).
   */
  #listPageOf(id: string | undefined): Page {
    const list = id === undefined ? undefined : this.#shopping.find(id);
    if (id === undefined || list?.main === true || (this.#shopping.loaded && list === undefined)) {
      return pages.shopping;
    }
    return { name: "shopping", list: id };
  }

  /** The meal shopping added: back to the target list, which tells what was added (C4, Q22). */
  readonly #mealShoppingAdded = (e: MealShoppingAddedEvent) => {
    this.#router.navigate(this.#listPage(e.list), { replace: true });
    // After the navigation, which clears the notice of the page left.
    this.#notice = { kind: "done", text: e.message };
    this.requestUpdate();
  };

  /** A write on a list deleted in between: the main list opens, and tells it (D11). */
  readonly #listDeleted = () => {
    this.#router.navigate(pages.shopping);
    // After the navigation, which clears the notice of the page left.
    this.#notice = { kind: "deleted" };
    this.requestUpdate();
  };

  /** Forgets the copies in memory: at the sign-out, or on a 401 (D25). */
  #clearCopies() {
    this.#shopping.clear();
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
