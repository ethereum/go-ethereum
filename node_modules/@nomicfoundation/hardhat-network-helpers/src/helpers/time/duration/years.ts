import { days } from "./days";

/**
 * Converts years into seconds
 */
export function years(n: number): number {
  return days(n) * 365;
}
