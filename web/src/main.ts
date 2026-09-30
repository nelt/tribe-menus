// Provisional skeleton, to be replaced by the router (ADR 0004).
import { LitElement, css, html } from "lit";
import { tribeIdFromPath } from "./route.ts";

export class App extends LitElement {
  static override styles = css`
    :host {
      display: block;
      padding: 1.5rem;
    }
    h1 {
      margin: 0 0 0.5rem;
      color: var(--color-action);
    }
    p {
      margin: 0;
      color: var(--color-text-secondary);
    }
  `;

  override render() {
    const tribeId = tribeIdFromPath(window.location.pathname);
    return html`
      <h1>Melting Tribe</h1>
      ${tribeId === undefined ? null : html`<p>tribu ${tribeId}</p>`}
    `;
  }
}

customElements.define("mt-app", App);

declare global {
  interface HTMLElementTagNameMap {
    "mt-app": App;
  }
}
