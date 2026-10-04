import assert from "node:assert/strict";
import { test } from "node:test";
import { type Page, Router, type RouterWindow, pageFromPath, pathOfPage } from "./router.ts";

const base = "/tribes/demo/";

const pageCases: { name: string; pathname: string; want: Page }[] = [
  { name: "tribe root", pathname: "/tribes/demo/", want: "home" },
  { name: "login", pathname: "/tribes/demo/connexion", want: "login" },
  { name: "login with trailing slash", pathname: "/tribes/demo/connexion/", want: "login" },
  { name: "unknown page", pathname: "/tribes/demo/planning/2026-09-29", want: "home" },
  { name: "login of another tribe", pathname: "/tribes/autre/connexion", want: "home" },
  { name: "prefix of the login", pathname: "/tribes/demo/connexionx", want: "home" },
];

for (const { name, pathname, want } of pageCases) {
  test(`pageFromPath: ${name}`, () => {
    assert.equal(pageFromPath(pathname, base), want);
  });
}

test("pathOfPage", () => {
  assert.equal(pathOfPage("home", base), "/tribes/demo/");
  assert.equal(pathOfPage("login", base), "/tribes/demo/connexion");
});

function fakeWindow(pathname: string) {
  const calls: string[] = [];
  let popstate = () => {};
  const window: RouterWindow = {
    location: { pathname },
    history: {
      pushState: (_data, _unused, url) => {
        calls.push(`push ${url}`);
        window.location.pathname = url;
      },
      replaceState: (_data, _unused, url) => {
        calls.push(`replace ${url}`);
        window.location.pathname = url;
      },
    },
    addEventListener: (_type, listener) => {
      popstate = listener;
    },
  };
  return { window, calls, back: (to: string) => ((window.location.pathname = to), popstate()) };
}

const navigateCases: { name: string; from: string; page: Page; replace: boolean; wantCalls: string[]; wantChanges: Page[] }[] = [
  { name: "push", from: "/tribes/demo/", page: "login", replace: false, wantCalls: ["push /tribes/demo/connexion"], wantChanges: ["login"] },
  { name: "replace", from: "/tribes/demo/", page: "login", replace: true, wantCalls: ["replace /tribes/demo/connexion"], wantChanges: ["login"] },
  { name: "to the home page", from: "/tribes/demo/connexion", page: "home", replace: false, wantCalls: ["push /tribes/demo/"], wantChanges: ["home"] },
  { name: "already there", from: "/tribes/demo/connexion", page: "login", replace: false, wantCalls: [], wantChanges: [] },
];

for (const { name, from, page, replace, wantCalls, wantChanges } of navigateCases) {
  test(`Router.navigate: ${name}`, () => {
    const { window, calls } = fakeWindow(from);
    const changes: Page[] = [];
    const router = new Router(window, base, (p) => changes.push(p));
    router.navigate(page, { replace });
    assert.deepEqual(calls, wantCalls);
    assert.deepEqual(changes, wantChanges);
    assert.equal(router.page, page);
  });
}

test("Router: back in the history tells the page", () => {
  const { window, back } = fakeWindow("/tribes/demo/");
  const changes: Page[] = [];
  const router = new Router(window, base, (p) => changes.push(p));
  router.navigate("login");
  back("/tribes/demo/");
  assert.deepEqual(changes, ["login", "home"]);
});
