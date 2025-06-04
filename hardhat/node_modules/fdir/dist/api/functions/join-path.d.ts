import { Options, PathSeparator } from "../../types";
export declare function joinPathWithBasePath(filename: string, directoryPath: string): string;
export declare function joinDirectoryPath(filename: string, directoryPath: string, separator: PathSeparator): string;
export type JoinPathFunction = (filename: string, directoryPath: string) => string;
export declare function build(root: string, options: Options): JoinPathFunction;
