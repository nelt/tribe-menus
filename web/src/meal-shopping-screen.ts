// Screen of the meal shopping, after the mock-up Doux-CoursesRepas (plan courses, C4, D15,
// D24): the fields « Du » and « Au », the calendar of the month of the field being set, with
// the points of the days with meals, the summary of the period, the target list and its
// sheet, the served dishes it has already received (Q22), and « Ajouter à « … » », inactive
// on a period refused. It waits for the copies of the lists and of the meal plan (with the
// library). The computation is the server's (D6): once added, the application goes back to
// the target list, which tells how many served dishes were added and ignored.
import { LitElement, type PropertyValues, html, nothing } from "lit";
import { repeat } from "lit/directives/repeat.js";
import { networkErrorMessage } from "./login.ts";
import { type Day, type ServedDish, today } from "./meal-plan.ts";
import type { Planning } from "./planning.ts";
import {
  type CalendarDay,
  type Month,
  type Period,
  type ShoppingList,
  addMonths,
  alreadyReceived,
  defaultPeriod,
  ignoredMessage,
  mealDays,
  mealShoppingErrorMessage,
  mealShoppingMessage,
  monthGrid,
  monthOf,
  monthTitle,
  periodError,
  periodSummary,
  servedInPeriod,
  shortDay,
  sortLists,
} from "./shopping.ts";
import type { ShoppingLists } from "./shopping-lists.ts";
import { alertMessage, networkNeededMessage } from "./templates.ts";

/** The event sent once the meal shopping is added: the target list, and the banner it shows. */
export class MealShoppingAddedEvent extends Event {
  readonly list: ShoppingList;
  readonly message: string;

  constructor(list: ShoppingList, message: string) {
    super("meal-shopping-added", { bubbles: true, composed: true });
    this.list = list;
    this.message = message;
  }
}

const weekdayInitials = ["L", "M", "M", "J", "V", "S", "D"];
const weekdayNames = ["lundi", "mardi", "mercredi", "jeudi", "vendredi", "samedi", "dimanche"];

const chevron = (d: string) =>
  html`<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d=${d}></path></svg>`;

export class MealShoppingScreen extends LitElement {
  static override properties = {
    lists: { attribute: false },
    planning: { attribute: false },
    listId: { attribute: false },
    backHref: { attribute: false },
    failed: { state: true },
    revision: { state: true },
    period: { state: true },
    field: { state: true },
    month: { state: true },
    target: { state: true },
    choosing: { state: true },
    sending: { state: true },
    error: { state: true },
  };

  declare lists: ShoppingLists;
  declare planning: Planning;
  /** The list we come from, preselected; the main list without one, or if it is unknown (D15). */
  declare listId: string | undefined;
  declare backHref: string;
  /** The last check failed for lack of network, the copies never loaded. */
  failed = false;
  /** Increases at each change of the copies, to render again. */
  revision = 0;
  /** The period chosen; from tomorrow to 7 days after today at the opening (D24). */
  period: Period = defaultPeriod(today());
  /** The field the calendar sets: « Du » at the opening, then « Au » (docs/design/README.md). */
  field: "from" | "to" = "from";
  /** The month shown: that of the field being set. */
  month: Month = monthOf(this.period.from);
  /** The target list chosen in the sheet; until then, the list we come from. */
  target: string | undefined = undefined;
  /** The sheet « Ajouter à quelle liste ? » is open. */
  choosing = false;
  /** The meal shopping is being added: the button waits. */
  sending = false;
  /** The message of a refusal, or « Réseau nécessaire » (D2). */
  error: string | undefined = undefined;
  readonly #onChange = () => this.revision++;

  protected override createRenderRoot() {
    return this;
  }

  override connectedCallback() {
    super.connectedCallback();
    this.lists.addEventListener("change", this.#onChange);
    this.planning.addEventListener("change", this.#onChange);
    void this.#check();
  }

  override disconnectedCallback() {
    super.disconnectedCallback();
    this.lists.removeEventListener("change", this.#onChange);
    this.planning.removeEventListener("change", this.#onChange);
  }

  protected override updated(changed: PropertyValues<this>) {
    const dialog = this.querySelector<HTMLDialogElement>("dialog.sheet");
    if (changed.has("choosing") && dialog !== null && this.choosing && !dialog.open) {
      dialog.showModal();
    }
  }

  /** Checks the lists, the meal plan and the library, at the same time. */
  async #check() {
    this.failed = false;
    const [lists, planning] = await Promise.all([this.lists.check(), this.planning.check()]);
    // A 401 has already brought back to the login.
    this.failed = (lists.kind === "network-error" || planning.kind === "network-error") && !this.#loaded();
  }

  #loaded(): boolean {
    return this.lists.loaded && this.planning.loaded;
  }

  /** The target list: the one chosen, else the one we come from, else the main list. */
  #target(): ShoppingList | undefined {
    const chosen = this.target ?? this.listId;
    return (chosen === undefined ? undefined : this.lists.find(chosen)) ?? this.lists.find(undefined);
  }

  override render() {
    return html`<main class="screen meal-shopping">
        <a class="back-link" href=${this.backHref}>${chevron("M15 18l-6-6 6-6")}Retour à la liste</a>
        <h1 class="screen-title">Courses des repas</h1>
        ${this.#body()}
      </main>
      ${this.#sheet()}`;
  }

  #body() {
    const target = this.#target();
    if (!this.#loaded() || target === undefined) {
      return this.failed
        ? html`${alertMessage("meal-shopping-loading-error", networkErrorMessage)}<button class="button-secondary" type="button" @click=${this.#check}>Réessayer</button>`
        : html`<div class="shopping-loading" role="status" aria-label="Chargement des repas">
            <div class="skeleton skeleton--button"></div>
            <div class="skeleton skeleton--calendar"></div>
            <div class="skeleton skeleton--button"></div>
          </div>`;
    }
    const served = this.planning.served.items ?? [];
    const refused = periodError(this.period);
    const inPeriod = refused === undefined ? servedInPeriod(served, this.period) : [];
    const summary = periodSummary(this.period, inPeriod);
    const ignored = refused === undefined ? ignoredMessage(alreadyReceived(target, inPeriod)) : undefined;
    const cta = `Ajouter à « ${target.name} »`;
    return html`${this.error === undefined
        ? nothing
        : html`<div class="banner-error" role="alert">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="12" r="9"></circle><path d="M12 7v6M12 16.5v.5"></path></svg>
            <span>${this.error}</span>
          </div>`}
      <div class="period-fields">
        ${this.#field("from", "Du", false)} ${this.#field("to", "Au", refused !== undefined)}
      </div>
      ${this.#calendar(served)}
      ${refused === undefined
        ? html`<div class="period-summary"><span>${summary.days}</span><span class="period-summary-dishes">${summary.dishes}</span></div>`
        : alertMessage("period-error", mealShoppingErrorMessage(refused) ?? "")}
      <button class="target-field" type="button" aria-haspopup="dialog" @click=${() => (this.choosing = true)}>
        <span class="target-field-text"><span class="target-field-label">Ajouter à la liste</span> <span class="target-field-name">${target.name}</span></span>
        ${target.main ? html`<span class="pill pill--main">principale</span>` : nothing} ${chevron("M6 9l6 6 6-6")}
      </button>
      ${ignored === undefined ? nothing : html`<p class="hint hint--left" role="status">${ignored}</p>`}
      ${refused === undefined && inPeriod.length === 0
        ? html`<p class="hint hint--left" role="status">Aucun repas planifié sur cette période : l’ajout ne changera rien à la liste.</p>`
        : nothing}
      <div class="meal-shopping-send">
        ${this.sending
          ? html`<button class="button-primary" type="button" disabled aria-busy="true">Ajout en cours…</button>`
          : html`<button class="button-primary" type="button" ?disabled=${refused !== undefined} @click=${() => this.#send(target)}>${cta}</button>`}
      </div>`;
  }

  #field(field: "from" | "to", label: string, invalid: boolean) {
    const active = this.field === field;
    return html`<button
      class="period-field ${active ? "period-field--active" : ""} ${invalid ? "period-field--invalid" : ""}"
      type="button"
      aria-pressed=${active ? "true" : "false"}
      aria-invalid=${invalid ? "true" : "false"}
      aria-describedby=${invalid ? "period-error" : nothing}
      @click=${() => this.#editField(field)}
    >
      <span class="period-field-label">${label}</span> <span class="period-field-day">${shortDay(this.period[field])}</span>
    </button>`;
  }

  #calendar(served: ServedDish[]) {
    const previous = addMonths(this.month, -1);
    const next = addMonths(this.month, 1);
    const cells = monthGrid(this.month, this.period, today(), mealDays(served));
    return html`<div class="calendar">
      <div class="calendar-head">
        <button class="calendar-nav" type="button" aria-label="Mois précédent" ?disabled=${previous === undefined} @click=${() => previous !== undefined && (this.month = previous)}>${chevron("M15 18l-6-6 6-6")}</button>
        <span class="calendar-title" aria-live="polite">${monthTitle(this.month)}</span>
        <button class="calendar-nav" type="button" aria-label="Mois suivant" ?disabled=${next === undefined} @click=${() => next !== undefined && (this.month = next)}>${chevron("M9 18l6-6-6-6")}</button>
      </div>
      <div class="calendar-grid">
        ${weekdayInitials.map((l, i) => html`<abbr class="calendar-weekday" title=${weekdayNames[i] ?? ""}>${l}</abbr>`)}
        ${cells.map((d) => (d === undefined ? html`<span></span>` : this.#day(d)))}
      </div>
      <p class="calendar-legend"><span class="calendar-legend-dot" aria-hidden="true"></span>jour avec des repas planifiés</p>
    </div>`;
  }

  #day(d: CalendarDay) {
    const classes = ["calendar-day", d.bound ? "calendar-day--bound" : "", d.between ? "calendar-day--between" : "", d.today ? "calendar-day--today" : "", d.past ? "calendar-day--past" : ""].join(" ");
    return html`<button class=${classes} type="button" aria-label=${d.label} aria-pressed=${d.bound ? "true" : "false"} @click=${() => this.#pick(d.date)}>
      <span>${d.number}</span><span class="calendar-dot ${d.meals ? "calendar-dot--meals" : ""}" aria-hidden="true"></span>
    </button>`;
  }

  #sheet() {
    if (!this.choosing) {
      return nothing;
    }
    const current = this.#target();
    return html`<dialog class="sheet" aria-labelledby="target-title" @close=${() => (this.choosing = false)} @click=${this.#closeOnBackdrop}>
      <h2 class="sheet-title" id="target-title">Ajouter à quelle liste ?</h2>
      <div class="sheet-actions">
        ${repeat(
          sortLists(this.lists.lists.items ?? []),
          (l) => l.id,
          (l) => html`<button class="target-option ${l.id === current?.id ? "target-option--current" : ""}" type="button" aria-pressed=${l.id === current?.id ? "true" : "false"} @click=${() => this.#choose(l)}>
            <span class="target-option-radio" aria-hidden="true"></span>
            <span class="target-option-name">${l.name}</span>
            ${l.main ? html`<span class="pill pill--main">principale</span>` : nothing}
            <span class="target-option-info">${l.items.length === 0 ? "vide" : l.items.length > 1 ? `${l.items.length} articles` : "1 article"}</span>
          </button>`,
        )}
      </div>
      <button class="button-text button-text--wide" type="button" @click=${this.#closeSheet}>Annuler</button>
    </dialog>`;
  }

  /** A touch on the veil, outside the sheet, closes it. */
  #closeOnBackdrop(e: MouseEvent) {
    if (e.target instanceof HTMLDialogElement) {
      this.#closeSheet();
    }
  }

  #closeSheet() {
    this.querySelector<HTMLDialogElement>("dialog.sheet")?.close();
    this.choosing = false;
  }

  #choose(list: ShoppingList) {
    this.target = list.id;
    this.error = undefined;
    this.#closeSheet();
  }

  /** A field touched: the calendar sets it, and shows the month of its day. */
  #editField(field: "from" | "to") {
    this.field = field;
    this.month = monthOf(this.period[field]);
  }

  /** A day touched sets the field being set; after « Du », « Au » (docs/design/README.md). */
  #pick(date: Day) {
    this.error = undefined;
    if (this.field === "from") {
      this.period = { ...this.period, from: date };
      this.field = "to";
    } else {
      this.period = { ...this.period, to: date };
    }
  }

  async #send(target: ShoppingList) {
    if (this.sending || periodError(this.period) !== undefined) {
      return;
    }
    this.error = undefined;
    this.sending = true;
    try {
      const result = await this.lists.addMealShopping(target.id, this.period);
      if (result.kind === "done") {
        this.dispatchEvent(new MealShoppingAddedEvent(this.lists.find(target.id) ?? target, mealShoppingMessage(result.added, result.ignored)));
        return;
      }
      if (result.kind === "deleted") {
        this.dispatchEvent(new Event("list-deleted", { bubbles: true, composed: true }));
        return;
      }
      // A 401 has already brought back to the login.
      if (result.kind !== "unauthorized") {
        this.error = (result.kind === "refused" ? mealShoppingErrorMessage(result.error) : undefined) ?? networkNeededMessage;
        window.scrollTo({ top: 0 });
      }
    } finally {
      this.sending = false;
    }
  }
}

customElements.define("mt-meal-shopping-screen", MealShoppingScreen);

declare global {
  interface HTMLElementTagNameMap {
    "mt-meal-shopping-screen": MealShoppingScreen;
  }
}
