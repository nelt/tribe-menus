// Formats of the screen (ADR 0030, plan grands-ecrans, D3, D11, D12): decided by the size of
// the window, never by the device, and set on the root of the document as data-format, which
// the stylesheet of the application reads instead of width media queries. The thresholds are
// here and nowhere else.

/** The three formats: phone; tablet in portrait or narrow window; tablet in landscape or computer. */
export type Format = "compact" | "moyen" | "etendu";

/** From this width, the format is moyen (720 to 1023 px wide). */
export const mediumMinWidth = 720;
/** From this width, the format is etendu. */
export const expandedMinWidth = 1024;
/** Below this height, the format is compact whatever the width: a phone in landscape. */
export const minHeight = 560;

/** The attribute of the root of the document that carries the format. */
export const formatAttribute = "data-format";

/** The format of a window of this size, in CSS pixels. */
export function formatOf(width: number, height: number): Format {
  if (height < minHeight || width < mediumMinWidth) {
    return "compact";
  }
  return width < expandedMinWidth ? "moyen" : "etendu";
}

/**
 * The format to show once the window is measured again. A larger format is kept while a field
 * has the focus rather than falling to compact: a browser that resizes the layout viewport at
 * the opening of the virtual keyboard must not change the layout of a tablet (D11).
 */
export function nextFormat(current: Format, measured: Format, fieldFocused: boolean): Format {
  if (fieldFocused && measured === "compact" && current !== "compact") {
    return current;
  }
  return measured;
}

/** Whether an element takes typing: what opens the virtual keyboard. */
export function isField(element: Element | null): boolean {
  if (element === null) {
    return false;
  }
  const tag = element.tagName.toLowerCase();
  if (tag === "textarea" || tag === "select") {
    return true;
  }
  if (tag === "input") {
    const type = (element.getAttribute("type") ?? "text").toLowerCase();
    return !["button", "checkbox", "radio", "submit", "reset", "image", "file", "range", "color", "hidden"].includes(type);
  }
  return element.getAttribute("contenteditable") !== null && element.getAttribute("contenteditable") !== "false";
}

/** What is left to scroll on each side of an element scrolled sideways (the planning by week, D6). */
export interface ScrollEdges {
  before: boolean;
  after: boolean;
}

/** A pixel or so is no day left: the sizes are rounded, and a zoom gives fractions. */
const scrollTolerance = 2;

/** What is left to scroll before and after, from the measures of the element scrolled. */
export function scrollEdges(scrollLeft: number, clientWidth: number, scrollWidth: number): ScrollEdges {
  return { before: scrollLeft > scrollTolerance, after: scrollLeft + clientWidth < scrollWidth - scrollTolerance };
}

/** What the tracking of the format needs from the window. */
export interface LayoutWindow {
  readonly innerWidth: number;
  readonly innerHeight: number;
  matchMedia(query: string): { addEventListener(type: "change", listener: () => void): void };
  setTimeout(handler: () => void, timeout: number): unknown;
  readonly document: {
    readonly documentElement: { setAttribute(name: string, value: string): void };
    readonly activeElement: Element | null;
    addEventListener(type: "focusout", listener: () => void): void;
  };
}

/** The media queries whose changes can change the format: one per threshold. */
export const thresholdQueries = [`(min-width: ${mediumMinWidth}px)`, `(min-width: ${expandedMinWidth}px)`, `(min-height: ${minHeight}px)`];

let current: Format | undefined;
const listeners = new Set<(format: Format) => void>();

/**
 * Sets the format on the root of the document, then follows the window: called once, before
 * the first render of the application (main.ts).
 */
export function trackFormat(win: LayoutWindow): Format {
  const update = () => {
    const measured = formatOf(win.innerWidth, win.innerHeight);
    const next = current === undefined ? measured : nextFormat(current, measured, isField(win.document.activeElement));
    if (next === current) {
      return;
    }
    current = next;
    win.document.documentElement.setAttribute(formatAttribute, next);
    for (const listener of listeners) {
      listener(next);
    }
  };
  update();
  for (const query of thresholdQueries) {
    win.matchMedia(query).addEventListener("change", update);
  }
  // A format kept for a field (D11) follows the window once the focus leaves it; the focus
  // moving to another field keeps it: measured after the focus has moved.
  win.document.addEventListener("focusout", () => win.setTimeout(update, 0));
  return current ?? "compact";
}

/** The format shown; compact before trackFormat. */
export function currentFormat(): Format {
  return current ?? "compact";
}

/** Calls the listener at each change of the format; returns what removes it. */
export function onFormatChange(listener: (format: Format) => void): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

/** Forgets the format and its listeners: for the tests. */
export function resetFormatForTests(): void {
  current = undefined;
  listeners.clear();
}
