// Builds the front end into web/dist (ADR 0004).
// Usage: node scripts/build.mjs [--watch]
//
// A build names the code, the stylesheet, the fonts and the identity files after their content
// (ADR 0012, point 2), writes index.html with these names and lists them in files.json, which the
// server reads to cache them for good. Watch mode keeps fixed names and writes no list
// (plan production, D10).
import { copyFile, mkdir, readdir, readFile, rm, writeFile } from "node:fs/promises";
import { dirname, join, relative } from "node:path";
import { fileURLToPath } from "node:url";
import * as esbuild from "esbuild";

const webDir = join(dirname(fileURLToPath(import.meta.url)), "..");
const srcDir = join(webDir, "src");
const distDir = join(webDir, "dist");
// The licences of the fonts (OFL) are copied next to them (ADR 0011, point 3).
const fontLicences = ["OFL-BricolageGrotesque.txt", "OFL-Figtree.txt"];
// The list of the files named after their content, read by the server (internal/server).
const filesList = "files.json";
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

// Returns the name, relative to dist, of the file produced for an entry point.
function outputOf(metafile, entryPoint) {
  for (const [output, info] of Object.entries(metafile.outputs)) {
    if (info.entryPoint === entryPoint) {
      return relative(distDir, join(webDir, output));
    }
  }
  throw new Error(`no output for ${entryPoint}`);
}

// Replaces in the index page the fixed name of a file by the name produced by the build.
function replaceName(html, fixed, produced) {
  const attribute = `"${fixed}"`;
  if (html.split(attribute).length !== 2) {
    throw new Error(`index.html must reference ${fixed} exactly once`);
  }
  return html.replace(attribute, `"${produced}"`);
}

// Writes index.html, the licences of the fonts and, outside watch mode, the list of the files.
async function writeStaticFiles(metafile) {
  let html = await readFile(join(srcDir, "index.html"), "utf8");
  html = replaceName(html, "main.js", outputOf(metafile, "src/main.ts"));
  html = replaceName(html, "app.css", outputOf(metafile, "src/app.css"));
  await writeFile(join(distDir, "index.html"), html);
  for (const file of fontLicences) {
    await copyFile(join(srcDir, "fonts", file), join(distDir, file));
  }
  if (!watch) {
    // Sourcemaps keep a fixed name relative to their file: they are not listed (ADR 0012, point 2).
    const files = Object.keys(metafile.outputs)
      .map((output) => relative(distDir, join(webDir, output)))
      .filter((file) => !file.endsWith(".map"))
      .sort();
    await writeFile(join(distDir, filesList), JSON.stringify(files, null, 2) + "\n");
  }
}

// Writes the files that esbuild does not produce after each build, so that watch mode picks up
// their changes too.
const staticFilesPlugin = {
  name: "static-files",
  setup(build) {
    build.onEnd(async (result) => {
      if (result.errors.length === 0 && result.metafile) {
        await writeStaticFiles(result.metafile);
      }
    });
  },
};

/** @type {esbuild.BuildOptions} */
const options = {
  absWorkingDir: webDir,
  entryPoints: ["src/main.ts", "src/app.css"],
  outdir: distDir,
  entryNames: watch ? "[name]" : "[name]-[hash]",
  // Flat: the identity files come from docs/design/identite, outside src.
  assetNames: watch ? "[name]" : "[name]-[hash]",
  loader: { ".woff2": "file", ".svg": "file" },
  bundle: true,
  format: "esm",
  target: "es2022",
  minify: !watch,
  sourcemap: watch ? "inline" : "linked",
  metafile: true,
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
