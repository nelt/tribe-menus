// The combobox of an ingredient on the catalog in memory (P4, Q13), shared by the screen of a
// dish (a line being typed) and the form of an item added by hand (plan courses, D18): an
// ARIA combobox, its list of suggestions and « Créer « … » », chosen with the pointer or the
// keyboard. A Lit controller: the host places the field and the list where its layout wants
// them, and gives the suggestions of the text typed.
import { type ReactiveController, type ReactiveControllerHost, html, nothing } from "lit";
import { live } from "lit/directives/live.js";
import { type Ingredient, highlight, unitLabel } from "./dishes.ts";

/** An option of the list: an ingredient of the catalog, or the creation of the text typed. */
export interface Option {
  ingredient?: Ingredient;
  create?: string;
}

export interface ComboboxOptions {
  /** The identifier of the list, for aria-controls; its options are <id>-<n>. */
  id: string;
  /** The class of the field, by which the closing after a blur recognizes it. */
  inputClass: string;
  label: string;
  placeholder?: string;
  /** The options of a text typed. */
  options: (text: string) => Option[];
  onType: (text: string) => void;
  onChoose: (option: Option) => void;
}

export class IngredientCombobox implements ReactiveController {
  /** The suggestions are shown; active, the option chosen with the keyboard. */
  open = false;
  active = -1;
  readonly #host: ReactiveControllerHost & HTMLElement;
  readonly #options: ComboboxOptions;
  /** A pointer is pressed: the suggestions close only once it is released (see #blur). */
  #pointerDown = false;
  #afterPointerUp: (() => void) | undefined;
  readonly #onPointerDown = () => (this.#pointerDown = true);
  readonly #onPointerUp = () => {
    this.#pointerDown = false;
    this.#afterPointerUp?.();
    this.#afterPointerUp = undefined;
  };

  constructor(host: ReactiveControllerHost & HTMLElement, options: ComboboxOptions) {
    this.#host = host;
    this.#options = options;
    host.addController(this);
  }

  hostConnected() {
    document.addEventListener("pointerdown", this.#onPointerDown, true);
    document.addEventListener("pointerup", this.#onPointerUp, true);
    document.addEventListener("pointercancel", this.#onPointerUp, true);
  }

  hostDisconnected() {
    document.removeEventListener("pointerdown", this.#onPointerDown, true);
    document.removeEventListener("pointerup", this.#onPointerUp, true);
    document.removeEventListener("pointercancel", this.#onPointerUp, true);
  }

  /** Closes the suggestions. */
  close() {
    this.#set(false, -1);
  }

  #set(open: boolean, active: number) {
    this.open = open;
    this.active = active;
    this.#host.requestUpdate();
  }

  /** The field, showing the text typed. */
  input(text: string, extra: { invalid?: boolean; describedBy?: string } = {}) {
    const id = this.#options.id;
    const expanded = this.open && this.#options.options(text).length > 0;
    return html`<input
      class=${this.#options.inputClass}
      type="text"
      role="combobox"
      autocomplete="off"
      maxlength="40"
      placeholder=${this.#options.placeholder ?? nothing}
      aria-label=${this.#options.label}
      aria-autocomplete="list"
      aria-expanded=${expanded ? "true" : "false"}
      aria-controls=${id}
      aria-activedescendant=${expanded && this.active >= 0 ? `${id}-${this.active}` : nothing}
      aria-invalid=${extra.invalid === true ? "true" : nothing}
      aria-describedby=${extra.describedBy ?? nothing}
      .value=${live(text)}
      @input=${(e: Event) => this.#type((e.target as HTMLInputElement).value)}
      @keydown=${(e: KeyboardEvent) => this.#key(text, e)}
      @blur=${this.#blur}
      @focus=${() => this.#set(true, this.active)}
    >`;
  }

  /** The list of the suggestions of the text typed; it stays in the DOM, empty, for aria-controls. */
  list(text: string) {
    const id = this.#options.id;
    const options = this.open ? this.#options.options(text) : [];
    return html`<ul class="suggestions" id=${id} role="listbox" aria-label="Ingrédients proposés" ?hidden=${options.length === 0} @mousedown=${(e: Event) => e.preventDefault()}>
      ${options.map((o, i) => {
        const selected = i === this.active ? "true" : "false";
        if (o.ingredient === undefined) {
          return html`<li class="suggestion suggestion--create" role="option" id=${`${id}-${i}`} aria-selected=${selected} @click=${() => this.#choose(o)}>
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" aria-hidden="true"><path d="M12 5v14M5 12h14"></path></svg>Créer « ${o.create} »
          </li>`;
        }
        const unit = o.ingredient.defaultUnit === undefined ? "" : unitLabel(o.ingredient.defaultUnit, "");
        return html`<li class="suggestion" role="option" id=${`${id}-${i}`} aria-selected=${selected} @click=${() => this.#choose(o)}>
          <span class="suggestion-name">${highlight(o.ingredient.name, text).map((s) => (s.found ? html`<strong>${s.text}</strong>` : s.text))}</span>
          ${unit === "" ? nothing : html`<span class="suggestion-unit">${unit}</span>`}
        </li>`;
      })}
    </ul>`;
  }

  /**
   * The suggestions close when the combobox loses the focus, but after the gesture that took
   * it: closed at once, the list would move what is below it, and the click would miss the
   * button pressed. With a mouse, the pointer is released after the blur; with a finger,
   * the blur, the release and the click come in the same task.
   */
  readonly #blur = () => {
    const close = () =>
      setTimeout(() => {
        const active = document.activeElement;
        if (!this.#host.contains(active) || !active?.classList.contains(this.#options.inputClass)) {
          this.close();
        }
      }, 0);
    if (this.#pointerDown) {
      this.#afterPointerUp = close;
    } else {
      close();
    }
  };

  #type(text: string) {
    this.#options.onType(text);
    this.#set(true, -1);
  }

  #key(text: string, e: KeyboardEvent) {
    const options = this.#options.options(text);
    switch (e.key) {
      case "ArrowDown":
        e.preventDefault();
        this.#set(true, Math.min(this.active + 1, options.length - 1));
        break;
      case "ArrowUp":
        e.preventDefault();
        this.#set(this.open, Math.max(this.active - 1, 0));
        break;
      case "Enter": {
        // Enter chooses an option, and never sends the form from the combobox.
        e.preventDefault();
        const option = this.open ? options[this.active] : undefined;
        if (option !== undefined) {
          this.#choose(option);
        }
        break;
      }
      case "Escape":
        this.close();
        break;
    }
  }

  #choose(option: Option) {
    this.close();
    this.#options.onChoose(option);
  }
}
