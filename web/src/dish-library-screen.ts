// Screen Plats: the dish library and its search (P3), after the mock-up Doux-Bibliotheque.
// The library is checked at each visit (plan bibliotheque, D25); the search runs in the
// browser, at each keystroke, on the copy in memory (ADR 0027).
import { LitElement, html, nothing } from "lit";
import { type Dish, type Result, searchDishes } from "./dishes.ts";
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

export class DishLibraryScreen extends LitElement {
  static override properties = {
    library: { attribute: false },
    newDishHref: { attribute: false },
    dishHref: { attribute: false },
    query: { state: true },
    failed: { state: true },
    revision: { state: true },
  };

  declare library: Library;
  declare newDishHref: string;
  declare dishHref: (dish: Dish) => string;
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
    return html`<main class="screen">
      <div class="screen-header">
        <h1 class="screen-title">Plats</h1>
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
    </main>`;
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
      <a class="dish-card ${colorClass(dish.color)}" href=${this.dishHref(dish)}>
        <span class="dish-initial" aria-hidden="true">${dish.name.charAt(0)}</span>
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
