// Rules of the meal plan in the browser (plan planning, D4, D11, D14): the day of the device,
// the ranges of 7 days and their titles, the meal tiles of a range. Pure functions, tested by
// node:test. A day is written YYYY-MM-DD, without time zone, as the API stores it. The
// screen of a meal reads here its dishes and the dish suggestions (Q21, D12).
import { type Dish, searchDishes, sortDishes } from "./dishes.ts";

/** A day, YYYY-MM-DD. */
export type Day = string;

/** The two meals of a day (Q1). */
export type Mealtime = "lunch" | "dinner";

export const mealtimes: readonly Mealtime[] = ["lunch", "dinner"];

/** A dish served in a meal, as GET meal-plan gives it. */
export interface ServedDish {
  id: string;
  date: Day;
  mealtime: Mealtime;
  dish: string;
  servings: number;
  rank: number;
}

/** The days a meal can be placed on (Q35, D4), as the API accepts them. */
export const firstDay: Day = "2000-01-01";
export const lastDay: Day = "2099-12-31";

/** The number of days of a range (R1). */
export const rangeLength = 7;

const dayMs = 86_400_000;

function pad(n: number, width = 2): string {
  return String(n).padStart(width, "0");
}

function utcOf(day: Day): Date {
  return new Date(`${day}T00:00:00Z`);
}

function dayOfUTC(date: Date): Day {
  return `${pad(date.getUTCFullYear(), 4)}-${pad(date.getUTCMonth() + 1)}-${pad(date.getUTCDate())}`;
}

/** The day of the device (D4, Q34): its local date, whatever the time zone. */
export function today(now: Date = new Date()): Day {
  return `${pad(now.getFullYear(), 4)}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`;
}

/** Whether a meal can be placed on this day (Q35). */
export function inBounds(day: Day): boolean {
  return day >= firstDay && day <= lastDay;
}

/**
 * Reads a day of a path (?du=, D14): YYYY-MM-DD exactly, a real date, from 2000 to 2099;
 * undefined otherwise.
 */
export function parseDay(text: string): Day | undefined {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(text)) {
    return undefined;
  }
  const date = utcOf(text);
  if (Number.isNaN(date.getTime()) || dayOfUTC(date) !== text) {
    return undefined;
  }
  return inBounds(text) ? text : undefined;
}

/** The day n days after (before, if negative) the one given. */
export function addDays(day: Day, n: number): Day {
  return dayOfUTC(new Date(utcOf(day).getTime() + n * dayMs));
}

/** The 7 days from the one given (R1). */
export function dayRange(start: Day): Day[] {
  return Array.from({ length: rangeLength }, (_, i) => addDays(start, i));
}

/**
 * The start of the range before: 7 days earlier, at the first day of the bounds at most, so
 * that every day of the bounds can be reached; undefined at the first day (review of PR #87).
 */
export function previousStart(start: Day): Day | undefined {
  if (start <= firstDay) {
    return undefined;
  }
  const previous = addDays(start, -rangeLength);
  return previous < firstDay ? firstDay : previous;
}

/** The start of the range after: 7 days later, undefined beyond the bounds (review of PR #87). */
export function nextStart(start: Day): Day | undefined {
  const next = addDays(start, rangeLength);
  return next > lastDay ? undefined : next;
}

const weekdays = ["Dimanche", "Lundi", "Mardi", "Mercredi", "Jeudi", "Vendredi", "Samedi"];
const months = ["janvier", "février", "mars", "avril", "mai", "juin", "juillet", "août", "septembre", "octobre", "novembre", "décembre"];
const shortMonths = ["janv.", "févr.", "mars", "avr.", "mai", "juin", "juil.", "août", "sept.", "oct.", "nov.", "déc."];

function parts(day: Day): { year: number; month: number; date: number; weekday: number } {
  const d = utcOf(day);
  return { year: d.getUTCFullYear(), month: d.getUTCMonth(), date: d.getUTCDate(), weekday: d.getUTCDay() };
}

/** The number of the day in its month, « 1er » for the first, as in French. */
function dateNumber(date: number): string {
  return date === 1 ? "1er" : String(date);
}

/** The name of the day of the week: « Lundi ». */
export function weekdayName(day: Day): string {
  return weekdays[parts(day).weekday] ?? "";
}

/** The short date of a day: « 5 oct. », « 1er janv. ». */
export function shortDate(day: Day): string {
  const { month, date } = parts(day);
  return `${dateNumber(date)} ${shortMonths[month] ?? ""}`;
}

/**
 * The day of a meal, as the screen of a meal titles it: « Mardi 6 octobre »; the year
 * follows in another year than that of today, as for the title of a range.
 */
export function longDate(day: Day, todayDay: Day): string {
  const { year, month, date } = parts(day);
  const other = year === parts(todayDay).year ? "" : ` ${year}`;
  return `${weekdayName(day)} ${dateNumber(date)} ${months[month] ?? ""}${other}`;
}

/**
 * The title of a range (docs/design/README.md): « 5 – 11 octobre », « 28 septembre –
 * 4 octobre », « 29 décembre 2026 – 4 janvier 2027 »; the year follows in another year than
 * that of today: « 6 – 12 octobre 2025 ».
 */
export function rangeTitle(days: Day[], todayDay: Day): string {
  const first = parts(days[0] ?? todayDay);
  const last = parts(days[days.length - 1] ?? todayDay);
  const end = `${dateNumber(last.date)} ${months[last.month] ?? ""}`;
  if (first.year !== last.year) {
    return `${dateNumber(first.date)} ${months[first.month] ?? ""} ${first.year} – ${end} ${last.year}`;
  }
  const year = first.year === parts(todayDay).year ? "" : ` ${first.year}`;
  if (first.month !== last.month) {
    return `${dateNumber(first.date)} ${months[first.month] ?? ""} – ${end}${year}`;
  }
  return `${dateNumber(first.date)} – ${end}${year}`;
}

/** A dish of a meal tile: the dish served and the dish of the library. */
export interface TileDish {
  served: ServedDish;
  dish: Dish;
}

/** A meal tile (Doux-Main): its dishes in the order they were added (Q35, D11). */
export interface MealTile {
  date: Day;
  mealtime: Mealtime;
  dishes: TileDish[];
  /** Out of the bounds of the API (Q35): the meal cannot be opened. */
  inactive: boolean;
}

/** A day of a range, with its two meals, lunch then dinner. */
export interface DayTiles {
  date: Day;
  today: boolean;
  meals: MealTile[];
}

/** The dishes of each meal, by "<date> <mealtime>", in the order they were added (D11). */
function byMeal(served: Iterable<ServedDish>, dishes: Iterable<Dish>): Map<string, TileDish[]> {
  const library = new Map<string, Dish>();
  for (const d of dishes) {
    library.set(d.id, d);
  }
  const meals = new Map<string, TileDish[]>();
  for (const s of served) {
    const dish = library.get(s.dish);
    if (dish === undefined) {
      continue;
    }
    const key = `${s.date} ${s.mealtime}`;
    const meal = meals.get(key) ?? [];
    meal.push({ served: s, dish });
    meals.set(key, meal);
  }
  for (const meal of meals.values()) {
    meal.sort((a, b) => a.served.rank - b.served.rank);
  }
  return meals;
}

/**
 * The days and meal tiles of a range, from the dishes served and the library. A dish served
 * whose dish the copy of the library does not hold yet is left out until the next check of
 * both copies (D15).
 */
export function mealTiles(days: Day[], served: Iterable<ServedDish>, dishes: Iterable<Dish>, todayDay: Day): DayTiles[] {
  const meals = byMeal(served, dishes);
  return days.map((date) => ({
    date,
    today: date === todayDay,
    meals: mealtimes.map((mealtime) => ({
      date,
      mealtime,
      dishes: meals.get(`${date} ${mealtime}`) ?? [],
      inactive: !inBounds(date),
    })),
  }));
}

/** The dishes of a meal, in the order they were added (D11), as mealTiles gives them. */
export function mealDishes(date: Day, mealtime: Mealtime, served: Iterable<ServedDish>, dishes: Iterable<Dish>): TileDish[] {
  return byMeal(served, dishes).get(`${date} ${mealtime}`) ?? [];
}

/**
 * The dish suggestions of a meal (D12): without search, the whole library by alphabetical
 * order (Q30); with one, the results of the search of the tab Plats (P3); never the dishes
 * already served in the meal (Q21).
 */
export function dishSuggestions(dishes: Iterable<Dish>, meal: TileDish[], query: string): Dish[] {
  const served = new Set(meal.map((d) => d.dish.id));
  const found = query.trim() === "" ? sortDishes(dishes) : searchDishes(dishes, query).map((r) => r.dish);
  return found.filter((d) => !served.has(d.id));
}

/** The total of the servings of a meal. */
export function totalServings(dishes: TileDish[]): number {
  return dishes.reduce((total, d) => total + d.served.servings, 0);
}

/** A number of servings: « 1 part », « 7 parts ». */
export function servingsLabel(n: number): string {
  return n > 1 ? `${n} parts` : `${n} part`;
}
