// Waiting for a smooth scroll to end, shared by the steps of the scenarios and the screen
// tests: a scroll started while another one runs would settle on another day (scroll-snap).
import { type Locator, expect } from "@playwright/test";

/**
 * Does an action that scrolls an element, then waits for the scroll to start and to end: its
 * position has changed, then no longer changes. Waiting only for a still position would end at
 * once on a busy machine, the smooth scroll not started yet.
 */
export async function scrolledBy(element: Locator, action: () => Promise<void>): Promise<void> {
  const start = await element.evaluate((e) => e.scrollLeft);
  await action();
  await expect.poll(() => element.evaluate((e) => e.scrollLeft)).not.toBe(start);
  let last = -1;
  await expect
    .poll(async () => {
      const now = await element.evaluate((e) => e.scrollLeft);
      const still = now === last;
      last = now;
      return still;
    })
    .toBe(true);
}
