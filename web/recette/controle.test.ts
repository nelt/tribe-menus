import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";
import { type RanTest, controlPage, matchPoints, parseControlPoints } from "./controle.ts";

const plan = (rows: string) => `# Plan de recette : v0.6.0

## Changements et voies de preuve

| # | Changement |
| --- | --- |
| 1 | \`courses-suite/D\` |

## Points de contrôle de la vidéo

| Point | Entrée | Parcours, projet | Étape (\`test.step\`) | Ce qu'il faut regarder |
| --- | --- | --- | --- | --- |
${rows}

## Séance manuelle

| M1 | x | \`autre.spec.ts\`, \`phone-chromium\` | « ailleurs » | hors section |
`;

test("parseControlPoints reads one point per row and per project", () => {
  const points = parseControlPoints(
    plan(
      "| V1 | 3 | `courses.spec.ts`, `phone-chromium` et `computer-chromium` | « Liste recalculée » | le bandeau vert « Liste recalculée. », comme sur `Doux-ListeSuite` |\n" +
        "| V2 | 4 | `connexion.setup.ts`, `phone-chromium` | « Signed in » | l'écran Menus |",
    ),
  );
  assert.deepEqual(points, [
    { id: "V1", entry: "3", file: "courses.spec.ts", project: "phone-chromium", step: "Liste recalculée", look: "le bandeau vert « Liste recalculée. », comme sur `Doux-ListeSuite`" },
    { id: "V1", entry: "3", file: "courses.spec.ts", project: "computer-chromium", step: "Liste recalculée", look: "le bandeau vert « Liste recalculée. », comme sur `Doux-ListeSuite`" },
    { id: "V2", entry: "4", file: "connexion.setup.ts", project: "phone-chromium", step: "Signed in", look: "l'écran Menus" },
  ]);
});

test("parseControlPoints accepts a plan without point", () => {
  assert.deepEqual(parseControlPoints(plan("")), []);
});

test("parseControlPoints refuses a row it cannot read, and the template", () => {
  const cases = [
    "| V1 | 3 | courses.spec.ts, phone-chromium | « Liste » | x |",
    "| V1 | 3 | `courses.spec.ts` | « Liste » | x |",
    "| V1 | 3 | `courses.spec.ts`, `phone-chromium` | Liste | x |",
  ];
  for (const row of cases) {
    assert.throws(() => parseControlPoints(plan(row)), /V1: journey, project or step unreadable/, row);
  }
  const template = readFileSync(new URL("../../docs/recettes/_template.md", import.meta.url), "utf8");
  assert.throws(() => parseControlPoints(template), /V1: journey, project or step unreadable/);
  assert.throws(() => parseControlPoints("# Plan\n"), /no section/);
});

const ran: RanTest[] = [
  {
    file: "courses.spec.ts",
    project: "phone-chromium",
    title: "recompute",
    scenario: "Liste périmée et <recalcul>",
    status: "passed",
    video: "resultats/a/video.webm",
    steps: [
      { title: "List up to date", offset: 0.5 },
      { title: "Stale list", offset: 8.2, gherkin: ["Quand je reviens sur l'application", "Alors un bandeau s'affiche"] },
      { title: "Liste recalculée", offset: 12.34, gherkin: ["Quand je touche « Recalculer »", "Et j'attends"], capture: "captures/V1-phone-chromium.png" },
    ],
  },
  { file: "courses.spec.ts", project: "computer-chromium", title: "recompute", status: "failed", steps: [{ title: "Stale list", offset: 3, gherkin: ["Quand je reviens"], failed: true }] },
];

test("matchPoints finds the step in the test of the file and the project, or reports it missing", () => {
  const points = parseControlPoints(plan("| V1 | 3 | `courses.spec.ts`, `phone-chromium` et `computer-chromium` | « Liste recalculée » | bandeau |"));
  const { shown, missing } = matchPoints(points, ran);
  assert.deepEqual(
    shown.map((s) => [s.point.project, s.step.offset]),
    [["phone-chromium", 12.34]],
  );
  assert.deepEqual(
    missing.map((p) => p.project),
    ["computer-chromium"],
  );
});

test("controlPage shows each point with its scenario, the designated step, its capture and its video", () => {
  const points = parseControlPoints(plan("| V1 | 3 | `courses.spec.ts`, `phone-chromium` et `computer-chromium` | « Liste recalculée » | <b>bandeau</b> & co |"));
  const { shown, missing } = matchPoints(points, ran);
  const page = controlPage("v0.6.0", shown, missing, ran);
  // The plan and the scenario are escaped.
  assert.match(page, /&#60;b&#62;bandeau&#60;\/b&#62; &#38; co/);
  assert.doesNotMatch(page, /<b>bandeau/);
  assert.match(page, /Liste périmée et &#60;recalcul&#62;/);
  // The video opens a second before the step; each step puts it at its instant.
  assert.match(page, /<video id="v0" controls preload="metadata" src="resultats\/a\/video.webm#t=11.3">/);
  assert.match(page, /<button type="button" class="at" data-video="v0" data-t="7.2"[^>]*>▶ 8.2 s<\/button>/);
  assert.match(page, /<img src="captures\/V1-phone-chromium.png"/);
  // The designated step stands out, its keywords in bold; a step without Gherkin is tooling, left out.
  assert.match(page, /<li class="current"><span class="lines"><span class="line"><strong>Quand<\/strong> je touche « Recalculer »<\/span><span class="line"><strong>Et<\/strong> j&#39;attends<\/span>/);
  assert.doesNotMatch(page, /List up to date/);
  // What ran, what is missing, what failed.
  assert.match(page, /<th>phone-chromium<\/th><th>computer-chromium<\/th>/);
  assert.match(page, /Étapes désignées introuvables.*V1 : <code>courses.spec.ts<\/code>, computer-chromium/s);
  assert.match(page, /<li class="failed"><span class="lines"><span class="line"><strong>Quand<\/strong> je reviens/);
  assert.match(page, /✗ failed/);
});
