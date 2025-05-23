import { PathSeparator } from "./types";
export declare function cleanPath(path: string): string;
export declare function convertSlashes(path: string, separator: PathSeparator): string;
export declare function isRootDirectory(path: string): boolean;
export declare function normalizePath(path: string, options: {
    resolvePaths?: boolean;
    normalizePath?: boolean;
    pathSeparator: PathSeparator;
}): string;
