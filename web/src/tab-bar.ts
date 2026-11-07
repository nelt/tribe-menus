// Tab bar of the application (plan bibliotheque, D11): Planning, Plats and Courses, which opens
// the main shopping list (plan courses, C1; docs/design/README.md).
import { LitElement, html } from "lit";

/** The active tab. */
export type Tab = "planning" | "dishes" | "shopping";

const icons = {
  planning: html`<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="5" width="18" height="16" rx="3"></rect><path d="M8 3v4M16 3v4M3 10h18"></path></svg>`,
  dishes: html`<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M4 5a2 2 0 0 1 2-2h13v16H6a2 2 0 0 0-2 2z"></path><path d="M4 21V5"></path><path d="M9 8h6"></path></svg>`,
  shopping: html`<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M3 4h2l2.4 11.2a2 2 0 0 0 2 1.6h7.7a2 2 0 0 0 2-1.5L21 8H6"></path><circle cx="10" cy="20" r="1.3"></circle><circle cx="17" cy="20" r="1.3"></circle></svg>`,
};

export class TabBar extends LitElement {
  static override properties = {
    active: {},
    planningHref: { attribute: false },
    dishesHref: { attribute: false },
    shoppingHref: { attribute: false },
  };

  declare active: Tab;
  declare planningHref: string;
  declare dishesHref: string;
  declare shoppingHref: string;

  protected override createRenderRoot() {
    return this;
  }

  override render() {
    const current = (tab: Tab) => (this.active === tab ? "page" : undefined);
    return html`<nav class="tab-bar" aria-label="Onglets">
      <a class="tab" href=${this.planningHref} aria-current=${current("planning") ?? "false"}>${icons.planning}Planning</a>
      <a class="tab" href=${this.dishesHref} aria-current=${current("dishes") ?? "false"}>${icons.dishes}Plats</a>
      <a class="tab" href=${this.shoppingHref} aria-current=${current("shopping") ?? "false"}>${icons.shopping}Courses</a>
    </nav>`;
  }
}

customElements.define("mt-tab-bar", TabBar);

declare global {
  interface HTMLElementTagNameMap {
    "mt-tab-bar": TabBar;
  }
}
