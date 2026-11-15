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
  /** Path of its capture, relative to the control page, if any. */
  capture?: string;
}

/** A test that ran, in a project. */
export interface RanTest {
  file: string;
  project: string;
  title: string;
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

/** The local control page: the points, then the tests that failed, with their videos. */
export function controlPage(version: string, shown: readonly Shown[], missing: readonly ControlPoint[], tests: readonly RanTest[]): string {
  const video = (path: string | undefined, offset: number) =>
    path === undefined ? "<p>Pas de vidéo.</p>" : `<video controls preload="metadata" src="${escape(`${path}#t=${Math.max(0, offset - lead).toFixed(1)}`)}"></video>`;
  const points = shown.map(
    ({ point, test, step }) => `
    <section>
      <h2>${escape(point.id)} · entrée ${escape(point.entry)} · ${escape(point.project)}</h2>
      <p class="look">${escape(point.look)}</p>
      <p class="where"><code>${escape(point.file)}</code>, « ${escape(test.title)} », étape « ${escape(point.step)} », à ${step.offset.toFixed(1)} s · ${escape(test.status)}</p>
      <div class="proof">
        ${step.capture === undefined ? "<p>Pas de capture.</p>" : `<a href="${escape(step.capture)}"><img src="${escape(step.capture)}" alt="Capture de l'étape « ${escape(point.step)} »"></a>`}
        ${video(test.video, step.offset)}
      </div>
    </section>`,
  );
  const absent = missing.map((p) => `<li>${escape(p.id)} : <code>${escape(p.file)}</code>, ${escape(p.project)}, étape « ${escape(p.step)} »</li>`);
  const failed = tests
    .filter((t) => t.status !== "passed" && t.status !== "skipped")
    .map((t) => `<li><code>${escape(t.file)}</code>, ${escape(t.project)} : « ${escape(t.title)} » (${escape(t.status)})${t.video === undefined ? "" : `, <a href="${escape(t.video)}">vidéo</a>`}</li>`);
  return `<!doctype html>
<html lang="fr">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Contrôle de la vidéo ${escape(version)}</title>
  <style>
    body { font: 15px/1.5 system-ui, sans-serif; margin: 0 auto; padding: 16px; max-width: 1400px; color: #222; background: #fff; }
    section { border-top: 1px solid #ccc; padding: 8px 0 16px; }
    h2 { font-size: 1.1em; margin: 8px 0 4px; }
    .look { font-weight: 600; margin: 4px 0; }
    .where { color: #555; margin: 4px 0 8px; }
    .proof { display: flex; flex-wrap: wrap; gap: 16px; align-items: flex-start; }
    .proof img, .proof video { max-width: min(100%, 640px); max-height: 80vh; border: 1px solid #ccc; }
    .missing { color: #a00; }
  </style>
</head>
<body>
  <h1>Contrôle de la vidéo ${escape(version)}</h1>
  <p>${shown.length} point(s) à contrôler ; verdicts dans le plan de recette. Une vidéo commence ${lead} s avant son étape.</p>
  ${absent.length === 0 ? "" : `<h2 class="missing">Étapes désignées introuvables</h2><ul class="missing">${absent.join("")}</ul>`}
  ${failed.length === 0 ? "" : `<h2 class="missing">Parcours en échec</h2><ul class="missing">${failed.join("")}</ul>`}
  ${points.join("")}
</body>
</html>
`;
}
