import assert from "node:assert/strict";
import { test } from "node:test";
import { parsePending } from "./pending.ts";

test("parsePending", () => {
  const keys = parsePending("# comment\n\na.feature: Un scénario\n  b.feature: Un autre  \n");
  assert.deepEqual([...keys], ["a.feature: Un scénario", "b.feature: Un autre"]);
});

test("parsePending: refused lines", () => {
  assert.throws(() => parsePending("Un scénario\n"), /pending.txt:1/);
  assert.throws(() => parsePending("a.feature: A\na.feature: A\n"), /listed twice/);
});
