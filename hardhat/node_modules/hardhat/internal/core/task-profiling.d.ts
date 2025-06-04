export interface TaskProfile {
    name: string;
    start: bigint;
    end?: bigint;
    children: TaskProfile[];
    parallel?: boolean;
}
export declare function createTaskProfile(name: string): TaskProfile;
export declare function completeTaskProfile(taskProfile: TaskProfile): void;
export declare function createParentTaskProfile(taskProfile: TaskProfile): TaskProfile;
/**
 * Sets `parallel` to `true` to any children that was running at the same time
 * of another.
 *
 * We assume `children[]` is in chronological `start` order.
 */
export declare function flagParallelChildren(profile: TaskProfile, isParentParallel?: boolean): void;
//# sourceMappingURL=task-profiling.d.ts.map