import assert from "node:assert/strict";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { test } from "node:test";
import { recetteSettings } from "./config.ts";

/** A volume of the configuration and a plan of the acceptance run, in a directory of their own. */
function setup(plan: string): NodeJS.ProcessEnv {
  const dir = mkdtempSync(join(tmpdir(), "recette-"));
  writeFileSync(join(dir, "recette-auto.json"), JSON.stringify({ tribeURL: "https://recette.example.org/tribes/x/", address: "a@example.org" }));
  writeFileSync(join(dir, "v0.6.0.md"), plan);
  return { RECETTE_CONFIG_DIR: dir, RECETTE_OUTPUT_DIR: join(dir, "out"), RECETTE_PLAN: join(dir, "v0.6.0.md") };
}

const table = "## Points de contrôle de la vidéo\n\n| Point | Entrée | Parcours, projet | Étape | Regarder |\n| --- | --- | --- | --- | --- |\n";

test("recetteSettings reads the control points of the plan", () => {
  const settings = recetteSettings(setup(table + "| V1 | 1 | `base.spec.ts`, `phone-chromium` | « Menus » | l'écran Menus |\n"));
  assert.equal(settings.tribeURL, "https://recette.example.org/tribes/x/");
  assert.deepEqual(
    settings.points.map((p) => [p.id, p.file, p.project, p.step]),
    [["V1", "base.spec.ts", "phone-chromium", "Menus"]],
  );
});

test("recetteSettings refuses a plan it cannot read, before any code request", () => {
  assert.throws(() => recetteSettings(setup("# Plan de recette\n")), /v0\.6\.0\.md: no section/);
  assert.throws(() => recetteSettings(setup(table + "| V1 | 1 | `….spec.ts`, `phone-chromium` | « … » | … |\n")), /v0\.6\.0\.md: control points: V1: journey, project or step unreadable/);
});

test("recetteSettings refuses to run without make recette-auto", () => {
  assert.throws(() => recetteSettings({}), /make recette-auto/);
});
