// Screen Planning, after the mock-up Doux-Main (R1, R5): the 7 days from today, or from the
// day of the path (?du=, D14), two meals a day; the arrows change the range without request
// (D1). The days wait for the copies of the meal plan and of the library (D15), never to show
// a week empty by mistake. Back in the foreground on another day, the range of today follows
// it; another range stays (D17). A meal tile opens the screen of its meal (D14).
import { LitElement, html, nothing } from "lit";
import type { Api, TribeInfo } from "./api.ts";
import { colorClass } from "./dish-library-screen.ts";
import { networkErrorMessage } from "./login.ts";
import {
  type Day,
  type DayTiles,
  type MealTile,
  type Mealtime,
  dayRange,
  mealTiles,
  nextStart,
  previousStart,
  rangeTitle,
  servingsLabel,
  shortDate,
  today,
  weekdayName,
} from "./meal-plan.ts";
import type { Planning } from "./planning.ts";
import { alertMessage } from "./templates.ts";

const mealtimeLabels = { lunch: "Midi", dinner: "Soir" } as const;

export class PlanningScreen extends LitElement {
  static override properties = {
    planning: { attribute: false },
    api: { attribute: false },
    start: { attribute: false },
    tribeHref: { attribute: false },
    mealHref: { attribute: false },
    go: { attribute: false },
    today: { state: true },
    tribe: { state: true },
    failed: { state: true },
    revision: { state: true },
  };

  declare planning: Planning;
  declare api: Api;
  /** The first day of the range of the path; undefined for the range of today. */
  declare start: Day | undefined;
  declare tribeHref: string;
  declare mealHref: (date: Day, mealtime: Mealtime) => string;
  /** Shows another range: undefined for that of today. */
  declare go: (start: Day | undefined) => void;
  /** The day of the device (D4), read again at each return to the foreground (D17). */
  today: Day = today();
  /** The tribe, for its chip; undefined until read, or if it cannot be. */
  tribe: TribeInfo | undefined = undefined;
  /** The last check, or the reading of the tribe, failed for lack of network. */
  failed = false;
  /** Increases at each change of the copies, to render again. */
  revision = 0;
  readonly #onChange = () => this.revision++;
  readonly #onVisibility = () => {
    if (document.visibilityState === "visible") {
      this.today = today();
      // The chip is the only way to the screen Tribu, so to the sign-out (review of PR #90).
      if (this.tribe === undefined) {
        void this.#readTribe();
      }
    }
  };

  protected override createRenderRoot() {
    return this;
  }

  override connectedCallback() {
    super.connectedCallback();
    this.today = today();
    this.planning.addEventListener("change", this.#onChange);
    this.planning.library.addEventListener("change", this.#onChange);
    document.addEventListener("visibilitychange", this.#onVisibility);
    void this.#retry();
  }

  override disconnectedCallback() {
    super.disconnectedCallback();
    this.planning.removeEventListener("change", this.#onChange);
    this.planning.library.removeEventListener("change", this.#onChange);
    document.removeEventListener("visibilitychange", this.#onVisibility);
  }

  /** Checks the copies and reads the tribe, if not read yet (review of PR #90, point 1). */
  async #retry() {
    this.failed = false;
    const [checked, tribe] = await Promise.all([this.planning.check(), this.tribe === undefined ? this.#readTribe() : true]);
    // A 401 has already brought back to the login.
    this.failed = checked.kind === "network-error" || !tribe;
  }

  /** Reads the tribe, for its chip; false for lack of network. */
  async #readTribe(): Promise<boolean> {
    const outcome = await this.api.getTribe();
    if (outcome.kind === "ok") {
      this.tribe = outcome.value;
    }
    return outcome.kind !== "network-error";
  }

  override render() {
    const start = this.start ?? this.today;
    const days = dayRange(start);
    const previous = previousStart(start);
    const next = nextStart(start);
    const served = this.planning.served.items;
    const dishes = this.planning.library.dishes.items;
    return html`<main class="screen planning">
      <div class="screen-header">
        <h1 class="screen-title">Menus</h1>
        ${this.tribe === undefined ? nothing : tribeChip(this.tribe, this.tribeHref)}
      </div>
      <div class="range">
        <button class="range-arrow" type="button" aria-label="Semaine précédente" ?disabled=${previous === undefined} @click=${() => this.#goTo(previous)}>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M15 18l-6-6 6-6"></path></svg>
        </button>
        <div class="range-text">
          <h2 class="range-title" aria-live="polite">${rangeTitle(days, this.today)}</h2>
          ${start === this.today
            ? html`<span class="range-today">À partir d'aujourd'hui</span>`
            : html`<button class="range-back" type="button" @click=${() => this.go(undefined)}>Revenir à aujourd'hui</button>`}
        </div>
        <button class="range-arrow" type="button" aria-label="Semaine suivante" ?disabled=${next === undefined} @click=${() => this.#goTo(next)}>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M9 18l6-6-6-6"></path></svg>
        </button>
      </div>
      ${this.failed ? html`${alertMessage("planning-error", networkErrorMessage)}<button class="button-secondary" type="button" @click=${this.#retry}>Réessayer</button>` : nothing}
      ${served === undefined || dishes === undefined ? nothing : html`<ol class="days">${mealTiles(days, served, dishes, this.today).map((d) => this.#day(d))}</ol>`}
    </main>`;
  }

  /** Shows the range from this day; that of today is the root of the tribe (D14). */
  #goTo(start: Day | undefined) {
    if (start !== undefined) {
      this.go(start === this.today ? undefined : start);
    }
  }

  #day(day: DayTiles) {
    return html`<li class="day">
      <div class="day-head">
        <h3 class="day-name">${weekdayName(day.date)} <span class="day-date">${shortDate(day.date)}</span></h3>
        ${day.today ? html`<span class="day-today">Aujourd'hui</span>` : nothing}
      </div>
      <div class="day-meals">${day.meals.map((m) => this.#meal(m))}</div>
    </li>`;
  }

  #meal(meal: MealTile) {
    const empty = meal.dishes.length === 0;
    const classes = ["meal-tile", `meal-tile--${meal.mealtime}`, empty ? "meal-tile--empty" : "", meal.inactive ? "meal-tile--inactive" : ""].filter((c) => c !== "").join(" ");
    const name = `${weekdayName(meal.date)} ${shortDate(meal.date)}, ${mealtimeLabels[meal.mealtime].toLowerCase()}`;
    const label = html`<span class="meal-tile-label" aria-hidden="true">${mealtimeLabels[meal.mealtime].toUpperCase()}</span>`;
    if (meal.inactive) {
      return html`<div class=${classes} role="group" aria-label=${name}>${label}</div>`;
    }
    // The whole tile opens the meal, full or empty (docs/design/README.md); read as its day,
    // its mealtime and its dishes.
    const dishes = empty ? "aucun plat" : meal.dishes.map(({ served, dish }) => `${dish.name}, ${servingsLabel(served.servings)}`).join(" ; ");
    return html`<a class=${classes} href=${this.mealHref(meal.date, meal.mealtime)} aria-label=${`${name} : ${dishes}`}>
      ${label}
      ${empty
        ? html`<span class="meal-tile-add"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" aria-hidden="true"><path d="M12 5v14M5 12h14"></path></svg>Ajouter</span>`
        : html`<ul class="meal-tile-dishes">
            ${meal.dishes.map(
              ({ served, dish }) => html`<li class="dish-pill ${colorClass(dish.color)}">
                <span class="dish-pill-name">${dish.name}</span><span class="dish-pill-servings">${served.servings}</span>
              </li>`,
            )}
          </ul>`}
    </a>`;
  }
}

/** The chip of the tribe (Doux-Main): its name and its number of active members. */
function tribeChip(tribe: TribeInfo, href: string) {
  const members = tribe.activeMembers > 1 ? `${tribe.activeMembers} membres` : `${tribe.activeMembers} membre`;
  return html`<a class="tribe-chip" href=${href} aria-label=${`Tribu ${tribe.name}, ${members}`}>
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="9" cy="8" r="3.2"></circle><path d="M3 20a6 6 0 0 1 12 0"></path><circle cx="17" cy="9" r="2.6"></circle><path d="M16 14.2a5 5 0 0 1 5.5 5.8"></path></svg>
    <span class="tribe-chip-name">${tribe.name}</span> · ${tribe.activeMembers}
  </a>`;
}

customElements.define("mt-planning-screen", PlanningScreen);

declare global {
  interface HTMLElementTagNameMap {
    "mt-planning-screen": PlanningScreen;
  }
}
