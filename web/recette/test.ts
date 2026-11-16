// The test of the journeys of the acceptance run (ADR 0032, point 11): proofs made to be
// checked. A journey is a scenario, titled in French by the annotation « scénario », cut into
// named steps (step). Each step tells what it does and checks in French Gherkin, as the
// .feature do (docs/specs/conventions-gherkin.md), and, on the page, ends with a short pause,
// so that the video shows the state reached and not only the transition, and a full page
// capture, shown by the control page when the plan of the acceptance run designates the step.
// The name of a step is the key of the control points; its Gherkin is for the reader.
import { type Page, test as base, expect } from "@playwright/test";

export { expect };

/** Annotation holding the time the video of the test starts, for the control page. */
export const videoStartAnnotation = "video-start";
/** Name of the capture attached at the end of each step. */
export const captureName = "capture";
/** Name of the Gherkin text attached at the start of each step. */
export const gherkinName = "gherkin";
/** Annotation holding the French title of the scenario of a journey. */
export const scenarioAnnotation = "scénario";

/** The details of a test: its scenario, in French. */
export function scenario(title: string) {
  return { annotation: { type: scenarioAnnotation, description: title } };
}

/** Pause after the last action of a step, before its capture. */
const pause = 800;

export const test = base.extend<{ page: Page }>({
  // The video starts with the context of the page, just before this fixture.
  page: async ({ page }, use, testInfo) => {
    testInfo.annotations.push({ type: videoStartAnnotation, description: String(Date.now()) });
    await use(page);
  },
});

/**
 * Runs a named step of a journey, described by its Gherkin lines; on a page, then pauses and
 * attaches a full page capture. Without page (another device, a tool), neither pause nor capture.
 */
export async function step<T>(page: Page | undefined, title: string, gherkin: readonly string[], body: () => Promise<T>): Promise<T> {
  return test.step(title, async () => {
    await test.info().attach(gherkinName, { body: gherkin.join("\n"), contentType: "text/plain; charset=utf-8" });
    const result = await body();
    if (page !== undefined) {
      await page.waitForTimeout(pause);
      await test.info().attach(captureName, { body: await page.screenshot({ fullPage: true }), contentType: "image/png" });
    }
    return result;
  });
}
