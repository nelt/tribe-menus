// Reporter of Playwright that writes the control page of a run (plan recette-auto, step 12):
// the control points of the plan of the acceptance run, each with its capture and its video at
// the instant of its step. A point whose step ran in no journey fails the run.
import { mkdir, writeFile } from "node:fs/promises";
import { basename, join, relative } from "node:path";
import type { FullResult, Reporter, TestCase, TestResult, TestStep } from "@playwright/test/reporter";
import { type ControlPoint, type RanStep, type RanTest, controlPage, matchPoints } from "./controle.ts";
import { captureName, gherkinName, scenarioAnnotation, videoStartAnnotation } from "./test.ts";

interface Options {
  /** docs/recettes/vX.Y.Z.md */
  plan: string;
  /** Its control points, read with the settings, before the run (recetteSettings). */
  points: ControlPoint[];
  /** Directory of the run, where the page and the captures are written. */
  outputDir: string;
}

/** The steps of test.step, nested ones included, in order. */
function flatten(steps: readonly TestStep[]): TestStep[] {
  return steps.flatMap((s) => (s.category === "test.step" ? [s, ...flatten(s.steps)] : flatten(s.steps)));
}

export default class ControlReporter implements Reporter {
  readonly #options: Options;
  readonly #tests: RanTest[] = [];
  readonly #captures: { test: RanTest; step: RanStep; body: Buffer }[] = [];

  constructor(options: Options) {
    this.#options = options;
  }

  onTestEnd(test: TestCase, result: TestResult) {
    const start = Number(result.annotations.find((a) => a.type === videoStartAnnotation)?.description ?? result.startTime.getTime());
    const video = result.attachments.find((a) => a.name === "video" && a.path !== undefined)?.path;
    const ran: RanTest = {
      file: basename(test.location.file),
      project: test.parent.project()?.name ?? "",
      title: test.title,
      status: result.status,
      steps: [],
    };
    if (video !== undefined) {
      ran.video = relative(this.#options.outputDir, video);
    }
    const scenario = [...test.annotations, ...result.annotations].find((a) => a.type === scenarioAnnotation)?.description;
    if (scenario !== undefined) {
      ran.scenario = scenario;
    }
    for (const s of flatten(result.steps)) {
      const step: RanStep = { title: s.title, offset: (s.startTime.getTime() - start) / 1000 };
      if (s.error !== undefined) {
        step.failed = true;
      }
      // Attachments are made by actions of the step, steps of their own.
      const attached = [s, ...s.steps.filter((c) => c.category !== "test.step")].flatMap((c) => c.attachments);
      const gherkin = attached.find((a) => a.name === gherkinName)?.body;
      if (gherkin !== undefined) {
        step.gherkin = gherkin.toString("utf8").split("\n");
      }
      const body = attached.filter((a) => a.name === captureName).at(-1)?.body;
      if (body !== undefined) {
        this.#captures.push({ test: ran, step, body });
      }
      ran.steps.push(step);
    }
    this.#tests.push(ran);
  }

  async onEnd(result: FullResult): Promise<{ status: FullResult["status"] } | undefined> {
    const { plan, points, outputDir } = this.#options;
    const version = basename(plan, ".md");
    const { shown, missing } = matchPoints(points, this.#tests);
    // Only the captures of the designated steps are written, named after their point.
    await mkdir(join(outputDir, "captures"), { recursive: true });
    for (const s of shown) {
      const capture = this.#captures.find((c) => c.test === s.test && c.step === s.step);
      if (capture !== undefined) {
        const name = join("captures", `${s.point.id}-${s.point.project}.png`);
        await writeFile(join(outputDir, name), capture.body);
        s.step.capture = name;
      }
    }
    const page = join(outputDir, "controle.html");
    await writeFile(page, controlPage(version, shown, missing, this.#tests));
    console.log(`\nControl page: ${page}`);
    if (missing.length === 0) {
      return undefined;
    }
    console.error(`\nDesignated by ${basename(plan)}, these steps ran in no journey:`);
    for (const p of missing) {
      console.error(`  ${p.id}: ${p.file}, ${p.project}, « ${p.step} »`);
    }
    return { status: result.status === "interrupted" ? "interrupted" : "failed" };
  }

  printsToStdio() {
    return false;
  }
}
