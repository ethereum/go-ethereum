import type { GlobOptions } from "tinyglobby";
import * as path from "path";

/**
 * DO NOT USE THIS FUNCTION. It's SLOW and its semantics are optimized for
 * user-facing CLI globs, not traversing the FS.
 *
 * It's not removed because unfortunately some plugins used it, like the truffle
 * ones.
 *
 * @deprecated
 */
export async function glob(
  pattern: string,
  options: GlobOptions = {}
): Promise<string[]> {
  const files = await (await import("tinyglobby")).glob([pattern], options);
  return files.map(path.normalize);
}

/**
 * @deprecated
 * @see glob
 */
export function globSync(pattern: string, options: GlobOptions = {}): string[] {
  const files = (require("tinyglobby") as typeof import("tinyglobby")).globSync(
    [pattern],
    options
  );
  return files.map(path.normalize);
}
