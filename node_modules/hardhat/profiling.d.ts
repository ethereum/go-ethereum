/**
 * Utility to create ad-hoc profiles when computing flamegraphs. You can think
 * of these as virtual tasks that execute the function `f`.
 *
 * This is an **unstable** feature, only meant for development. DO NOT use in
 * production code nor plugins.
 *
 * @param name The name of the profile. Think of it as a virtual task name.
 * @param f The function you want to profile.
 */
export declare function adhocProfile<T>(name: string, f: () => Promise<T>): Promise<T>;
/**
 * Sync version of adhocProfile
 *
 * @see adhocProfile
 */
export declare function adhocProfileSync<T>(name: string, f: () => T): T;
//# sourceMappingURL=profiling.d.ts.map