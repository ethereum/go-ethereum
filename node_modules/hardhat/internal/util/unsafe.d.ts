/**
 * This function is a typed version of `Object.keys`. Note that it's type
 * unsafe. You have to be sure that `o` has exactly the same keys as `T`.
 */
export declare const unsafeObjectKeys: <T>(o: T) => Extract<keyof T, string>[];
/**
 * This function is a typed version of `Object.entries`. Note that it's type
 * unsafe. You have to be sure that `o` has exactly the same keys as `T`.
 */
export declare function unsafeObjectEntries<T extends object>(o: T): [keyof T, T[keyof T]][];
//# sourceMappingURL=unsafe.d.ts.map