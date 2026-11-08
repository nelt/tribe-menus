// Waiting for a smooth scroll to end, shared by the steps of the scenarios and the screen
// tests: a scroll started while another one runs would settle on another day (scroll-snap).
import { type Locator, expect } from "@playwright/test";

/** Waits for the scroll of an element to end: its position no longer changes. */
export async function scrollEnded(element: Locator): Promise<void> {
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
