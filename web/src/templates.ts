// Templates shared by the screens.
import { html, type TemplateResult } from "lit";

const repositoryURL = "https://github.com/nelt/tribe-menus";

/** The address of the source code of this version, given by the server (ADR 0011, point 2). */
export function sourceURL(doc: Document = document): string {
  return doc.querySelector<HTMLMetaElement>('meta[name="source-url"]')?.content || repositoryURL;
}

/** An error message, with its icon, announced to screen readers. */
export function alertMessage(id: string, text: string): TemplateResult {
  return html`<p class="alert" role="alert" id=${id}>
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" aria-hidden="true">
      <circle cx="12" cy="12" r="9"></circle>
      <path d="M12 8v5M12 16.5v.5"></path>
    </svg>
    <span>${text}</span>
  </p>`;
}

/** The footer of the screens: the brand and the link to the source code. */
export function footer(): TemplateResult {
  return html`<footer class="app-footer">
    Melting Tribe ·
    <a href=${sourceURL()}>logiciel libre, code source</a>
  </footer>`;
}
