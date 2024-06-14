import { Comonad1 } from './Comonad';
import { FoldableWithIndex1 } from './FoldableWithIndex';
import { Predicate, Refinement } from './function';
import { FunctorWithIndex1 } from './FunctorWithIndex';
import { Monad1 } from './Monad';
import { Option } from './Option';
import { Ord } from './Ord';
import { Semigroup } from './Semigroup';
import { Eq } from './Eq';
import { Show } from './Show';
import { TraversableWithIndex1 } from './TraversableWithIndex';
declare module './HKT' {
    interface URItoKind<A> {
        NonEmptyArray2v: NonEmptyArray<A>;
    }
}
export declare const URI = "NonEmptyArray2v";
export declare type URI = typeof URI;
/**
 * @since 1.15.0
 */
export interface NonEmptyArray<A> extends Array<A> {
    0: A;
    map<B>(f: (a: A, index: number, nea: NonEmptyArray<A>) => B): NonEmptyArray<B>;
    concat(as: Array<A>): NonEmptyArray<A>;
}
/**
 * @since 1.17.0
 */
export declare const getShow: <A>(S: Show<A>) => Show<NonEmptyArray<A>>;
/**
 * Use `cons` instead
 *
 * @since 1.15.0
 * @deprecated
 */
export declare function make<A>(head: A, tail: Array<A>): NonEmptyArray<A>;
/**
 * @since 1.15.0
 */
export declare function head<A>(nea: NonEmptyArray<A>): A;
/**
 * @since 1.15.0
 */
export declare function tail<A>(nea: NonEmptyArray<A>): Array<A>;
/**
 * @since 1.17.3
 */
export declare const reverse: <A>(nea: NonEmptyArray<A>) => NonEmptyArray<A>;
/**
 * @since 1.15.0
 */
export declare function min<A>(ord: Ord<A>): (nea: NonEmptyArray<A>) => A;
/**
 * @since 1.15.0
 */
export declare function max<A>(ord: Ord<A>): (nea: NonEmptyArray<A>) => A;
/**
 * Builds a `NonEmptyArray` from an `Array` returning `none` if `as` is an empty array
 *
 * @since 1.15.0
 */
export declare function fromArray<A>(as: Array<A>): Option<NonEmptyArray<A>>;
/**
 * Builds a `NonEmptyArray` from a provably (compile time) non empty `Array`.
 *
 * @since 1.15.0
 */
export declare function fromNonEmptyArray<A>(as: Array<A> & {
    0: A;
}): NonEmptyArray<A>;
/**
 * Builds a `Semigroup` instance for `NonEmptyArray`
 *
 * @since 1.15.0
 */
export declare const getSemigroup: <A = never>() => Semigroup<NonEmptyArray<A>>;
/**
 * Use `getEq`
 *
 * @since 1.15.0
 * @deprecated
 */
export declare const getSetoid: <A>(E: Eq<A>) => Eq<NonEmptyArray<A>>;
/**
 * @example
 * import { fromNonEmptyArray, getEq, make } from 'fp-ts/lib/NonEmptyArray2v'
 * import { eqNumber } from 'fp-ts/lib/Eq'
 *
 * const S = getEq(eqNumber)
 * assert.strictEqual(S.equals(make(1, [2]), fromNonEmptyArray([1, 2])), true)
 * assert.strictEqual(S.equals(make(1, [2]), fromNonEmptyArray([1, 3])), false)
 *
 * @since 1.19.0
 */
export declare function getEq<A>(E: Eq<A>): Eq<NonEmptyArray<A>>;
/**
 * Group equal, consecutive elements of an array into non empty arrays.
 *
 * @example
 * import { make, group } from 'fp-ts/lib/NonEmptyArray2v'
 * import { ordNumber } from 'fp-ts/lib/Ord'
 *
 * assert.deepStrictEqual(group(ordNumber)([1, 2, 1, 1]), [
 *   make(1, []),
 *   make(2, []),
 *   make(1, [1])
 * ])
 *
 * @since 1.15.0
 */
export declare const group: <A>(E: Eq<A>) => (as: A[]) => NonEmptyArray<A>[];
/**
 * Sort and then group the elements of an array into non empty arrays.
 *
 * @example
 * import { make, groupSort } from 'fp-ts/lib/NonEmptyArray2v'
 * import { ordNumber } from 'fp-ts/lib/Ord'
 *
 * assert.deepStrictEqual(groupSort(ordNumber)([1, 2, 1, 1]), [make(1, [1, 1]), make(2, [])])
 *
 * @since 1.15.0
 */
export declare const groupSort: <A>(O: Ord<A>) => (as: A[]) => NonEmptyArray<A>[];
/**
 * Splits an array into sub-non-empty-arrays stored in an object, based on the result of calling a `string`-returning
 * function on each element, and grouping the results according to values returned
 *
 * @example
 * import { cons, groupBy } from 'fp-ts/lib/NonEmptyArray2v'
 *
 * assert.deepStrictEqual(groupBy((s: string) => String(s.length))(['foo', 'bar', 'foobar']), {
 *   '3': cons('foo', ['bar']),
 *   '6': cons('foobar', [])
 * })
 *
 * @since 1.15.0
 */
export declare function groupBy<A>(f: (a: A) => string): (as: Array<A>) => {
    [key: string]: NonEmptyArray<A>;
};
/** @deprecated */
export declare function groupBy<A>(as: Array<A>, f: (a: A) => string): {
    [key: string]: NonEmptyArray<A>;
};
/**
 * @since 1.15.0
 */
export declare function last<A>(nea: NonEmptyArray<A>): A;
/**
 * @since 1.15.0
 */
export declare function sort<A>(O: Ord<A>): (nea: NonEmptyArray<A>) => NonEmptyArray<A>;
/**
 * Use `Array`'s `findFirst`
 *
 * @since 1.15.0
 * @deprecated
 */
export declare function findFirst<A, B extends A>(nea: NonEmptyArray<A>, refinement: Refinement<A, B>): Option<B>;
export declare function findFirst<A>(nea: NonEmptyArray<A>, predicate: Predicate<A>): Option<A>;
/**
 * Use `Array`'s `findLast`
 *
 * @since 1.15.0
 * @deprecated
 */
export declare function findLast<A, B extends A>(nea: NonEmptyArray<A>, refinement: Refinement<A, B>): Option<B>;
export declare function findLast<A>(nea: NonEmptyArray<A>, predicate: Predicate<A>): Option<A>;
/**
 * Use `Array`'s `findIndex`
 *
 * @since 1.15.0
 * @deprecated
 */
export declare function findIndex<A>(nea: NonEmptyArray<A>, predicate: Predicate<A>): Option<number>;
/**
 * Use `Array`'s `findLastIndex`
 *
 * @since 1.15.0
 * @deprecated
 */
export declare function findLastIndex<A>(nea: NonEmptyArray<A>, predicate: Predicate<A>): Option<number>;
/**
 * @since 1.15.0
 */
export declare function insertAt<A>(i: number, a: A): (nea: NonEmptyArray<A>) => Option<NonEmptyArray<A>>;
/** @deprecated */
export declare function insertAt<A>(i: number, a: A, nea: NonEmptyArray<A>): Option<NonEmptyArray<A>>;
/**
 * @since 1.15.0
 */
export declare function updateAt<A>(i: number, a: A): (nea: NonEmptyArray<A>) => Option<NonEmptyArray<A>>;
/** @deprecated */
export declare function updateAt<A>(i: number, a: A, nea: NonEmptyArray<A>): Option<NonEmptyArray<A>>;
/**
 * @since 1.17.0
 */
export declare function modifyAt<A>(i: number, f: (a: A) => A): (nea: NonEmptyArray<A>) => Option<NonEmptyArray<A>>;
/** @deprecated */
export declare function modifyAt<A>(nea: NonEmptyArray<A>, i: number, f: (a: A) => A): Option<NonEmptyArray<A>>;
/**
 * @since 1.17.0
 */
export declare const copy: <A>(nea: NonEmptyArray<A>) => NonEmptyArray<A>;
/**
 * @since 1.15.0
 */
export declare function filter<A, B extends A>(refinement: Refinement<A, B>): (nea: NonEmptyArray<A>) => Option<NonEmptyArray<A>>;
export declare function filter<A>(predicate: Predicate<A>): (nea: NonEmptyArray<A>) => Option<NonEmptyArray<A>>;
/** @deprecated */
export declare function filter<A, B extends A>(nea: NonEmptyArray<A>, refinement: Refinement<A, B>): Option<NonEmptyArray<A>>;
/** @deprecated */
export declare function filter<A>(nea: NonEmptyArray<A>, predicate: Predicate<A>): Option<NonEmptyArray<A>>;
/**
 * @since 1.15.0
 */
export declare function filterWithIndex<A>(predicate: (i: number, a: A) => boolean): (nea: NonEmptyArray<A>) => Option<NonEmptyArray<A>>;
/** @deprecated */
export declare function filterWithIndex<A>(nea: NonEmptyArray<A>, predicate: (i: number, a: A) => boolean): Option<NonEmptyArray<A>>;
/**
 * Append an element to the end of an array, creating a new non empty array
 *
 * @example
 * import { snoc } from 'fp-ts/lib/NonEmptyArray2v'
 *
 * assert.deepStrictEqual(snoc([1, 2, 3], 4), [1, 2, 3, 4])
 *
 * @since 1.16.0
 */
export declare const snoc: <A>(as: Array<A>, a: A) => NonEmptyArray<A>;
/**
 * Append an element to the front of an array, creating a new non empty array
 *
 * @example
 * import { cons } from 'fp-ts/lib/NonEmptyArray2v'
 *
 * assert.deepStrictEqual(cons(1, [2, 3, 4]), [1, 2, 3, 4])
 *
 * @since 1.16.0
 */
export declare const cons: <A>(a: A, as: Array<A>) => NonEmptyArray<A>;
/**
 * @since 1.15.0
 */
export declare const nonEmptyArray: Monad1<URI> & Comonad1<URI> & TraversableWithIndex1<URI, number> & FunctorWithIndex1<URI, number> & FoldableWithIndex1<URI, number>;
/**
 * @since 1.19.0
 */
export declare const of: <A>(a: A) => NonEmptyArray<A>;
declare const ap: <A>(fa: NonEmptyArray<A>) => <B>(fab: NonEmptyArray<(a: A) => B>) => NonEmptyArray<B>, apFirst: <B>(fb: NonEmptyArray<B>) => <A>(fa: NonEmptyArray<A>) => NonEmptyArray<A>, apSecond: <B>(fb: NonEmptyArray<B>) => <A>(fa: NonEmptyArray<A>) => NonEmptyArray<B>, chain: <A, B>(f: (a: A) => NonEmptyArray<B>) => (ma: NonEmptyArray<A>) => NonEmptyArray<B>, chainFirst: <A, B>(f: (a: A) => NonEmptyArray<B>) => (ma: NonEmptyArray<A>) => NonEmptyArray<A>, duplicate: <A>(ma: NonEmptyArray<A>) => NonEmptyArray<NonEmptyArray<A>>, extend: <A, B>(f: (fa: NonEmptyArray<A>) => B) => (ma: NonEmptyArray<A>) => NonEmptyArray<B>, flatten: <A>(mma: NonEmptyArray<NonEmptyArray<A>>) => NonEmptyArray<A>, foldMap: <M>(M: import("./Monoid").Monoid<M>) => <A>(f: (a: A) => M) => (fa: NonEmptyArray<A>) => M, foldMapWithIndex: <M>(M: import("./Monoid").Monoid<M>) => <A>(f: (i: number, a: A) => M) => (fa: NonEmptyArray<A>) => M, map: <A, B>(f: (a: A) => B) => (fa: NonEmptyArray<A>) => NonEmptyArray<B>, mapWithIndex: <A, B>(f: (i: number, a: A) => B) => (fa: NonEmptyArray<A>) => NonEmptyArray<B>, reduce: <A, B>(b: B, f: (b: B, a: A) => B) => (fa: NonEmptyArray<A>) => B, reduceRight: <A, B>(b: B, f: (a: A, b: B) => B) => (fa: NonEmptyArray<A>) => B, reduceRightWithIndex: <A, B>(b: B, f: (i: number, a: A, b: B) => B) => (fa: NonEmptyArray<A>) => B, reduceWithIndex: <A, B>(b: B, f: (i: number, b: B, a: A) => B) => (fa: NonEmptyArray<A>) => B;
export { ap, apFirst, apSecond, chain, chainFirst, duplicate, extend, flatten, foldMap, foldMapWithIndex, map, mapWithIndex, reduce, reduceRight, reduceRightWithIndex, reduceWithIndex };
