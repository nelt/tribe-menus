// Screen of a shopping list, after the mock-ups Doux-Liste and Doux-ArticleMain (plan courses,
// C1, C3, C6, C11) and Doux-ListeSuite (plan courses-suite, C8 to C10): its name, which opens
// the list picker, and « Sélectionner »; the banner of a stale list with « Recalculer », the
// progress, the items (checkbox, quantity, what remains to take, « ajouté » and the cross of an
// item added), « Ajouter un article à la main » with the combobox of the catalog, and « Courses
// faites » with its sheet. In selection mode (D14), the items are selected, then moved to
// another list or to a new list, each through its sheet; a selection is not an input: a
// navigation loses it without confirmation, a change of format keeps it. The list waits for
// the copy of the lists, checked at each visit. Every write needs the network (D2): without
// it, « Réseau nécessaire » above the list, which stays shown. A list deleted in between (D11)
// sends to the main list, which names it and tells what was not done (plan courses-suite,
// D12).
import { LitElement, type PropertyValues, type TemplateResult, html, nothing } from "lit";
import { live } from "lit/directives/live.js";
import { repeat } from "lit/directives/repeat.js";
import { type Ingredient, hasControl, initialOf, hasVisible, maxQuantityLength, normalizeName, parseQuantity, suggestIngredients, trimSpaces, unitLabel, units } from "./dishes.ts";
import { IngredientCombobox, type Option } from "./ingredient-combobox.ts";
import { networkErrorMessage } from "./login.ts";
import { colorClass } from "./dish-library-screen.ts";
import {
  type Item,
  type LostWrite,
  type MovedMessage,
  type ShoppingList,
  checkListName,
  checkoutOf,
  deletedListMessage,
  dishCount,
  itemQuantity,
  itemRowLabel,
  maxItems,
  maxListNameLength,
  moveErrorMessage,
  moveTargets,
  moveTitle,
  movedMessage,
  newListSheetMessage,
  progressOf,
  recomputeErrorMessage,
  recomputedMessage,
  remainingQuantity,
  removedMessage,
  selectAllLabel,
  selectedIn,
  selectionHint,
  selectionLabel,
  shoppingErrorMessage,
  sortItems,
  sourceRows,
  staleMessage,
  takenSentence,
  targetSummary,
  toggleAll,
  toggleSelected,
} from "./shopping.ts";
import type { ShoppingLists, WriteResult } from "./shopping-lists.ts";
import { alertMessage, networkNeededMessage, warningBanner } from "./templates.ts";

/** How long the crosses of the items added wait after a removal: a slow double touch removes one item only. */
const removalPauseMs = 500;

/**
 * What the screen tells above the list: a green banner, the green banner of a move with the
 * link of its target list, a warning, an error, or « Réseau nécessaire ».
 */
export type Notice =
  | { kind: "done"; text: string }
  | { kind: "moved"; message: MovedMessage; href: string }
  | { kind: "deleted"; text: string }
  | { kind: "error"; text: string }
  | { kind: "network" };

/** The sheet open: « Courses faites ? » with the items it counted (D9), « Déplacer vers… », « Nouvelle liste ». */
type Sheet = { kind: "checkout"; leaving: string[]; message: string } | { kind: "move" } | { kind: "new-list"; name: string; error: string | undefined };

/**
 * The event of a write refused for a list deleted in between (D11): its detail is the warning
 * the main list shows, which names the list and tells what was not done (plan courses-suite, D12).
 */
export function listDeletedEvent(name: string | undefined, lost?: LostWrite): CustomEvent<string> {
  return new CustomEvent("list-deleted", { detail: deletedListMessage(name, lost), bubbles: true, composed: true });
}

/** The item being added by hand: the ingredient typed or chosen, the quantity, the unit. */
interface Draft {
  ingredient: string;
  quantity: string;
  unit: string;
}

const emptyDraft: Draft = { ingredient: "", quantity: "", unit: "" };

const checkIcon = html`<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12.5l4.5 4.5L19 7.5"></path></svg>`;
const crossIcon = html`<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" aria-hidden="true"><path d="M6 6l12 12M18 6L6 18"></path></svg>`;
const calendarIcon = html`<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="5" width="18" height="16" rx="3"></rect><path d="M8 3v4M16 3v4M3 10h18M12 13v5M9.5 15.5h5"></path></svg>`;
/** The icon of the recomputation (Doux-ListeSuite), in the stale banner and the pill « Repas modifiés ». */
export function recomputeIcon(size: number, stroke: number) {
  return html`<svg width=${size} height=${size} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width=${stroke} stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M21 12a9 9 0 1 1-2.6-6.4"></path><path d="M21 4v5h-5"></path></svg>`;
}
const arrowIcon = html`<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M4 12h15M13 6l6 6-6 6"></path></svg>`;
const chevronIcon = html`<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M9 18l6-6-6-6"></path></svg>`;
const plusIcon = html`<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" aria-hidden="true"><path d="M12 5v14M5 12h14"></path></svg>`;

/** The green banner of a write done, with its cross (Doux-Courses). */
export function doneBanner(text: string | TemplateResult, close: () => void) {
  return html`<div class="banner-done" role="status">
    ${checkIcon}<span>${text}</span>
    <button class="banner-close" type="button" aria-label="Fermer le message" @click=${close}>${crossIcon}</button>
  </div>`;
}

/** An error banner: a write without network (D2), a refusal at the top of the screen. */
function errorBanner(text: string) {
  return html`<div class="banner-error" role="alert">
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="12" r="9"></circle><path d="M12 7v6M12 16.5v.5"></path></svg>
    <span>${text}</span>
  </div>`;
}

/** The error banner of a write without network (D2). */
export function networkBanner() {
  return errorBanner(networkNeededMessage);
}

/** Under the sources of an item with a taken quantity: what was taken and what remains (D9). */
function takenNote(item: Item) {
  const sentence = takenSentence(item);
  return sentence === undefined ? nothing : html`<p class="item-taken-note">${sentence}</p>`;
}

export class ShoppingListScreen extends LitElement {
  static override properties = {
    lists: { attribute: false },
    listId: { attribute: false },
    pickerHref: { attribute: false },
    mealShoppingHref: { attribute: false },
    listHref: { attribute: false },
    notice: { attribute: false },
    beside: { attribute: false },
    failed: { state: true },
    revision: { state: true },
    adding: { state: true },
    draft: { state: true },
    formError: { state: true },
    sheet: { state: true },
    removalPaused: { state: true },
    open: { state: true },
    recomputing: { state: true },
    recomputeError: { state: true },
    selected: { state: true },
  };

  declare lists: ShoppingLists;
  /** The list shown; undefined for the main list (C1). */
  declare listId: string | undefined;
  declare pickerHref: string;
  /** The meal shopping, toward a list (D15). */
  declare mealShoppingHref: (list: ShoppingList) => string;
  /** A list, from the banner after a move (D14). */
  declare listHref: (list: ShoppingList) => string;
  /** What the screen tells above the list; given by the application after a list deleted. */
  declare notice: Notice | undefined;
  /** Beside the list picker, in etendu: the name is a title alone, the picker is beside (plan grands-ecrans, D5). */
  beside = false;
  /** The last check failed for lack of network, the copy never loaded. */
  failed = false;
  /** Increases at each change of the copy, to render again. */
  revision = 0;
  /** The form of an item added by hand is open. */
  adding = false;
  draft: Draft = emptyDraft;
  /** The code of the refusal of the item added. */
  formError: string | undefined = undefined;
  /** The sheet open. */
  sheet: Sheet | undefined = undefined;
  /** The computed items unfolded (C5). */
  open: ReadonlySet<string> = new Set();
  /** « Recalculer » runs (D13). */
  recomputing = false;
  /** The message of a refusal of « Recalculer », under the stale banner (Q38). */
  recomputeError: string | undefined = undefined;
  /**
   * The items selected, in selection mode (D14); undefined out of it. An item gone from the
   * copy leaves the selection (selectedIn).
   */
  selected: ReadonlySet<string> | undefined = undefined;
  /** The dishes of the sources already looked for by a check of the library (plan courses-suite, D15). */
  readonly #lookedFor = new Set<string>();
  /** A removal runs, or has just ended: the crosses wait. */
  removalPaused = false;
  #removalPause: ReturnType<typeof setTimeout> | undefined;
  /** A write runs that a second touch must not repeat: an item added, a checkout, a move. */
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
    // The lists, with the library: the catalog of the suggestions, the dishes of the unfolded lines.
    void this.#check();
  }

  override disconnectedCallback() {
    super.disconnectedCallback();
    this.lists.removeEventListener("change", this.#onChange);
    clearTimeout(this.#removalPause);
  }

  /**
   * The selection follows the copy (D14): an item gone leaves it, and comes back unselected; a
   * list emptied ends the selection mode, with its sheet (review of PR #121, point 1).
   */
  protected override willUpdate() {
    const list = this.#list();
    if (this.selected === undefined || list === undefined) {
      return;
    }
    if (list.items.length === 0) {
      this.selected = undefined;
      if (this.sheet?.kind === "move" || this.sheet?.kind === "new-list") {
        this.sheet = undefined;
      }
      return;
    }
    const kept = selectedIn(this.selected, list);
    if (kept.length !== this.selected.size) {
      this.selected = new Set(kept);
    }
  }

  protected override updated(changed: PropertyValues<this>) {
    const dialog = this.querySelector<HTMLDialogElement>("dialog.sheet");
    if (changed.has("sheet") && dialog !== null && this.sheet !== undefined && !dialog.open) {
      dialog.showModal();
    }
    this.#lookForDishes();
  }

  /**
   * A source unfolded whose dish the copy of the library does not hold (a dish created by
   * another member since its check) makes it check once, then the source shows (plan
   * courses-suite, D15; review of PR #101, point 3).
   */
  #lookForDishes() {
    const list = this.#list();
    const dishes = this.lists.library.dishes.items;
    if (list === undefined || dishes === undefined) {
      return;
    }
    const known = new Set(dishes.map((d) => d.id));
    const missing = list.items
      .filter((i) => this.open.has(i.id))
      .flatMap((i) => i.sources.map((s) => s.dish))
      .filter((d) => !known.has(d) && !this.#lookedFor.has(d));
    if (missing.length > 0) {
      for (const d of missing) {
        this.#lookedFor.add(d);
      }
      void this.lists.checkLibrary();
    }
  }

  /** An item is being typed by hand, not added yet (plan grands-ecrans, D9). */
  get unsaved(): boolean {
    return this.adding && (trimSpaces(this.draft.ingredient) !== "" || trimSpaces(this.draft.quantity) !== "");
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
    // Never empty, nor holding an item gone (willUpdate).
    if (this.selected !== undefined) {
      const selection = selectedIn(this.selected, list);
      return html`<main class="screen shopping shopping--selecting">
          ${this.#selectionHeader(list, this.selected)} ${this.#notice(true)} ${this.#items(list, this.selected)} ${this.#selectionActions(selection)}
        </main>
        ${this.#sheet(list, selection)}`;
    }
    return html`<main class="screen shopping">
      ${this.#header(list)} ${this.#notice(false)} ${this.#items(list, undefined)} ${this.#mealShopping(list)} ${this.#addItem(list)} ${this.#checkout(list)}
    </main>
    ${this.#sheet(list, [])}`;
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
    return html`<div class="shopping-head">
        <h1 class="screen-title shopping-title" tabindex="-1">
        ${this.beside
          ? html`<span class="shopping-title-text"><span class="shopping-title-name">${list.name}</span>${list.main ? html`<span class="pill pill--main">principale</span>` : nothing}</span>`
          : html`<a class="shopping-title-link" href=${this.pickerHref} aria-label=${label}>
              <span class="shopping-title-name">${list.name}</span>
              <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M6 9l6 6 6-6"></path></svg>
              ${list.main ? html`<span class="pill pill--main">principale</span>` : nothing}
            </a>`}
        </h1>
        ${list.items.length === 0 ? nothing : html`<button class="button-text shopping-select" type="button" @click=${this.#startSelecting}>Sélectionner</button>`}
      </div>
      ${this.#stale(list)}
      ${progress.total === 0
        ? nothing
        : html`<div class="progress">
            <div class="progress-text"><span>${progress.label}</span><span class="progress-remaining">${progress.remaining}</span></div>
            <progress class="progress-bar" max=${progress.total} .value=${progress.checked} aria-hidden="true"></progress>
          </div>`}`;
  }

  /** The banner of a stale list, with « Recalculer » (C8, D13), and the refusal of 300 items under it. */
  #stale(list: ShoppingList) {
    if (!this.lists.isStale(list.id)) {
      return nothing;
    }
    return html`<div class="banner-stale">
        <p class="banner-stale-text">${recomputeIcon(22, 2.2)}<span>${staleMessage}</span></p>
        <button class="button-recompute" type="button" ?disabled=${this.recomputing} @click=${() => this.#recompute(list)}>${this.recomputing ? "Recalcul en cours…" : "Recalculer"}</button>
      </div>
      ${this.recomputeError === undefined ? nothing : errorBanner(this.recomputeError)}`;
  }

  /** The header of the selection mode (D14): « Annuler », the number of items selected, « Tout » or « Aucun ». */
  #selectionHeader(list: ShoppingList, selected: ReadonlySet<string>) {
    return html`<div class="selection-head">
        <button class="button-text selection-cancel" type="button" @click=${this.#stopSelecting}>Annuler</button>
        <h1 class="screen-title selection-title" aria-live="polite">${selectionLabel(selectedIn(selected, list).length)}</h1>
        <button class="button-text selection-all" type="button" @click=${() => (this.selected = toggleAll(selected, list))}>${selectAllLabel(selected, list)}</button>
      </div>
      <p class="screen-text selection-hint">${selectionHint(list.name)}</p>`;
  }

  /**
   * The actions of the selection, inactive without item selected (C9, C10): in compact in place
   * of the tab bar, at the bottom of the pane of the list in moyen and etendu (D14).
   */
  #selectionActions(selection: string[]) {
    const none = selection.length === 0;
    return html`<div class="selection-actions" role="group" aria-label="Actions de la sélection">
      <button class="selection-action selection-action--move" type="button" ?disabled=${none} @click=${() => this.#openSheet({ kind: "move" })}>${none ? nothing : arrowIcon}Déplacer vers…</button>
      <button class="selection-action selection-action--new" type="button" ?disabled=${none} @click=${this.#openNewList}>${none ? nothing : plusIcon}Nouvelle liste</button>
    </div>`;
  }

  /** The banners; in selection mode, the green ones are left out (Doux-ListeSuite). */
  #notice(selecting: boolean) {
    const notice = this.notice;
    switch (notice?.kind) {
      case "done":
        return selecting ? nothing : doneBanner(notice.text, () => (this.notice = undefined));
      case "moved": {
        const { start, link, end } = notice.message;
        return selecting ? nothing : doneBanner(html`${start}<a class="banner-link" href=${notice.href}>${link}</a>${end}`, () => (this.notice = undefined));
      }
      case "deleted":
        return warningBanner(notice.text);
      case "error":
        return errorBanner(notice.text);
      case "network":
        return networkBanner();
      default:
        return nothing;
    }
  }

  /** The items; in selection mode, with those selected (D14). */
  #items(list: ShoppingList, selection: ReadonlySet<string> | undefined) {
    if (list.items.length === 0) {
      return html`<p class="shopping-empty">La liste est vide. Ajoutez les courses des repas ou un article à la main.</p>`;
    }
    return html`<ul class="items">${repeat(
      sortItems(list.items),
      (i) => i.id,
      (i) => (selection === undefined ? this.#item(list, i) : this.#pickable(i, selection.has(i.id))),
    )}</ul>`;
  }

  /** What a line shows of an item: its name and what remains to take, « ajouté » or its dishes, its quantity. */
  #shown(item: Item) {
    const quantity = itemQuantity(item);
    const rest = remainingQuantity(item);
    const name = html`<span class="item-name">${item.ingredient}</span>`;
    return html`${rest === undefined ? name : html`<span class="item-names">${name} <span class="item-remaining">${rest}</span></span>`}
      ${item.manual ? html`<span class="pill pill--dashed">ajouté</span>` : html`<span class="pill">${dishCount(item)}</span>`}
      ${quantity === "" ? nothing : html`<span class="item-quantity">${quantity}</span>`}`;
  }

  /**
   * An item in selection mode (D14): one button, a round mark, without unfolding; checked
   * items too. Named with its quantity: two items added may share an ingredient (D23).
   */
  #pickable(item: Item, picked: boolean) {
    const quantity = itemQuantity(item);
    return html`<li class="item ${item.checked ? "item--checked" : ""} ${picked ? "item--picked" : ""}">
      <button
        class="item-pick"
        type="button"
        aria-pressed=${picked ? "true" : "false"}
        aria-label=${`Sélectionner ${item.ingredient}${quantity === "" ? "" : `, ${quantity}`}`}
        @click=${() => (this.selected = toggleSelected(this.selected ?? new Set(), item.id))}
      >
        <span class="item-pick-mark" aria-hidden="true">${checkIcon}</span>
        ${this.#shown(item)}
      </button>
    </li>`;
  }

  #item(list: ShoppingList, item: Item) {
    const open = !item.manual && this.open.has(item.id);
    const shown = this.#shown(item);
    return html`<li class="item ${item.checked ? "item--checked" : ""} ${open ? "item--open" : ""}">
      <div class="item-row">
        <input class="item-check" type="checkbox" aria-label=${`Cocher ${item.ingredient}`} .checked=${live(item.checked)} @change=${(e: Event) => this.#setChecked(list, item, (e.target as HTMLInputElement).checked)}>
        ${item.manual
          ? html`${shown}<button class="item-remove" type="button" aria-label=${`Retirer ${item.ingredient}`} ?disabled=${this.removalPaused} @click=${() => this.#remove(list, item)}>${crossIcon}</button>`
          : html`<button class="item-unfold" type="button" aria-expanded=${open ? "true" : "false"} aria-label=${itemRowLabel(item, open)} @click=${() => this.#toggle(item)}>
              ${shown}
              <svg class="item-chevron" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M6 9l6 6 6-6"></path></svg>
            </button>`}
      </div>
      ${open ? this.#sources(item) : nothing}
    </li>`;
  }

  /** The lines of an item unfolded (C5): the dish and its servings, the meal, the quantity brought. */
  #sources(item: Item) {
    const rows = sourceRows(item, this.lists.library.dishes.items ?? []);
    return html`<ul class="item-sources" aria-label=${`Repas de ${item.ingredient}`}>
      ${rows.map(
        (r) => html`<li class="item-source ${colorClass(r.dish.color)}">
          <span class="dish-initial dish-initial--tiny" aria-hidden="true">${initialOf(r.dish.name)}</span>
          <span class="item-source-text">
            <span class="item-source-dish">${r.dish.name} · ${r.servings}</span>
            <span class="item-source-meal">${r.day} · <span class="item-source-mealtime item-source-mealtime--${r.mealtime}">${r.mealtime === "lunch" ? "midi" : "soir"}</span></span>
          </span>
          ${r.quantity === "" ? nothing : html`<span class="item-source-quantity">${r.quantity}</span>`}
        </li>`,
      )}
    </ul>
    ${takenNote(item)}`;
  }

  /** Unfolds or folds a computed item (C5). */
  #toggle(item: Item) {
    const open = new Set(this.open);
    if (!open.delete(item.id)) {
      open.add(item.id);
    }
    this.open = open;
  }

  /** « Ajouter les courses des repas », under the items (Doux-Liste). */
  #mealShopping(list: ShoppingList) {
    return html`<a class="button-meal-shopping" href=${this.mealShoppingHref(list)}>
      ${calendarIcon}<span>Ajouter les courses des repas</span>
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M9 18l6-6-6-6"></path></svg>
    </a>`;
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

  #sheet(list: ShoppingList, selection: string[]) {
    const sheet = this.sheet;
    switch (sheet?.kind) {
      case undefined:
        return nothing;
      case "checkout":
        return html`<dialog class="sheet" aria-labelledby="checkout-title" @close=${() => (this.sheet = undefined)} @click=${this.#closeOnBackdrop}>
          <h2 class="sheet-title" id="checkout-title">Courses faites ?</h2>
          <p class="sheet-text">${sheet.message}</p>
          <button class="button-primary" type="button" @click=${this.#confirmCheckout}>Retirer les articles cochés</button>
          <button class="button-text button-text--wide" type="button" @click=${this.#closeSheet}>Annuler</button>
        </dialog>`;
      case "move":
        return this.#moveSheet(list, selection);
      case "new-list":
        return this.#newListSheet(list, selection, sheet);
    }
  }

  /** « Déplacer N articles vers… » (C9): the other lists, in the order of the list picker, each with its number of items. */
  #moveSheet(list: ShoppingList, selection: string[]) {
    const targets = moveTargets(this.lists.lists.items ?? [], list.id);
    return html`<dialog class="sheet" aria-labelledby="move-title" @close=${() => (this.sheet = undefined)} @click=${this.#closeOnBackdrop}>
      <h2 class="sheet-title" id="move-title">${moveTitle(selection.length)}</h2>
      <p class="sheet-text">Un article calculé rejoint la ligne du même ingrédient s’il y en a une ; un article ajouté à la main reste à part.</p>
      ${targets.length === 0
        ? html`<p class="sheet-text">Aucune autre liste : créez-en une avec « Nouvelle liste ».</p>`
        : html`<div class="sheet-actions">
            ${targets.map(
              (t) => html`<button class="sheet-action sheet-target" type="button" @click=${() => this.#move(list, selection, { list: t.id }, t.name)}>
                <span class="sheet-target-name">${t.name}</span><span class="sheet-target-summary">${targetSummary(t)}</span>${chevronIcon}
              </button>`,
            )}
          </div>`}
      <button class="button-text button-text--wide" type="button" @click=${this.#closeSheet}>Annuler</button>
    </dialog>`;
  }

  /** « Nouvelle liste » from a selection (C10): its name, then « Créer la liste », inactive while the name is empty. */
  #newListSheet(list: ShoppingList, selection: string[], sheet: Sheet & { kind: "new-list" }) {
    const message = sheet.error === undefined ? undefined : shoppingErrorMessage(sheet.error);
    return html`<dialog class="sheet" aria-labelledby="new-list-title" @close=${() => (this.sheet = undefined)} @click=${this.#closeOnBackdrop}>
      <h2 class="sheet-title" id="new-list-title">Nouvelle liste</h2>
      <p class="sheet-text">${newListSheetMessage(selection.length, list.name)}</p>
      <form class="sheet-form" novalidate @submit=${(e: Event) => this.#submitNewList(e, list, selection)}>
        <label class="item-form-label">Nom de la liste
          <input
            class="field-input"
            type="text"
            name="list-name"
            maxlength=${maxListNameLength}
            autocomplete="off"
            placeholder="ex. Marché, Boucherie"
            aria-invalid=${message === undefined ? "false" : "true"}
            aria-describedby=${message === undefined ? nothing : "new-list-error"}
            .value=${live(sheet.name)}
            @input=${(e: Event) => (this.sheet = { ...sheet, name: (e.target as HTMLInputElement).value, error: undefined })}
          >
        </label>
        ${message === undefined ? nothing : alertMessage("new-list-error", message)}
        <button class="button-primary" type="submit" ?disabled=${trimSpaces(sheet.name) === ""}>Créer la liste</button>
        <button class="button-text button-text--wide" type="button" @click=${this.#closeSheet}>Annuler</button>
      </form>
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
    this.#clearNotices();
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
    this.#clearNotices();
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
      this.#failedWrite(result, "add-item");
    } finally {
      this.#busy = false;
    }
  }

  async #setChecked(list: ShoppingList, item: Item, checked: boolean) {
    this.#clearNotices();
    const result = await this.lists.setChecked(list.id, item.id, checked);
    if (result.kind !== "done") {
      this.#failedWrite(result, "check");
    }
  }

  async #remove(list: ShoppingList, item: Item) {
    this.#clearNotices();
    this.removalPaused = true;
    clearTimeout(this.#removalPause);
    try {
      const result = await this.lists.removeItem(list.id, item.id);
      if (result.kind !== "done") {
        this.#failedWrite(result, "remove");
      }
    } finally {
      this.#removalPause = setTimeout(() => (this.removalPaused = false), removalPauseMs);
    }
  }

  /** The banners disappear at the next action (docs/design/README.md, « Messages des listes de courses »). */
  #clearNotices() {
    this.notice = undefined;
    this.recomputeError = undefined;
  }

  /** « Recalculer » (C8, D13): « Liste recalculée. » once done; refused beyond 300 items (Q38). */
  async #recompute(list: ShoppingList) {
    if (this.recomputing) {
      return;
    }
    this.#clearNotices();
    this.recomputing = true;
    try {
      const result = await this.lists.recompute(list.id);
      if (result.kind === "done") {
        this.notice = { kind: "done", text: recomputedMessage };
        return;
      }
      const message = result.kind === "refused" ? recomputeErrorMessage(result.error) : undefined;
      if (message !== undefined) {
        this.recomputeError = message;
        return;
      }
      this.#failedWrite(result, "recompute");
    } finally {
      this.recomputing = false;
    }
  }

  #askCheckout(list: ShoppingList) {
    this.#clearNotices();
    const { leaving, message } = checkoutOf(list);
    this.sheet = { kind: "checkout", leaving, message };
  }

  async #confirmCheckout() {
    const sheet = this.sheet;
    const list = this.#list();
    if (sheet?.kind !== "checkout" || list === undefined || this.#busy) {
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
      this.#failedWrite(result, "checkout");
    } finally {
      this.#busy = false;
    }
  }

  /** « Sélectionner » (D14): the selection mode, nothing selected; the focus to « Annuler », the button gone. */
  #startSelecting() {
    this.#clearNotices();
    this.#cancelAdding();
    this.selected = new Set();
    void this.updateComplete.then(() => this.querySelector<HTMLElement>(".selection-cancel")?.focus());
  }

  /**
   * Out of the selection mode, by « Annuler » or a move done: the focus back to
   * « Sélectionner », or to the title of a list emptied (review of PR #121, point 2).
   */
  #stopSelecting() {
    this.selected = undefined;
    void this.updateComplete.then(() => (this.querySelector<HTMLElement>(".shopping-select") ?? this.querySelector<HTMLElement>(".shopping-title"))?.focus());
  }

  #openSheet(sheet: Sheet) {
    this.#clearNotices();
    this.sheet = sheet;
  }

  #openNewList() {
    this.#openSheet({ kind: "new-list", name: "", error: undefined });
    void this.updateComplete.then(() => this.querySelector<HTMLInputElement>("dialog.sheet input")?.focus());
  }

  async #submitNewList(e: Event, list: ShoppingList, selection: string[]) {
    e.preventDefault();
    const sheet = this.sheet;
    if (sheet?.kind !== "new-list") {
      return;
    }
    const refused = checkListName(sheet.name);
    if (refused !== undefined) {
      this.sheet = { ...sheet, error: refused };
      return;
    }
    await this.#move(list, selection, { name: sheet.name }, trimSpaces(sheet.name));
  }

  /**
   * Moves the items selected (C9, C10, D10): back out of the selection mode, with the green
   * banner whose link opens the target list. Refused, the selection is kept: the target list
   * deleted in between or full, above the list (Q42, Q38); a name or 20 lists, in the sheet.
   */
  async #move(list: ShoppingList, selection: string[], to: { list: string } | { name: string }, target: string) {
    if (this.#busy) {
      return;
    }
    // Every item selected has left the copy in between: nothing to send.
    if (selection.length === 0) {
      this.#closeSheet();
      return;
    }
    this.#busy = true;
    try {
      const result = await this.lists.move(list.id, selection, to);
      if (result.kind === "done") {
        this.#closeSheet();
        this.#stopSelecting();
        this.notice = { kind: "moved", message: movedMessage(result.moved, result.missing, result.target.name, "name" in to), href: this.listHref(result.target) };
        return;
      }
      const sheet = this.sheet;
      if (result.kind === "refused" && sheet?.kind === "new-list" && shoppingErrorMessage(result.error) !== undefined) {
        this.sheet = { ...sheet, error: result.error };
        return;
      }
      this.#closeSheet();
      const refusal = result.kind === "refused" ? moveErrorMessage(result.error, target) : undefined;
      if (refusal !== undefined) {
        this.notice = refusal.kind === "warning" ? { kind: "deleted", text: refusal.text } : { kind: "error", text: refusal.text };
        this.#scrollToTop();
        return;
      }
      this.#failedWrite(result, "move");
    } finally {
      this.#busy = false;
    }
  }

  /** A write not done: the list deleted in between, or « Réseau nécessaire » (D2, D11). */
  #failedWrite(result: Exclude<WriteResult, { kind: "done" }>, lost: LostWrite) {
    if (result.kind === "deleted") {
      this.dispatchEvent(listDeletedEvent(result.name, lost));
      return;
    }
    // A 401 has already brought back to the login.
    if (result.kind !== "unauthorized") {
      this.notice = { kind: "network" };
      this.#scrollToTop();
    }
  }

  /** Up to the banners: the window, or the pane of the list in etendu. */
  #scrollToTop() {
    window.scrollTo({ top: 0 });
    this.scrollTop = 0;
  }
}

customElements.define("mt-shopping-list-screen", ShoppingListScreen);

declare global {
  interface HTMLElementTagNameMap {
    "mt-shopping-list-screen": ShoppingListScreen;
  }
}
