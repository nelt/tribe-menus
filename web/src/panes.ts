// The panes of a page (plan grands-ecrans, D5, D8, D9; ADR 0030, point 4): the pane of a list
// (the library, the list picker) and the detail beside it, after the format. In etendu, the
// pages of the tab Plats show the library beside the dish, those of the tab Courses the list
// picker beside the list; in compact and moyen, one pane at a time. The application renders
// each pane in a place of its own, so that an element is created again only when its pane
// changes: what tells which input a navigation would lose.
import type { Format } from "./layout.ts";
import { type Page, tabOf } from "./router.ts";

/** The pane of a list: the dish library, or the list picker. */
export type ListPane = "library" | "lists";

/** The detail of a page, beside the pane of a list in etendu. */
export type Detail =
  | { kind: "choose-dish" }
  | { kind: "dish"; id: string | undefined }
  | { kind: "list"; list: string | undefined }
  | { kind: "meal-shopping"; list: string | undefined };

/** The pane of a list of a page; undefined for a page without one. */
export function listPaneOf(page: Page, format: Format): ListPane | undefined {
  if (format === "etendu") {
    switch (tabOf(page)) {
      case "dishes":
        return "library";
      case "shopping":
        return "lists";
      default:
        return undefined;
    }
  }
  switch (page.name) {
    case "dishes":
      return "library";
    case "list-picker":
      return "lists";
    default:
      return undefined;
  }
}

/**
 * The detail of a page of the tabs Plats and Courses; undefined for the other pages, and for
 * the library and the list picker alone. In etendu, Plats alone invites to choose a dish, and
 * the list picker shows the main list (D5).
 */
export function detailOf(page: Page, format: Format): Detail | undefined {
  switch (page.name) {
    case "dishes":
      return format === "etendu" ? { kind: "choose-dish" } : undefined;
    case "new-dish":
      return { kind: "dish", id: undefined };
    case "dish":
      return { kind: "dish", id: page.id };
    case "shopping":
      return { kind: "list", list: page.list };
    case "list-picker":
      return format === "etendu" ? { kind: "list", list: undefined } : undefined;
    case "meal-shopping":
      return { kind: "meal-shopping", list: page.list };
    default:
      return undefined;
  }
}

/** The key of a detail: the same key, the same element (D8). */
export function detailKey(detail: Detail): string {
  switch (detail.kind) {
    case "choose-dish":
      return "choose-dish";
    case "dish":
      return `dish ${detail.id ?? "new"}`;
    case "list":
      return `list ${detail.list ?? "main"}`;
    case "meal-shopping":
      return `meal-shopping ${detail.list ?? "main"}`;
  }
}

/** What going from a page to another loses, in a format: the pane of the list, the detail. */
export interface Lost {
  list: boolean;
  detail: boolean;
}

/** What a navigation loses: a pane whose content is no longer the same. */
export function lostBy(from: Page, to: Page, format: Format): Lost {
  const before = detailOf(from, format);
  const after = detailOf(to, format);
  return {
    list: listPaneOf(from, format) !== listPaneOf(to, format),
    detail: before !== undefined && (after === undefined || detailKey(before) !== detailKey(after)),
  };
}

/** The attribute that marks the element of a pane, which the application asks for an input not saved (D9). */
export const paneAttribute = "data-pane";

/** A screen that can hold an input not saved (D9). */
export interface Guarded {
  /** An input differs from what is saved. */
  readonly unsaved: boolean;
}

/** Whether an element holds an input not saved. */
export function holdsUnsaved(element: Element | null): boolean {
  return element !== null && "unsaved" in element && (element as unknown as Guarded).unsaved;
}
