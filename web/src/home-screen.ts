// Provisional home once signed in: the name of the tribe and the sign-out, until the
// planning comes.
import { LitElement, html, nothing } from "lit";
import type { Api, Session } from "./api.ts";
import { networkErrorMessage } from "./login.ts";
import { alertMessage, footer } from "./templates.ts";
import { symbolURL } from "./identity.ts";

export class HomeScreen extends LitElement {
  static override properties = {
    api: { attribute: false },
    session: { attribute: false },
    failed: { state: true },
  };

  declare api: Api;
  declare session: Session;
  /** The sign-out failed for lack of network. */
  failed = false;
  #busy = false;

  protected override createRenderRoot() {
    return this;
  }

  override render() {
    return html`<div class="home">
      <main class="home-main">
        <img class="home-symbol" src=${symbolURL} alt="" width="320" height="320">
        <h1 class="home-title">${this.session.tribe.name}</h1>
        <button class="button-danger" type="button" @click=${this.#signOut}>Se déconnecter</button>
        ${this.failed ? alertMessage("home-error", networkErrorMessage) : nothing}
      </main>
      ${footer()}
    </div>`;
  }

  async #signOut() {
    if (this.#busy) {
      return;
    }
    this.#busy = true;
    try {
      const outcome = await this.api.deleteSession();
      // A 401 has already brought back to the login.
      if (outcome.kind === "ok") {
        this.dispatchEvent(new Event("signed-out", { bubbles: true, composed: true }));
      }
      this.failed = outcome.kind === "network-error";
    } finally {
      this.#busy = false;
    }
  }
}

customElements.define("mt-home", HomeScreen);

declare global {
  interface HTMLElementTagNameMap {
    "mt-home": HomeScreen;
  }
}
