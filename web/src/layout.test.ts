import assert from "node:assert/strict";
import { afterEach, test } from "node:test";
import { type Format, type LayoutWindow, currentFormat, formatOf, isField, nextFormat, onFormatChange, onFormatWillChange, resetFormatForTests, scrollEdges, thresholdQueries, trackFormat } from "./layout.ts";

const formatCases: { name: string; width: number; height: number; want: Format }[] = [
  { name: "narrowest supported", width: 320, height: 640, want: "compact" },
  { name: "phone in portrait (Pixel 7)", width: 412, height: 839, want: "compact" },
  { name: "phone in landscape stays compact (D3)", width: 844, height: 390, want: "compact" },
  { name: "just below moyen", width: 719, height: 900, want: "compact" },
  { name: "first width of moyen", width: 720, height: 900, want: "moyen" },
  { name: "tablet in portrait", width: 820, height: 1180, want: "moyen" },
  { name: "last width of moyen", width: 1023, height: 768, want: "moyen" },
  { name: "first width of etendu", width: 1024, height: 768, want: "etendu" },
  { name: "tablet in landscape", width: 1180, height: 820, want: "etendu" },
  { name: "computer", width: 1440, height: 900, want: "etendu" },
  { name: "widest supported", width: 2560, height: 1440, want: "etendu" },
  { name: "wide but low window", width: 1366, height: 520, want: "compact" },
  { name: "just below the height", width: 1366, height: 559, want: "compact" },
  { name: "first height", width: 1366, height: 560, want: "etendu" },
  { name: "moyen width, low", width: 800, height: 559, want: "compact" },
];

for (const { name, width, height, want } of formatCases) {
  test(`formatOf: ${name}`, () => {
    assert.equal(formatOf(width, height), want);
  });
}

const nextCases: { name: string; current: Format; measured: Format; field: boolean; want: Format }[] = [
  { name: "the keyboard of a tablet in landscape keeps etendu (D11)", current: "etendu", measured: "compact", field: true, want: "etendu" },
  { name: "the keyboard of a tablet in portrait keeps moyen", current: "moyen", measured: "compact", field: true, want: "moyen" },
  { name: "without a field, the window decides", current: "etendu", measured: "compact", field: false, want: "compact" },
  { name: "a rotation with a field is followed", current: "etendu", measured: "moyen", field: true, want: "moyen" },
  { name: "growing with a field is followed", current: "compact", measured: "etendu", field: true, want: "etendu" },
  { name: "unchanged", current: "moyen", measured: "moyen", field: false, want: "moyen" },
];

for (const { name, current, measured, field, want } of nextCases) {
  test(`nextFormat: ${name}`, () => {
    assert.equal(nextFormat(current, measured, field), want);
  });
}

/** A fake element: a tag and attributes. */
function element(tag: string, attributes: Record<string, string> = {}): Element {
  return { tagName: tag.toUpperCase(), getAttribute: (name: string) => attributes[name] ?? null } as unknown as Element;
}

const edgeCases: { name: string; left: number; client: number; scroll: number; before: boolean; after: boolean }[] = [
  { name: "everything fits (computer)", left: 0, client: 1280, scroll: 1280, before: false, after: false },
  { name: "at the start, days after (tablet)", left: 0, client: 864, scroll: 1180, before: false, after: true },
  { name: "in the middle", left: 170, client: 864, scroll: 1180, before: true, after: true },
  { name: "at the end", left: 316, client: 864, scroll: 1180, before: true, after: false },
  { name: "a fraction from the end is the end", left: 314.5, client: 864, scroll: 1180, before: true, after: false },
  { name: "a fraction from the start is the start", left: 1.5, client: 864, scroll: 1180, before: false, after: true },
];

for (const { name, left, client, scroll, before, after } of edgeCases) {
  test(`scrollEdges: ${name}`, () => {
    assert.deepEqual(scrollEdges(left, client, scroll), { before, after });
  });
}

test("isField", () => {
  assert.equal(isField(null), false);
  assert.equal(isField(element("input")), true);
  assert.equal(isField(element("input", { type: "search" })), true);
  assert.equal(isField(element("input", { type: "number" })), true);
  assert.equal(isField(element("input", { type: "checkbox" })), false);
  assert.equal(isField(element("input", { type: "submit" })), false);
  assert.equal(isField(element("textarea")), true);
  assert.equal(isField(element("select")), true);
  assert.equal(isField(element("button")), false);
  assert.equal(isField(element("a")), false);
  assert.equal(isField(element("div", { contenteditable: "" })), true);
  assert.equal(isField(element("div", { contenteditable: "false" })), false);
});

/** A window whose size, focus and timers the test drives. */
class FakeWindow implements LayoutWindow {
  innerWidth: number;
  innerHeight: number;
  readonly queries: string[] = [];
  readonly #changes: (() => void)[] = [];
  readonly #focusOut: (() => void)[] = [];
  readonly #timers: (() => void)[] = [];
  readonly attributes = new Map<string, string>();
  readonly document: LayoutWindow["document"] & { activeElement: Element | null };

  constructor(width: number, height: number) {
    this.innerWidth = width;
    this.innerHeight = height;
    const attributes = this.attributes;
    const focusOut = this.#focusOut;
    this.document = {
      documentElement: { setAttribute: (name: string, value: string) => attributes.set(name, value) },
      activeElement: null,
      addEventListener: (_type: "focusout", listener: () => void) => focusOut.push(listener),
    };
  }

  matchMedia(query: string) {
    this.queries.push(query);
    return { addEventListener: (_type: "change", listener: () => void) => this.#changes.push(listener) };
  }

  setTimeout(handler: () => void): unknown {
    this.#timers.push(handler);
    return this.#timers.length;
  }

  /** Resizes the window and tells the media queries. */
  resize(width: number, height: number) {
    this.innerWidth = width;
    this.innerHeight = height;
    this.#changes.forEach((c) => c());
  }

  /** Moves the focus away, then runs the timers. */
  blur() {
    this.document.activeElement = null;
    this.#focusOut.forEach((f) => f());
    this.#timers.splice(0).forEach((t) => t());
  }
}

afterEach(() => resetFormatForTests());

test("trackFormat: sets the attribute at once, then follows the thresholds", () => {
  assert.equal(currentFormat(), "compact");
  const win = new FakeWindow(1180, 820);
  const changes: Format[] = [];
  onFormatChange((f) => changes.push(f));
  assert.equal(trackFormat(win), "etendu");
  assert.equal(win.attributes.get("data-format"), "etendu");
  assert.deepEqual(win.queries, thresholdQueries);
  win.resize(820, 1180);
  assert.equal(win.attributes.get("data-format"), "moyen");
  assert.equal(currentFormat(), "moyen");
  win.resize(820, 1100);
  assert.deepEqual(changes, ["etendu", "moyen"], "a change within a format tells nothing");
});

test("trackFormat: kept while a field has the focus, followed once it leaves it (D11)", () => {
  const win = new FakeWindow(1180, 820);
  trackFormat(win);
  win.document.activeElement = element("input");
  win.resize(1180, 400);
  assert.equal(win.attributes.get("data-format"), "etendu");
  win.blur();
  assert.equal(win.attributes.get("data-format"), "compact");
});

test("trackFormat: the keyboard closed with the focus, nothing changes", () => {
  const win = new FakeWindow(1180, 820);
  trackFormat(win);
  win.document.activeElement = element("textarea");
  win.resize(1180, 400);
  win.resize(1180, 820);
  win.blur();
  assert.equal(win.attributes.get("data-format"), "etendu");
});

test("onFormatWillChange: told before the attribute changes, the format left still set", () => {
  const win = new FakeWindow(1180, 820);
  const seen: (string | undefined)[] = [];
  trackFormat(win);
  onFormatWillChange(() => seen.push(win.attributes.get("data-format")));
  win.resize(820, 1180);
  win.resize(820, 1100);
  win.resize(390, 844);
  assert.deepEqual(seen, ["etendu", "moyen"]);
});
