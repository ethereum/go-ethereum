import { minutes } from "./minutes";

/**
 * Converts hours into seconds
 */
export function hours(n: number): number {
  return minutes(n) * 60;
}
