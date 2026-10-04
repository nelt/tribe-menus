// Login screens (ENF-01): email, then code. Forms in the classic DOM (ADR 0004, point 3).
import { LitElement, type PropertyValues, html, nothing } from "lit";
import type { Api, Session } from "./api.ts";
import {
  type LoginState,
  afterCodeRequest,
  afterResend,
  afterSessionFailure,
  changeEmail,
  emailEdited,
  errorMessage,
  initialLoginState,
  isInstalledApp,
  normalizeCode,
  pasteIntoCode,
} from "./login.ts";
import { alertMessage, footer } from "./templates.ts";

const errorId = "login-error";

/** Fired with the session once it is open. */
export class SignedInEvent extends Event {
  readonly session: Session;

  constructor(session: Session) {
    super("signed-in", { bubbles: true, composed: true });
    this.session = session;
  }
}

export class LoginScreen extends LitElement {
  static override properties = {
    api: { attribute: false },
    state: { state: true },
    code: { state: true },
    busy: { state: true },
  };

  declare api: Api;
  state: LoginState = initialLoginState;
  code = "";
  /** A request is in progress: a second submission is ignored. */
  busy = false;

  protected override createRenderRoot() {
    return this;
  }

  protected override updated(changed: PropertyValues<this>) {
    const previous = changed.get("state") as LoginState | undefined;
    if (previous !== undefined && previous.step !== this.state.step) {
      this.querySelector<HTMLInputElement>(this.state.step === "code" ? ".code-input" : ".field-input")?.focus();
    }
  }

  override render() {
    const step = this.state.step;
    return html`<div class="login ${step === "code" ? "login--code" : ""}">
      <header class="login-brand">
        <img class="login-symbol" src="symbole.svg" alt="" width="320" height="320">
        <h1 class="login-logotype"><img src="logotype.svg" alt="Melting Tribe" width="411" height="100"></h1>
        <p class="login-tagline">Les menus de la semaine, en tribu</p>
      </header>
      <main class="login-main">${step === "email" ? this.#emailForm() : this.#codeForm()}</main>
      ${footer()}
    </div>`;
  }

  #emailForm() {
    const state = this.state;
    if (state.step !== "email") {
      return nothing;
    }
    const error = state.error;
    return html`<form class="login-form" novalidate @submit=${this.#requestCode}>
      <h2 class="login-title">Bienvenue !</h2>
      <label class="field">
        Votre adresse e-mail
        <input
          class="field-input"
          type="email"
          name="email"
          inputmode="email"
          autocomplete="email"
          .value=${state.email}
          aria-invalid=${error === "invalid-email" ? "true" : "false"}
          aria-describedby=${error === undefined ? nothing : errorId}
          @input=${(e: InputEvent) => (this.state = emailEdited((e.target as HTMLInputElement).value))}
        >
      </label>
      ${error === undefined ? nothing : alertMessage(errorId, errorMessage(error))}
      <button class="button-primary" type="submit" ?disabled=${state.blocked}>Recevoir un code</button>
      <p class="login-hint">Pas de mot de passe : un code à 6 chiffres vous est envoyé par e-mail.</p>
    </form>`;
  }

  #codeForm() {
    const state = this.state;
    if (state.step !== "code") {
      return nothing;
    }
    const error = state.error;
    const cells = [0, 1, 2, 3, 4, 5].map(
      (i) => html`<span class="code-cell ${i < this.code.length ? "code-cell--filled" : ""}"></span>`,
    );
    const fieldClass = state.dead ? "code-field--dead" : error === "incorrect-code" ? "code-field--error" : "";
    return html`<form class="login-form" novalidate @submit=${this.#openSession}>
      <div class="login-intro">
        <h2 class="login-title">Vérifiez vos e-mails</h2>
        <p class="login-text">
          Si <strong>${state.email}</strong> fait partie de la tribu, un code à 6 chiffres vient d’y être envoyé.
          <button class="link-button" type="button" @click=${() => (this.state = changeEmail(this.state))}>Modifier</button>
        </p>
      </div>
      <label class="code-field ${fieldClass}">
        <span class="visually-hidden">Code de connexion</span>
        <span class="code-cells" aria-hidden="true">${cells}</span>
        <input
          class="code-input"
          type="text"
          name="code"
          inputmode="numeric"
          autocomplete="one-time-code"
          maxlength="6"
          .value=${this.code}
          ?disabled=${state.dead}
          aria-invalid=${error === "incorrect-code" ? "true" : "false"}
          aria-describedby=${error === undefined ? nothing : errorId}
          @input=${this.#codeInput}
          @paste=${this.#codePaste}
        >
      </label>
      ${error === undefined ? nothing : alertMessage(errorId, errorMessage(error, state.attemptsLeft))}
      ${state.dead
        ? html`<button class="button-primary" type="button" @click=${this.#resend}>Recevoir un nouveau code</button>`
        : html`<button class="button-primary" type="submit">Se connecter</button>
            <button class="button-text" type="button" @click=${this.#resend}>Je n’ai rien reçu : renvoyer un code</button>`}
    </form>`;
  }

  #codeInput(e: InputEvent) {
    const input = e.target as HTMLInputElement;
    const code = normalizeCode(input.value);
    // The field shows the digits only, even when other characters were pasted.
    if (input.value !== code) {
      input.value = code;
    }
    this.code = code;
  }

  // maxlength would cut a pasted "123 456" before the digits are kept: the digits of the
  // paste take the place of the selection, as a native paste would.
  #codePaste(e: ClipboardEvent) {
    const text = e.clipboardData?.getData("text");
    if (text === undefined) {
      return;
    }
    e.preventDefault();
    const input = e.target as HTMLInputElement;
    const pasted = pasteIntoCode(input.value, input.selectionStart ?? input.value.length, input.selectionEnd ?? input.value.length, text);
    if (pasted === undefined) {
      return;
    }
    input.value = pasted.code;
    input.setSelectionRange(pasted.caret, pasted.caret);
    this.code = pasted.code;
  }

  async #requestCode(e: SubmitEvent) {
    e.preventDefault();
    if (this.busy || this.state.step !== "email" || this.state.blocked) {
      return;
    }
    // The field stays editable during the request: the next screen follows the address
    // requested, to which the code was sent, not the one typed meanwhile.
    const requested = this.state;
    this.busy = true;
    try {
      const outcome = await this.api.requestCode(requested.email);
      this.code = "";
      this.state = afterCodeRequest(requested, outcome);
    } finally {
      this.busy = false;
    }
  }

  async #resend() {
    if (this.busy) {
      return;
    }
    this.busy = true;
    try {
      const outcome = await this.api.requestCode(this.state.email);
      if (outcome.kind === "ok") {
        this.code = "";
      }
      this.state = afterResend(this.state, outcome);
      if (outcome.kind === "ok") {
        // The field, enabled again after a dead code, waits for the new one.
        await this.updateComplete;
        this.querySelector<HTMLInputElement>(".code-input")?.focus();
      }
    } finally {
      this.busy = false;
    }
  }

  async #openSession(e: SubmitEvent) {
    e.preventDefault();
    if (this.busy || this.state.step !== "code" || this.state.dead) {
      return;
    }
    // An incomplete code would cost an attempt: the field waits for its six digits.
    if (this.code.length !== 6) {
      this.querySelector<HTMLInputElement>(".code-input")?.focus();
      return;
    }
    this.busy = true;
    try {
      const outcome = await this.api.openSession(this.state.email, this.code, isInstalledApp(window));
      if (outcome.kind === "ok") {
        this.dispatchEvent(new SignedInEvent(outcome.value));
        return;
      }
      this.state = afterSessionFailure(this.state, outcome);
    } finally {
      this.busy = false;
    }
  }
}

customElements.define("mt-login", LoginScreen);

declare global {
  interface HTMLElementTagNameMap {
    "mt-login": LoginScreen;
  }
}
