// Loading at startup (docs/design maquette Chargement): the symbol breathes and drops of
// soup fall; on a slow or absent network, a message and "Réessayer". No tribe name.
import { LitElement, html } from "lit";
import { symbolURL } from "./identity.ts";

const drop = (n: number) =>
  html`<svg class="drop drop-${n}" width="12" height="16" viewBox="0 0 12 16" aria-hidden="true">
    <path d="M6 0 C6 0 12 8 12 11 A6 6 0 0 1 0 11 C0 8 6 0 6 0 Z" fill="#F2C94C"></path>
  </svg>`;

export class LoadingScreen extends LitElement {
  static override properties = {
    slow: { type: Boolean },
  };

  /** The network is slow or absent: the message and "Réessayer" replace the drops. */
  slow = false;

  protected override createRenderRoot() {
    return this;
  }

  override render() {
    return html`<div class="loading">
      <img class="loading-symbol" src=${symbolURL} alt="" width="320" height="320">
      ${this.slow
        ? html`<div class="loading-slow" role="status">
            <p class="loading-message">Le réseau est lent ou absent.</p>
            <button class="button-secondary" type="button" @click=${() => this.dispatchEvent(new Event("retry", { bubbles: true, composed: true }))}>Réessayer</button>
          </div>`
        : html`<div class="loading-drops" role="status" aria-label="Chargement">${[1, 2, 3].map(drop)}</div>`}
    </div>`;
  }
}

customElements.define("mt-loading", LoadingScreen);

declare global {
  interface HTMLElementTagNameMap {
    "mt-loading": LoadingScreen;
  }
}
