import type { IOptions as GlobOptions } from "glob";

import * as path from "path";
import util from "util";

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
  const { default: globModule } = await import("glob");
  const files = await util.promisify(globModule)(pattern, options);
  return files.map(path.normalize);
}

/**
 * @deprecated
 * @see glob
 */
export function globSync(pattern: string, options: GlobOptions = {}): string[] {
  const files = require("glob").sync(pattern, options);
  return files.map(path.normalize);
}
