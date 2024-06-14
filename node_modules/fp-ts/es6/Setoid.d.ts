/**
 * @file This type class is deprecated, please use `Eq` instead.
 */
/**
 * Use `Eq` instead
 * @since 1.0.0
 * @deprecated
 */
export interface Setoid<A> {
    readonly equals: (x: A, y: A) => boolean;
}
/**
 * Use `Eq.fromEquals` instead
 * @since 1.14.0
 * @deprecated
 */
export declare const fromEquals: <A>(equals: (x: A, y: A) => boolean) => Setoid<A>;
/**
 * Use `Eq.strictEqual` instead
 * @since 1.0.0
 * @deprecated
 */
export declare const strictEqual: <A>(a: A, b: A) => boolean;
/**
 * Use `Eq.eqString` instead
 * @since 1.0.0
 * @deprecated
 */
export declare const setoidString: Setoid<string>;
/**
 * Use `Eq.eqNumber` instead
 * @since 1.0.0
 * @deprecated
 */
export declare const setoidNumber: Setoid<number>;
/**
 * Use `Eq.eqBoolean` instead
 * @since 1.0.0
 * @deprecated
 */
export declare const setoidBoolean: Setoid<boolean>;
/**
 * Use `Array.getMonoid` instead
 * @since 1.0.0
 * @deprecated
 */
export declare const getArraySetoid: <A>(S: Setoid<A>) => Setoid<A[]>;
/**
 * Use `Eq.getStructEq` instead
 * @since 1.14.2
 * @deprecated
 */
export declare const getStructSetoid: <O extends {
    [key: string]: any;
}>(setoids: { [K in keyof O]: Setoid<O[K]>; }) => Setoid<O>;
/**
 * Use `Eq.getStructEq` instead
 * @since 1.0.0
 * @deprecated
 */
export declare const getRecordSetoid: <O extends {
    [key: string]: any;
}>(setoids: { [K in keyof O]: Setoid<O[K]>; }) => Setoid<O>;
/**
 * Use `Eq.getTupleEq` instead
 * @since 1.14.2
 * @deprecated
 */
export declare const getTupleSetoid: <T extends Setoid<any>[]>(...setoids: T) => Setoid<{ [K in keyof T]: T[K] extends Setoid<infer A> ? A : never; }>;
/**
 * Use `Eq.getTupleEq` instead
 * @since 1.0.0
 * @deprecated
 */
export declare const getProductSetoid: <A, B>(SA: Setoid<A>, SB: Setoid<B>) => Setoid<[A, B]>;
/**
 * Use `Eq.contramap` instead
 * @since 1.2.0
 * @deprecated
 */
export declare const contramap: <A, B>(f: (b: B) => A, fa: Setoid<A>) => Setoid<B>;
/**
 * Use `Eq.eqDate` instead
 * @since 1.4.0
 * @deprecated
 */
export declare const setoidDate: Setoid<Date>;
