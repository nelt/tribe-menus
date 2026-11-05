import assert from "node:assert/strict";
import { test } from "node:test";
import { type Page, Router, type RouterWindow, pageFromPath, pages, pathOfPage, planningAround, tabOf } from "./router.ts";

const base = "/tribes/demo/";

const id = "0123456789abcdef0123456789abcdef";

const pageCases: { name: string; pathname: string; search?: string; want: Page }[] = [
  { name: "tribe root", pathname: "/tribes/demo/", want: pages.planning },
  { name: "login", pathname: "/tribes/demo/connexion", want: pages.login },
  { name: "login with trailing slash", pathname: "/tribes/demo/connexion/", want: pages.login },
  { name: "dish library", pathname: "/tribes/demo/plats", want: pages.dishes },
  { name: "new dish", pathname: "/tribes/demo/plats/nouveau", want: pages.newDish },
  { name: "a dish", pathname: `/tribes/demo/plats/${id}`, want: { name: "dish", id } },
  { name: "a dish, identifier in upper case", pathname: `/tribes/demo/plats/${id.toUpperCase()}`, want: pages.planning },
  { name: "a dish, identifier too short", pathname: "/tribes/demo/plats/0123", want: pages.planning },
  { name: "tribe", pathname: "/tribes/demo/tribu", want: pages.tribe },
  { name: "unknown page", pathname: "/tribes/demo/planning/2026-09-29", want: pages.planning },
  { name: "planning from a day (D14)", pathname: "/tribes/demo/", search: "?du=2026-09-28", want: { name: "planning", start: "2026-09-28" } },
  { name: "planning from the first day of the bounds", pathname: "/tribes/demo/", search: "?du=2000-01-01", want: { name: "planning", start: "2000-01-01" } },
  { name: "planning from a day out of the bounds", pathname: "/tribes/demo/", search: "?du=2100-01-01", want: pages.planning },
  { name: "planning from a day that does not exist", pathname: "/tribes/demo/", search: "?du=2026-02-30", want: pages.planning },
  { name: "planning from an unreadable day", pathname: "/tribes/demo/", search: "?du=lundi", want: pages.planning },
  { name: "planning, another parameter", pathname: "/tribes/demo/", search: "?de=2026-09-28", want: pages.planning },
  { name: "the day is a query of the planning only", pathname: "/tribes/demo/plats", search: "?du=2026-09-28", want: pages.dishes },
  { name: "a meal, lunch (D14)", pathname: "/tribes/demo/repas/2026-10-06/midi", want: { name: "meal", date: "2026-10-06", mealtime: "lunch" } },
  { name: "a meal, dinner", pathname: "/tribes/demo/repas/2026-10-06/soir", want: { name: "meal", date: "2026-10-06", mealtime: "dinner" } },
  { name: "a meal, trailing slash", pathname: "/tribes/demo/repas/2026-10-06/soir/", want: { name: "meal", date: "2026-10-06", mealtime: "dinner" } },
  { name: "a meal, unknown mealtime", pathname: "/tribes/demo/repas/2026-10-06/matin", want: pages.planning },
  { name: "a meal, mealtime in English", pathname: "/tribes/demo/repas/2026-10-06/dinner", want: pages.planning },
  { name: "a meal, day out of the bounds", pathname: "/tribes/demo/repas/2100-01-01/midi", want: pages.planning },
  { name: "a meal, day that does not exist", pathname: "/tribes/demo/repas/2026-02-30/midi", want: pages.planning },
  { name: "a meal without mealtime", pathname: "/tribes/demo/repas/2026-10-06", want: pages.planning },
  { name: "login of another tribe", pathname: "/tribes/autre/connexion", want: pages.planning },
  { name: "prefix of the login", pathname: "/tribes/demo/connexionx", want: pages.planning },
];

for (const { name, pathname, search, want } of pageCases) {
  test(`pageFromPath: ${name}`, () => {
    assert.deepEqual(pageFromPath(pathname, base, search), want);
  });
}

test("pathOfPage", () => {
  assert.equal(pathOfPage(pages.planning, base), "/tribes/demo/");
  assert.equal(pathOfPage({ name: "planning", start: "2026-09-28" }, base), "/tribes/demo/?du=2026-09-28");
  assert.equal(pathOfPage(pages.login, base), "/tribes/demo/connexion");
  assert.equal(pathOfPage(pages.dishes, base), "/tribes/demo/plats");
  assert.equal(pathOfPage(pages.newDish, base), "/tribes/demo/plats/nouveau");
  assert.equal(pathOfPage({ name: "dish", id }, base), `/tribes/demo/plats/${id}`);
  assert.equal(pathOfPage(pages.tribe, base), "/tribes/demo/tribu");
  assert.equal(pathOfPage({ name: "meal", date: "2026-10-06", mealtime: "lunch" }, base), "/tribes/demo/repas/2026-10-06/midi");
  assert.equal(pathOfPage({ name: "meal", date: "2026-10-06", mealtime: "dinner" }, base), "/tribes/demo/repas/2026-10-06/soir");
});

test("planningAround: the range of today if it holds the day, else the range from the day", () => {
  assert.deepEqual(planningAround("2026-10-05", "2026-10-05"), pages.planning);
  assert.deepEqual(planningAround("2026-10-11", "2026-10-05"), pages.planning);
  assert.deepEqual(planningAround("2026-10-12", "2026-10-05"), { name: "planning", start: "2026-10-12" });
  assert.deepEqual(planningAround("2026-10-04", "2026-10-05"), { name: "planning", start: "2026-10-04" });
});

test("tabOf", () => {
  assert.equal(tabOf(pages.planning), "planning");
  assert.equal(tabOf(pages.tribe), "planning");
  assert.equal(tabOf({ name: "meal", date: "2026-10-06", mealtime: "dinner" }), "planning");
  assert.equal(tabOf(pages.dishes), "dishes");
  assert.equal(tabOf(pages.newDish), "dishes");
  assert.equal(tabOf({ name: "dish", id }), "dishes");
});

function fakeWindow(from: string) {
  const go = (url: string) => {
    const [pathname = "", search = ""] = url.split(/(?=\?)/);
    window.location.pathname = pathname;
    window.location.search = search;
  };
  const calls: string[] = [];
  let popstate = () => {};
  const window: RouterWindow = {
    location: { pathname: "", search: "" },
    history: {
      pushState: (_data, _unused, url) => {
        calls.push(`push ${url}`);
        go(url);
      },
      replaceState: (_data, _unused, url) => {
        calls.push(`replace ${url}`);
        go(url);
      },
    },
    addEventListener: (_type, listener) => {
      popstate = listener;
    },
  };
  go(from);
  return { window, calls, back: (to: string) => (go(to), popstate()) };
}

const navigateCases: { name: string; from: string; page: Page; replace: boolean; wantCalls: string[]; wantChanges: Page[] }[] = [
  { name: "push", from: "/tribes/demo/", page: pages.login, replace: false, wantCalls: ["push /tribes/demo/connexion"], wantChanges: [pages.login] },
  { name: "replace", from: "/tribes/demo/", page: pages.login, replace: true, wantCalls: ["replace /tribes/demo/connexion"], wantChanges: [pages.login] },
  { name: "to the home page", from: "/tribes/demo/connexion", page: pages.planning, replace: false, wantCalls: ["push /tribes/demo/"], wantChanges: [pages.planning] },
  { name: "already there", from: "/tribes/demo/connexion", page: pages.login, replace: false, wantCalls: [], wantChanges: [] },
  { name: "to another range (D14)", from: "/tribes/demo/", page: { name: "planning", start: "2026-10-12" }, replace: false, wantCalls: ["push /tribes/demo/?du=2026-10-12"], wantChanges: [{ name: "planning", start: "2026-10-12" }] },
  { name: "back to today (D14)", from: "/tribes/demo/?du=2026-10-12", page: pages.planning, replace: false, wantCalls: ["push /tribes/demo/"], wantChanges: [pages.planning] },
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
  assert.deepEqual(changes, [pages.login, pages.planning]);
});

test("Router.follow: a page of the tribe, or the browser", () => {
  const { window, calls } = fakeWindow("/tribes/demo/");
  const changes: Page[] = [];
  const router = new Router(window, base, (p) => changes.push(p));
  assert.equal(router.follow("/tribes/demo/plats"), true);
  assert.equal(router.follow("/tribes/demo/", "?du=2026-09-28"), true);
  assert.equal(router.follow("/tribes/autre/plats"), false);
  assert.equal(router.follow("/mentions-legales.html"), false);
  assert.deepEqual(calls, ["push /tribes/demo/plats", "push /tribes/demo/?du=2026-09-28"]);
  assert.deepEqual(changes, [pages.dishes, { name: "planning", start: "2026-09-28" }]);
  assert.equal(router.href(pages.tribe), "/tribes/demo/tribu");
});
