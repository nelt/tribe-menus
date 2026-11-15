// Settings of the automated acceptance run (plan recette-auto, D10 and D11), read in the
// volume of the Dev Container and in the environment that make recette-auto sets: nothing of
// the instance in the repository (ADR 0011).
import { readFileSync } from "node:fs";
import { homedir } from "node:os";
import { join } from "node:path";

export interface RecetteSettings {
  /** https://<host>/tribes/<slug>/, the tribe recette-auto on the staging instance. */
  tribeURL: string;
  /** The only member of the tribe, a real box of the MX Plan. */
  address: string;
  /** Directory of recette-auto.json, recette-auto.pop3 and the state of the session. */
  configDir: string;
  /** The state of the session written by the project setup, taken up by the others (D8). */
  storageState: string;
  /** Directory of this run in the volume of the videos: vX.Y.Z/<time>/ (D11). */
  outputDir: string;
  /** The plan of the acceptance run of the version, docs/recettes/vX.Y.Z.md (D12). */
  plan: string;
}

/** Reads the settings; throws when run otherwise than by make recette-auto. */
export function recetteSettings(env: NodeJS.ProcessEnv = process.env): RecetteSettings {
  const configDir = env["RECETTE_CONFIG_DIR"] || join(homedir(), ".config", "tribe-menus");
  const outputDir = env["RECETTE_OUTPUT_DIR"];
  const plan = env["RECETTE_PLAN"];
  if (!outputDir || !plan) {
    throw new Error("RECETTE_OUTPUT_DIR and RECETTE_PLAN are set by make recette-auto: run it rather than Playwright");
  }
  const file = JSON.parse(readFileSync(join(configDir, "recette-auto.json"), "utf8")) as { tribeURL?: unknown; address?: unknown };
  if (typeof file.tribeURL !== "string" || !/^https:\/\/[^/]+\/tribes\/[^/]+\/$/.test(file.tribeURL) || typeof file.address !== "string") {
    throw new Error("recette-auto.json: tribeURL https://<host>/tribes/<slug>/ and address needed");
  }
  return {
    tribeURL: file.tribeURL,
    address: file.address,
    configDir,
    storageState: join(configDir, "recette-auto.state.json"),
    outputDir,
    plan,
  };
}
