import type { IOptions as GlobOptions } from "glob";
/**
 * DO NOT USE THIS FUNCTION. It's SLOW and its semantics are optimized for
 * user-facing CLI globs, not traversing the FS.
 *
 * It's not removed because unfortunately some plugins used it, like the truffle
 * ones.
 *
 * @deprecated
 */
export declare function glob(pattern: string, options?: GlobOptions): Promise<string[]>;
/**
 * @deprecated
 * @see glob
 */
export declare function globSync(pattern: string, options?: GlobOptions): string[];
//# sourceMappingURL=glob.d.ts.map