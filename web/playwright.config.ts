import { defineConfig, devices } from "@playwright/test";

const port = 8090;
// Fixed data directory, emptied at each start (D15): the demonstration tribe, the databases
// and the file of the emails that the login journey reads (D6).
const dataDir = ".e2e-data";
const mailFile = `${dataDir}/mails.jsonl`;

export default defineConfig({
  testDir: "e2e",
  // The *.test.ts of e2e/ are unit tests, run by node:test.
  testMatch: "*.spec.ts",
  forbidOnly: !!process.env["CI"],
  reporter: [["list"], ["html", { open: "never" }]],
  use: {
    baseURL: `http://localhost:${port}`,
    trace: "retain-on-failure",
  },
  projects: [
    { name: "chromium", use: { ...devices["Pixel 7"] } },
    { name: "webkit", use: { ...devices["iPhone 15"] } },
    // Smallest current iPhone width (375 px): catches layouts that overflow on a small screen.
    { name: "webkit-small", use: { ...devices["iPhone SE (3rd gen)"] } },
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
