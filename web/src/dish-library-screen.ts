// Screen Plats: the dish library and its search (P3), after the mock-up Doux-Bibliotheque.
// The library is checked at each visit (plan bibliotheque, D25); the search runs in the
// browser, at each keystroke, on the copy in memory (ADR 0027).
import { LitElement, html, nothing } from "lit";
import { type Dish, type Result, initialOf, searchDishes } from "./dishes.ts";
import type { Library } from "./library.ts";
import { networkErrorMessage } from "./login.ts";
import { alertMessage } from "./templates.ts";

/** The hues of the palette, in its order (docs/design/README.md): class dish-color-<n>. */
const palette = ["#c2502e", "#3f7f25", "#2f6fb0", "#b0306a", "#b36a0b", "#8a45b3", "#2e8a6e", "#9c7400", "#6e6680"];

/** The class of the color of a dish; the neutral hue for a color out of the palette. */
export function colorClass(color: string): string {
  const index = palette.indexOf(color.toLowerCase());
  return `dish-color-${index < 0 ? palette.length : index + 1}`;
}

/** The count above the list: the dishes, or the results of the search (Q30). */
export function countLabel(total: number, results: number, query: string): string {
  if (query.trim() === "") {
    return total > 1 ? `${total} plats` : `${total} plat`;
  }
  return `${results} ${results > 1 ? "résultats" : "résultat"} pour « ${query.trim()} »`;
}

/**
 * The detail of Plats in etendu, without dish open (plan grands-ecrans, D5; GE-Plats-*):
 * « Choisissez un plat ou créez-en un », and « Nouveau plat ».
 */
export function chooseDish(newDishHref: string) {
  return html`<main class="screen dish-choose" aria-labelledby="dish-choose-title">
    <div class="dish-choose-box">
      <span class="dish-choose-icon" aria-hidden="true">
        <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 5a2 2 0 0 1 2-2h13v16H6a2 2 0 0 0-2 2z"></path><path d="M4 21V5"></path><path d="M9 8h6"></path></svg>
      </span>
      <h1 class="dish-choose-title" id="dish-choose-title">Choisissez un plat ou créez-en un</h1>
      <p class="dish-choose-text">Le plat s’ouvre ici, à côté de la bibliothèque.</p>
      <a class="button-secondary" href=${newDishHref}>
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" aria-hidden="true"><path d="M12 5v14M5 12h14"></path></svg>Nouveau plat
      </a>
    </div>
  </main>`;
}

export class DishLibraryScreen extends LitElement {
  static override properties = {
    library: { attribute: false },
    newDishHref: { attribute: false },
    dishHref: { attribute: false },
    current: { attribute: false },
    beside: { attribute: false },
    query: { state: true },
    failed: { state: true },
    revision: { state: true },
  };

  declare library: Library;
  declare newDishHref: string;
  declare dishHref: (dish: Dish) => string;
  /** The dish open beside the library, whose card is marked (plan grands-ecrans, D5). */
  current: string | undefined = undefined;
  /** A pane beside a dish, in etendu: the main of the page is the dish (D5). */
  beside = false;
  query = "";
  /** The last check failed for lack of network. */
  failed = false;
  /** Increases at each change of the library, to render again. */
  revision = 0;
  readonly #onChange = () => this.revision++;

  protected override createRenderRoot() {
    return this;
  }

  override connectedCallback() {
    super.connectedCallback();
    this.library.addEventListener("change", this.#onChange);
    void this.#check();
  }

  override disconnectedCallback() {
    super.disconnectedCallback();
    this.library.removeEventListener("change", this.#onChange);
  }

  async #check() {
    this.failed = false;
    const outcome = await this.library.check();
    // A 401 has already brought back to the login.
    this.failed = outcome.kind === "network-error";
  }

  override render() {
    const dishes = this.library.dishes.items;
    // Beside a dish, the library is a region of the page, titled at the level under the dish:
    // only the role and the title change, the search keeps its field (D8).
    return html`<section class="screen library" role=${this.beside ? nothing : "main"} aria-labelledby="library-title">
      <div class="screen-header">
        ${this.beside ? html`<h2 class="screen-title" id="library-title">Plats</h2>` : html`<h1 class="screen-title" id="library-title">Plats</h1>`}
        <a class="button-new" href=${this.newDishHref}>
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" aria-hidden="true"><path d="M12 5v14M5 12h14"></path></svg>Nouveau
        </a>
      </div>
      <label class="search">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" aria-hidden="true"><circle cx="11" cy="11" r="7"></circle><path d="M20 20l-4-4"></path></svg>
        <span class="visually-hidden">Rechercher un plat ou un ingrédient</span>
        <input class="search-input" type="search" placeholder="Nom du plat ou ingrédient" autocomplete="off" .value=${this.query} @input=${this.#search}>
      </label>
      ${this.failed ? html`${alertMessage("library-error", networkErrorMessage)}<button class="button-secondary" type="button" @click=${this.#check}>Réessayer</button>` : nothing}
      ${dishes === undefined ? nothing : this.#list(dishes)}
    </section>`;
  }

  #list(dishes: Dish[]) {
    if (dishes.length === 0) {
      return html`<p class="dish-list-empty">Aucun plat pour l'instant</p>`;
    }
    const results = searchDishes(dishes, this.query);
    return html`<p class="dish-count" role="status">${countLabel(dishes.length, results.length, this.query)}</p>
      ${results.length === 0
        ? html`<p class="dish-list-empty">Aucun plat ne correspond</p>`
        : html`<ul class="dish-list">${results.map((r) => this.#card(r))}</ul>`}`;
  }

  #card({ dish, found }: Result) {
    return html`<li>
      <a class="dish-card ${colorClass(dish.color)}" href=${this.dishHref(dish)} aria-current=${dish.id === this.current ? "page" : nothing}>
        <span class="dish-initial" aria-hidden="true">${initialOf(dish.name)}</span>
        <span class="dish-card-body">
          <span class="dish-card-head">
            <span class="dish-name">${dish.name}</span>
            <span class="dish-servings">pour ${dish.servings}</span>
          </span>
          ${dish.ingredients.length === 0
            ? nothing
            : html`<span class="dish-ingredients">${dish.ingredients.map(
                (l) => html`<span class="chip ${found.has(l.name) ? "chip--found" : ""}">${l.name}</span>`,
              )}</span>`}
        </span>
      </a>
    </li>`;
  }

  #search(e: Event) {
    this.query = (e.target as HTMLInputElement).value;
  }
}

customElements.define("mt-dish-library", DishLibraryScreen);

declare global {
  interface HTMLElementTagNameMap {
    "mt-dish-library": DishLibraryScreen;
  }
}
