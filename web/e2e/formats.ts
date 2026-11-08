// Formats of the scenarios @ui and judgement of acceptance/pending.txt across the projects
// (ADR 0030, points 6 and 8): each project of Playwright has a format; a scenario tagged
// @compact, @moyen or @etendu is for the projects of these formats only, one without such a tag
// for all. A pending scenario is passed when it passes in every project of its formats.
import type { Format } from "../src/layout.ts";

export const formats: readonly Format[] = ["compact", "moyen", "etendu"];

/** The formats of a scenario, from its tags: all of them without a tag of format. */
export function formatsOf(tags: readonly string[]): Format[] {
  const tagged = formats.filter((f) => tags.includes(`@${f}`));
  return tagged.length === 0 ? [...formats] : tagged;
}

/** The outcome of a pending scenario, or of an example of it, in a project. */
export interface PendingOutcome {
  key: string;
  project: string;
  passed: boolean;
}

/**
 * The pending scenarios to take out of the list: those that passed in every project of their
 * formats, each of their examples included. A scenario not run in one of these projects (make
 * e2e runs part of them) is not judged: it has not been tried everywhere.
 */
export function passedPending(outcomes: readonly PendingOutcome[], projectsOf: (key: string) => readonly string[]): string[] {
  const byKey = new Map<string, PendingOutcome[]>();
  for (const o of outcomes) {
    byKey.set(o.key, [...(byKey.get(o.key) ?? []), o]);
  }
  const passed: string[] = [];
  for (const [key, list] of byKey) {
    const ran = new Set(list.map((o) => o.project));
    if (projectsOf(key).every((p) => ran.has(p)) && list.every((o) => o.passed)) {
      passed.push(key);
    }
  }
  return passed.sort();
}
