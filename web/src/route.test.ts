import assert from "node:assert/strict";
import { test } from "node:test";
import { tribeIdFromPath } from "./route.ts";

const cases: { name: string; pathname: string; want: string | undefined }[] = [
  { name: "tribe root", pathname: "/tribes/demo/", want: "demo" },
  { name: "without trailing slash", pathname: "/tribes/demo", want: "demo" },
  { name: "sub-path", pathname: "/tribes/demo/planning/2026-09-29", want: "demo" },
  { name: "encoded identifier", pathname: "/tribes/les%20martin/", want: "les martin" },
  { name: "malformed encoding", pathname: "/tribes/%E0%A4%A/", want: undefined },
  { name: "no identifier", pathname: "/tribes/", want: undefined },
  { name: "site root", pathname: "/", want: undefined },
  { name: "other prefix", pathname: "/tribesx/demo/", want: undefined },
];

for (const { name, pathname, want } of cases) {
  test(`tribeIdFromPath: ${name}`, () => {
    assert.equal(tribeIdFromPath(pathname), want);
  });
}
