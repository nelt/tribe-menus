import assert from "node:assert/strict";
import { test } from "node:test";
import type { CollectionPage, Outcome, PageCondition } from "./api.ts";
import { CachedCollection, type PageReader, afterWrite, check, maxAttempts, versionOf } from "./collections.ts";

test("versionOf", () => {
  assert.equal(versionOf('"dishes-42"'), 42);
  assert.equal(versionOf('"ingredients-0"'), 0);
  assert.equal(versionOf("dishes"), undefined);
});

const writeCases: { current: number | undefined; written: number; want: string }[] = [
  { current: 42, written: 43, want: "take" },
  { current: 42, written: 42, want: "keep" },
  { current: 42, written: 40, want: "keep" },
  { current: 42, written: 44, want: "check" },
  { current: undefined, written: 1, want: "check" },
];

for (const { current, written, want } of writeCases) {
  test(`afterWrite: ${current} then ${written}`, () => {
    assert.equal(afterWrite(current, written), want);
  });
}

type Reply = Outcome<CollectionPage>;

/** A reader that answers in turn, and records the requests. */
function reader(replies: Reply[]) {
  const requests: { after: string | undefined; condition: PageCondition }[] = [];
  const read: PageReader = async (after, condition) => {
    requests.push({ after, condition });
    const reply = replies.shift();
    if (reply === undefined) {
      throw new Error("unexpected request");
    }
    return reply;
  };
  return { read, requests };
}

const page = (items: unknown[], etag: string, next?: string): Reply => ({ kind: "ok", value: { kind: "page", items, next, etag } });
const notModified: Reply = { kind: "ok", value: { kind: "not-modified" } };
const changed: Reply = { kind: "ok", value: { kind: "changed" } };

test("check: no copy, every page with If-Match", async () => {
  const { read, requests } = reader([page([1, 2], '"d-5"', "2"), page([3], '"d-5"')]);
  assert.deepEqual(await check(read, undefined), { kind: "loaded", items: [1, 2, 3], etag: '"d-5"' });
  assert.deepEqual(requests, [
    { after: undefined, condition: {} },
    { after: "2", condition: { ifMatch: '"d-5"' } },
  ]);
});

test("check: unchanged, one conditional request", async () => {
  const { read, requests } = reader([notModified]);
  assert.deepEqual(await check(read, '"d-5"'), { kind: "unchanged" });
  assert.deepEqual(requests, [{ after: undefined, condition: { ifNoneMatch: '"d-5"' } }]);
});

test("check: changed during the loading, started again", async () => {
  const { read, requests } = reader([page([1], '"d-6"', "1"), changed, page([1], '"d-7"', "1"), page([2], '"d-7"')]);
  assert.deepEqual(await check(read, '"d-5"'), { kind: "loaded", items: [1, 2], etag: '"d-7"' });
  assert.equal(requests.length, 4);
  assert.deepEqual(requests[2], { after: undefined, condition: { ifNoneMatch: '"d-5"' } });
});

test(`check: changing ${maxAttempts} times, an error`, async () => {
  const replies: Reply[] = [];
  for (let i = 0; i < maxAttempts; i++) {
    replies.push(page([1], `"d-${i}"`, "1"), changed);
  }
  const { read } = reader(replies);
  assert.deepEqual(await check(read, undefined), { kind: "network-error" });
});

test("check: failures", async () => {
  assert.deepEqual(await check(reader([{ kind: "network-error" }]).read, undefined), { kind: "network-error" });
  assert.deepEqual(await check(reader([{ kind: "unauthorized" }]).read, '"d-1"'), { kind: "unauthorized" });
  assert.deepEqual(await check(reader([page([1], '"d-1"', "1"), { kind: "network-error" }]).read, undefined), { kind: "network-error" });
  // A next page of another version without 412: the contract is broken.
  assert.deepEqual(await check(reader([page([1], '"d-1"', "1"), page([2], '"d-2"')]).read, undefined), { kind: "network-error" });
});

interface Item {
  id: string;
  n: number;
}

const parse = (raw: unknown): Item | undefined => (typeof raw === "object" && raw !== null && "id" in raw ? (raw as Item) : undefined);

test("CachedCollection: load, keep, reload", async () => {
  const { read } = reader([page([{ id: "a", n: 1 }], '"d-1"'), notModified, page([{ id: "b", n: 1 }], '"d-2"')]);
  const copy = new CachedCollection<Item>("d", read, parse, (i) => i.id);
  assert.equal(copy.items, undefined);
  assert.deepEqual(await copy.check(), { kind: "ok", value: undefined });
  assert.deepEqual(copy.items, [{ id: "a", n: 1 }]);
  assert.equal(copy.version, 1);
  await copy.check();
  assert.equal(copy.version, 1);
  await copy.check();
  assert.deepEqual(copy.items, [{ id: "b", n: 1 }]);
  assert.equal(copy.version, 2);
  copy.clear();
  assert.equal(copy.items, undefined);
  assert.equal(copy.version, undefined);
});

test("CachedCollection: a check already running is shared", async () => {
  const { read, requests } = reader([page([], '"d-1"')]);
  const copy = new CachedCollection<Item>("d", read, parse, (i) => i.id);
  await Promise.all([copy.check(), copy.check()]);
  assert.equal(requests.length, 1);
});

test("CachedCollection: an item that cannot be read fails the loading, the copy stays", async () => {
  const { read } = reader([page([{ id: "a", n: 1 }], '"d-1"'), page([{ nope: true }], '"d-2"')]);
  const copy = new CachedCollection<Item>("d", read, parse, (i) => i.id);
  await copy.check();
  assert.deepEqual(await copy.check(), { kind: "network-error" });
  assert.deepEqual(copy.items, [{ id: "a", n: 1 }]);
  assert.equal(copy.version, 1);
});

test("CachedCollection.applyWrite: the next version is taken, a gap checks again", async () => {
  const { read, requests } = reader([page([{ id: "a", n: 1 }], '"d-1"'), page([{ id: "a", n: 3 }, { id: "c", n: 1 }], '"d-4"')]);
  const copy = new CachedCollection<Item>("d", read, parse, (i) => i.id);
  await copy.check();
  await copy.applyWrite([{ id: "a", n: 2 }], 2);
  assert.deepEqual(copy.items, [{ id: "a", n: 2 }]);
  assert.equal(copy.version, 2);
  assert.equal(requests.length, 1);
  await copy.applyWrite([{ id: "b", n: 1 }], 4);
  assert.equal(requests.length, 2);
  assert.deepEqual(requests[1]?.condition, { ifNoneMatch: '"d-2"' });
  assert.deepEqual(copy.items, [{ id: "a", n: 3 }, { id: "c", n: 1 }]);
  assert.equal(copy.version, 4);
});

/** A reader whose replies are released by the test, one by one. */
function heldReader() {
  const held: ((reply: Reply) => void)[] = [];
  const read: PageReader = () => new Promise((resolve) => held.push(resolve));
  const release = (reply: Reply) => {
    const next = held.shift();
    if (next === undefined) {
      throw new Error("no request held");
    }
    next(reply);
  };
  return { read, release, pending: () => held.length };
}

test("CachedCollection: a check begun before clear() writes nothing (review of PR #77, point 1)", async () => {
  const { read, release } = heldReader();
  const copy = new CachedCollection<Item>("d", read, parse, (i) => i.id);
  const checking = copy.check();
  copy.clear();
  release(page([{ id: "a", n: 1 }], '"d-5"'));
  await checking;
  assert.equal(copy.items, undefined);
  assert.equal(copy.version, undefined);
});

test("CachedCollection: a check asked after clear() does not share the one begun before", async () => {
  const { read, release, pending } = heldReader();
  const copy = new CachedCollection<Item>("d", read, parse, (i) => i.id);
  const before = copy.check();
  copy.clear();
  const after = copy.check();
  assert.equal(pending(), 2);
  release(page([{ id: "a", n: 1 }], '"d-5"'));
  release(page([{ id: "b", n: 1 }], '"d-6"'));
  await Promise.all([before, after]);
  assert.deepEqual(copy.items, [{ id: "b", n: 1 }]);
  assert.equal(copy.version, 6);
});
test("CachedCollection: a check begun before a write does not undo it (review of PR #77, point 2)", async () => {
  const { read, release, pending } = heldReader();
  const copy = new CachedCollection<Item>("d", read, parse, (i) => i.id);
  const first = copy.check();
  release(page([{ id: "a", n: 1 }], '"d-5"'));
  await first;

  // A check reads version 5 while the member writes version 6.
  const checking = copy.check();
  await copy.applyWrite([{ id: "b", n: 1 }], 6);
  release(page([{ id: "a", n: 1 }], '"d-5"'));
  await checking;
  assert.deepEqual(copy.items, [{ id: "a", n: 1 }, { id: "b", n: 1 }]);
  assert.equal(copy.version, 6);
  assert.equal(pending(), 0);
});

test("CachedCollection: after a write with a gap, a new check begins after the running one", async () => {
  const { read, release, pending } = heldReader();
  const copy = new CachedCollection<Item>("d", read, parse, (i) => i.id);
  const first = copy.check();
  release(page([{ id: "a", n: 1 }], '"d-5"'));
  await first;

  const running = copy.check();
  const written = copy.applyWrite([{ id: "b", n: 1 }], 8);
  // The running check, begun before the write, ends; its result is not taken.
  release(page([{ id: "a", n: 1 }], '"d-5"'));
  await running;
  await new Promise((resolve) => setTimeout(resolve, 0));
  assert.equal(pending(), 1, "a new check after the running one");
  release(page([{ id: "a", n: 1 }, { id: "b", n: 1 }, { id: "c", n: 1 }], '"d-8"'));
  await written;
  assert.deepEqual(copy.items?.map((i) => i.id), ["a", "b", "c"]);
  assert.equal(copy.version, 8);
});

test("CachedCollection.applyWrite: several items of one write, one version", async () => {
  const { read } = reader([page([{ id: "a", n: 1 }], '"d-1"')]);
  const copy = new CachedCollection<Item>("d", read, parse, (i) => i.id);
  await copy.check();
  await copy.applyWrite([{ id: "a", n: 2 }, { id: "b", n: 1 }], 2);
  assert.deepEqual(copy.items, [{ id: "a", n: 2 }, { id: "b", n: 1 }]);
  assert.equal(copy.version, 2);
});

test("CachedCollection.put: an item without version is taken, the version stays (D26)", async () => {
  const { read, release, pending } = heldReader();
  const copy = new CachedCollection<Item>("d", read, parse, (i) => i.id);
  const first = copy.check();
  release(page([{ id: "a", n: 1 }], '"d-5"'));
  await first;

  // A check begun before does not undo the item put.
  const checking = copy.check();
  copy.put([{ id: "a", n: 3 }]);
  release(page([{ id: "a", n: 2 }], '"d-6"'));
  await checking;
  assert.deepEqual(copy.items, [{ id: "a", n: 3 }]);
  assert.equal(copy.version, 5, "checked again at the next visit");
  assert.equal(pending(), 0);
});

test("CachedCollection.put: nothing before a first loading", () => {
  const { read } = reader([]);
  const copy = new CachedCollection<Item>("d", read, parse, (i) => i.id);
  copy.put([{ id: "a", n: 1 }]);
  assert.equal(copy.items, undefined);
});
