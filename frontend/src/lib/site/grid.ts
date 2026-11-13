/**
 * Balanced card grid: 1 column on phones, 2 from `sm`, 3 from `lg` (on a 6-track
 * grid so a short last row can use wider cards). No breakpoint ever leaves a single
 * card alone on its last row.
 */

export const BALANCED_GRID = "grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-6";

/** Classes for the card at `index` of `count` cards in BALANCED_GRID. */
export function balancedCardClass(index: number, count: number): string {
  const classes: string[] = [];

  // Two columns: an odd count makes the last card span the row.
  if (count % 2 === 1 && index === count - 1) classes.push("sm:col-span-2");

  // Three columns on six tracks: a remainder of 2 gives the last two cards half
  // the row each; a remainder of 1 turns the last four into two rows of two.
  const remainder = count % 3;
  const wideFrom = count === 1 ? 0 : remainder === 2 ? count - 2 : remainder === 1 ? count - 4 : count;
  if (count === 1) classes.push("lg:col-span-6");
  else if (index >= wideFrom) classes.push("lg:col-span-3");
  else classes.push("lg:col-span-2");

  return classes.join(" ");
}
