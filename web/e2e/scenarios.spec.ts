// The scenarios @ui of docs/specs/features, exported by uiexport (ADR 0028): a test per
// scenario or example of an outline, whose steps are run by the registry. A scenario of
// acceptance/pending.txt must fail, and fails the run if it passes; any other must pass.
import { readFileSync } from "node:fs";
import { test } from "@playwright/test";
import { parsePending } from "./pending.ts";
import { World, findStep } from "./steps.ts";
import "./steps-dishes.ts";
import "./steps-planning.ts";

interface Scenario {
  file: string;
  title: string;
  tags: string[];
  example: number;
  steps: { type: string; text: string; table?: string[][] }[];
}

// Written by npm run e2e before Playwright starts (package.json).
const exported = ".e2e-tools/scenarios.json";
const scenarios = JSON.parse(readFileSync(exported, "utf8")) as Scenario[];
const pending = parsePending(readFileSync("../acceptance/pending.txt", "utf8"));

// A request of the page may still be in a route of Tribe.signInPage when the scenario ends:
// waiting for it here, its error ignored, keeps it from failing the test ("route.fetch: Test
// ended").
test.afterEach(async ({ page }) => {
  await page.unrouteAll({ behavior: "ignoreErrors" });
});

for (const scenario of scenarios) {
  const key = `${scenario.file}: ${scenario.title}`;
  const name = scenario.example > 0 ? `${key} (exemple ${scenario.example})` : key;
  test(name, { tag: scenario.tags }, async ({ page, request, baseURL }) => {
    test.fail(pending.has(key), "listed in acceptance/pending.txt: passes, remove it from the list");
    const world = new World(page, request, baseURL ?? "");
    for (const s of scenario.steps) {
      const definition = findStep(s.text);
      if (definition === undefined) {
        throw new Error(`undefined step: ${s.text}`);
      }
      world.table = s.table ?? [];
      world.stepType = s.type;
      await test.step(s.text, () => definition.fn(world, ...definition.args));
    }
  });
}
