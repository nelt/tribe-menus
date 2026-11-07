// Screen of a shopping list, after the mock-ups Doux-Liste and Doux-ArticleMain (plan courses,
// C1, C3, C6, C11): its name, which opens the list picker, the progress, the items (checkbox,
// quantity, « ajouté » and the cross of an item added), « Ajouter un article à la main » with
// the combobox of the catalog, and « Courses faites » with its sheet. The list waits for the
// copy of the lists, checked at each visit. Every write needs the network (D2): without it,
// « Réseau nécessaire » above the list, which stays shown. A list deleted in between (D11)
// sends to the main list, which tells it.
import { LitElement, type PropertyValues, html, nothing } from "lit";
import { live } from "lit/directives/live.js";
import { repeat } from "lit/directives/repeat.js";
import { type Ingredient, hasControl, hasVisible, maxQuantityLength, normalizeName, parseQuantity, suggestIngredients, trimSpaces, unitLabel, units } from "./dishes.ts";
import { IngredientCombobox, type Option } from "./ingredient-combobox.ts";
import { networkErrorMessage } from "./login.ts";
import { type Item, type ShoppingList, checkoutOf, dishCount, itemQuantity, maxItems, progressOf, removedMessage, shoppingErrorMessage, sortItems } from "./shopping.ts";
import type { ShoppingLists, WriteResult } from "./shopping-lists.ts";
import { alertMessage, networkNeededMessage, warningBanner } from "./templates.ts";

/** How long the crosses of the items added wait after a removal: a slow double touch removes one item only. */
const removalPauseMs = 500;

/** What the screen tells above the list: a green banner, a warning, or « Réseau nécessaire ». */
export type Notice = { kind: "done"; text: string } | { kind: "deleted" } | { kind: "network" };

/** The item being added by hand: the ingredient typed or chosen, the quantity, the unit. */
interface Draft {
  ingredient: string;
  quantity: string;
  unit: string;
}

const emptyDraft: Draft = { ingredient: "", quantity: "", unit: "" };

const checkIcon = html`<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12.5l4.5 4.5L19 7.5"></path></svg>`;
const crossIcon = html`<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" aria-hidden="true"><path d="M6 6l12 12M18 6L6 18"></path></svg>`;
const plusIcon = html`<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" aria-hidden="true"><path d="M12 5v14M5 12h14"></path></svg>`;

/** The green banner of a write done, with its cross (Doux-Courses). */
export function doneBanner(text: string, close: () => void) {
  return html`<div class="banner-done" role="status">
    ${checkIcon}<span>${text}</span>
    <button class="banner-close" type="button" aria-label="Fermer le message" @click=${close}>${crossIcon}</button>
  </div>`;
}

/** The error banner of a write without network (D2). */
export function networkBanner() {
  return html`<div class="banner-error" role="alert">
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="12" r="9"></circle><path d="M12 7v6M12 16.5v.5"></path></svg>
    <span>${networkNeededMessage}</span>
  </div>`;
}

export class ShoppingListScreen extends LitElement {
  static override properties = {
    lists: { attribute: false },
    listId: { attribute: false },
    pickerHref: { attribute: false },
    notice: { attribute: false },
    failed: { state: true },
    revision: { state: true },
    adding: { state: true },
    draft: { state: true },
    formError: { state: true },
    sheet: { state: true },
    removalPaused: { state: true },
  };

  declare lists: ShoppingLists;
  /** The list shown; undefined for the main list (C1). */
  declare listId: string | undefined;
  declare pickerHref: string;
  /** What the screen tells above the list; given by the application after a list deleted. */
  declare notice: Notice | undefined;
  /** The last check failed for lack of network, the copy never loaded. */
  failed = false;
  /** Increases at each change of the copy, to render again. */
  revision = 0;
  /** The form of an item added by hand is open. */
  adding = false;
  draft: Draft = emptyDraft;
  /** The code of the refusal of the item added. */
  formError: string | undefined = undefined;
  /** The sheet « Courses faites ? », with the items it counted (D9). */
  sheet: { leaving: string[]; message: string } | undefined = undefined;
  /** A removal runs, or has just ended: the crosses wait. */
  removalPaused = false;
  #removalPause: ReturnType<typeof setTimeout> | undefined;
  /** A write runs that a second touch must not repeat: an item added, a checkout. */
  #busy = false;
  readonly #onChange = () => this.revision++;
  readonly #combobox = new IngredientCombobox(this, {
    id: "item-suggestions",
    inputClass: "item-combobox",
    label: "Article",
    placeholder: "ex. pain, lessive",
    options: (text) => this.#options(text),
    onType: (text) => this.#type(text),
    onChoose: (option) => this.#choose(option),
  });

  protected override createRenderRoot() {
    return this;
  }

  override connectedCallback() {
    super.connectedCallback();
    this.lists.addEventListener("change", this.#onChange);
    void this.#check();
    // The catalog, for the suggestions, if the tab Plats was not visited yet.
    if (this.lists.library.catalog.items === undefined) {
      void this.lists.library.catalog.check();
    }
  }

  override disconnectedCallback() {
    super.disconnectedCallback();
    this.lists.removeEventListener("change", this.#onChange);
    clearTimeout(this.#removalPause);
  }

  protected override updated(changed: PropertyValues<this>) {
    const dialog = this.querySelector<HTMLDialogElement>("dialog.sheet");
    if (changed.has("sheet") && dialog !== null && this.sheet !== undefined && !dialog.open) {
      dialog.showModal();
    }
  }

  async #check() {
    this.failed = false;
    const outcome = await this.lists.check();
    // A 401 has already brought back to the login.
    this.failed = outcome.kind === "network-error" && !this.lists.loaded;
    if (this.lists.loaded && this.#list() === undefined) {
      this.dispatchEvent(new Event("unknown-list", { bubbles: true, composed: true }));
    }
  }

  #list(): ShoppingList | undefined {
    return this.lists.find(this.listId);
  }

  override render() {
    const list = this.#list();
    if (list === undefined) {
      return html`<main class="screen">${this.failed ? this.#failed() : this.#loading()}</main>`;
    }
    return html`<main class="screen shopping">
      ${this.#header(list)} ${this.#notice()} ${this.#items(list)} ${this.#addItem(list)} ${this.#checkout(list)}
    </main>
    ${this.#sheet()}`;
  }

  #loading() {
    return html`<div class="shopping-loading" role="status" aria-label="Chargement de la liste">
      <div class="skeleton skeleton--title"></div>
      <div class="skeleton skeleton--button"></div>
      ${[1, 2, 3, 4, 5, 6].map(() => html`<div class="skeleton skeleton--item"><span class="skeleton-box"></span><span class="skeleton-line"></span></div>`)}
    </div>`;
  }

  #failed() {
    return html`<h1 class="screen-title shopping-title">Courses</h1>
      ${alertMessage("shopping-error", networkErrorMessage)}
      <button class="button-secondary" type="button" @click=${this.#check}>Réessayer</button>`;
  }

  #header(list: ShoppingList) {
    const label = list.main ? `${list.name}, liste principale. Changer de liste` : `${list.name}. Changer de liste`;
    const progress = progressOf(list);
    return html`<h1 class="screen-title shopping-title">
        <a class="shopping-title-link" href=${this.pickerHref} aria-label=${label}>
          <span class="shopping-title-name">${list.name}</span>
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M6 9l6 6 6-6"></path></svg>
          ${list.main ? html`<span class="pill pill--main">principale</span>` : nothing}
        </a>
      </h1>
      ${progress.total === 0
        ? nothing
        : html`<div class="progress">
            <div class="progress-text"><span>${progress.label}</span><span class="progress-remaining">${progress.remaining}</span></div>
            <progress class="progress-bar" max=${progress.total} .value=${progress.checked} aria-hidden="true"></progress>
          </div>`}`;
  }

  #notice() {
    switch (this.notice?.kind) {
      case "done":
        return doneBanner(this.notice.text, () => (this.notice = undefined));
      case "deleted":
        return warningBanner("Cette liste a été supprimée entre-temps.");
      case "network":
        return networkBanner();
      default:
        return nothing;
    }
  }

  #items(list: ShoppingList) {
    if (list.items.length === 0) {
      return html`<p class="shopping-empty">La liste est vide. Ajoutez les courses des repas ou un article à la main.</p>`;
    }
    return html`<ul class="items">${repeat(sortItems(list.items), (i) => i.id, (i) => this.#item(list, i))}</ul>`;
  }

  #item(list: ShoppingList, item: Item) {
    const quantity = itemQuantity(item);
    return html`<li class="item ${item.checked ? "item--checked" : ""}">
      <input class="item-check" type="checkbox" aria-label=${`Cocher ${item.ingredient}`} .checked=${live(item.checked)} @change=${(e: Event) => this.#setChecked(list, item, (e.target as HTMLInputElement).checked)}>
      <span class="item-name">${item.ingredient}</span>
      ${item.manual ? html`<span class="pill pill--dashed">ajouté</span>` : html`<span class="pill">${dishCount(item)}</span>`}
      ${quantity === "" ? nothing : html`<span class="item-quantity">${quantity}</span>`}
      ${item.manual
        ? html`<button class="item-remove" type="button" aria-label=${`Retirer ${item.ingredient}`} ?disabled=${this.removalPaused} @click=${() => this.#remove(list, item)}>${crossIcon}</button>`
        : nothing}
    </li>`;
  }

  #addItem(list: ShoppingList) {
    if (list.items.length >= maxItems) {
      return html`<div class="button-with-hint">
        <button class="button-add" type="button" disabled>Ajouter un article à la main</button>
        <span class="hint">300 articles au plus.</span>
      </div>`;
    }
    if (!this.adding) {
      return html`<button class="button-add" type="button" @click=${this.#startAdding}>${plusIcon}Ajouter un article à la main</button>`;
    }
    return this.#form(list);
  }

  #form(list: ShoppingList) {
    const { ingredient, quantity, unit } = this.draft;
    const hasQuantity = trimSpaces(quantity) !== "";
    const known = this.#known(ingredient);
    const defaultUnit = known?.defaultUnit;
    const message = this.formError === undefined ? undefined : shoppingErrorMessage(this.formError);
    const onIngredient = this.formError === "ingredient_name_required" || this.formError === "ingredient_name_invalid";
    const onQuantity = this.formError === "invalid_quantity" || this.formError === "quantity_too_precise";
    const onUnit = this.formError === "unit_required";
    let hint = "Quantité facultative.";
    if (!hasQuantity && hasVisible(ingredient) && known !== undefined) {
      hint = defaultUnit === undefined ? "Sans quantité, l’article s’affiche seul." : `Sans quantité, l’article s’affiche seul. Une quantité prend l’unité « ${unitLabel(defaultUnit, "")} ».`;
    }
    return html`<form class="item-form" novalidate @submit=${(e: Event) => this.#submit(e, list)}>
      <span class="item-form-label" aria-hidden="true">Article</span>
      <div class="item-form-ingredient">
        ${this.#combobox.input(ingredient, { invalid: onIngredient, ...(onIngredient ? { describedBy: "item-error" } : {}) })}
        ${this.#combobox.list(ingredient)}
      </div>
      <div class="item-form-quantity">
        <label class="item-form-label">Quantité
          <input
            class="item-form-input"
            type="text"
            inputmode="decimal"
            autocomplete="off"
            maxlength=${maxQuantityLength}
            placeholder="Qté"
            aria-invalid=${onQuantity ? "true" : "false"}
            aria-describedby=${onQuantity ? "item-error" : "item-hint"}
            .value=${live(quantity)}
            @input=${(e: Event) => this.#setQuantity((e.target as HTMLInputElement).value)}
          >
        </label>
        <label class="item-form-label">Unité
          <select class="item-form-input" ?disabled=${!hasQuantity} aria-invalid=${onUnit ? "true" : "false"} @change=${(e: Event) => this.#setDraft({ unit: (e.target as HTMLSelectElement).value })}>
            <option value="" ?selected=${unit === ""}>${hasQuantity ? "Choisir" : defaultUnit === undefined ? "—" : unitLabel(defaultUnit, "")}</option>
            ${units.map((u) => html`<option value=${u.code} aria-label=${u.name} ?selected=${unit === u.code}>${u.label}</option>`)}
          </select>
        </label>
      </div>
      <p class="hint hint--left" id="item-hint">${hint}</p>
      ${message === undefined ? nothing : alertMessage("item-error", message)}
      <div class="item-form-actions">
        <button class="button-primary button-primary--small" type="submit">Ajouter à la liste</button>
        <button class="button-text" type="button" @click=${this.#cancelAdding}>Annuler</button>
      </div>
    </form>`;
  }

  #checkout(list: ShoppingList) {
    const { leaving, hint } = checkoutOf(list);
    return html`<div class="checkout">
      <button class="button-primary" type="button" ?disabled=${leaving.length === 0} @click=${() => this.#askCheckout(list)}>
        ${leaving.length === 0 ? nothing : checkIcon}Courses faites
      </button>
      <p class="hint">${hint}</p>
    </div>`;
  }

  #sheet() {
    if (this.sheet === undefined) {
      return nothing;
    }
    return html`<dialog class="sheet" aria-labelledby="checkout-title" @close=${() => (this.sheet = undefined)} @click=${this.#closeOnBackdrop}>
      <h2 class="sheet-title" id="checkout-title">Courses faites ?</h2>
      <p class="sheet-text">${this.sheet.message}</p>
      <button class="button-primary" type="button" @click=${this.#confirmCheckout}>Retirer les articles cochés</button>
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
    this.sheet = undefined;
  }

  /** The options of the item typed: the ingredients suggested, then « Créer « … » ». Two items of one ingredient may be added (D23). */
  #options(text: string): Option[] {
    const { ingredients, create } = suggestIngredients(this.lists.library.catalog.items ?? [], text, []);
    const options: Option[] = ingredients.map((ingredient) => ({ ingredient }));
    if (create) {
      options.push({ create: trimSpaces(text) });
    }
    return options;
  }

  /** The ingredient of the catalog that the text names, in the sense of the normalized name. */
  #known(text: string): Ingredient | undefined {
    if (!hasVisible(text)) {
      return undefined;
    }
    return this.lists.library.catalog.items?.find((i) => normalizeName(i.name) === normalizeName(text));
  }

  #setDraft(change: Partial<Draft>) {
    this.draft = { ...this.draft, ...change };
    this.formError = undefined;
  }

  #type(ingredient: string) {
    this.#setDraft({ ingredient });
  }

  /** An ingredient chosen: the item takes it, and its default unit if a quantity is there. */
  #choose(option: Option) {
    const ingredient = option.ingredient?.name ?? option.create ?? "";
    const unit = this.draft.unit === "" && trimSpaces(this.draft.quantity) !== "" ? (option.ingredient?.defaultUnit ?? "") : this.draft.unit;
    this.#setDraft({ ingredient, unit });
  }

  /** A quantity typed takes the default unit of the ingredient (D23); without quantity, no unit. */
  #setQuantity(quantity: string) {
    if (trimSpaces(quantity) === "") {
      this.#setDraft({ quantity, unit: "" });
      return;
    }
    const unit = this.draft.unit === "" ? (this.#known(this.draft.ingredient)?.defaultUnit ?? "") : this.draft.unit;
    this.#setDraft({ quantity, unit });
  }

  #startAdding() {
    this.notice = undefined;
    this.adding = true;
    this.draft = emptyDraft;
    this.formError = undefined;
    void this.updateComplete.then(() => this.querySelector<HTMLInputElement>(".item-combobox")?.focus());
  }

  #cancelAdding() {
    this.adding = false;
    this.draft = emptyDraft;
    this.formError = undefined;
    this.#combobox.close();
  }

  /** Checks before sending, in the order of the server (dish.ParseLine): the name, the quantity, the unit. */
  #refusal(): string | undefined {
    const { ingredient, quantity, unit } = this.draft;
    if (hasControl(ingredient)) {
      return "ingredient_name_invalid";
    }
    if (!hasVisible(ingredient)) {
      return "ingredient_name_required";
    }
    if (trimSpaces(quantity) === "") {
      return undefined;
    }
    if (unit === "") {
      return "unit_required";
    }
    const reading = parseQuantity(quantity, unit);
    return "error" in reading ? reading.error : undefined;
  }

  async #submit(e: Event, list: ShoppingList) {
    e.preventDefault();
    if (this.#busy) {
      return;
    }
    this.notice = undefined;
    const refused = this.#refusal();
    if (refused !== undefined) {
      this.formError = refused;
      return;
    }
    const { ingredient, quantity, unit } = this.draft;
    this.#busy = true;
    try {
      const result = await this.lists.addItem(list.id, trimSpaces(quantity) === "" ? { ingredient } : { ingredient, quantity, unit });
      if (result.kind === "done") {
        // The form stays open, emptied, for another item (docs/design/README.md).
        this.draft = emptyDraft;
        this.formError = undefined;
        this.querySelector<HTMLInputElement>(".item-combobox")?.focus();
        return;
      }
      if (result.kind === "refused" && shoppingErrorMessage(result.error) !== undefined) {
        this.formError = result.error;
        return;
      }
      this.#failedWrite(result);
    } finally {
      this.#busy = false;
    }
  }

  async #setChecked(list: ShoppingList, item: Item, checked: boolean) {
    this.notice = undefined;
    const result = await this.lists.setChecked(list.id, item.id, checked);
    if (result.kind !== "done") {
      this.#failedWrite(result);
    }
  }

  async #remove(list: ShoppingList, item: Item) {
    this.notice = undefined;
    this.removalPaused = true;
    clearTimeout(this.#removalPause);
    try {
      const result = await this.lists.removeItem(list.id, item.id);
      if (result.kind !== "done") {
        this.#failedWrite(result);
      }
    } finally {
      this.#removalPause = setTimeout(() => (this.removalPaused = false), removalPauseMs);
    }
  }

  #askCheckout(list: ShoppingList) {
    this.notice = undefined;
    const { leaving, message } = checkoutOf(list);
    this.sheet = { leaving, message };
  }

  async #confirmCheckout() {
    const sheet = this.sheet;
    const list = this.#list();
    if (sheet === undefined || list === undefined || this.#busy) {
      return;
    }
    this.#busy = true;
    try {
      const result = await this.lists.checkOut(list.id, sheet.leaving);
      this.#closeSheet();
      if (result.kind === "done") {
        this.notice = { kind: "done", text: removedMessage(result.removed) };
        return;
      }
      this.#failedWrite(result);
    } finally {
      this.#busy = false;
    }
  }

  /** A write not done: the list deleted in between, or « Réseau nécessaire » (D2, D11). */
  #failedWrite(result: Exclude<WriteResult, { kind: "done" }>) {
    if (result.kind === "deleted") {
      this.dispatchEvent(new Event("list-deleted", { bubbles: true, composed: true }));
      return;
    }
    // A 401 has already brought back to the login.
    if (result.kind !== "unauthorized") {
      this.notice = { kind: "network" };
      window.scrollTo({ top: 0 });
    }
  }
}

customElements.define("mt-shopping-list-screen", ShoppingListScreen);

declare global {
  interface HTMLElementTagNameMap {
    "mt-shopping-list-screen": ShoppingListScreen;
  }
}
