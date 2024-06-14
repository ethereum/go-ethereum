import { CustomError } from "../core/errors";
export declare class FileSystemAccessError extends CustomError {
}
export declare class FileNotFoundError extends CustomError {
    constructor(filePath: string, parent?: Error);
}
export declare class InvalidDirectoryError extends CustomError {
    constructor(filePath: string, parent: Error);
}
/**
 * Returns the real path of absolutePath, resolving symlinks.
 *
 * @throws FileNotFoundError if absolutePath doesn't exist.
 */
export declare function getRealPath(absolutePath: string): Promise<string>;
/**
 * Sync version of getRealPath
 *
 * @see getRealCase
 */
export declare function getRealPathSync(absolutePath: string): string;
/**
 * Returns an array of files (not dirs) that match a condition.
 *
 * @param absolutePathToDir A directory. If it doesn't exist `[]` is returned.
 * @param matches A function to filter files (not directories)
 * @returns An array of absolute paths. Each file has its true case, except
 *  for the initial absolutePathToDir part, which preserves the given casing.
 *  No order is guaranteed.
 */
export declare function getAllFilesMatching(absolutePathToDir: string, matches?: (absolutePathToFile: string) => boolean): Promise<string[]>;
/**
 * Sync version of getAllFilesMatching
 *
 * @see getAllFilesMatching
 */
export declare function getAllFilesMatchingSync(absolutePathToDir: string, matches?: (absolutePathToFile: string) => boolean): string[];
/**
 * Returns the true case relative path of `relativePath` from `from`, without
 * resolving symlinks.
 */
export declare function getFileTrueCase(from: string, relativePath: string): Promise<string>;
/**
 * Sync version of getFileTrueCase
 *
 * @see getFileTrueCase
 */
export declare function getFileTrueCaseSync(from: string, relativePath: string): string;
//# sourceMappingURL=fs-utils.d.ts.map