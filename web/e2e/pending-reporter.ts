// Reporter of Playwright that judges acceptance/pending.txt across the projects (ADR 0030,
// point 8): scenarios.spec.ts lets a pending scenario fail without failing the run, and notes
// whether it passed; this reporter fails the run when a pending scenario passed in every project
// of its formats. make e2e-formats, which runs all the projects, so judges both ways; make e2e
// only the scenarios it ran in all their projects.
import type { FullConfig, FullResult, Reporter, TestCase, TestResult } from "@playwright/test/reporter";
import { type PendingOutcome, formatsOf, passedPending } from "./formats.ts";

/** The annotations written by scenarios.spec.ts. */
export const scenarioAnnotation = "scenario";
export const pendingAnnotation = "pending";
export const pendingPasses = "passes";

export default class PendingReporter implements Reporter {
  #config: FullConfig | undefined;
  readonly #outcomes: (PendingOutcome & { id: string })[] = [];
  readonly #tags = new Map<string, readonly string[]>();

  onBegin(config: FullConfig) {
    this.#config = config;
  }

  onTestEnd(test: TestCase, result: TestResult) {
    // The annotations of this attempt, or those of the test in older versions of Playwright.
    const annotations = [...result.annotations, ...test.annotations];
    const key = annotations.find((a) => a.type === scenarioAnnotation)?.description;
    const pending = annotations.find((a) => a.type === pendingAnnotation)?.description;
    if (key === undefined || pending === undefined || result.status === "skipped") {
      return;
    }
    // A retry replaces the outcome of the attempt before it.
    const project = test.parent.project()?.name ?? "";
    const previous = this.#outcomes.findIndex((o) => o.key === key && o.project === project && o.id === test.id);
    const outcome = { key, project, id: test.id, passed: result.status === "passed" && pending === pendingPasses };
    if (previous >= 0) {
      this.#outcomes[previous] = outcome;
    } else {
      this.#outcomes.push(outcome);
    }
    this.#tags.set(key, test.tags);
  }

  async onEnd(result: FullResult): Promise<{ status: FullResult["status"] } | undefined> {
    const projects = this.#config?.projects ?? [];
    const projectsOf = (key: string) => {
      const wanted = formatsOf(this.#tags.get(key) ?? []);
      return projects.filter((p) => wanted.includes(p.metadata["format"] as never)).map((p) => p.name);
    };
    const passed = passedPending(this.#outcomes, projectsOf);
    if (passed.length === 0) {
      return undefined;
    }
    console.error(`\nListed in acceptance/pending.txt, these scenarios pass in all the projects of their formats: remove them from the list.`);
    for (const key of passed) {
      console.error(`  ${key}`);
    }
    return { status: result.status === "interrupted" ? "interrupted" : "failed" };
  }

  printsToStdio() {
    return false;
  }
}
