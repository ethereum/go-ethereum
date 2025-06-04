import { Options } from "../../types";
export type GetArrayFunction = (paths: string[]) => string[];
export declare function build(options: Options): GetArrayFunction;
