import { Group, Options } from "../../types";
export type GroupFilesFunction = (groups: Group[], directory: string, files: string[]) => void;
export declare function build(options: Options): GroupFilesFunction;
