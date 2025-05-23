import { TaskProfile } from "./task-profiling";
export interface Flamegraph {
    name: string;
    value: number;
    children: Flamegraph[];
    parallel: boolean;
}
export declare function profileToFlamegraph(profile: TaskProfile): Flamegraph;
export declare function createFlamegraphHtmlFile(flamegraph: Flamegraph): string;
/**
 * Converts the TaskProfile into a flamegraph, saves it, and returns its path.
 */
export declare function saveFlamegraph(profile: TaskProfile): string;
//# sourceMappingURL=flamegraph.d.ts.map