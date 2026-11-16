// Targeted journey of v0.6.0 (docs/recettes/v0.6.0.md, entries 5 and 11): selection, « Nouvelle
// liste » and « Déplacer vers… » in each format. Two items added by hand to « Courses », both
// selected and sent to a new list; then sent back one by one; the empty list deleted (C12: the
// tribe keeps no list of the runs, 20 at most); the items checked and taken out of « Courses ».
// The lists left by a run stopped halfway are removed first.
import type { Page } from "@playwright/test";
import { check, checkout, itemLine, listCard, listTitle, openCourses, openListPicker, runSuffix } from "./helpers.ts";
import { expect, scenario, step, test } from "./test.ts";

function actions(page: Page) {
  return page.getByRole("group", { name: "Actions de la sélection" });
}

/** Enters the selection mode, unless the list already is in it, and selects these items. */
async function select(page: Page, ...ingredients: string[]) {
  const start = page.getByRole("button", { name: "Sélectionner", exact: true });
  if ((await start.count()) > 0) {
    await start.click();
  }
  await expect(page.getByRole("main").locator(".selection-hint")).toBeVisible();
  for (const ingredient of ingredients) {
    const pick = page.getByRole("button", { name: new RegExp(`^Sélectionner ${ingredient}(,|$)`) });
    await pick.click();
    await expect(pick).toHaveAttribute("aria-pressed", "true");
  }
}

/** The green banner of a move, with its link to the target list. */
function movedBanner(page: Page) {
  return page.locator(".banner-done").filter({ has: page.locator(".banner-link") });
}

/** Deletes a list from the picker: it must be empty and not the main one (C12). */
async function deleteList(page: Page, name: string) {
  await openListPicker(page);
  await page.getByRole("button", { name: `Actions pour la liste ${name}` }).click();
  await page.getByRole("dialog").getByRole("button", { name: "Supprimer la liste" }).click();
  await expect(listCard(page, name)).toHaveCount(0);
}

/** Opens the move sheet for the selection, then moves it to « Courses ». */
async function moveToCourses(page: Page) {
  await actions(page).getByRole("button", { name: "Déplacer vers…" }).click();
  await page.getByRole("dialog").locator(".sheet-target").filter({ has: page.locator(".sheet-target-name").getByText("Courses", { exact: true }) }).click();
  await expect(movedBanner(page)).toContainText("1 article déplacé vers « Courses ».");
}

test("selection, new list and move", scenario("Sélection d'articles, nouvelle liste et déplacement"), async ({ page }, testInfo) => {
  const run = runSuffix(testInfo.project.name);
  const milk = `lait ${run}`;
  const eggs = `œufs ${run}`;
  const list = `Sélection ${run}`;

  // A run stopped halfway leaves its list, items added by hand in it: removed, then the list.
  // Tooling, not the application: no Gherkin, left out of the scenario on the control page.
  await test.step("Lists of stopped runs removed", async () => {
    await openCourses(page);
    await openListPicker(page);
    const left = (await page.locator(".list-card-name").allInnerTexts()).filter((n) => n.startsWith("Sélection "));
    for (const name of left) {
      await listCard(page, name).locator(".list-card-link").click();
      await expect(listTitle(page)).toHaveText(name);
      const remove = page.getByRole("button", { name: /^Retirer / });
      while ((await remove.count()) > 0) {
        const before = await remove.count();
        await remove.first().click();
        await expect(remove).toHaveCount(before - 1);
      }
      await deleteList(page, name);
    }
  });

  await step(page, "Two items added by hand", [`Étant donné la liste "Courses" contenant les articles ajoutés "${milk}" et "${eggs}", ajoutés à la main`], async () => {
    await openCourses(page);
    await page.getByRole("button", { name: "Ajouter un article à la main" }).click();
    for (const ingredient of [milk, eggs]) {
      await page.getByRole("combobox", { name: "Article" }).fill(ingredient);
      await page.getByRole("button", { name: "Ajouter à la liste" }).click();
      await expect(itemLine(page, ingredient)).toBeVisible();
    }
    await page.getByRole("button", { name: "Annuler" }).click();
  });

  await step(page, "Two items selected", [`Quand je touche « Sélectionner » puis sélectionne "${milk}" et "${eggs}" dans la liste "Courses"`, "Alors les actions « Déplacer vers… » et « Nouvelle liste » sont disponibles"], async () => {
    await select(page, milk, eggs);
    await expect(actions(page).getByRole("button", { name: "Déplacer vers…" })).toBeEnabled();
    await expect(actions(page).getByRole("button", { name: "Nouvelle liste" })).toBeEnabled();
  });

  await step(page, "New list from the selection", [`Quand je crée la liste "${list}" à partir de la sélection`, `Alors un message m'indique que 2 articles ont été déplacés vers la nouvelle liste "${list}"`, `Et la liste "Courses" ne contient plus "${milk}"`], async () => {
    await actions(page).getByRole("button", { name: "Nouvelle liste" }).click();
    const sheet = page.getByRole("dialog", { name: "Nouvelle liste" });
    await sheet.getByRole("textbox", { name: "Nom de la liste" }).fill(list);
    await sheet.getByRole("button", { name: "Créer la liste" }).click();
    await expect(movedBanner(page)).toContainText(`2 articles déplacés vers la nouvelle liste « ${list} ».`);
    await expect(itemLine(page, milk)).toHaveCount(0);
  });

  await step(page, "New list holds the items", [`Quand je touche le lien du message`, `Alors la liste "${list}" s'affiche, avec "${milk}" et "${eggs}"`], async () => {
    await movedBanner(page).locator(".banner-link").click();
    await expect(listTitle(page)).toHaveText(list);
    await expect(itemLine(page, milk)).toBeVisible();
    await expect(itemLine(page, eggs)).toBeVisible();
  });

  await step(page, "Move sheet", [`Quand je sélectionne "${milk}" dans la liste "${list}" et touche « Déplacer vers… »`, `Alors la feuille des autres listes s'affiche, avec la liste "Courses"`], async () => {
    await select(page, milk);
    await actions(page).getByRole("button", { name: "Déplacer vers…" }).click();
    await expect(page.getByRole("dialog").locator(".sheet-target-name").getByText("Courses", { exact: true })).toBeVisible();
  });

  await step(page, "Items sent back to Courses", [`Quand je déplace la sélection vers la liste "Courses"`, `Alors un message m'indique qu'1 article a été déplacé vers la liste "Courses"`, `Et quand je déplace de même "${eggs}" vers la liste "Courses"`, `Alors la liste "${list}" ne contient plus "${eggs}"`], async () => {
    await page.getByRole("dialog").locator(".sheet-target").filter({ has: page.locator(".sheet-target-name").getByText("Courses", { exact: true }) }).click();
    await expect(movedBanner(page)).toContainText("1 article déplacé vers « Courses ».");
    await select(page, eggs);
    await moveToCourses(page);
    await expect(itemLine(page, eggs)).toHaveCount(0);
  });

  await step(page, "Empty list deleted", [`Quand je supprime la liste "${list}", vide, depuis le choix des listes`, "Alors elle n'y figure plus"], () => deleteList(page, list));

  await step(page, "Items taken out of Courses", [`Quand je coche "${milk}" et "${eggs}" dans la liste "Courses" puis termine les courses`, "Alors ils ne figurent plus dans la liste"], async () => {
    await openCourses(page);
    await check(page, milk);
    await check(page, eggs);
    await checkout(page, milk, eggs);
  });
});
