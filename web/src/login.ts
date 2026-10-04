// States of the login screens (ENF-01, docs/design maquettes Connexion and Code), as pure
// functions of the outcomes of the API. The address stays in memory only.

import type { Failure, Outcome } from "./api.ts";

/** The errors the email screen shows. */
export type EmailError = "invalid-email" | "too-many-requests" | "network-error";

/** The errors the code screen shows. */
export type CodeError =
  | "incorrect-code"
  | "attempts-exhausted"
  | "code-expired"
  | "invalid-email"
  | "too-many-requests"
  | "network-error";

export type LoginState =
  | {
      step: "email";
      email: string;
      error?: EmailError;
      /** Too many requests: the button is inactive until the address is changed. */
      blocked: boolean;
    }
  | {
      step: "code";
      email: string;
      error?: CodeError;
      attemptsLeft?: number;
      /** The code can no longer succeed: the only action is to receive a new one. */
      dead: boolean;
    };

export const initialLoginState: LoginState = { step: "email", email: "", blocked: false };

/** The address is being typed: an error about the previous one no longer applies. */
export function emailEdited(email: string): LoginState {
  return { step: "email", email, blocked: false };
}

/** "Modifier", on the code screen: back to the email screen, address kept. */
export function changeEmail(state: LoginState): LoginState {
  return { step: "email", email: state.email, blocked: false };
}

/** Outcome of a code request from the email screen. */
export function afterCodeRequest(state: LoginState, outcome: Outcome<undefined>): LoginState {
  if (outcome.kind === "ok") {
    return { step: "code", email: state.email, dead: false };
  }
  const error = emailError(outcome);
  return { step: "email", email: state.email, error, blocked: error === "too-many-requests" };
}

/**
 * Outcome of a new code request from the code screen ("Je n'ai rien reçu" or "Recevoir un
 * nouveau code"). Refused, the code already received stays usable unless it was dead.
 */
export function afterResend(state: LoginState, outcome: Outcome<undefined>): LoginState {
  if (state.step !== "code") {
    return state;
  }
  if (outcome.kind === "ok") {
    return { step: "code", email: state.email, dead: false };
  }
  return { step: "code", email: state.email, error: codeError(outcome), dead: state.dead };
}

/** Outcome of a failed attempt to open a session. */
export function afterSessionFailure(state: LoginState, failure: Failure): LoginState {
  if (state.step !== "code") {
    return state;
  }
  switch (failure.kind) {
    case "incorrect-code":
      return { step: "code", email: state.email, error: "incorrect-code", attemptsLeft: failure.attemptsLeft, dead: false };
    case "attempts-exhausted":
    case "code-expired":
      return { step: "code", email: state.email, error: failure.kind, dead: true };
    default:
      return { step: "code", email: state.email, error: codeError(failure), dead: state.dead };
  }
}

function emailError(failure: Failure): EmailError {
  switch (failure.kind) {
    case "invalid-email":
    case "too-many-requests":
      return failure.kind;
    default:
      return "network-error";
  }
}

function codeError(failure: Failure): CodeError {
  switch (failure.kind) {
    case "invalid-email":
    case "too-many-requests":
    case "attempts-exhausted":
    case "code-expired":
      return failure.kind;
    case "incorrect-code":
      return "incorrect-code";
    default:
      return "network-error";
  }
}

export const networkErrorMessage = "Le réseau est lent ou absent. Réessayez.";

/** The message of an error, as the mockups word it. */
export function errorMessage(error: EmailError | CodeError, attemptsLeft = 0): string {
  switch (error) {
    case "invalid-email":
      return "Cette adresse ne semble pas complète. Vérifiez-la.";
    case "too-many-requests":
      return "Trop de demandes depuis cet appareil. Réessayez dans quelques minutes.";
    case "incorrect-code":
      return attemptsLeft === 1
        ? "Code incorrect. Il vous reste 1 essai."
        : `Code incorrect. Il vous reste ${attemptsLeft} essais.`;
    case "attempts-exhausted":
      return "Trop d’essais : ce code n’est plus valable. Demandez-en un nouveau.";
    case "code-expired":
      return "Ce code a expiré. Demandez-en un nouveau.";
    case "network-error":
      return networkErrorMessage;
  }
}

/** Keeps the digits of a typed or pasted code, six at most. */
export function normalizeCode(input: string): string {
  return input.replace(/\D/g, "").slice(0, 6);
}

/** What tells an installed app from a browser tab (display-mode, and navigator.standalone on iOS). */
export interface DisplayEnvironment {
  matchMedia(query: string): { matches: boolean };
  navigator: { standalone?: boolean };
}

/** Whether the application runs installed, sent when the session opens (EF-04). */
export function isInstalledApp(env: DisplayEnvironment): boolean {
  return env.matchMedia("(display-mode: standalone)").matches || env.navigator.standalone === true;
}
