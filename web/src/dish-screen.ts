// Screen of a dish, after the mock-ups Doux-Plat (creation) and Doux-PlatModif (change),
// plan bibliotheque, D19: name, servings, lines (optional quantity, unit of the closed list,
// ingredient in an ARIA combobox on the catalog in memory, P4), « Enregistrer le plat ». A
// crossed change refused shows the banner of Doux-PlatConflit and the current dish (D5).
// Without network, nothing changes (D18).
import { LitElement, type PropertyValues, html, nothing } from "lit";
import { live } from "lit/directives/live.js";
import type { Api } from "./api.ts";
import { colorClass } from "./dish-library-screen.ts";
import {
  type Dish,
  type DishError,
  type Ingredient,
  checkDish,
  dishErrorMessage,
  hasVisible,
  initialOf,
  inputDiffers,
  inputOfDish,
  maxQuantityLength,
  normalizeName,
  quantityInput,
  suggestIngredients,
  trimSpaces,
  unitLabel,
  units,
} from "./dishes.ts";
import { IngredientCombobox, type Option } from "./ingredient-combobox.ts";
import type { Library } from "./library.ts";
import { networkErrorMessage } from "./login.ts";
import { alertMessage, networkNeededMessage, warningBanner } from "./templates.ts";

/** At most this many lines in a dish (Q31). */
const maxLines = 40;
/** The servings proposed when the size of the tribe cannot be read. */
const defaultServings = 4;

/** A line of the form: its ingredient chosen, or a draft whose ingredient is being typed. */
interface FormLine {
  key: number;
  /** The ingredient chosen; empty for a draft. */
  name: string;
  /** The text typed in the combobox of a draft. */
  text: string;
  quantity: string;
  unit: string;
}

type Load = "loading" | "ready" | "not-found" | "failed";

export class DishScreen extends LitElement {
  static override properties = {
    library: { attribute: false },
    api: { attribute: false },
    dishId: { attribute: false },
    libraryHref: { attribute: false },
    beside: { attribute: false },
    load: { state: true },
    name: { state: true },
    servings: { state: true },
    lines: { state: true },
    error: { state: true },
    conflict: { state: true },
  };

  declare library: Library;
  declare api: Api;
  /** The dish changed; undefined for a creation. */
  declare dishId: string | undefined;
  declare libraryHref: string;
  /** Beside the library, in etendu: no link back to it (plan grands-ecrans, D5). */
  beside = false;
  load: Load = "loading";
  name = "";
  /** Undefined until the size of the tribe is read, for a creation (D16). */
  servings: number | undefined = undefined;
  lines: FormLine[] = [];
  /** The refusal shown: a dish refused, or "network" for a save without network. */
  error: DishError | "network" | undefined = undefined;
  /** The last save was refused for a crossed change: the banner of Doux-PlatConflit. */
  conflict = false;
  /** The dish as read, for the title and the revision of the next save. */
  #dish: Dish | undefined;
  /** The servings were proposed without the size of the tribe: sent only if changed. */
  #servingsGuessed = false;
  /** The servings proposed to a creation: what a creation not changed shows. */
  #proposedServings: number | undefined;
  #nextKey = 0;
  #busy = false;
  #focusDraft = false;
  readonly #onChange = () => this.#libraryChanged();
  /** The combobox of the draft, the only line whose ingredient is typed. */
  readonly #combobox = new IngredientCombobox(this, {
    id: "ingredient-suggestions",
    inputClass: "line-combobox",
    label: "Ingrédient",
    options: (text) => this.#options(text),
    onType: (text) => this.#updateDraft({ text }),
    onChoose: (option) => this.#choose(option),
  });

  protected override createRenderRoot() {
    return this;
  }

  override connectedCallback() {
    super.connectedCallback();
    this.library.addEventListener("change", this.#onChange);
    void this.#start();
  }

  override disconnectedCallback() {
    super.disconnectedCallback();
    this.library.removeEventListener("change", this.#onChange);
  }

  protected override updated(changed: PropertyValues<this>) {
    if (changed.has("lines") && this.#focusDraft) {
      this.#focusDraft = false;
      this.querySelector<HTMLInputElement>(".line-combobox")?.focus();
    }
  }

  async #start() {
    if (this.dishId === undefined) {
      this.load = "ready";
      void this.#readTribeSize();
      // The catalog, for the suggestions, if the tab Plats was not visited yet (D25).
      if (!this.library.loaded) {
        await this.library.check();
      }
      return;
    }
    this.#libraryChanged();
    if (this.load === "loading") {
      await this.#loadLibrary();
    }
  }

  async #loadLibrary() {
    this.load = "loading";
    const outcome = await this.library.check();
    this.#libraryChanged();
    if (this.load === "loading") {
      // A 401 has already brought back to the login.
      this.load = outcome.kind === "ok" ? "not-found" : "failed";
    }
  }

  /** The dish changed is taken from the copy once, when it is there. */
  #libraryChanged() {
    if (this.dishId === undefined || this.load === "ready") {
      this.requestUpdate();
      return;
    }
    const dish = this.library.dishes.items?.find((d) => d.id === this.dishId);
    if (dish !== undefined) {
      this.#fill(dish);
      this.load = "ready";
    }
  }

  async #readTribeSize() {
    const outcome = await this.api.getTribe();
    if (this.servings !== undefined) {
      return;
    }
    this.servings = outcome.kind === "ok" ? outcome.value.size : defaultServings;
    this.#proposedServings = this.servings;
    this.#servingsGuessed = outcome.kind !== "ok";
  }

  /**
   * The form differs from what is saved, or from a creation not changed (plan grands-ecrans,
   * D9): what a navigation in moyen or etendu asks to confirm before losing it.
   */
  get unsaved(): boolean {
    if (this.load !== "ready") {
      return false;
    }
    const saved = this.#dish === undefined ? { name: "", servings: this.#proposedServings, lines: [] } : inputOfDish(this.#dish);
    const lines = this.#sentLines().map((l) => ({ name: l.sentName, quantity: l.quantity, unit: l.unit }));
    return inputDiffers(saved, { name: this.name, servings: this.servings, lines });
  }

  /** The form shows a dish: the one read, or the current one sent back with a refusal. */
  #fill(dish: Dish) {
    this.#dish = dish;
    this.name = dish.name;
    this.servings = dish.servings;
    this.lines = dish.ingredients.map((l) => ({ key: this.#nextKey++, name: l.name, text: "", quantity: quantityInput(l.quantity), unit: l.unit ?? "" }));
    this.#combobox.close();
  }

  override render() {
    return html`<main class="screen screen--form">
      ${this.beside
        ? nothing
        : html`<a class="back-link" href=${this.libraryHref}>
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M15 18l-6-6 6-6"></path></svg>Plats
          </a>`}
      ${this.#title()} ${this.#body()}
    </main>`;
  }

  #title() {
    if (this.dishId === undefined) {
      return html`<h1 class="screen-title screen-title--form">Nouveau plat</h1>`;
    }
    const dish = this.#dish;
    if (dish === undefined) {
      return html`<h1 class="screen-title screen-title--form">Plat</h1>`;
    }
    return html`<div class="dish-title ${colorClass(dish.color)}">
      <span class="dish-initial" aria-hidden="true">${initialOf(dish.name)}</span>
      <h1 class="screen-title screen-title--form">${dish.name}</h1>
    </div>`;
  }

  #body() {
    switch (this.load) {
      case "loading":
        return nothing;
      case "not-found":
        return html`<p class="screen-text">Ce plat est introuvable.</p>`;
      case "failed":
        return html`${alertMessage("dish-error", networkErrorMessage)}<button class="button-secondary" type="button" @click=${this.#loadLibrary}>Réessayer</button>`;
      case "ready":
        return this.#form();
    }
  }

  #form() {
    const nameError = this.#errorOf(undefined);
    return html`${this.conflict ? conflictBanner() : nothing}
      ${this.error === "network" ? alertMessage("dish-error", networkNeededMessage) : nothing}
      <form class="dish-form" novalidate @submit=${this.#submit}>
        <label class="field">Nom du plat
          <input
            class="field-input"
            type="text"
            name="name"
            maxlength="60"
            autocomplete="off"
            .value=${live(this.name)}
            aria-invalid=${nameError === undefined ? "false" : "true"}
            aria-describedby=${nameError === undefined ? nothing : "name-error"}
            @input=${(e: Event) => (this.name = (e.target as HTMLInputElement).value)}
          >
        </label>
        ${nameError === undefined ? nothing : alertMessage("name-error", nameError)}
        <div class="servings-box">
          <span>Pour combien de parts ?</span>
          ${this.servings === undefined
            ? nothing
            : html`<mt-stepper
                name="servings"
                label="Pour combien de parts ?"
                less-label="Une part de moins"
                more-label="Une part de plus"
                min="1"
                max="50"
                .value=${this.servings}
                @change=${this.#servingsChanged}
              ></mt-stepper>`}
        </div>
        <section class="dish-lines" aria-labelledby="lines-title">
          <h2 class="section-title" id="lines-title">Ingrédients</h2>
          ${this.lines.length === 0 ? nothing : html`<ul class="lines">${this.lines.map((l, i) => this.#line(l, i))}</ul>`}
          <button class="button-add" type="button" ?disabled=${this.lines.length >= maxLines} @click=${this.#addLine}>
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" aria-hidden="true"><path d="M12 5v14M5 12h14"></path></svg>Ajouter un ingrédient
          </button>
        </section>
        <div class="save-bar"><button class="button-primary" type="submit">Enregistrer le plat</button></div>
      </form>`;
  }

  #line(line: FormLine, index: number) {
    const draft = line.name === "";
    const error = this.#errorOf(line.key);
    const errorId = `line-error-${line.key}`;
    const of = draft ? "" : ` de ${line.name}`;
    const hasQuantity = trimSpaces(line.quantity) !== "";
    // Without quantity, the default unit of the ingredient chosen is announced, not chosen (D4).
    const hint = hasQuantity || draft ? "" : unitLabel(this.#known(line)?.defaultUnit ?? "", "");
    // A draft holding something can be removed, like a line chosen (recette du plan bibliotheque).
    const removable = !draft || line.text !== "" || hasQuantity;
    const removeLabel = draft ? "Retirer l’ingrédient en cours de saisie" : `Retirer l’ingrédient ${line.name}`;
    return html`<li class="line ${draft ? "line--draft" : ""}">
      <div class="line-row ${removable ? "" : "line-row--fixed"}">
        <input
          class="line-quantity"
          type="text"
          inputmode="decimal"
          autocomplete="off"
          maxlength=${maxQuantityLength}
          placeholder="Qté"
          aria-label=${`Quantité${of}`}
          aria-invalid=${error === undefined ? "false" : "true"}
          aria-describedby=${error === undefined ? nothing : errorId}
          .value=${live(line.quantity)}
          @input=${(e: Event) => this.#setQuantity(line, (e.target as HTMLInputElement).value)}
        >
        <span class="line-unit ${hasQuantity ? "" : "line-unit--empty"}">
          <span class="line-unit-label ${hasQuantity ? "" : "line-unit-label--hint"}" aria-hidden="true">${hasQuantity ? unitLabel(line.unit, line.quantity) : hint}</span>
          <select aria-label=${`Unité${of}`} ?disabled=${!hasQuantity} @change=${(e: Event) => this.#update(line, { unit: (e.target as HTMLSelectElement).value })}>
            <option value="" ?selected=${line.unit === ""}></option>
            ${units.map((u) => html`<option value=${u.code} aria-label=${u.name} ?selected=${line.unit === u.code}>${u.label}</option>`)}
          </select>
        </span>
        ${draft ? this.#combobox.input(line.text) : html`<span class="line-name">${line.name}</span>`}
        ${removable
          ? html`<button class="line-remove" type="button" aria-label=${removeLabel} @click=${() => this.#removeLine(index)}>
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" aria-hidden="true"><path d="M6 6l12 12M18 6L6 18"></path></svg>
            </button>`
          : nothing}
      </div>
      ${draft ? this.#combobox.list(line.text) : nothing}
      ${error === undefined ? nothing : alertMessage(errorId, error)}
    </li>`;
  }

  /** The options of the draft: the ingredients suggested, then « Créer « … » ». */
  #options(text: string): Option[] {
    const inDish = this.lines.filter((l) => l.name !== "").map((l) => l.name);
    const { ingredients, create } = suggestIngredients(this.library.catalog.items ?? [], text, inDish);
    const options: Option[] = ingredients.map((ingredient) => ({ ingredient }));
    if (create) {
      options.push({ create: trimSpaces(text) });
    }
    return options;
  }

  #updateDraft(change: Partial<FormLine>) {
    const draft = this.lines.find((l) => l.name === "");
    if (draft !== undefined) {
      this.#update(draft, change);
    }
  }

  /** An ingredient chosen: the draft takes it, and its default unit if a quantity is there. */
  #choose(option: Option) {
    const line = this.lines.find((l) => l.name === "");
    if (line === undefined) {
      return;
    }
    const name = option.ingredient?.name ?? option.create ?? "";
    const unit = line.unit === "" && trimSpaces(line.quantity) !== "" ? (option.ingredient?.defaultUnit ?? "") : line.unit;
    this.#update(line, { name, text: "", unit });
  }

  /** A quantity typed takes the default unit of the ingredient; without quantity, no unit. */
  #setQuantity(line: FormLine, quantity: string) {
    if (trimSpaces(quantity) === "") {
      this.#update(line, { quantity, unit: "" });
      return;
    }
    const unit = line.unit === "" && line.name !== "" ? (this.#known(line)?.defaultUnit ?? "") : line.unit;
    this.#update(line, { quantity, unit });
  }

  /** The ingredient of the referential that a line names, chosen or typed. */
  #known(line: FormLine): Ingredient | undefined {
    const name = line.name || line.text;
    if (!hasVisible(name)) {
      return undefined;
    }
    return this.library.catalog.items?.find((i) => normalizeName(i.name) === normalizeName(name));
  }

  #update(line: FormLine, change: Partial<FormLine>) {
    this.lines = this.lines.map((l) => (l.key === line.key ? { ...l, ...change } : l));
  }

  #addLine() {
    if (this.lines.some((l) => l.name === "")) {
      this.querySelector<HTMLInputElement>(".line-combobox")?.focus();
      return;
    }
    // The new line takes the focus once rendered, and only it.
    this.#focusDraft = true;
    this.lines = [...this.lines, { key: this.#nextKey++, name: "", text: "", quantity: "", unit: "" }];
  }

  #removeLine(index: number) {
    this.lines = this.lines.filter((_, i) => i !== index);
    // An error on a line points at it by position: once the lines shift, it would land elsewhere.
    if (this.error !== undefined && this.error !== "network" && this.error.line !== undefined) {
      this.error = undefined;
    }
  }

  #servingsChanged(e: Event) {
    this.servings = (e.target as HTMLInputElement & { value: number }).value;
    this.#servingsGuessed = false;
  }

  /** The message of the error shown on the name (key undefined) or on a line. */
  #errorOf(key: number | undefined): string | undefined {
    const error = this.error;
    if (error === undefined || error === "network") {
      return undefined;
    }
    const sent = this.#sentLines();
    const onLine = error.line === undefined ? undefined : sent[error.line]?.key;
    return onLine === key ? dishErrorMessage(error) : undefined;
  }

  /**
   * The lines sent: those chosen, and a draft taken as typed; a draft without visible
   * character is dropped as an empty one (D30), the server refusing it.
   */
  #sentLines(): (FormLine & { sentName: string })[] {
    return this.lines
      .map((l) => ({ ...l, sentName: l.name === "" ? l.text : l.name }))
      .filter((l) => hasVisible(l.sentName));
  }

  async #submit(e: Event) {
    e.preventDefault();
    if (this.#busy) {
      return;
    }
    const sent = this.#sentLines();
    const lines = sent.map((l) => ({ name: l.sentName, quantity: l.quantity, unit: trimSpaces(l.quantity) === "" ? "" : l.unit }));
    // The banner of a crossed change goes at the next save (docs/design/README.md).
    this.conflict = false;
    const refused = checkDish({ name: this.name, lines });
    if (refused !== undefined) {
      this.error = refused;
      return;
    }
    this.#busy = true;
    try {
      const existing = this.#dish === undefined ? undefined : { id: this.#dish.id, revision: this.#dish.revision };
      const result = await this.library.save(existing, {
        name: this.name,
        ...(this.servings === undefined || this.#servingsGuessed ? {} : { servings: this.servings }),
        ingredients: lines.map((l) => (trimSpaces(l.quantity) === "" ? { name: l.name } : l)),
      });
      switch (result.kind) {
        case "saved":
          this.error = undefined;
          this.dispatchEvent(new Event("saved", { bubbles: true, composed: true }));
          return;
        case "changed":
          this.error = undefined;
          this.conflict = true;
          this.#fill(result.dish);
          // The window, or the pane of the dish in etendu.
          window.scrollTo({ top: 0 });
          this.scrollTop = 0;
          return;
        case "refused":
          this.error = dishErrorMessage(result.error) === undefined ? "network" : result.error;
          return;
        default:
          // A 401 has already brought back to the login.
          this.error = "network";
      }
    } finally {
      this.#busy = false;
    }
  }
}

/** The banner of a crossed change refused (Doux-PlatConflit, Q25, D5). */
function conflictBanner() {
  return warningBanner("Ce plat a été modifié entre-temps. Voici sa version à jour : refaites votre modification.");
}

customElements.define("mt-dish-screen", DishScreen);

declare global {
  interface HTMLElementTagNameMap {
    "mt-dish-screen": DishScreen;
  }
}
