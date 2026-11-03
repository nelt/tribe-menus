// Copy in memory of a collection of the API (ADR 0029, plan bibliotheque, D29): its items and
// the ETag of their version, checked by a conditional request, reloaded page by page when the
// collection changed, and resumed from the first page when it changes during the loading.
// Nothing is written on the device. The decisions are pure functions, tested by node:test.
import type { CollectionPage, Failure, Outcome, PageCondition } from "./api.ts";

/** At most this many loadings in a row, the collection changing each time, before an error. */
export const maxAttempts = 3;

/** The version of a collection read in its ETag, "<collection>-<version>". */
export function versionOf(etag: string): number | undefined {
  const m = /-(\d+)"$/.exec(etag);
  return m === null ? undefined : Number(m[1]);
}

/**
 * What a copy does after a write of the member (D29): it takes the version written if it
 * follows its own immediately; keeps its own if the write did not change the collection or
 * if it already holds a later version; otherwise another member wrote in between, and it
 * checks the collection again. A copy never loaded is loaded at its next check.
 */
export function afterWrite(current: number | undefined, written: number): "take" | "keep" | "check" {
  if (current === undefined) {
    return "check";
  }
  if (written === current + 1) {
    return "take";
  }
  return written <= current ? "keep" : "check";
}

/** Reads a page of the collection. */
export type PageReader = (after: string | undefined, condition: PageCondition) => Promise<Outcome<CollectionPage>>;

/** The result of a check of a copy. */
export type Check = { kind: "unchanged" } | { kind: "loaded"; items: unknown[]; etag: string } | Failure;

/**
 * Checks a copy whose ETag is given (undefined without copy): the first page, conditional;
 * unchanged, the copy stays; changed, every page is read with If-Match, and a change during
 * the loading starts it again, maxAttempts times at most, then fails as a network error.
 */
export async function check(read: PageReader, etag: string | undefined): Promise<Check> {
  for (let attempt = 0; attempt < maxAttempts; attempt++) {
    const first = await read(undefined, etag === undefined ? {} : { ifNoneMatch: etag });
    if (first.kind !== "ok") {
      return first;
    }
    if (first.value.kind === "not-modified") {
      return { kind: "unchanged" };
    }
    if (first.value.kind !== "page") {
      return { kind: "network-error" };
    }
    const loaded = await readNext(read, first.value);
    if (loaded.kind !== "changed") {
      return loaded;
    }
  }
  return { kind: "network-error" };
}

async function readNext(
  read: PageReader,
  first: { items: unknown[]; next: string | undefined; etag: string },
): Promise<Check | { kind: "changed" }> {
  const items = [...first.items];
  for (let next = first.next; next !== undefined; ) {
    const page = await read(next, { ifMatch: first.etag });
    if (page.kind !== "ok") {
      return page;
    }
    if (page.value.kind === "changed") {
      return { kind: "changed" };
    }
    if (page.value.kind !== "page" || page.value.etag !== first.etag) {
      return { kind: "network-error" };
    }
    items.push(...page.value.items);
    next = page.value.next;
  }
  return { kind: "loaded", items, etag: first.etag };
}

/** CachedCollection is the copy in memory of a collection. */
export class CachedCollection<T> {
  readonly name: string;
  readonly #read: PageReader;
  readonly #parse: (raw: unknown) => T | undefined;
  readonly #key: (item: T) => string;
  #items: Map<string, T> | undefined;
  #etag: string | undefined;
  #checking: Promise<Outcome<undefined>> | undefined;
  /**
   * Increases when the copy is cleared: a check begun before does not write its result into
   * the copy, which must not survive the sign-out or a 401 (D25).
   */
  #generation = 0;

  constructor(name: string, read: PageReader, parse: (raw: unknown) => T | undefined, key: (item: T) => string) {
    this.name = name;
    this.#read = read;
    this.#parse = parse;
    this.#key = key;
  }

  /** The items of the copy, undefined until a first loading. */
  get items(): T[] | undefined {
    return this.#items === undefined ? undefined : [...this.#items.values()];
  }

  /** The version of the copy. */
  get version(): number | undefined {
    return this.#etag === undefined ? undefined : versionOf(this.#etag);
  }

  /** Checks the copy against the API; a check already running is shared. */
  check(): Promise<Outcome<undefined>> {
    if (this.#checking === undefined) {
      const checking = this.#check().finally(() => {
        if (this.#checking === checking) {
          this.#checking = undefined;
        }
      });
      this.#checking = checking;
    }
    return this.#checking;
  }

  async #check(): Promise<Outcome<undefined>> {
    const generation = this.#generation;
    const result = await check(this.#read, this.#etag);
    if (generation !== this.#generation) {
      return { kind: "ok", value: undefined };
    }
    if (result.kind === "unchanged") {
      return { kind: "ok", value: undefined };
    }
    if (result.kind !== "loaded") {
      return result;
    }
    const items = new Map<string, T>();
    for (const raw of result.items) {
      const item = this.#parse(raw);
      if (item === undefined) {
        return { kind: "network-error" };
      }
      items.set(this.#key(item), item);
    }
    this.#items = items;
    this.#etag = result.etag;
    return { kind: "ok", value: undefined };
  }

  /**
   * Takes an item written by the member, and the version of the collection after the write
   * (D26, D29); checks the collection again if another member wrote in between.
   */
  async applyWrite(item: T, version: number): Promise<Outcome<undefined>> {
    if (this.#items !== undefined) {
      this.#items.set(this.#key(item), item);
    }
    switch (afterWrite(this.version, version)) {
      case "take":
        this.#etag = `"${this.name}-${version}"`;
        return { kind: "ok", value: undefined };
      case "keep":
        return { kind: "ok", value: undefined };
      case "check":
        return this.check();
    }
  }

  /** Forgets the copy: at the sign-out, or on a 401 (D25). */
  clear(): void {
    this.#generation++;
    this.#checking = undefined;
    this.#items = undefined;
    this.#etag = undefined;
  }
}
