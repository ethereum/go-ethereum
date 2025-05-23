/**
 * @file See [Getting started with fp-ts: Semigroup](https://dev.to/gcanti/getting-started-with-fp-ts-semigroup-2mf7)
 */
import { Ord } from './Ord';
import { Magma } from './Magma';
/**
 * A `Semigroup` is a `Magma` where `concat` is associative, that is:
 *
 * Associativiy: `concat(concat(x, y), z) = concat(x, concat(y, z))`
 *
 * @since 1.0.0
 */
export interface Semigroup<A> extends Magma<A> {
}
/**
 * @since 1.0.0
 */
export declare const fold: <A>(S: Semigroup<A>) => (a: A) => (as: A[]) => A;
/**
 * @since 1.0.0
 */
export declare const getFirstSemigroup: <A = never>() => Semigroup<A>;
/**
 * @since 1.0.0
 */
export declare const getLastSemigroup: <A = never>() => Semigroup<A>;
/**
 * Given a tuple of semigroups returns a semigroup for the tuple
 *
 * @example
 * import { getTupleSemigroup, semigroupString, semigroupSum, semigroupAll } from 'fp-ts/lib/Semigroup'
 *
 * const S1 = getTupleSemigroup(semigroupString, semigroupSum)
 * assert.deepStrictEqual(S1.concat(['a', 1], ['b', 2]), ['ab', 3])
 *
 * const S2 = getTupleSemigroup(semigroupString, semigroupSum, semigroupAll)
 * assert.deepStrictEqual(S2.concat(['a', 1, true], ['b', 2, false]), ['ab', 3, false])
 *
 * @since 1.14.0
 */
export declare const getTupleSemigroup: <T extends Semigroup<any>[]>(...semigroups: T) => Semigroup<{ [K in keyof T]: T[K] extends Semigroup<infer A> ? A : never; }>;
/**
 * Use `getTupleSemigroup` instead
 * @since 1.0.0
 * @deprecated
 */
export declare const getProductSemigroup: <A, B>(SA: Semigroup<A>, SB: Semigroup<B>) => Semigroup<[A, B]>;
/**
 * @since 1.0.0
 */
export declare const getDualSemigroup: <A>(S: Semigroup<A>) => Semigroup<A>;
/**
 * @since 1.0.0
 */
export declare const getFunctionSemigroup: <S>(S: Semigroup<S>) => <A = never>() => Semigroup<(a: A) => S>;
/**
 * @since 1.14.0
 */
export declare const getStructSemigroup: <O extends {
    [key: string]: any;
}>(semigroups: { [K in keyof O]: Semigroup<O[K]>; }) => Semigroup<O>;
/**
 * Use `getStructSemigroup` instead
 * @since 1.0.0
 * @deprecated
 */
export declare const getRecordSemigroup: <O extends {
    [key: string]: any;
}>(semigroups: { [K in keyof O]: Semigroup<O[K]>; }) => Semigroup<O>;
/**
 * @since 1.0.0
 */
export declare const getMeetSemigroup: <A>(O: Ord<A>) => Semigroup<A>;
/**
 * @since 1.0.0
 */
export declare const getJoinSemigroup: <A>(O: Ord<A>) => Semigroup<A>;
/**
 * Boolean semigroup under conjunction
 * @since 1.0.0
 */
export declare const semigroupAll: Semigroup<boolean>;
/**
 * Boolean semigroup under disjunction
 * @since 1.0.0
 */
export declare const semigroupAny: Semigroup<boolean>;
/**
 * Use `Array`'s `getMonoid`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const getArraySemigroup: <A = never>() => Semigroup<A[]>;
/**
 * Use `Record`'s `getMonoid`
 * @since 1.4.0
 * @deprecated
 */
export declare function getDictionarySemigroup<K extends string, A>(S: Semigroup<A>): Semigroup<Record<K, A>>;
export declare function getDictionarySemigroup<A>(S: Semigroup<A>): Semigroup<{
    [key: string]: A;
}>;
/**
 * Returns a `Semigroup` instance for objects preserving their type
 *
 * @example
 * import { getObjectSemigroup } from 'fp-ts/lib/Semigroup'
 *
 * interface Person {
 *   name: string
 *   age: number
 * }
 *
 * const S = getObjectSemigroup<Person>()
 * assert.deepStrictEqual(S.concat({ name: 'name', age: 23 }, { name: 'name', age: 24 }), { name: 'name', age: 24 })
 *
 * @since 1.4.0
 */
export declare const getObjectSemigroup: <A extends object = never>() => Semigroup<A>;
/**
 * Number `Semigroup` under addition
 * @since 1.0.0
 */
export declare const semigroupSum: Semigroup<number>;
/**
 * Number `Semigroup` under multiplication
 * @since 1.0.0
 */
export declare const semigroupProduct: Semigroup<number>;
/**
 * @since 1.0.0
 */
export declare const semigroupString: Semigroup<string>;
/**
 * @since 1.0.0
 */
export declare const semigroupVoid: Semigroup<void>;
