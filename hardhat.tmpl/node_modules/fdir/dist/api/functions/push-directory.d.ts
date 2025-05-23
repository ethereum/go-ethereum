import { FilterPredicate, Options } from "../../types";
export type PushDirectoryFunction = (directoryPath: string, paths: string[], filters?: FilterPredicate[]) => void;
export declare function build(root: string, options: Options): PushDirectoryFunction;
