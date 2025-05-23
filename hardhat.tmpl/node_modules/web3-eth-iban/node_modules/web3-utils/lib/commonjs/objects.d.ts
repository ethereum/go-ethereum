/**
 * Deep merge two objects.
 * @param destination - The destination object.
 * @param sources - An array of source objects.
 * @returns - The merged object.
 */
export declare const mergeDeep: (destination: Record<string, unknown>, ...sources: Record<string, unknown>[]) => Record<string, unknown>;
