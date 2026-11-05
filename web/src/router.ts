// Client-side router (ADR 0004, point 6), on the History API, under the tribe prefix
// /tribes/<identifier>/ given by the <base> of the page. Paths in French (plan bibliotheque,
// D11; plan planning, D14; docs/design/README.md).
import { type Day, type Mealtime, addDays, parseDay, rangeLength } from "./meal-plan.ts";

/** The pages of the application. */
export type Page =
  /** The 7 days from today, or from the day given (?du=, D14). */
  | { name: "planning"; start?: Day }
  /** A meal: its day and its mealtime (D14). */
  | { name: "meal"; date: Day; mealtime: Mealtime }
  | { name: "login" }
  | { name: "dishes" }
  | { name: "new-dish" }
  | { name: "dish"; id: string }
  | { name: "tribe" };

export const pages = {
  planning: { name: "planning" },
  login: { name: "login" },
  dishes: { name: "dishes" },
  newDish: { name: "new-dish" },
  tribe: { name: "tribe" },
} as const satisfies Record<string, Page>;

const fixedPaths: [string, Page][] = [
  ["", pages.planning],
  ["connexion", pages.login],
  ["plats", pages.dishes],
  ["plats/nouveau", pages.newDish],
  ["tribu", pages.tribe],
];

/** A dish identifier: 128 random bits in hexadecimal (plan bibliotheque, D10). */
const dishPath = /^plats\/([0-9a-f]{32})$/;

/** A meal: repas/YYYY-MM-DD/midi or soir (D14). */
const mealPath = /^repas\/([^/]+)\/(midi|soir)$/;

const mealtimePaths: Record<Mealtime, string> = { lunch: "midi", dinner: "soir" };

/** The query of the planning: the first day of the range shown (D14). */
const startParameter = "du";

/**
 * Returns the page of a path and its query, relative to the base of the tribe (with its
 * trailing slash). An unknown path, inside the tribe or not, gives the planning; so does a
 * day of the planning unreadable or out of the bounds (D14).
 */
export function pageFromPath(pathname: string, base: string, search = ""): Page {
  if (!pathname.startsWith(base)) {
    return pages.planning;
  }
  const rest = pathname.slice(base.length).replace(/\/+$/, "");
  if (rest === "") {
    const start = parseDay(new URLSearchParams(search).get(startParameter) ?? "");
    return start === undefined ? pages.planning : { name: "planning", start };
  }
  const fixed = fixedPaths.find(([path]) => path === rest);
  if (fixed !== undefined) {
    return fixed[1];
  }
  const meal = mealPath.exec(rest);
  if (meal !== null) {
    const date = parseDay(meal[1] ?? "");
    return date === undefined ? pages.planning : { name: "meal", date, mealtime: meal[2] === "midi" ? "lunch" : "dinner" };
  }
  const id = dishPath.exec(rest)?.[1];
  return id === undefined ? pages.planning : { name: "dish", id };
}

/** Returns the path of a page under the base of the tribe, with its query. */
export function pathOfPage(page: Page, base: string): string {
  if (page.name === "dish") {
    return `${base}plats/${page.id}`;
  }
  if (page.name === "meal") {
    return `${base}repas/${page.date}/${mealtimePaths[page.mealtime]}`;
  }
  if (page.name === "planning" && page.start !== undefined) {
    return `${base}?${startParameter}=${page.start}`;
  }
  const fixed = fixedPaths.find(([, p]) => p.name === page.name);
  return base + (fixed?.[0] ?? "");
}

/**
 * The planning to go back to from a meal opened without coming from the planning: the range
 * of today if it holds the day of the meal, the range from that day otherwise.
 */
export function planningAround(day: Day, today: Day): Page {
  return day >= today && day < addDays(today, rangeLength) ? pages.planning : { name: "planning", start: day };
}

/** The tab of a page: the dish pages are under Plats, the others under Planning. */
export function tabOf(page: Page): "planning" | "dishes" {
  return page.name === "dishes" || page.name === "new-dish" || page.name === "dish" ? "dishes" : "planning";
}

/** What the router needs from the browser, so that the tests can provide it. */
export interface RouterWindow {
  location: { pathname: string; search: string };
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
    return pageFromPath(this.#window.location.pathname, this.#base, this.#window.location.search);
  }

  /** The path of a page, for a link. */
  href(page: Page): string {
    return pathOfPage(page, this.#base);
  }

  /**
   * Goes to a page; replace does not add an entry to the history, for a redirection.
   * Nothing happens when the page is already the current one.
   */
  navigate(page: Page, { replace = false }: { replace?: boolean } = {}): void {
    const url = pathOfPage(page, this.#base);
    if (url === pathOfPage(this.page, this.#base)) {
      return;
    }
    if (replace) {
      this.#window.history.replaceState(null, "", url);
    } else {
      this.#window.history.pushState(null, "", url);
    }
    this.#onChange(this.page);
  }

  /**
   * Follows a link of the application: true when the path is a page of the tribe, which the
   * router then shows; false for any other address, left to the browser.
   */
  follow(pathname: string, search = ""): boolean {
    if (!pathname.startsWith(this.#base)) {
      return false;
    }
    this.navigate(pageFromPath(pathname, this.#base, search));
    return true;
  }
}
