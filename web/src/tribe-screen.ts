// Screen Tribu, after the mock-up Doux-Tribu, reduced to the name of the tribe, its number of
// active members and the setting of its size (T3, plan bibliotheque, D3), with the sign-out
// at the bottom (plan planning, D5) and, under it, the link to the source code that the
// provisional home carried (ADR 0011, point 2); opened by the chip of the planning. The members and the
// sessions come with their own plan. Without network, a change of the size fails and changes
// nothing (D18).
import { LitElement, html, nothing } from "lit";
import type { Api, TribeInfo } from "./api.ts";
import { networkErrorMessage } from "./login.ts";
import { alertMessage, footer, networkNeededMessage } from "./templates.ts";

/** The number of active members, under the name of the tribe. */
export function activeMembersLabel(n: number): string {
  return n > 1 ? `${n} membres actifs` : `${n} membre actif`;
}

export class TribeScreen extends LitElement {
  static override properties = {
    api: { attribute: false },
    planningHref: { attribute: false },
    tribe: { state: true },
    size: { state: true },
    failed: { state: true },
    signOutFailed: { state: true },
  };

  declare api: Api;
  declare planningHref: string;
  /** The tribe as the API last gave it; undefined until read. */
  tribe: TribeInfo | undefined = undefined;
  /** The size shown by the stepper, possibly not saved yet. */
  size = 0;
  /** Why the last call failed: the tribe could not be read, or the size not saved. */
  failed: "read" | "save" | undefined = undefined;
  /** The sign-out failed for lack of network. */
  signOutFailed = false;
  #saving = false;
  #signingOut = false;

  protected override createRenderRoot() {
    return this;
  }

  override connectedCallback() {
    super.connectedCallback();
    void this.#read();
  }

  async #read() {
    this.failed = undefined;
    const outcome = await this.api.getTribe();
    // A 401 has already brought back to the login.
    if (outcome.kind === "ok") {
      this.tribe = outcome.value;
      this.size = outcome.value.size;
    } else if (outcome.kind === "network-error") {
      this.failed = "read";
    }
  }

  override render() {
    return html`<main class="screen screen--sub">
      <a class="back-link" href=${this.planningHref}>
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M15 18l-6-6 6-6"></path></svg>Planning
      </a>
      ${this.failed === "read" ? html`${alertMessage("tribe-error", networkErrorMessage)}<button class="button-secondary" type="button" @click=${this.#read}>Réessayer</button>` : nothing}
      ${this.tribe === undefined ? nothing : this.#tribe(this.tribe)}
      <div class="sign-out">
        <button class="button-danger" type="button" @click=${this.#signOut}>Se déconnecter</button>
        ${this.signOutFailed ? alertMessage("sign-out-error", networkErrorMessage) : nothing}
      </div>
      ${footer()}
    </main>`;
  }

  /** Signs out, without confirmation (D5): the application goes back to the login. */
  async #signOut() {
    if (this.#signingOut) {
      return;
    }
    this.#signingOut = true;
    try {
      const outcome = await this.api.deleteSession();
      // A 401 has already brought back to the login.
      if (outcome.kind === "ok") {
        this.dispatchEvent(new Event("signed-out", { bubbles: true, composed: true }));
      }
      this.signOutFailed = outcome.kind === "network-error";
    } finally {
      this.#signingOut = false;
    }
  }

  #tribe(tribe: TribeInfo) {
    return html`<div class="tribe-head">
        <span class="eyebrow">TRIBU</span>
        <h1 class="screen-title">${tribe.name}</h1>
        <span class="tribe-members">${activeMembersLabel(tribe.activeMembers)}</span>
      </div>
      <div class="setting setting--neutral">
        <div class="setting-text">
          <span class="setting-title">Taille de la tribu</span>
          <span class="setting-help">Parts proposées pour chaque plat ajouté à un repas et pour chaque nouveau plat.</span>
        </div>
        <mt-stepper
          name="size"
          label="Taille de la tribu"
          less-label="Une personne de moins"
          more-label="Une personne de plus"
          min="1"
          max="50"
          .value=${this.size}
          @change=${this.#sizeChanged}
        ></mt-stepper>
      </div>
      ${this.failed === "save" ? alertMessage("tribe-error", networkNeededMessage) : nothing}`;
  }

  /**
   * Saves the size shown. The changes made while a save runs are sent after it, the last
   * one only; a save that fails brings back the size saved.
   */
  async #sizeChanged(e: Event) {
    this.size = (e.target as HTMLElement & { value: number }).value;
    this.failed = undefined;
    if (this.#saving) {
      return;
    }
    this.#saving = true;
    try {
      while (this.tribe !== undefined && this.size !== this.tribe.size) {
        const outcome = await this.api.setTribeSize(this.size);
        if (outcome.kind !== "ok") {
          // A 401 has already brought back to the login.
          this.size = this.tribe.size;
          this.failed = outcome.kind === "network-error" ? "save" : undefined;
          return;
        }
        this.tribe = outcome.value;
      }
    } finally {
      this.#saving = false;
    }
  }
}

customElements.define("mt-tribe-screen", TribeScreen);

declare global {
  interface HTMLElementTagNameMap {
    "mt-tribe-screen": TribeScreen;
  }
}
