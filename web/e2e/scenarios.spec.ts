// The scenarios @ui of docs/specs/features, exported by uiexport (ADR 0028): a test per
// scenario or example of an outline, whose steps are run by the registry, in the projects of
// its formats (ADR 0030, point 6). Any scenario must pass, except those of
// acceptance/pending.txt, whose failure is noted; pending-reporter.ts fails the run if one of
// them passes in every project of its formats (ADR 0030, point 8).
import { readFileSync } from "node:fs";
import { test } from "@playwright/test";
import type { Format } from "../src/layout.ts";
import { formatsOf } from "./formats.ts";
import { parsePending } from "./pending.ts";
import { pendingAnnotation, pendingPasses, scenarioAnnotation } from "./pending-reporter.ts";
import { World, findStep } from "./steps.ts";
import "./steps-dishes.ts";
import "./steps-planning.ts";
import "./steps-formats.ts";
import "./steps-shopping.ts";

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
  test(name, { tag: scenario.tags, annotation: { type: scenarioAnnotation, description: key } }, async ({ page, request, baseURL }, testInfo) => {
    const format = testInfo.project.metadata["format"] as Format;
    test.skip(!formatsOf(scenario.tags).includes(format), `not a scenario of the format ${format}`);
    const world = new World(page, request, baseURL ?? "");
    const run = async () => {
      for (const s of scenario.steps) {
        const definition = findStep(s.text);
        if (definition === undefined) {
          throw new Error(`undefined step: ${s.text}`);
        }
        world.table = s.table ?? [];
        world.stepType = s.type;
        await test.step(s.text, () => definition.fn(world, ...definition.args));
      }
    };
    if (!pending.has(key)) {
      await run();
      return;
    }
    try {
      await run();
    } catch (err) {
      testInfo.annotations.push({ type: pendingAnnotation, description: `fails, as listed in acceptance/pending.txt: ${err instanceof Error ? err.message.split("\n")[0] : String(err)}` });
      return;
    }
    testInfo.annotations.push({ type: pendingAnnotation, description: pendingPasses });
  });
}
