import { type Project, defineConfig, devices } from "@playwright/test";
import type { Format } from "./src/layout.ts";

const port = 8090;
// Fixed data directory, emptied at each start (D15): the demonstration tribe, the databases
// and the file of the emails that the login journey reads (D6).
const dataDir = ".e2e-data";
const mailFile = `${dataDir}/mails.jsonl`;

// make e2e-formats runs everything in every project; make e2e, on each PR, runs under the
// tablets and the computer only the scenarios of formats-ecran.feature (plan grands-ecrans,
// D14; ADR 0030, point 7).
const allFormats = process.env["E2E_ALL_FORMATS"] === "1";
const formatsOnly: Partial<Project> = allFormats ? {} : { testMatch: "scenarios.spec.ts", grep: /formats-ecran\.feature: / };

/** A project of a format (ADR 0030, point 6). */
function project(name: string, format: Format, use: NonNullable<Project["use"]>, more: Partial<Project> = {}): Project {
  return { name, use, metadata: { format }, ...more };
}

const tabletPortrait = { width: 820, height: 1180 };
const tabletLandscape = { width: 1180, height: 820 };

export default defineConfig({
  testDir: "e2e",
  // The *.test.ts of e2e/ are unit tests, run by node:test.
  testMatch: "*.spec.ts",
  forbidOnly: !!process.env["CI"],
  reporter: [["list"], ["html", { open: "never" }], ["./e2e/pending-reporter.ts"]],
  use: {
    baseURL: `http://localhost:${port}`,
    trace: "retain-on-failure",
  },
  projects: [
    project("phone-chromium", "compact", { ...devices["Pixel 7"] }),
    project("phone-webkit", "compact", { ...devices["iPhone 15"] }),
    project("tablet-portrait-chromium", "moyen", { ...devices["Pixel 7"], viewport: tabletPortrait, screen: tabletPortrait }, formatsOnly),
    project("tablet-portrait-webkit", "moyen", { ...devices["iPad Pro 11"], viewport: tabletPortrait, screen: tabletPortrait }, formatsOnly),
    project("tablet-landscape-chromium", "etendu", { ...devices["Pixel 7"], viewport: tabletLandscape, screen: tabletLandscape }, formatsOnly),
    project("tablet-landscape-webkit", "etendu", { ...devices["iPad Pro 11 landscape"], viewport: tabletLandscape, screen: tabletLandscape }, formatsOnly),
    project("computer-chromium", "etendu", { ...devices["Desktop Chrome"], viewport: { width: 1440, height: 900 } }, formatsOnly),
  ],
  webServer: {
    command: [
      "npm run build",
      `rm -rf ${dataDir}`,
      `go run ../cmd/tribe-menus admin seed -data ${dataDir}`,
      `go run ../cmd/tribe-menus serve -dev -root .. -data ${dataDir} -mail-file ${mailFile} -addr localhost:${port}`,
    ].join(" && "),
    url: `http://localhost:${port}/healthz`,
    // Never reuse a server: the rate limits start from zero at each run (D15).
    reuseExistingServer: false,
    timeout: 120_000,
  },
});
