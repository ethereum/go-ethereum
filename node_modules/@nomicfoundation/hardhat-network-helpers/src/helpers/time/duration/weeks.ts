import { days } from "./days";

/**
 * Converts weeks into seconds
 */
export function weeks(n: number): number {
  return days(n) * 7;
}
