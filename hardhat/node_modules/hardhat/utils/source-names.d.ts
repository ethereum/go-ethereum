/**
 * This function validates the source name's format.
 *
 * It throws if the format is invalid.
 * If it doesn't throw all you know is that the format is valid.
 */
export declare function validateSourceNameFormat(sourceName: string): void;
/**
 * This function returns true if the sourceName is, potentially, from a local
 * file. It doesn't validate that the file actually exists.
 *
 * The source name must be in a valid format.
 */
export declare function isLocalSourceName(projectRoot: string, sourceName: string): Promise<boolean>;
/**
 * Validates that a source name exists, starting from `fromDir`, and has the
 * right casing.
 *
 * The source name must be in a valid format.
 */
export declare function validateSourceNameExistenceAndCasing(fromDir: string, sourceName: string): Promise<void>;
/**
 * Returns the source name of an existing local file's absolute path.
 *
 * Throws is the file doesn't exist, it's not inside the project, or belongs
 * to a library.
 */
export declare function localPathToSourceName(projectRoot: string, localFileAbsolutePath: string): Promise<string>;
/**
 * This function takes a valid local source name and returns its path. The
 * source name doesn't need to point to an existing file.
 */
export declare function localSourceNameToPath(projectRoot: string, sourceName: string): string;
/**
 * Normalizes the source name, for example, by replacing `a/./b` with `a/b`.
 *
 * The sourceName param doesn't have to be a valid source name. It can,
 * for example, be denormalized.
 */
export declare function normalizeSourceName(sourceName: string): string;
/**
 * This function returns true if the sourceName is a unix absolute path or a
 * platform-dependent one.
 *
 * This function is used instead of just `path.isAbsolute` to ensure that
 * source names never start with `/`, even on Windows.
 */
export declare function isAbsolutePathSourceName(sourceName: string): boolean;
/**
 * This function replaces backslashes (\\) with slashes (/).
 *
 * Note that a source name must not contain backslashes.
 */
export declare function replaceBackslashes(str: string): string;
/**
 * This function returns true if the sourceName contains the current package's name
 * as a substring
 */
export declare function includesOwnPackageName(sourceName: string): Promise<boolean>;
//# sourceMappingURL=source-names.d.ts.map