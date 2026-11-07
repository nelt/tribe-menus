// Screen of the list picker, after the mock-up Doux-ChoixListes (plan courses, C1, C2, C12):
// the lists in the order of C2, each with its progress, which opens it, and its button « ⋮ »
// for its sheet of actions (rename, make it the main list, delete it, inactive with its
// reason); « Nouvelle liste », which opens the list created. Every write needs the network
// (D2). Crossed writes (D11): a list no longer empty is told in the sheet, with the list up
// to date; a list deleted in between sends to the main list, which tells it.
import { LitElement, type PropertyValues, html, nothing } from "lit";
import { live } from "lit/directives/live.js";
import { repeat } from "lit/directives/repeat.js";
import { networkErrorMessage } from "./login.ts";
import { type ShoppingList, checkListName, deleteBlocked, listSummary, maxListNameLength, maxLists, progressOf, shoppingErrorMessage, sortLists } from "./shopping.ts";
import { networkBanner } from "./shopping-list-screen.ts";
import type { ShoppingLists, WriteResult } from "./shopping-lists.ts";
import { alertMessage, warningBanner } from "./templates.ts";

/** The sheet « ⋮ » of a list: its actions, or its renaming; a warning after a refusal. */
interface Sheet {
  id: string;
  renaming: boolean;
  name: string;
  error: string | undefined;
  warning: string | undefined;
}

const tooManyListsMessage = "20 listes au plus : supprimez une liste vide pour en créer une autre.";

export class ListPickerScreen extends LitElement {
  static override properties = {
    lists: { attribute: false },
    backHref: { attribute: false },
    listHref: { attribute: false },
    failed: { state: true },
    revision: { state: true },
    network: { state: true },
    creating: { state: true },
    newName: { state: true },
    nameError: { state: true },
    sheet: { state: true },
  };

  declare lists: ShoppingLists;
  /** The list the member comes from. */
  declare backHref: string;
  declare listHref: (list: ShoppingList) => string;
  failed = false;
  revision = 0;
  /** The last write failed for lack of network (D2). */
  network = false;
  creating = false;
  newName = "";
  nameError: string | undefined = undefined;
  sheet: Sheet | undefined = undefined;
  #busy = false;
  readonly #onChange = () => this.revision++;

  protected override createRenderRoot() {
    return this;
  }

  override connectedCallback() {
    super.connectedCallback();
    this.lists.addEventListener("change", this.#onChange);
    void this.#check();
  }

  override disconnectedCallback() {
    super.disconnectedCallback();
    this.lists.removeEventListener("change", this.#onChange);
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
    this.failed = outcome.kind === "network-error" && !this.lists.loaded;
  }

  override render() {
    const lists = this.lists.lists.items;
    return html`<main class="screen shopping">
        <a class="back-link" href=${this.backHref}>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M15 18l-6-6 6-6"></path></svg>Retour à la liste
        </a>
        <h1 class="screen-title shopping-title">Listes</h1>
        ${this.network ? networkBanner() : nothing}
        ${this.failed ? html`${alertMessage("picker-error", networkErrorMessage)}<button class="button-secondary" type="button" @click=${this.#check}>Réessayer</button>` : nothing}
        ${lists === undefined ? nothing : this.#lists(lists)}
      </main>
      ${lists === undefined ? nothing : this.#sheet(lists)}`;
  }

  #lists(lists: ShoppingList[]) {
    return html`<ul class="list-cards">${repeat(sortLists(lists), (l) => l.id, (l) => this.#card(l))}</ul>
      <p class="screen-text">La liste principale s’ouvre dans l’onglet Courses et reçoit par défaut les courses des repas.</p>
      ${this.#create(lists)}`;
  }

  #card(list: ShoppingList) {
    const progress = progressOf(list);
    return html`<li class="list-card ${list.main ? "list-card--main" : ""}">
      <a class="list-card-link" href=${this.listHref(list)}>
        <span class="list-card-head">
          <span class="list-card-name">${list.name}</span>
          ${list.main ? html`<span class="pill pill--main">principale</span>` : nothing}
        </span>
        <span class="list-card-summary">${listSummary(list)}</span>
        ${progress.total === 0
          ? nothing
          : html`<span class="list-card-progress">
              <progress class="progress-bar progress-bar--small" max=${progress.total} .value=${progress.checked} aria-hidden="true"></progress>
              <span>${progress.checked} sur ${progress.total}</span>
            </span>`}
      </a>
      <button class="list-card-actions" type="button" aria-label=${`Actions pour la liste ${list.name}`} @click=${() => this.#openSheet(list)}>
        <svg width="22" height="22" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><circle cx="12" cy="5" r="1.9"></circle><circle cx="12" cy="12" r="1.9"></circle><circle cx="12" cy="19" r="1.9"></circle></svg>
      </button>
    </li>`;
  }

  #create(lists: ShoppingList[]) {
    if (this.creating) {
      const message = this.nameError === undefined ? undefined : shoppingErrorMessage(this.nameError);
      return html`<form class="item-form" novalidate @submit=${this.#submitCreate}>
        <label class="item-form-label">Nom de la liste
          <input
            class="field-input"
            type="text"
            name="list-name"
            maxlength=${maxListNameLength}
            autocomplete="off"
            placeholder="ex. Marché, Drive"
            aria-invalid=${message === undefined ? "false" : "true"}
            aria-describedby=${message === undefined ? nothing : "new-list-error"}
            .value=${live(this.newName)}
            @input=${(e: Event) => {
              this.newName = (e.target as HTMLInputElement).value;
              this.nameError = undefined;
            }}
          >
        </label>
        ${message === undefined ? nothing : alertMessage("new-list-error", message)}
        <div class="item-form-actions">
          <button class="button-primary button-primary--small" type="submit">Créer la liste</button>
          <button class="button-text" type="button" @click=${this.#cancelCreate}>Annuler</button>
        </div>
      </form>`;
    }
    if (lists.length >= maxLists) {
      return html`<div class="button-with-hint">
        <button class="button-primary" type="button" disabled>Nouvelle liste</button>
        <span class="hint">${tooManyListsMessage}</span>
      </div>`;
    }
    return html`<button class="button-primary" type="button" @click=${this.#startCreate}>
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" aria-hidden="true"><path d="M12 5v14M5 12h14"></path></svg>Nouvelle liste
    </button>`;
  }

  #sheet(lists: ShoppingList[]) {
    const sheet = this.sheet;
    const list = sheet === undefined ? undefined : lists.find((l) => l.id === sheet.id);
    if (sheet === undefined || list === undefined) {
      return nothing;
    }
    const blocked = deleteBlocked(list);
    const message = sheet.error === undefined ? undefined : shoppingErrorMessage(sheet.error);
    return html`<dialog class="sheet" aria-labelledby="list-sheet-title" @close=${() => (this.sheet = undefined)} @click=${this.#closeOnBackdrop}>
      <div class="sheet-head">
        <h2 class="sheet-title" id="list-sheet-title">${list.name}</h2>
        ${list.main ? html`<span class="pill pill--main">principale</span>` : nothing}
      </div>
      ${sheet.warning === undefined ? nothing : warningBanner(sheet.warning)}
      ${sheet.renaming
        ? html`<form class="sheet-form" novalidate @submit=${(e: Event) => this.#submitRename(e, list)}>
            <label class="item-form-label">Nom de la liste
              <input
                class="field-input"
                type="text"
                name="list-name"
                maxlength=${maxListNameLength}
                autocomplete="off"
                aria-invalid=${message === undefined ? "false" : "true"}
                aria-describedby=${message === undefined ? nothing : "rename-error"}
                .value=${live(sheet.name)}
                @input=${(e: Event) => (this.sheet = { ...sheet, name: (e.target as HTMLInputElement).value, error: undefined })}
              >
            </label>
            ${message === undefined ? nothing : alertMessage("rename-error", message)}
            <button class="button-primary" type="submit">Enregistrer</button>
            <button class="button-text button-text--wide" type="button" @click=${() => (this.sheet = { ...sheet, renaming: false, error: undefined })}>Annuler</button>
          </form>`
        : html`<div class="sheet-actions">
              <button class="sheet-action" type="button" @click=${() => this.#startRename(sheet, list)}>
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M4 20h4L19 9l-4-4L4 16z"></path><path d="M13.5 6.5l4 4"></path></svg>Renommer
              </button>
              ${list.main
                ? nothing
                : html`<button class="sheet-action" type="button" @click=${() => this.#setMain(list)}>
                    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 3.5l2.6 5.4 5.9.8-4.3 4.1 1 5.8L12 16.8l-5.2 2.8 1-5.8-4.3-4.1 5.9-.8z"></path></svg>Définir comme liste principale
                  </button>`}
              ${blocked === undefined
                ? html`<button class="sheet-action sheet-action--danger" type="button" @click=${() => this.#delete(list)}>
                    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M4 7h16M9 7V4h6v3M6 7l1 13h10l1-13"></path></svg>Supprimer la liste
                  </button>`
                : html`<button class="sheet-action sheet-action--inactive" type="button" disabled aria-describedby="delete-reason">
                      Supprimer la liste<span class="sheet-action-reason" id="delete-reason">${blocked}</span>
                    </button>`}
            </div>
            <button class="button-text button-text--wide" type="button" @click=${this.#closeSheet}>Fermer</button>`}
    </dialog>`;
  }

  #closeOnBackdrop(e: MouseEvent) {
    if (e.target instanceof HTMLDialogElement) {
      this.#closeSheet();
    }
  }

  #closeSheet() {
    this.querySelector<HTMLDialogElement>("dialog.sheet")?.close();
    this.sheet = undefined;
  }

  #openSheet(list: ShoppingList) {
    this.network = false;
    this.sheet = { id: list.id, renaming: false, name: list.name, error: undefined, warning: undefined };
  }

  #startRename(sheet: Sheet, list: ShoppingList) {
    this.sheet = { ...sheet, renaming: true, name: list.name, error: undefined, warning: undefined };
    void this.updateComplete.then(() => this.querySelector<HTMLInputElement>("dialog.sheet input")?.focus());
  }

  #startCreate() {
    this.network = false;
    this.creating = true;
    this.newName = "";
    this.nameError = undefined;
    void this.updateComplete.then(() => this.querySelector<HTMLInputElement>(".item-form input")?.focus());
  }

  #cancelCreate() {
    this.creating = false;
    this.newName = "";
    this.nameError = undefined;
  }

  async #submitCreate(e: Event) {
    e.preventDefault();
    if (this.#busy) {
      return;
    }
    this.network = false;
    const refused = checkListName(this.newName);
    if (refused !== undefined) {
      this.nameError = refused;
      return;
    }
    this.#busy = true;
    try {
      const result = await this.lists.create(this.newName);
      if (result.kind === "done") {
        this.creating = false;
        if (result.list !== undefined) {
          this.dispatchEvent(new CustomEvent("open-list", { detail: result.list, bubbles: true, composed: true }));
        }
        return;
      }
      if (result.kind === "refused" && shoppingErrorMessage(result.error) !== undefined) {
        this.nameError = result.error;
        return;
      }
      this.#failedWrite(result);
    } finally {
      this.#busy = false;
    }
  }

  async #submitRename(e: Event, list: ShoppingList) {
    e.preventDefault();
    const sheet = this.sheet;
    if (sheet === undefined || this.#busy) {
      return;
    }
    const refused = checkListName(sheet.name);
    if (refused !== undefined) {
      this.sheet = { ...sheet, error: refused };
      return;
    }
    this.#busy = true;
    try {
      const result = await this.lists.rename(list.id, sheet.name);
      if (result.kind === "done") {
        this.#closeSheet();
        return;
      }
      if (result.kind === "refused" && shoppingErrorMessage(result.error) !== undefined) {
        this.sheet = { ...sheet, error: result.error };
        return;
      }
      this.#failedWrite(result);
    } finally {
      this.#busy = false;
    }
  }

  async #setMain(list: ShoppingList) {
    if (this.#busy) {
      return;
    }
    this.#busy = true;
    try {
      const result = await this.lists.setMain(list.id);
      if (result.kind === "done") {
        this.#closeSheet();
        return;
      }
      this.#failedWrite(result);
    } finally {
      this.#busy = false;
    }
  }

  /**
   * Deletes a list, without confirmation (C12). A list no longer empty in between: the sheet
   * tells it, with the list up to date; become the main list: the sheet shows the reason.
   */
  async #delete(list: ShoppingList) {
    const sheet = this.sheet;
    if (sheet === undefined || this.#busy) {
      return;
    }
    this.#busy = true;
    try {
      const result = await this.lists.delete(list.id);
      if (result.kind === "done") {
        this.#closeSheet();
        return;
      }
      if (result.kind === "refused" && result.error === "list_not_empty") {
        this.sheet = { ...sheet, warning: "Cette liste n’est plus vide." };
        return;
      }
      if (result.kind === "refused" && result.error === "main_list") {
        this.sheet = { ...sheet };
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
      this.#closeSheet();
      this.dispatchEvent(new Event("list-deleted", { bubbles: true, composed: true }));
      return;
    }
    if (result.kind !== "unauthorized") {
      this.#closeSheet();
      this.network = true;
      window.scrollTo({ top: 0 });
    }
  }
}

customElements.define("mt-list-picker", ListPickerScreen);

declare global {
  interface HTMLElementTagNameMap {
    "mt-list-picker": ListPickerScreen;
  }
}
