// Templates shared by the screens.
import { html, type TemplateResult } from "lit";

const repositoryURL = "https://github.com/nelt/tribe-menus";

/** The message of a change without network, or of an unforeseen response (plan
 * bibliotheque, D18; docs/design/README.md, « Messages »): nothing changed. */
export const networkNeededMessage = "Réseau nécessaire. Le réseau est lent ou absent : réessayez.";

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

/** A warning banner, announced to screen readers: a crossed change (Q25, Q33). */
export function warningBanner(text: string): TemplateResult {
  return html`<div class="banner-warning" role="alert">
    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
      <path d="M10.3 3.9L2.4 17.6A2 2 0 0 0 4.1 20.6h15.8a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0z"></path>
      <path d="M12 9.5v4.5"></path>
      <circle cx="12" cy="17.2" r="0.6" fill="currentColor"></circle>
    </svg>
    <span>${text}</span>
  </div>`;
}

/** What a chip of the tribe reads: its name and its number of active members (Doux-Main). */
export function tribeChipLabel(tribe: { name: string; activeMembers: number }): string {
  const members = tribe.activeMembers > 1 ? `${tribe.activeMembers} membres` : `${tribe.activeMembers} membre`;
  return `Tribu ${tribe.name}, ${members}`;
}

/** The footer of the screens: the brand and the link to the source code. */
export function footer(): TemplateResult {
  return html`<footer class="app-footer">
    Melting Tribe ·
    <a href=${sourceURL()}>logiciel libre, code source</a>
  </footer>`;
}
