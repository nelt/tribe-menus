// Registry of the steps of the scenarios @ui (ADR 0028, point 2): each wording of a step,
// following docs/specs/conventions-gherkin.md, is a regular expression and a function.
// Distinct from the steps of godog (acceptance/), which prove the other scenarios.
import type { APIRequestContext, Page } from "@playwright/test";
import { Tribe } from "./tribe.ts";

/** The state of a scenario, from step to step. */
export class World {
  readonly page: Page;
  readonly request: APIRequestContext;
  readonly baseURL: string;
  /** The data table of the current step, the header first. */
  table: string[][] = [];
  /** Context, Action or Outcome. */
  stepType = "";
  #tribe: Tribe | undefined;
  /** Who uses the page of the browser, once it is open. */
  pageOwner: string | undefined;
  /** Free state of the step definitions. */
  readonly memory = new Map<string, unknown>();

  constructor(page: Page, request: APIRequestContext, baseURL: string) {
    this.page = page;
    this.request = request;
    this.baseURL = baseURL;
  }

  /** The tribe of the scenario, created on first use with the people of the scenarios. */
  async tribe(): Promise<Tribe> {
    this.#tribe ??= await Tribe.create(this.request, this.baseURL);
    return this.#tribe;
  }

  /** Opens a page of the tribe in the browser, as this person. */
  async open(who: string, path: string): Promise<void> {
    const tribe = await this.tribe();
    if (this.pageOwner === undefined) {
      await tribe.signInPage(this.page, who);
      this.pageOwner = who;
    } else if (this.pageOwner !== who) {
      throw new Error(`the browser is ${this.pageOwner}'s, not ${who}'s`);
    }
    await this.page.goto(tribe.path(path));
  }
}

export type StepFunction = (world: World, ...args: string[]) => Promise<void>;

interface Definition {
  pattern: RegExp;
  fn: StepFunction;
}

const definitions: Definition[] = [];

/** Defines a step: the pattern must match the whole text; its groups are the arguments. */
export function step(pattern: RegExp, fn: StepFunction): void {
  definitions.push({ pattern: new RegExp(`^(?:${pattern.source})$`, pattern.flags), fn });
}

/** The definition of a step and its arguments; undefined without definition. */
export function findStep(text: string): { fn: StepFunction; args: string[] } | undefined {
  const found = definitions.flatMap(({ pattern, fn }) => {
    const m = pattern.exec(text);
    return m === null ? [] : [{ fn, args: m.slice(1).map((a) => a ?? "") }];
  });
  if (found.length > 1) {
    throw new Error(`ambiguous step, ${found.length} definitions: ${text}`);
  }
  return found[0];
}
