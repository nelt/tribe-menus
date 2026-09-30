import { defineConfig, devices } from "@playwright/test";

const port = 8090;

export default defineConfig({
  testDir: "e2e",
  forbidOnly: !!process.env["CI"],
  reporter: [["list"], ["html", { open: "never" }]],
  use: {
    baseURL: `http://localhost:${port}`,
    trace: "retain-on-failure",
  },
  projects: [
    { name: "chromium", use: { ...devices["Pixel 7"] } },
    { name: "webkit", use: { ...devices["iPhone 15"] } },
  ],
  webServer: {
    command: `npm run build && go run ../cmd/tribe-menus serve -dev -root .. -addr localhost:${port}`,
    url: `http://localhost:${port}/`,
    reuseExistingServer: !process.env["CI"],
    timeout: 120_000,
  },
});
