import { FilterPredicate, Options, Counts } from "../../types";
export type PushFileFunction = (directoryPath: string, paths: string[], counts: Counts, filters?: FilterPredicate[]) => void;
export declare function build(options: Options): PushFileFunction;
