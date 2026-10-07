// The name of a built file follows its content (ADR 0012, point 2): what makes it safe to serve
// the files of the list as immutable.
import assert from "node:assert/strict";
import { appendFile, cp, mkdtemp, readFile, rm, symlink } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { after, before, test } from "node:test";
import { fileURLToPath } from "node:url";
import { build } from "./build.mjs";

const webDir = join(dirname(fileURLToPath(import.meta.url)), "..");
let root = "";
let copy = "";

// A copy of web/ and of the identity files, laid out as in the repository.
before(async () => {
  root = await mkdtemp(join(tmpdir(), "tribe-menus-build-"));
  copy = join(root, "web");
  await cp(join(webDir, "src"), join(copy, "src"), { recursive: true });
  await cp(join(webDir, "..", "docs", "design", "identite"), join(root, "docs", "design", "identite"), { recursive: true });
  await symlink(join(webDir, "node_modules"), join(copy, "node_modules"));
});

after(async () => {
  await rm(root, { recursive: true, force: true });
});

/** Builds the copy and returns the produced name of each file, by its fixed name. */
async function names() {
  await build(copy, { logLevel: "silent" });
  const files = JSON.parse(await readFile(join(copy, "dist", "files.json"), "utf8"));
  return Object.fromEntries(files.map((file) => [file.replace(/-[A-Z0-9]{8}\./, "."), file]));
}

/** The fixed names whose produced name differs between two builds. */
function changed(before, after) {
  assert.deepEqual(Object.keys(after).sort(), Object.keys(before).sort());
  return Object.keys(before).filter((name) => before[name] !== after[name]).sort();
}

test("each name changes with its content, and only with it", async () => {
  const first = await names();
  assert.deepEqual(Object.keys(first).sort(), [
    "BricolageGrotesque[opsz,wdth,wght].woff2",
    "Figtree[wght].woff2",
    "app.css",
    "logotype.svg",
    "main.js",
    "symbole.svg",
  ]);
  for (const [fixed, produced] of Object.entries(first)) {
    assert.notEqual(produced, fixed, `${fixed} has no hash`);
  }

  const again = await names();
  assert.deepEqual(changed(first, again), [], "a build without change keeps every name");

  await appendFile(join(copy, "src", "main.ts"), '\nconsole.info("changed");\n');
  const code = await names();
  assert.deepEqual(changed(again, code), ["main.js"]);

  await appendFile(join(copy, "src", "app.css"), "\n.changed { color: red; }\n");
  const style = await names();
  assert.deepEqual(changed(code, style), ["app.css"]);

  // The stylesheet names the font: its name changes with it.
  await appendFile(join(copy, "src", "fonts", "Figtree[wght].woff2"), "\0");
  const font = await names();
  assert.deepEqual(changed(style, font), ["Figtree[wght].woff2", "app.css"]);

  const index = await readFile(join(copy, "dist", "index.html"), "utf8");
  assert.ok(index.includes(`href="${font["app.css"]}"`), "index.html links the new stylesheet");
  assert.ok(index.includes(`src="${font["main.js"]}"`), "index.html loads the new code");
});
