import { hours } from "./hours";

/**
 * Converts days into seconds
 */
export function days(n: number): number {
  return hours(n) * 24;
}
