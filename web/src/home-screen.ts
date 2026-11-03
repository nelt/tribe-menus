// Provisional home once signed in, under the tab Planning (plan bibliotheque, D11): the chip
// of the tribe, which opens the screen Tribu, the name of the tribe and the sign-out, until
// the planning comes.
import { LitElement, html, nothing } from "lit";
import type { Api, Session, TribeInfo } from "./api.ts";
import { networkErrorMessage } from "./login.ts";
import { alertMessage, footer } from "./templates.ts";
import { symbolURL } from "./identity.ts";

export class HomeScreen extends LitElement {
  static override properties = {
    api: { attribute: false },
    session: { attribute: false },
    tribeHref: { attribute: false },
    tribe: { state: true },
    failed: { state: true },
  };

  declare api: Api;
  declare session: Session;
  declare tribeHref: string;
  /** The tribe, for its chip; undefined until read, or if it cannot be. */
  tribe: TribeInfo | undefined = undefined;
  /** The sign-out failed for lack of network. */
  failed = false;
  #busy = false;

  protected override createRenderRoot() {
    return this;
  }

  override connectedCallback() {
    super.connectedCallback();
    void this.#readTribe();
  }

  async #readTribe() {
    const outcome = await this.api.getTribe();
    if (outcome.kind === "ok") {
      this.tribe = outcome.value;
    }
  }

  override render() {
    return html`<div class="home">
      ${this.tribe === undefined ? nothing : tribeChip(this.tribe, this.tribeHref)}
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

/** The chip of the tribe: its name and its number of active members. */
function tribeChip(tribe: TribeInfo, href: string) {
  const members = tribe.activeMembers > 1 ? `${tribe.activeMembers} membres` : `${tribe.activeMembers} membre`;
  return html`<a class="tribe-chip" href=${href}><span class="tribe-chip-name">${tribe.name}</span><span class="tribe-chip-members">${members}</span></a>`;
}

customElements.define("mt-home", HomeScreen);

declare global {
  interface HTMLElementTagNameMap {
    "mt-home": HomeScreen;
  }
}
