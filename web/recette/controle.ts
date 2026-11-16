// The control points of the video (plan recette-auto, D9 and step 12): read in the plan of the
// acceptance run of the version, docs/recettes/vX.Y.Z.md, they are matched with the steps of
// the journeys that ran, and listed on a local page, each with its capture and the video at
// the instant of its step.

/** A point of the table « Points de contrôle de la vidéo », for one project. */
export interface ControlPoint {
  /** V1, V2… */
  id: string;
  /** Number of the entry of the table of the changes. */
  entry: string;
  /** File of the journey, web/recette/<file>. */
  file: string;
  project: string;
  /** Title of the step (test.step), between « and ». */
  step: string;
  /** What to look at. */
  look: string;
}

const section = "## Points de contrôle de la vidéo";

/**
 * Reads the control points of a plan of the acceptance run: one per row of the table of its
 * section and per project named in the row. Throws on a row whose journey, project or step
 * cannot be read, or that still holds the « … » of the template.
 */
export function parseControlPoints(markdown: string): ControlPoint[] {
  const lines = markdown.split("\n");
  const start = lines.findIndex((l) => l.trim() === section);
  if (start < 0) {
    throw new Error(`no section « ${section.slice(3)} » in the plan`);
  }
  const rows: string[] = [];
  for (const line of lines.slice(start + 1)) {
    if (line.startsWith("## ")) {
      break;
    }
    if (line.trim().startsWith("|")) {
      rows.push(line.trim());
    }
  }
  const points: ControlPoint[] = [];
  const errors: string[] = [];
  // The header and the separator of the table come first.
  for (const row of rows.slice(2)) {
    const cells = row.replace(/^\||\|$/g, "").split("|").map((c) => c.trim());
    const [id = "", entry = "", where = "", stepCell = "", look = ""] = cells;
    const names = [...where.matchAll(/`([^`]+)`/g)].map((m) => m[1] ?? "");
    const file = names.find((n) => n.endsWith(".ts"));
    const projects = names.filter((n) => !n.endsWith(".ts"));
    const step = /«\s*(.+?)\s*»/.exec(stepCell)?.[1];
    if (cells.length < 5 || file === undefined || projects.length === 0 || step === undefined || [file, step].some((s) => s.includes("…"))) {
      errors.push(`${id || row}: journey, project or step unreadable`);
      continue;
    }
    for (const project of projects) {
      points.push({ id, entry, file, project, step, look });
    }
  }
  if (errors.length > 0) {
    throw new Error(`control points: ${errors.join("; ")}`);
  }
  return points;
}

/** A step that ran, as the control page needs it. */
export interface RanStep {
  title: string;
  /** Seconds from the start of the video of its test. */
  offset: number;
  /** What it does and checks, in French Gherkin, one line each. */
  gherkin?: string[];
  /** It failed: the test stopped there. */
  failed?: boolean;
  /** Path of its capture, relative to the control page, if any. */
  capture?: string;
}

/** A test that ran, in a project. */
export interface RanTest {
  file: string;
  project: string;
  title: string;
  /** Its scenario, in French. */
  scenario?: string;
  status: string;
  /** Path of its video, relative to the control page, if any. */
  video?: string;
  steps: RanStep[];
}

/** A control point with what shows it. */
export interface Shown {
  point: ControlPoint;
  test: RanTest;
  step: RanStep;
}

/**
 * Finds the step of each point among the tests that ran; a point whose step is in no test of
 * its file and project is missing: designated by the plan, it was never played.
 */
export function matchPoints(points: readonly ControlPoint[], tests: readonly RanTest[]): { shown: Shown[]; missing: ControlPoint[] } {
  const shown: Shown[] = [];
  const missing: ControlPoint[] = [];
  for (const point of points) {
    let found: Shown | undefined;
    for (const test of tests) {
      const step = test.file === point.file && test.project === point.project ? test.steps.find((s) => s.title === point.step) : undefined;
      if (step !== undefined) {
        found = { point, test, step };
        break;
      }
    }
    if (found === undefined) {
      missing.push(point);
    } else {
      shown.push(found);
    }
  }
  return { shown, missing };
}

/** Seconds of video shown before the step, to see the action and not only its result. */
const lead = 1;

function escape(s: string): string {
  return s.replace(/[&<>"']/g, (c) => `&#${c.charCodeAt(0)};`);
}

/** The instant of the video to open at for a step. */
function instant(step: RanStep): number {
  return Math.max(0, step.offset - lead);
}

/** A Gherkin line, its keyword in bold. */
function gherkinLine(line: string): string {
  const m = /^(Étant donné que|Étant donné qu'|Étant donné|Quand|Alors|Et|Mais)\b(.*)$/s.exec(line);
  return m === null ? escape(line) : `<strong>${escape(m[1] ?? "")}</strong>${escape(m[2] ?? "")}`;
}

/** The steps of a test as a scenario: each with its Gherkin and, with a video, a link to its instant. */
function scenarioSteps(test: RanTest, video: string | undefined, current?: RanStep): string {
  // The steps without Gherkin are tooling (a box emptied, data of a stopped run removed), not
  // the application: left out, unless one of them failed.
  const items = test.steps.filter((step) => step.gherkin !== undefined || step.failed === true || step === current).map((step) => {
    const lines = (step.gherkin ?? [step.title]).map((l) => `<span class="line">${gherkinLine(l)}</span>`).join("");
    const classes = [step === current ? "current" : "", step.failed === true ? "failed" : ""].filter((c) => c !== "").join(" ");
    const at = video === undefined ? "" : `<button type="button" class="at" data-video="${escape(video)}" data-t="${instant(step).toFixed(1)}" title="Voir la vidéo à ce moment">▶ ${step.offset.toFixed(1)} s</button>`;
    return `<li${classes === "" ? "" : ` class="${classes}"`}><span class="lines">${lines}</span><span class="name">${escape(step.title)}</span>${at}</li>`;
  });
  const end = test.status === "passed" ? "" : `<li class="failed"><span class="lines"><span class="line"><strong>Arrêté</strong> : ${escape(test.status)}${test.steps.some((s) => s.failed) ? ", à l'étape marquée" : ""}</span></span></li>`;
  return `<ol class="steps">${items.join("")}${end}</ol>`;
}

/** The mark of an outcome. */
function outcome(status: string): string {
  return status === "passed" ? `<span class="ok">✓ passé</span>` : status === "skipped" ? `<span class="skip">— ignoré</span>` : `<span class="ko">✗ ${escape(status)}</span>`;
}

/**
 * The local control page: what ran, by journey and project; the points to check, each with
 * what to look at, its scenario with the designated step, its capture and its video; then each
 * scenario as it ran. A step's time puts the video of its block at that instant.
 */
export function controlPage(version: string, shown: readonly Shown[], missing: readonly ControlPoint[], tests: readonly RanTest[]): string {
  const files = [...new Set(tests.map((t) => t.file))];
  const projects = [...new Set(tests.map((t) => t.project))];
  const byFile = (file: string) => tests.filter((t) => t.file === file);
  const scenarioOf = (file: string) => byFile(file).find((t) => t.scenario !== undefined)?.scenario ?? file;

  const summary = `<table class="summary">
    <thead><tr><th>Scénario</th>${projects.map((p) => `<th>${escape(p)}</th>`).join("")}</tr></thead>
    <tbody>${files
      .map(
        (file) => `<tr><th><a href="#${escape(file)}">${escape(scenarioOf(file))}</a><br><code>${escape(file)}</code></th>${projects
          .map((project) => {
            const t = byFile(file).find((x) => x.project === project);
            return `<td>${t === undefined ? "" : `${outcome(t.status)}${t.video === undefined ? "" : ` · <a href="${escape(t.video)}">vidéo</a>`}`}</td>`;
          })
          .join("")}</tr>`,
      )
      .join("")}</tbody>
  </table>`;

  const absent = missing.map((p) => `<li>${escape(p.id)} : <code>${escape(p.file)}</code>, ${escape(p.project)}, étape « ${escape(p.step)} »</li>`);

  const points = shown.map(({ point, test, step }, i) => {
    const id = `v${i}`;
    return `
    <section class="point">
      <h3>${escape(point.id)} · ${escape(point.project)} <span class="entry">entrée ${escape(point.entry)}</span> ${outcome(test.status)}</h3>
      <p class="look"><span class="label">À regarder</span> ${escape(point.look)}</p>
      <div class="point-body">
        <div class="scenario">
          <p class="scenario-title">${escape(test.scenario ?? test.title)} <code>${escape(point.file)}</code></p>
          ${scenarioSteps(test, test.video === undefined ? undefined : id, step)}
        </div>
        <div class="proof">
          ${test.video === undefined ? "<p>Pas de vidéo.</p>" : `<video id="${id}" controls preload="metadata" src="${escape(`${test.video}#t=${instant(step).toFixed(1)}`)}"></video>`}
          ${step.capture === undefined ? "<p>Pas de capture.</p>" : `<a href="${escape(step.capture)}"><img src="${escape(step.capture)}" alt="Capture de l'étape « ${escape(point.step)} »"></a>`}
        </div>
      </div>
    </section>`;
  });

  const scenarios = files.map((file, i) => {
    const runs = byFile(file);
    const first = runs.find((t) => t.status === "passed") ?? runs[0];
    const id = `s${i}`;
    return `
    <section class="journey" id="${escape(file)}">
      <h3>${escape(scenarioOf(file))} <code>${escape(file)}</code></h3>
      <p class="runs">${runs.map((t) => `${escape(t.project)} ${outcome(t.status)}${t.video === undefined ? "" : ` (<a href="${escape(t.video)}">vidéo</a>)`}`).join(" · ")}</p>
      <div class="point-body">
        <div class="scenario">${first === undefined ? "" : scenarioSteps(first, first.video === undefined ? undefined : id)}</div>
        <div class="proof">${first?.video === undefined ? "" : `<p class="small">Vidéo de ${escape(first.project)}</p><video id="${id}" controls preload="metadata" src="${escape(first.video)}"></video>`}</div>
      </div>
      ${runs
        .filter((t) => t !== first && t.status !== "passed" && t.status !== "skipped")
        .map((t) => `<p class="small ko">En échec dans ${escape(t.project)} :</p>${scenarioSteps(t, undefined)}`)
        .join("")}
    </section>`;
  });

  return `<!doctype html>
<html lang="fr">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Contrôle de la vidéo ${escape(version)}</title>
  <style>
    body { font: 15px/1.5 system-ui, sans-serif; margin: 0 auto; padding: 16px; max-width: 1500px; color: #222; background: #fff; }
    h1 { margin: 0 0 4px; } h2 { margin: 32px 0 8px; border-bottom: 2px solid #222; } h3 { font-size: 1.1em; margin: 0 0 6px; }
    code { font-size: .85em; color: #555; }
    .summary { border-collapse: collapse; margin: 8px 0; } .summary th, .summary td { border: 1px solid #ccc; padding: 4px 8px; text-align: left; vertical-align: top; }
    .ok { color: #1a7f37; font-weight: 600; } .ko { color: #b00; font-weight: 600; } .skip { color: #777; }
    .point, .journey { border-top: 1px solid #ccc; padding: 12px 0 20px; }
    .entry { font-weight: normal; color: #555; font-size: .9em; }
    .look { background: #fff6d6; border-left: 4px solid #d4a000; padding: 6px 10px; margin: 6px 0 10px; }
    .label { font-weight: 700; margin-right: 6px; }
    .point-body { display: flex; flex-wrap: wrap; gap: 20px; align-items: flex-start; }
    .scenario { flex: 1 1 420px; min-width: 0; } .scenario-title { font-weight: 600; margin: 0 0 4px; }
    .steps { margin: 0; padding-left: 22px; } .steps li { margin: 2px 0; padding: 3px 6px; border-radius: 4px; }
    .steps .line { display: block; } .steps .name { display: block; font-size: .8em; color: #777; }
    .steps li.current { background: #fff6d6; outline: 2px solid #d4a000; } .steps li.failed { background: #fde8e8; }
    .at { font: inherit; font-size: .85em; border: 1px solid #aaa; background: #f6f6f6; border-radius: 4px; cursor: pointer; padding: 0 6px; margin-top: 2px; }
    .proof { flex: 1 1 420px; display: flex; flex-wrap: wrap; gap: 12px; align-items: flex-start; }
    .proof img, .proof video { max-width: min(100%, 640px); max-height: 75vh; border: 1px solid #ccc; }
    .small { font-size: .9em; color: #555; margin: 0 0 4px; } .missing { color: #b00; }
  </style>
</head>
<body>
  <h1>Contrôle de la vidéo ${escape(version)}</h1>
  <p>${shown.length} point(s) à contrôler ; verdicts dans le plan de recette. Les parcours se lisent comme des scénarios ; « ▶ » place la vidéo du bloc au moment de l'étape (${lead} s avant).</p>
  <h2>Ce qui a été joué</h2>
  ${summary}
  ${absent.length === 0 ? "" : `<h2 class="missing">Étapes désignées introuvables</h2><ul class="missing">${absent.join("")}</ul>`}
  <h2>Points à contrôler</h2>
  ${points.length === 0 ? "<p>Aucun point désigné par le plan de recette.</p>" : points.join("")}
  <h2>Scénarios joués</h2>
  ${scenarios.join("")}
  <script>
    document.addEventListener("click", (e) => {
      const at = e.target.closest("button.at");
      const video = at && document.getElementById(at.dataset.video);
      if (video) { video.currentTime = Number(at.dataset.t); video.play(); video.scrollIntoView({ block: "nearest" }); }
    });
  </script>
</body>
</html>
`;
}
