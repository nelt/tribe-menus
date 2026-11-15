// Journeys of the acceptance run against the staging instance (ADR 0032, points 6, 9 to 11;
// plan recette-auto, D8 to D11), launched by make recette-auto from the Dev Container: no
// server of the tests, the project setup signs in once, the four projects take up its session.
// Everything is filmed at the size of the window, traced and captured, in the volume of the
// videos; no retry, a journey that fails fails the run.
import { join } from "node:path";
import { type PlaywrightTestOptions, type Project, type ViewportSize, defineConfig, devices } from "@playwright/test";
import type { Format } from "./src/layout.ts";
import { recetteSettings } from "./recette/config.ts";

const settings = recetteSettings();

const tabletLandscape = { width: 1180, height: 820 };
const computer = { width: 1440, height: 900 };

/** A project of a format, filmed at the size of its window (ADR 0032, point 11). */
function project(name: string, format: Format, use: Partial<PlaywrightTestOptions> & { viewport: ViewportSize; screen?: ViewportSize }, more: Partial<Project> = {}): Project {
  return { name, metadata: { format }, use: { ...use, video: { mode: "on", size: use.viewport } }, ...more };
}

const journeys: Partial<Project> = { testMatch: "*.spec.ts", dependencies: ["setup"] };
const signedIn = { storageState: settings.storageState };

export default defineConfig({
  testDir: "recette",
  // The *.test.ts of recette/ are unit tests, run by node:test.
  testMatch: "*.spec.ts",
  forbidOnly: true,
  retries: 0,
  // One journey at a time: the tribe recette-auto is shared, and the video reads in order.
  workers: 1,
  timeout: 2 * 60_000,
  expect: { timeout: 15_000 },
  outputDir: join(settings.outputDir, "resultats"),
  reporter: [
    ["list"],
    ["html", { open: "never", outputFolder: join(settings.outputDir, "rapport") }],
    ["./recette/controle-reporter.ts", { plan: settings.plan, points: settings.points, outputDir: settings.outputDir }],
  ],
  use: {
    baseURL: settings.tribeURL,
    trace: "on",
    screenshot: "on",
    // A short pause before each action, so that the video follows (ADR 0032, point 11).
    launchOptions: { slowMo: 250 },
  },
  projects: [
    project("setup", "compact", { ...devices["Pixel 7"] }, { testMatch: "connexion.setup.ts" }),
    project("phone-chromium", "compact", { ...devices["Pixel 7"], ...signedIn }, journeys),
    project("phone-webkit", "compact", { ...devices["iPhone 15"], ...signedIn }, journeys),
    project("tablet-landscape-chromium", "etendu", { ...devices["Pixel 7"], viewport: tabletLandscape, screen: tabletLandscape, ...signedIn }, journeys),
    project("computer-chromium", "etendu", { ...devices["Desktop Chrome"], viewport: computer, ...signedIn }, journeys),
  ],
});
