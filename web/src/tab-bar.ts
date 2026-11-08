// Navigation of the application (plan bibliotheque, D11; plan grands-ecrans, D4): Planning,
// Plats and Courses, which opens the main shopping list (plan courses, C1). The same
// navigation « Onglets », with the same links, in every format: a bar at the bottom in compact,
// a rail on the left in moyen and etendu, topped with the symbol and the chip of the tribe,
// which opens the screen Tribu (docs/design/README.md, « Formats d'écran »). The stylesheet
// places it after data-format (ADR 0030).
import { LitElement, html, nothing } from "lit";
import { type Api, type TribeInfo, tribeEvent } from "./api.ts";
import { reducedSymbolURL } from "./identity.ts";
import { type Format, currentFormat, onFormatChange } from "./layout.ts";
import { tribeChipLabel } from "./templates.ts";

/** The active tab. */
export type Tab = "planning" | "dishes" | "shopping";

const icons = {
  planning: html`<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="5" width="18" height="16" rx="3"></rect><path d="M8 3v4M16 3v4M3 10h18"></path></svg>`,
  dishes: html`<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M4 5a2 2 0 0 1 2-2h13v16H6a2 2 0 0 0-2 2z"></path><path d="M4 21V5"></path><path d="M9 8h6"></path></svg>`,
  shopping: html`<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M3 4h2l2.4 11.2a2 2 0 0 0 2 1.6h7.7a2 2 0 0 0 2-1.5L21 8H6"></path><circle cx="10" cy="20" r="1.3"></circle><circle cx="17" cy="20" r="1.3"></circle></svg>`,
  members: html`<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="9" cy="8" r="3.2"></circle><path d="M3 20a6 6 0 0 1 12 0"></path><circle cx="17" cy="9" r="2.6"></circle><path d="M16 14.2a5 5 0 0 1 5.5 5.8"></path></svg>`,
};

export class TabBar extends LitElement {
  static override properties = {
    active: {},
    sub: { type: Boolean, reflect: true },
    api: { attribute: false },
    planningHref: { attribute: false },
    dishesHref: { attribute: false },
    shoppingHref: { attribute: false },
    tribeHref: { attribute: false },
    tribe: { state: true },
  };

  /** The active tab; none on the screen Tribu. */
  declare active: Tab | undefined;
  /** A screen opened from a tab, with its link back: no bar in compact, as its mock-up. */
  declare sub: boolean;
  declare api: Api;
  declare planningHref: string;
  declare dishesHref: string;
  declare shoppingHref: string;
  declare tribeHref: string;
  /** The tribe, for the chip of the rail; undefined until read, or if it cannot be. */
  tribe: TribeInfo | undefined = undefined;
  readonly #onTribe = (e: Event) => {
    this.tribe = (e as CustomEvent<TribeInfo>).detail;
  };
  /** Stops following the format. */
  #unfollow: (() => void) | undefined;
  readonly #onVisibility = () => {
    if (document.visibilityState === "visible") {
      this.#readTribeForRail(currentFormat());
    }
  };

  protected override createRenderRoot() {
    return this;
  }

  override connectedCallback() {
    super.connectedCallback();
    document.addEventListener("visibilitychange", this.#onVisibility);
    this.api.addEventListener(tribeEvent, this.#onTribe);
    this.#unfollow = onFormatChange((format) => this.#readTribeForRail(format));
    this.#readTribeForRail(currentFormat());
  }

  override disconnectedCallback() {
    super.disconnectedCallback();
    document.removeEventListener("visibilitychange", this.#onVisibility);
    this.api.removeEventListener(tribeEvent, this.#onTribe);
    this.#unfollow?.();
  }

  /** Reads the tribe once the rail shows its chip: not in compact, where the planning has it. */
  #readTribeForRail(format: Format) {
    if (format !== "compact" && this.tribe === undefined) {
      // What is read comes back by tribeEvent.
      void this.api.getTribe();
    }
  }

  override render() {
    const current = (tab: Tab) => (this.active === tab ? "page" : "false");
    return html`<div class="rail-head">
        <img class="rail-symbol" src=${reducedSymbolURL} alt="" width="44" height="44">
        ${this.tribe === undefined ? nothing : this.#chip(this.tribe)}
      </div>
      <nav class="tab-bar" aria-label="Onglets">
        <a class="tab" href=${this.planningHref} aria-current=${current("planning")}>${icons.planning}Planning</a>
        <a class="tab" href=${this.dishesHref} aria-current=${current("dishes")}>${icons.dishes}Plats</a>
        <a class="tab" href=${this.shoppingHref} aria-current=${current("shopping")}>${icons.shopping}Courses</a>
      </nav>`;
  }

  /** The chip of the tribe in the rail: the number of active members, then the name. */
  #chip(tribe: TribeInfo) {
    return html`<a class="rail-tribe" href=${this.tribeHref} aria-label=${tribeChipLabel(tribe)}>
      <span class="rail-tribe-members">${icons.members}${tribe.activeMembers}</span>
      <span class="rail-tribe-name">${tribe.name}</span>
    </a>`;
  }
}

customElements.define("mt-tab-bar", TabBar);

declare global {
  interface HTMLElementTagNameMap {
    "mt-tab-bar": TabBar;
  }
}
