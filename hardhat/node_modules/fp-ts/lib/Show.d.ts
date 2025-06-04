/**
 * The `Show` type class represents those types which can be converted into
 * a human-readable `string` representation.
 *
 * While not required, it is recommended that for any expression `x`, the
 * string `show x` be executable TypeScript code which evaluates to the same
 * value as the expression `x`.
 */
export interface Show<A> {
    show: (a: A) => string;
}
/**
 * @since 1.17.0
 */
export declare const showString: Show<string>;
/**
 * @since 1.17.0
 */
export declare const showNumber: Show<number>;
/**
 * @since 1.17.0
 */
export declare const showBoolean: Show<boolean>;
/**
 * @since 1.17.0
 */
export declare const getStructShow: <O extends {
    [key: string]: any;
}>(shows: { [K in keyof O]: Show<O[K]>; }) => Show<O>;
/**
 * @since 1.17.0
 */
export declare const getTupleShow: <T extends Show<any>[]>(...shows: T) => Show<{ [K in keyof T]: T[K] extends Show<infer A> ? A : never; }>;
