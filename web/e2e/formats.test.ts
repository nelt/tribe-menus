import assert from "node:assert/strict";
import { test } from "node:test";
import { formatsOf, passedPending } from "./formats.ts";

test("formatsOf", () => {
  assert.deepEqual(formatsOf(["@C1", "@ui"]), ["compact", "moyen", "etendu"]);
  assert.deepEqual(formatsOf(["@ENF-03", "@ui", "@compact"]), ["compact"]);
  assert.deepEqual(formatsOf(["@etendu", "@ui", "@moyen"]), ["moyen", "etendu"]);
});

const projects: Record<string, string[]> = {
  "a.feature: Partout": ["phone", "tablet", "computer"],
  "a.feature: Grand": ["tablet", "computer"],
};
const projectsOf = (key: string) => projects[key] ?? [];

test("passedPending: passed in every project of its formats", () => {
  const outcomes = ["tablet", "computer"].map((project) => ({ key: "a.feature: Grand", project, passed: true }));
  assert.deepEqual(passedPending(outcomes, projectsOf), ["a.feature: Grand"]);
});

test("passedPending: failed in one project, still pending", () => {
  const outcomes = [
    { key: "a.feature: Grand", project: "tablet", passed: true },
    { key: "a.feature: Grand", project: "computer", passed: false },
  ];
  assert.deepEqual(passedPending(outcomes, projectsOf), []);
});

test("passedPending: an example failed, still pending", () => {
  const outcomes = [
    { key: "a.feature: Grand", project: "tablet", passed: true },
    { key: "a.feature: Grand", project: "computer", passed: true },
    { key: "a.feature: Grand", project: "computer", passed: false },
  ];
  assert.deepEqual(passedPending(outcomes, projectsOf), []);
});

test("passedPending: not run in all its projects (make e2e), not judged", () => {
  const outcomes = [{ key: "a.feature: Partout", project: "phone", passed: true }];
  assert.deepEqual(passedPending(outcomes, projectsOf), []);
});
