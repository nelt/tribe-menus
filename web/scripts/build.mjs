// Builds the front end into web/dist (ADR 0004).
// Usage: node scripts/build.mjs [--watch]
import { copyFile, cp, mkdir, readdir, rm } from "node:fs/promises";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import * as esbuild from "esbuild";

const webDir = join(dirname(fileURLToPath(import.meta.url)), "..");
const srcDir = join(webDir, "src");
const distDir = join(webDir, "dist");
const identityDir = join(webDir, "..", "docs", "design", "identite");
const staticFiles = ["index.html", "app.css"];
// The identity files are read from docs/design/identite: a single source, no copy in web/src.
const identityFiles = ["symbole.svg", "logotype.svg"];
const watch = process.argv.includes("--watch");

// Empties dist, keeping the placeholder required by //go:embed.
async function cleanDist() {
  await mkdir(distDir, { recursive: true });
  for (const entry of await readdir(distDir)) {
    if (entry !== ".gitkeep") {
      await rm(join(distDir, entry), { recursive: true, force: true });
    }
  }
}

// Copies the static files, the fonts with their licence (OFL) and the identity files.
async function copyStaticFiles() {
  for (const file of staticFiles) {
    await copyFile(join(srcDir, file), join(distDir, file));
  }
  await cp(join(srcDir, "fonts"), join(distDir, "fonts"), { recursive: true });
  for (const file of identityFiles) {
    await copyFile(join(identityDir, file), join(distDir, file));
  }
}

// Copies static files after each build, so that watch mode picks up their changes too.
const staticFilesPlugin = {
  name: "static-files",
  setup(build) {
    build.onEnd(async (result) => {
      if (result.errors.length === 0) {
        await copyStaticFiles();
      }
    });
  },
};

/** @type {esbuild.BuildOptions} */
const options = {
  entryPoints: [join(srcDir, "main.ts")],
  outdir: distDir,
  bundle: true,
  format: "esm",
  target: "es2022",
  minify: !watch,
  sourcemap: watch ? "inline" : "linked",
  logLevel: "info",
  plugins: [staticFilesPlugin],
};

await cleanDist();
if (watch) {
  const context = await esbuild.context(options);
  await context.watch();
} else {
  await esbuild.build(options);
}
