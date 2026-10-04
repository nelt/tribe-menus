// Client-side router (ADR 0004, point 6), on the History API, under the tribe prefix
// /tribes/<identifier>/ given by the <base> of the page.

/** The pages of the application. */
export type Page = "home" | "login";

const pagePaths: Record<Page, string> = {
  home: "",
  login: "connexion",
};

/**
 * Returns the page of a path, relative to the base of the tribe (with its trailing slash).
 * An unknown path, inside the tribe or not, gives the home page.
 */
export function pageFromPath(pathname: string, base: string): Page {
  if (!pathname.startsWith(base)) {
    return "home";
  }
  const rest = pathname.slice(base.length).replace(/\/+$/, "");
  for (const [page, path] of Object.entries(pagePaths) as [Page, string][]) {
    if (rest === path) {
      return page;
    }
  }
  return "home";
}

/** Returns the path of a page under the base of the tribe. */
export function pathOfPage(page: Page, base: string): string {
  return base + pagePaths[page];
}

/** What the router needs from the browser, so that the tests can provide it. */
export interface RouterWindow {
  location: { pathname: string };
  history: {
    pushState(data: unknown, unused: string, url: string): void;
    replaceState(data: unknown, unused: string, url: string): void;
  };
  addEventListener(type: "popstate", listener: () => void): void;
}

/** Router keeps the page in the address bar and tells each change of page. */
export class Router {
  readonly #window: RouterWindow;
  readonly #base: string;
  readonly #onChange: (page: Page) => void;

  constructor(window: RouterWindow, base: string, onChange: (page: Page) => void) {
    this.#window = window;
    this.#base = base;
    this.#onChange = onChange;
    window.addEventListener("popstate", () => this.#onChange(this.page));
  }

  /** The current page. */
  get page(): Page {
    return pageFromPath(this.#window.location.pathname, this.#base);
  }

  /**
   * Goes to a page; replace does not add an entry to the history, for a redirection.
   * Nothing happens when the page is already the current one.
   */
  navigate(page: Page, { replace = false }: { replace?: boolean } = {}): void {
    if (page === this.page) {
      return;
    }
    const url = pathOfPage(page, this.#base);
    if (replace) {
      this.#window.history.replaceState(null, "", url);
    } else {
      this.#window.history.pushState(null, "", url);
    }
    this.#onChange(page);
  }
}
