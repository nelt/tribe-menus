// The test of the journeys of the acceptance run (ADR 0032, point 11): proofs made to be
// checked. A journey is cut into named steps (step), each ending with a short pause, so that
// the video shows the state reached and not only the transition, and a full page capture,
// shown by the control page when the plan of the acceptance run designates the step.
import { type Page, test as base, expect } from "@playwright/test";

export { expect };

/** Annotation holding the time the video of the test starts, for the control page. */
export const videoStartAnnotation = "video-start";
/** Name of the capture attached at the end of each step. */
export const captureName = "capture";

/** Pause after the last action of a step, before its capture. */
const pause = 800;

export const test = base.extend<{ page: Page }>({
  // The video starts with the context of the page, just before this fixture.
  page: async ({ page }, use, testInfo) => {
    testInfo.annotations.push({ type: videoStartAnnotation, description: String(Date.now()) });
    await use(page);
  },
});

/** Runs a named step of a journey, then pauses and attaches a full page capture. */
export async function step<T>(page: Page, title: string, body: () => Promise<T>): Promise<T> {
  return test.step(title, async () => {
    const result = await body();
    await page.waitForTimeout(pause);
    await test.info().attach(captureName, { body: await page.screenshot({ fullPage: true }), contentType: "image/png" });
    return result;
  });
}
