// Stepper − / value / + (docs/design/README.md, « Stepper de parts »): the servings of a dish
// and the size of the tribe. A field of its own, so in a shadow DOM and declared as a form
// field by ElementInternals (ADR 0004, point 3): its value goes with the form that holds it.
import { LitElement, css, html } from "lit";

export class Stepper extends LitElement {
  static formAssociated = true;

  static override properties = {
    value: { type: Number },
    min: { type: Number },
    max: { type: Number },
    name: {},
    label: {},
    lessLabel: { attribute: "less-label" },
    moreLabel: { attribute: "more-label" },
    disabled: { type: Boolean, reflect: true },
  };

  static override styles = css`
    :host {
      display: inline-flex;
      flex-shrink: 0;
    }

    .stepper {
      display: flex;
      align-items: center;
      gap: 2px;
      padding: 2px;
      border: var(--border-thin) solid var(--color-text);
      border-radius: 14px;
    }

    button {
      width: 40px;
      height: 42px;
      border: none;
      border-radius: 11px;
      background: transparent;
      color: var(--color-text);
      font: inherit;
      font-size: 22px;
      font-weight: 700;
      cursor: pointer;
    }

    button.more {
      color: var(--color-action);
      font-weight: 800;
    }

    button:disabled {
      color: var(--color-dashed);
      cursor: default;
    }

    button:focus-visible {
      outline: var(--border-strong) solid var(--color-text);
      outline-offset: -2px;
    }

    output {
      min-width: 24px;
      font-family: var(--font-heading);
      font-size: 20px;
      font-weight: 800;
      text-align: center;
    }
  `;

  declare value: number;
  declare min: number;
  declare max: number;
  declare name: string;
  /** The name of the group, read by screen readers. */
  declare label: string;
  declare lessLabel: string;
  declare moreLabel: string;
  declare disabled: boolean;
  readonly #internals: ElementInternals;
  #initial: number | undefined;

  constructor() {
    super();
    this.value = 1;
    this.min = 1;
    this.max = 50;
    this.name = "";
    this.label = "";
    this.lessLabel = "Moins";
    this.moreLabel = "Plus";
    this.disabled = false;
    this.#internals = this.attachInternals();
  }

  override willUpdate() {
    this.#initial ??= this.value;
    this.#internals.setFormValue(String(this.value));
  }

  override render() {
    return html`<div class="stepper" role="group" aria-label=${this.label}>
      <button type="button" aria-label=${this.lessLabel} ?disabled=${this.disabled || this.value <= this.min} @click=${() => this.#step(-1)}>−</button>
      <output aria-live="polite">${this.value}</output>
      <button class="more" type="button" aria-label=${this.moreLabel} ?disabled=${this.disabled || this.value >= this.max} @click=${() => this.#step(1)}>+</button>
    </div>`;
  }

  #step(delta: number) {
    const value = Math.min(this.max, Math.max(this.min, this.value + delta));
    if (value === this.value) {
      return;
    }
    this.value = value;
    this.dispatchEvent(new Event("change", { bubbles: true }));
  }

  formResetCallback() {
    this.value = this.#initial ?? this.min;
  }

  formDisabledCallback(disabled: boolean) {
    this.disabled = disabled;
  }
}

customElements.define("mt-stepper", Stepper);

declare global {
  interface HTMLElementTagNameMap {
    "mt-stepper": Stepper;
  }
}
