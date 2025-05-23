import { Bounded } from './Bounded';
import { Endomorphism } from './function';
import { Semigroup } from './Semigroup';
/**
 * @since 1.0.0
 */
export interface Monoid<A> extends Semigroup<A> {
    readonly empty: A;
}
/**
 * @since 1.0.0
 */
export declare const fold: <A>(M: Monoid<A>) => (as: A[]) => A;
/**
 * Given a tuple of monoids returns a monoid for the tuple
 *
 * @example
 * import { getTupleMonoid, monoidString, monoidSum, monoidAll } from 'fp-ts/lib/Monoid'
 *
 * const M1 = getTupleMonoid(monoidString, monoidSum)
 * assert.deepStrictEqual(M1.concat(['a', 1], ['b', 2]), ['ab', 3])
 *
 * const M2 = getTupleMonoid(monoidString, monoidSum, monoidAll)
 * assert.deepStrictEqual(M2.concat(['a', 1, true], ['b', 2, false]), ['ab', 3, false])
 *
 * @since 1.0.0
 */
export declare const getTupleMonoid: <T extends Monoid<any>[]>(...monoids: T) => Monoid<{ [K in keyof T]: T[K] extends Semigroup<infer A> ? A : never; }>;
/**
 * Use `getTupleMonoid` instead
 * @since 1.0.0
 * @deprecated
 */
export declare const getProductMonoid: <A, B>(MA: Monoid<A>, MB: Monoid<B>) => Monoid<[A, B]>;
/**
 * @since 1.0.0
 */
export declare const getDualMonoid: <A>(M: Monoid<A>) => Monoid<A>;
/**
 * Boolean monoid under conjunction
 * @since 1.0.0
 */
export declare const monoidAll: Monoid<boolean>;
/**
 * Boolean monoid under disjunction
 * @since 1.0.0
 */
export declare const monoidAny: Monoid<boolean>;
/**
 * @since 1.0.0
 */
export declare const unsafeMonoidArray: Monoid<Array<any>>;
/**
 * Use `Array`'s `getMonoid`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const getArrayMonoid: <A = never>() => Monoid<A[]>;
/**
 * Use `Record`'s `getMonoid`
 * @since 1.4.0
 * @deprecated
 */
export declare function getDictionaryMonoid<K extends string, A>(S: Semigroup<A>): Monoid<Record<K, A>>;
export declare function getDictionaryMonoid<A>(S: Semigroup<A>): Monoid<{
    [key: string]: A;
}>;
/**
 * Number monoid under addition
 * @since 1.0.0
 */
export declare const monoidSum: Monoid<number>;
/**
 * Number monoid under multiplication
 * @since 1.0.0
 */
export declare const monoidProduct: Monoid<number>;
/**
 * @since 1.0.0
 */
export declare const monoidString: Monoid<string>;
/**
 * @since 1.0.0
 */
export declare const monoidVoid: Monoid<void>;
/**
 * @since 1.0.0
 */
export declare const getFunctionMonoid: <M>(M: Monoid<M>) => <A = never>() => Monoid<(a: A) => M>;
/**
 * @since 1.0.0
 */
export declare const getEndomorphismMonoid: <A = never>() => Monoid<Endomorphism<A>>;
/**
 * @since 1.14.0
 */
export declare const getStructMonoid: <O extends {
    [key: string]: any;
}>(monoids: { [K in keyof O]: Monoid<O[K]>; }) => Monoid<O>;
/**
 * Use `getStructMonoid` instead
 * @since 1.0.0
 * @deprecated
 */
export declare const getRecordMonoid: <O extends {
    [key: string]: any;
}>(monoids: { [K in keyof O]: Monoid<O[K]>; }) => Monoid<O>;
/**
 * @since 1.9.0
 */
export declare const getMeetMonoid: <A>(B: Bounded<A>) => Monoid<A>;
/**
 * @since 1.9.0
 */
export declare const getJoinMonoid: <A>(B: Bounded<A>) => Monoid<A>;
