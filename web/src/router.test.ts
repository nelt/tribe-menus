import assert from "node:assert/strict";
import { test } from "node:test";
import { type Page, Router, type RouterWindow, pageFromPath, pages, pathOfPage, tabOf } from "./router.ts";

const base = "/tribes/demo/";

const id = "0123456789abcdef0123456789abcdef";

const pageCases: { name: string; pathname: string; want: Page }[] = [
  { name: "tribe root", pathname: "/tribes/demo/", want: pages.home },
  { name: "login", pathname: "/tribes/demo/connexion", want: pages.login },
  { name: "login with trailing slash", pathname: "/tribes/demo/connexion/", want: pages.login },
  { name: "dish library", pathname: "/tribes/demo/plats", want: pages.dishes },
  { name: "new dish", pathname: "/tribes/demo/plats/nouveau", want: pages.newDish },
  { name: "a dish", pathname: `/tribes/demo/plats/${id}`, want: { name: "dish", id } },
  { name: "a dish, identifier in upper case", pathname: `/tribes/demo/plats/${id.toUpperCase()}`, want: pages.home },
  { name: "a dish, identifier too short", pathname: "/tribes/demo/plats/0123", want: pages.home },
  { name: "tribe", pathname: "/tribes/demo/tribu", want: pages.tribe },
  { name: "unknown page", pathname: "/tribes/demo/planning/2026-09-29", want: pages.home },
  { name: "login of another tribe", pathname: "/tribes/autre/connexion", want: pages.home },
  { name: "prefix of the login", pathname: "/tribes/demo/connexionx", want: pages.home },
];

for (const { name, pathname, want } of pageCases) {
  test(`pageFromPath: ${name}`, () => {
    assert.deepEqual(pageFromPath(pathname, base), want);
  });
}

test("pathOfPage", () => {
  assert.equal(pathOfPage(pages.home, base), "/tribes/demo/");
  assert.equal(pathOfPage(pages.login, base), "/tribes/demo/connexion");
  assert.equal(pathOfPage(pages.dishes, base), "/tribes/demo/plats");
  assert.equal(pathOfPage(pages.newDish, base), "/tribes/demo/plats/nouveau");
  assert.equal(pathOfPage({ name: "dish", id }, base), `/tribes/demo/plats/${id}`);
  assert.equal(pathOfPage(pages.tribe, base), "/tribes/demo/tribu");
});

test("tabOf", () => {
  assert.equal(tabOf(pages.home), "planning");
  assert.equal(tabOf(pages.tribe), "planning");
  assert.equal(tabOf(pages.dishes), "dishes");
  assert.equal(tabOf(pages.newDish), "dishes");
  assert.equal(tabOf({ name: "dish", id }), "dishes");
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
  { name: "push", from: "/tribes/demo/", page: pages.login, replace: false, wantCalls: ["push /tribes/demo/connexion"], wantChanges: [pages.login] },
  { name: "replace", from: "/tribes/demo/", page: pages.login, replace: true, wantCalls: ["replace /tribes/demo/connexion"], wantChanges: [pages.login] },
  { name: "to the home page", from: "/tribes/demo/connexion", page: pages.home, replace: false, wantCalls: ["push /tribes/demo/"], wantChanges: [pages.home] },
  { name: "already there", from: "/tribes/demo/connexion", page: pages.login, replace: false, wantCalls: [], wantChanges: [] },
  { name: "from a dish to another", from: `/tribes/demo/plats/${id}`, page: { name: "dish", id: "f".repeat(32) }, replace: false, wantCalls: [`push /tribes/demo/plats/${"f".repeat(32)}`], wantChanges: [{ name: "dish", id: "f".repeat(32) }] },
];

for (const { name, from, page, replace, wantCalls, wantChanges } of navigateCases) {
  test(`Router.navigate: ${name}`, () => {
    const { window, calls } = fakeWindow(from);
    const changes: Page[] = [];
    const router = new Router(window, base, (p) => changes.push(p));
    router.navigate(page, { replace });
    assert.deepEqual(calls, wantCalls);
    assert.deepEqual(changes, wantChanges);
    assert.deepEqual(router.page, page);
  });
}

test("Router: back in the history tells the page", () => {
  const { window, back } = fakeWindow("/tribes/demo/");
  const changes: Page[] = [];
  const router = new Router(window, base, (p) => changes.push(p));
  router.navigate(pages.login);
  back("/tribes/demo/");
  assert.deepEqual(changes, [pages.login, pages.home]);
});

test("Router.follow: a page of the tribe, or the browser", () => {
  const { window, calls } = fakeWindow("/tribes/demo/");
  const changes: Page[] = [];
  const router = new Router(window, base, (p) => changes.push(p));
  assert.equal(router.follow("/tribes/demo/plats"), true);
  assert.equal(router.follow("/tribes/autre/plats"), false);
  assert.equal(router.follow("/mentions-legales.html"), false);
  assert.deepEqual(calls, ["push /tribes/demo/plats"]);
  assert.deepEqual(changes, [pages.dishes]);
  assert.equal(router.href(pages.tribe), "/tribes/demo/tribu");
});
