// Screen of a meal, after the mock-up Doux-Repas (R2 to R4): its dishes served, their
// servings set by the stepper, their removal, and the dish suggestions of the library
// (Q21, D12). The meal waits for the copies of the meal plan and of the library, checked at
// each visit (D15). The servings are saved at each step; while a save runs, the next steps
// are shown and the last one only is sent after it, as the size of the tribe. Crossed writes
// (D3) and writes without network (D13) are told above the meal, which stays shown. The rows
// of the dishes served are keyed by dish served, and the removal buttons wait while a removal
// runs and a moment after, the next dish taking the place of the one removed under the finger.
// In etendu, the meal is a panel over the planning by week (plan grands-ecrans, D6): the cross
// « Fermer le repas » of its head takes the place of the link « Planning », after data-format.
import { LitElement, html, nothing } from "lit";
import { live } from "lit/directives/live.js";
import { repeat } from "lit/directives/repeat.js";
import { colorClass } from "./dish-library-screen.ts";
import { type Dish, initialOf } from "./dishes.ts";
import { networkErrorMessage } from "./login.ts";
import { type Day, type Mealtime, type TileDish, dishSuggestions, longDate, mealDishes, servingsLabel, today } from "./meal-plan.ts";
import type { Planning } from "./planning.ts";
import { alertMessage, networkNeededMessage, warningBanner } from "./templates.ts";

const mealtimeLabels = { lunch: "Midi", dinner: "Soir" } as const;

/** How long the removal buttons wait after a removal: a slow double touch goes to the first only. */
const removalPauseMs = 500;

/** What the last write on the meal tells (docs/design/README.md, « Messages du planning »). */
type Message = { kind: "already-served"; name: string } | { kind: "removed" } | { kind: "network" };

export class MealScreen extends LitElement {
  static override properties = {
    planning: { attribute: false },
    date: { attribute: false },
    mealtime: { attribute: false },
    planningHref: { attribute: false },
    newDishHref: { attribute: false },
    query: { state: true },
    failed: { state: true },
    message: { state: true },
    revision: { state: true },
    removalPaused: { state: true },
  };

  declare planning: Planning;
  declare date: Day;
  declare mealtime: Mealtime;
  /** The range the link « Planning », or the cross of the panel, goes back to. */
  declare planningHref: string;
  declare newDishHref: string;
  query = "";
  /** The last check failed for lack of network. */
  failed = false;
  message: Message | undefined = undefined;
  /** Increases at each change of the copies or of a write running, to render again. */
  revision = 0;
  /** A removal runs, or has just ended: the removal buttons wait. */
  removalPaused = false;
  #removalPause: ReturnType<typeof setTimeout> | undefined;
  /** The servings shown by a stepper and not saved yet, by dish served. */
  readonly #shown = new Map<string, number>();
  /** The dishes served whose servings are being saved. */
  readonly #saving = new Set<string>();
  /** The dishes being served, by dish, and the dishes served being removed: one write each. */
  readonly #serving = new Set<string>();
  readonly #removing = new Set<string>();
  readonly #onChange = () => this.revision++;

  protected override createRenderRoot() {
    return this;
  }

  override connectedCallback() {
    super.connectedCallback();
    this.planning.addEventListener("change", this.#onChange);
    this.planning.library.addEventListener("change", this.#onChange);
    void this.#check();
  }

  override disconnectedCallback() {
    super.disconnectedCallback();
    this.planning.removeEventListener("change", this.#onChange);
    this.planning.library.removeEventListener("change", this.#onChange);
    clearTimeout(this.#removalPause);
  }

  async #check() {
    this.failed = false;
    const outcome = await this.planning.check();
    // A 401 has already brought back to the login.
    this.failed = outcome.kind === "network-error";
  }

  override render() {
    const served = this.planning.served.items;
    const dishes = this.planning.library.dishes.items;
    const meal = served === undefined || dishes === undefined ? undefined : mealDishes(this.date, this.mealtime, served, dishes);
    // The total follows the steppers, before their save.
    const total = meal === undefined ? 0 : meal.reduce((sum, d) => sum + this.#servings(d), 0);
    return html`<main class="screen meal">
      <a class="back-link" href=${this.planningHref}>
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M15 18l-6-6 6-6"></path></svg>Planning
      </a>
      <div class="meal-head meal-head--${this.mealtime}">
        <h1 class="meal-head-heading">
          <span class="meal-head-date">${longDate(this.date, today())}</span>
          <span class="meal-head-time">${mealtimeLabels[this.mealtime]}</span>
        </h1>
        ${total === 0 ? nothing : html`<span class="meal-head-total">${servingsLabel(total)}</span>`}
        <a class="meal-close" href=${this.planningHref} aria-label="Fermer le repas">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" aria-hidden="true"><path d="M6 6l12 12M18 6L6 18"></path></svg>
        </a>
      </div>
      ${this.#message()}
      ${this.failed ? html`${alertMessage("meal-check-error", networkErrorMessage)}<button class="button-secondary" type="button" @click=${this.#check}>Réessayer</button>` : nothing}
      ${meal === undefined || dishes === undefined ? nothing : html`${this.#served(meal)}${this.#add(dishes, meal)}`}
    </main>`;
  }

  #message() {
    switch (this.message?.kind) {
      case "already-served":
        return warningBanner(`« ${this.message.name} » est déjà servi dans ce repas.`);
      case "removed":
        return warningBanner("Ce plat a été retiré du repas entre-temps.");
      case "network":
        return alertMessage("meal-error", networkNeededMessage);
      default:
        return nothing;
    }
  }

  /** The servings shown for a dish served: those of the stepper, or those saved. */
  #servings(d: TileDish): number {
    return this.#shown.get(d.served.id) ?? d.served.servings;
  }

  #served(meal: TileDish[]) {
    return html`<section class="meal-section">
      <h2 class="section-title">Plats servis</h2>
      ${meal.length === 0
        ? html`<p class="meal-empty">Aucun plat pour ce repas</p>`
        : html`<ul class="served-dishes">
            ${repeat(
              meal,
              (d) => d.served.id,
              (d) => html`<li class="served-dish ${colorClass(d.dish.color)}">
                <span class="dish-initial dish-initial--small" aria-hidden="true">${initialOf(d.dish.name)}</span>
                <span class="served-dish-text">
                  <span class="served-dish-name">${d.dish.name}</span>
                  <span class="served-dish-reference">Défini pour ${servingsLabel(d.dish.servings)}</span>
                </span>
                <mt-stepper
                  label=${`Parts de « ${d.dish.name} »`}
                  less-label="Une part de moins"
                  more-label="Une part de plus"
                  min="1"
                  max="50"
                  .value=${live(this.#servings(d))}
                  @change=${(e: Event) => this.#servingsChanged(d.served.id, (e.target as HTMLElement & { value: number }).value)}
                ></mt-stepper>
                <button class="served-dish-remove" type="button" aria-label=${`Retirer « ${d.dish.name} » du repas`} ?disabled=${this.removalPaused} @click=${() => this.#remove(d.served.id)}>
                  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" aria-hidden="true"><path d="M6 6l12 12M18 6L6 18"></path></svg>
                </button>
              </li>`,
            )}
          </ul>`}
    </section>`;
  }

  #add(dishes: Dish[], meal: TileDish[]) {
    return html`<section class="meal-section">
      <h2 class="section-title">Ajouter un plat</h2>
      ${dishes.length === 0
        ? html`<p class="meal-empty">Aucun plat dans la bibliothèque</p>
            <a class="button-secondary" href=${this.newDishHref}>Créer un plat</a>`
        : html`<label class="search">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" aria-hidden="true"><circle cx="11" cy="11" r="7"></circle><path d="M20 20l-4-4"></path></svg>
              <span class="visually-hidden">Rechercher un plat</span>
              <input class="search-input" type="search" placeholder="Nom du plat ou ingrédient" autocomplete="off" .value=${this.query} @input=${this.#search}>
            </label>
            ${meal.length === 0 ? nothing : html`<p class="meal-hint">Les plats déjà servis dans ce repas ne sont pas proposés : réglez leurs parts ci-dessus.</p>`}
            ${this.#suggestions(dishSuggestions(dishes, meal, this.query), meal.length === dishes.length)}`}
    </section>`;
  }

  #suggestions(suggestions: Dish[], allServed: boolean) {
    if (suggestions.length === 0) {
      const text = allServed ? "Tous les plats de la bibliothèque sont déjà dans ce repas." : "Aucun plat ne correspond";
      return html`<p class="meal-empty" role="status">${text}</p>`;
    }
    return html`<ul class="dish-suggestions" aria-label="Plats proposés">
      ${suggestions.map(
        (d) => html`<li>
          <button class="dish-suggestion ${colorClass(d.color)}" type="button" @click=${() => this.#serve(d)}>
            <span class="dish-initial dish-initial--small" aria-hidden="true">${initialOf(d.name)}</span>
            <span class="dish-suggestion-name"><span class="visually-hidden">Ajouter </span>${d.name}</span>
            <span class="dish-suggestion-add" aria-hidden="true">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round"><path d="M12 5v14M5 12h14"></path></svg>
            </span>
          </button>
        </li>`,
      )}
    </ul>`;
  }

  #search(e: Event) {
    this.query = (e.target as HTMLInputElement).value;
  }

  /** Serves a dish at once, for the size of the tribe (D9, D12). */
  async #serve(dish: Dish) {
    if (this.#serving.has(dish.id)) {
      return;
    }
    this.#serving.add(dish.id);
    this.message = undefined;
    try {
      const result = await this.planning.serve(this.date, this.mealtime, dish.id);
      // A 401 has already brought back to the login.
      if (result.kind === "already-served") {
        this.message = { kind: "already-served", name: dish.name };
      } else if (result.kind === "network-error") {
        this.message = { kind: "network" };
      }
    } finally {
      this.#serving.delete(dish.id);
    }
  }

  /**
   * Saves the servings shown. The steps made while a save runs are sent after it, the last
   * one only; a save that fails brings back the servings saved.
   */
  async #servingsChanged(id: string, servings: number) {
    this.#shown.set(id, servings);
    this.message = undefined;
    this.revision++;
    if (this.#saving.has(id)) {
      return;
    }
    this.#saving.add(id);
    try {
      for (;;) {
        const wanted = this.#shown.get(id);
        const saved = this.planning.served.items?.find((s) => s.id === id);
        if (wanted === undefined || saved === undefined || wanted === saved.servings) {
          return;
        }
        const result = await this.planning.setServings(id, wanted);
        if (result.kind === "removed") {
          this.message = { kind: "removed" };
          return;
        }
        if (result.kind !== "saved") {
          // A 401 has already brought back to the login.
          this.message = result.kind === "network-error" ? { kind: "network" } : undefined;
          return;
        }
      }
    } finally {
      this.#saving.delete(id);
      this.#shown.delete(id);
      this.revision++;
    }
  }

  /** Removes a dish served, without confirmation (R4); one already removed just goes (D19). */
  async #remove(id: string) {
    if (this.#removing.has(id)) {
      return;
    }
    this.#removing.add(id);
    clearTimeout(this.#removalPause);
    this.removalPaused = true;
    this.message = undefined;
    try {
      const outcome = await this.planning.remove(id);
      // A 401 has already brought back to the login.
      if (outcome.kind === "network-error") {
        this.message = { kind: "network" };
      }
    } finally {
      this.#removing.delete(id);
      if (this.#removing.size === 0) {
        this.#removalPause = setTimeout(() => (this.removalPaused = false), removalPauseMs);
      }
    }
  }
}

customElements.define("mt-meal-screen", MealScreen);

declare global {
  interface HTMLElementTagNameMap {
    "mt-meal-screen": MealScreen;
  }
}
