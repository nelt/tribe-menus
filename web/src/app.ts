// The application: checks the session at startup, then shows the login or the planning of
// the tribe, under /tribes/<identifier>/ (ADR 0004, 0006).
import { LitElement, type PropertyValues, html, nothing } from "lit";
import { keyed } from "lit/directives/keyed.js";
import { Api, type Session } from "./api.ts";
import { chooseDish } from "./dish-library-screen.ts";
import "./dish-screen.ts";
import type { Dish } from "./dishes.ts";
import { type Format, currentFormat, onFormatChange, onFormatWillChange } from "./layout.ts";
import { Library } from "./library.ts";
import "./list-picker-screen.ts";
import "./loading-screen.ts";
import "./login-screen.ts";
import type { SignedInEvent } from "./login-screen.ts";
import "./meal-screen.ts";
import "./meal-shopping-screen.ts";
import type { MealShoppingAddedEvent } from "./meal-shopping-screen.ts";
import { type Detail, type ListPane, detailKey, detailOf, holdsUnsaved, listPaneOf, lostBy, paneAttribute } from "./panes.ts";
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
    format: { state: true },
    abandoning: { state: true },
  };

  startup: Startup = "checking";
  session: Session | undefined = undefined;
  declare page: Page;
  /** The format of the screen: in etendu, a meal is a panel over the planning (D6). */
  format: Format = currentFormat();
  /** The page a navigation would show, once the loss of an input not saved is confirmed (D9). */
  abandoning: Page | undefined = undefined;
  #detailScroll: number | undefined;
  #unfollowFormat: (() => void) | undefined;
  #unfollowFormatWill: (() => void) | undefined;
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
  /** The dish created beside the library, opened with « Plat enregistré. »; cleared at each page. */
  #created: string | undefined;

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
    document.addEventListener("keydown", (e) => this.#closeMealOnEscape(e));
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
    this.#unfollowFormat = onFormatChange((format) => (this.format = format));
    this.#unfollowFormatWill = onFormatWillChange(() => this.#keepDetailScroll());
    this.format = currentFormat();
    void this.#checkSession();
  }

  override disconnectedCallback() {
    super.disconnectedCallback();
    this.#unfollowFormat?.();
    this.#unfollowFormatWill?.();
  }

  /**
   * In etendu, the focus goes to the panel of a meal opened from the planning, and back to the
   * tile of the meal once it is closed (D10).
   */
  /**
   * How far the detail is scrolled, just before a change of format, kept for the new one (D8):
   * the window scrolls with one pane, the detail on its own beside the pane of a list.
   */
  #keepDetailScroll() {
    const detail = this.#detailElement();
    this.#detailScroll = detail === null ? undefined : this.querySelector(".app-content--panes") === null ? window.scrollY : detail.scrollTop;
  }

  /** The element of the detail: a dish, a list, the meal shopping (panes.ts). */
  #detailElement(): HTMLElement | null {
    return this.querySelector<HTMLElement>(`.app-content > [${paneAttribute}="detail"]`);
  }

  protected override updated(changed: PropertyValues<this>) {
    // The detail scrolls on its own beside the pane of a list, with the window otherwise.
    const scroll = this.#detailScroll;
    this.#detailScroll = undefined;
    const detail = this.#detailElement();
    if (scroll !== undefined && detail !== null) {
      if (this.querySelector(".app-content--panes") === null) {
        window.scrollTo({ top: scroll });
      } else {
        detail.scrollTop = scroll;
      }
    }
    const dialog = this.querySelector<HTMLDialogElement>("dialog.abandon");
    if (changed.has("abandoning") && dialog !== null && !dialog.open) {
      dialog.showModal();
    }
    const before = changed.get("page") as Page | undefined;
    if (before === undefined || this.format !== "etendu") {
      return;
    }
    if (this.page.name === "meal" && before.name === "planning") {
      void this.querySelector("mt-meal-screen")?.updateComplete.then(() => this.querySelector<HTMLElement>(".meal-close")?.focus());
    } else if (this.page.name === "planning" && before.name === "meal") {
      // Once the planning is no longer inert.
      const tile = `a.meal-tile[href="${this.#router.href(before)}"]`;
      void this.querySelector("mt-planning-screen")?.updateComplete.then(() => this.querySelector<HTMLElement>(tile)?.focus());
    }
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
    // The pane of a list and the detail, each in a place of its own (plan grands-ecrans, D5,
    // D8): side by side in etendu.
    const listPane = listPaneOf(this.page, this.format);
    const detail = detailOf(this.page, this.format);
    const panes = this.format === "etendu" && listPane !== undefined;
    return html`<mt-tab-bar
        .active=${this.page.name === "tribe" ? undefined : tabOf(this.page)}
        ?sub=${subScreen}
        .api=${this.#api}
        .planningHref=${this.#router.href(pages.planning)}
        .dishesHref=${this.#router.href(pages.dishes)}
        .shoppingHref=${this.#router.href(pages.shopping)}
        .tribeHref=${this.#router.href(pages.tribe)}
      ></mt-tab-bar>
      <div class="app-content ${panes ? "app-content--panes" : ""}">
        ${this.#planningBehind(this.page)}${this.#listPane(this.page, listPane, panes)}${this.#detail(detail, panes)}${this.#screen(this.page)}
      </div>
      ${this.#abandonDialog()}`;
  }

  /** The range of the planning a meal goes back to: the last one shown, or the one that holds it. */
  #planningOf(page: Page & { name: "meal" }): Page {
    return this.#lastPlanning ?? planningAround(page.date, today());
  }

  /**
   * The planning, in the first place of the content: on its page, and in etendu under the panel
   * of a meal, where it is the same element, so the planning by week keeps its scroll (D6). Under
   * the panel, it is covered: one main, the meal's.
   */
  #planningBehind(page: Page) {
    let start: Day | undefined;
    if (page.name === "planning") {
      start = page.start;
    } else if (page.name === "meal" && this.format === "etendu") {
      const behind = this.#planningOf(page);
      start = behind.name === "planning" ? behind.start : undefined;
    } else {
      return nothing;
    }
    return html`<mt-planning-screen
      .covered=${page.name === "meal"}
      .planning=${this.#planning}
      .api=${this.#api}
      .start=${start}
      .tribeHref=${this.#router.href(pages.tribe)}
      .mealHref=${(date: Day, mealtime: Mealtime) => this.#router.href({ name: "meal", date, mealtime })}
      .go=${(s: Day | undefined) => this.#router.navigate(s === undefined ? pages.planning : { name: "planning", start: s })}
    ></mt-planning-screen>`;
  }

  /** Échap closes the panel of a meal, in etendu, unless a dialog is open (D10). */
  #closeMealOnEscape(e: KeyboardEvent) {
    if (e.key !== "Escape" || e.defaultPrevented || this.page.name !== "meal" || this.format !== "etendu" || this.querySelector("dialog[open]") !== null) {
      return;
    }
    this.#router.navigate(this.#planningOf(this.page));
  }

  #screen(page: Page) {
    switch (page.name) {
      case "planning":
        // In the first place of the content (#planningBehind).
        return nothing;
      case "meal":
        // Always in the same place, a panel over the planning in etendu after data-format: a
        // change of format never creates the meal again (D8).
        return html`<div class="meal-overlay" @click=${() => this.#router.navigate(this.#planningOf(page))}></div>
          <div class="meal-panel">
            ${keyed(
              `${page.date} ${page.mealtime}`,
              html`<mt-meal-screen
                .planning=${this.#planning}
                .date=${page.date}
                .mealtime=${page.mealtime}
                .planningHref=${this.#router.href(this.#planningOf(page))}
                .newDishHref=${this.#router.href(pages.newDish)}
              ></mt-meal-screen>`,
            )}
          </div>`;
      case "dishes":
      case "new-dish":
      case "dish":
      case "shopping":
      case "list-picker":
      case "meal-shopping":
        // In the panes (#listPane, #detail).
        return nothing;
      case "tribe":
        return html`<mt-tribe-screen .api=${this.#api} .planningHref=${this.#router.href(pages.planning)} @signed-out=${this.#signedOut}></mt-tribe-screen>`;
      case "login":
        return nothing;
    }
  }

  /** The pane of a list: the library, or the list picker; beside the detail in etendu (D5). */
  #listPane(page: Page, pane: ListPane | undefined, beside: boolean) {
    switch (pane) {
      case "library":
        return html`<mt-dish-library
          data-pane="list"
          .library=${this.#library}
          .newDishHref=${this.#router.href(pages.newDish)}
          .dishHref=${(dish: Dish) => this.#router.href({ name: "dish", id: dish.id })}
          .current=${page.name === "dish" ? page.id : undefined}
          .beside=${beside}
        ></mt-dish-library>`;
      case "lists": {
        const list = page.name === "shopping" || page.name === "meal-shopping" ? page.list : undefined;
        return html`<mt-list-picker
          data-pane="list"
          .lists=${this.#shopping}
          .backHref=${this.#router.href(this.#lastList)}
          .listHref=${(l: ShoppingList) => this.#router.href(this.#listPage(l))}
          .current=${beside ? { list } : undefined}
          .beside=${beside}
          @open-list=${(e: CustomEvent<ShoppingList>) => this.#go(this.#listPage(e.detail))}
          @list-deleted=${this.#listDeleted}
        ></mt-list-picker>`;
      }
      case undefined:
        return nothing;
    }
  }

  /**
   * The detail: a dish, a list, the meal shopping, or in etendu the invitation to choose a dish.
   * The same detail is the same element, whatever the format (D8).
   */
  #detail(detail: Detail | undefined, beside: boolean) {
    if (detail === undefined) {
      return nothing;
    }
    switch (detail.kind) {
      case "choose-dish":
        return chooseDish(this.#router.href(pages.newDish));
      case "dish":
        return keyed(
          detailKey(detail),
          html`<mt-dish-screen
            data-pane="detail"
            .library=${this.#library}
            .api=${this.#api}
            .dishId=${detail.id}
            .libraryHref=${this.#router.href(pages.dishes)}
            .beside=${beside}
            .created=${detail.id !== undefined && detail.id === this.#created}
            @saved=${this.#dishSaved}
          ></mt-dish-screen>`,
        );
      case "list":
        return keyed(
          detailKey(detail),
          html`<mt-shopping-list-screen
            data-pane="detail"
            .lists=${this.#shopping}
            .listId=${detail.list}
            .pickerHref=${this.#router.href(pages.listPicker)}
            .mealShoppingHref=${(list: ShoppingList) => this.#router.href({ name: "meal-shopping", list: list.id })}
            .notice=${this.#notice}
            .beside=${beside}
            @unknown-list=${() => this.#router.navigate(pages.shopping, { replace: true })}
            @list-deleted=${this.#listDeleted}
          ></mt-shopping-list-screen>`,
        );
      case "meal-shopping":
        return html`<mt-meal-shopping-screen
          data-pane="detail"
          .lists=${this.#shopping}
          .planning=${this.#planning}
          .listId=${detail.list}
          .listHref=${(id: string | undefined) => this.#router.href(this.#listPageOf(id))}
          @meal-shopping-added=${this.#mealShoppingAdded}
          @list-deleted=${this.#listDeleted}
        ></mt-meal-shopping-screen>`;
    }
  }

  /** « Abandonner la modification ? » (D9; GE-Plats-*), while a navigation waits for it. */
  #abandonDialog() {
    if (this.abandoning === undefined) {
      return nothing;
    }
    return html`<dialog class="sheet abandon" aria-labelledby="abandon-title" aria-describedby="abandon-text" @close=${() => (this.abandoning = undefined)} @click=${this.#closeAbandonOnBackdrop}>
      <h2 class="sheet-title" id="abandon-title">Abandonner la modification ?</h2>
      <p class="sheet-text" id="abandon-text">Ce que vous avez saisi sera perdu.</p>
      <button class="button-primary" type="button" @click=${this.#abandon}>Abandonner</button>
      <button class="button-text button-text--wide" type="button" autofocus @click=${this.#keepEditing}>Continuer la modification</button>
    </dialog>`;
  }

  #closeAbandonOnBackdrop(e: MouseEvent) {
    if (e.target instanceof HTMLDialogElement) {
      this.#keepEditing();
    }
  }

  #keepEditing() {
    this.querySelector<HTMLDialogElement>("dialog.abandon")?.close();
    this.abandoning = undefined;
  }

  /** The input is lost: the navigation waiting goes on. */
  #abandon() {
    const page = this.abandoning;
    this.#keepEditing();
    if (page !== undefined) {
      this.#router.navigate(page);
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
    const page = this.#router.pageAt(link.pathname, link.search);
    if (page === undefined) {
      return;
    }
    e.preventDefault();
    this.#go(page);
  }

  /**
   * Goes to a page asked by the member. In moyen and etendu, a navigation that would lose an
   * input not saved waits for a confirmation (D9): the format counts at the time of the
   * navigation. Every navigation asked by the member goes through here.
   */
  #go(page: Page) {
    if (this.format !== "compact" && this.#loses(page)) {
      this.abandoning = page;
      return;
    }
    this.#router.navigate(page);
  }

  /** Whether going to a page loses an input not saved: in a pane whose content changes (D8, D9). */
  #loses(page: Page): boolean {
    const lost = lostBy(this.page, page, this.format);
    const unsaved = (pane: "list" | "detail") => holdsUnsaved(this.querySelector(`.app-content > [${paneAttribute}="${pane}"]`));
    return (lost.list && unsaved("list")) || (lost.detail && unsaved("detail"));
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
    this.#created = undefined;
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

  /**
   * A dish saved: back to the library; beside it, in etendu, the dish stays open (GE-Plats-*),
   * a dish created opening at its own path with « Plat enregistré. ».
   */
  readonly #dishSaved = (e: CustomEvent<Dish>) => {
    if (this.format !== "etendu") {
      this.#router.navigate(pages.dishes, { replace: true });
      return;
    }
    if (this.page.name === "new-dish") {
      this.#router.navigate({ name: "dish", id: e.detail.id }, { replace: true });
      // After the navigation, which clears it.
      this.#created = e.detail.id;
      this.requestUpdate();
    }
  };

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
