/**
 * @file Adapted from https://github.com/purescript/purescript-arrays
 */
import { Alternative1 } from './Alternative';
import { Applicative, Applicative1, Applicative2, Applicative2C, Applicative3, Applicative3C } from './Applicative';
import { Compactable1, Separated } from './Compactable';
import { Either } from './Either';
import { Extend1 } from './Extend';
import { FilterableWithIndex1 } from './FilterableWithIndex';
import { Foldable2v1 } from './Foldable2v';
import { FoldableWithIndex1 } from './FoldableWithIndex';
import { Endomorphism, Predicate, Refinement } from './function';
import { FunctorWithIndex1 } from './FunctorWithIndex';
import { HKT, Kind, Kind2, Kind3, URIS, URIS2, URIS3 } from './HKT';
import { Monad1 } from './Monad';
import { Monoid } from './Monoid';
import { Option } from './Option';
import { Ord } from './Ord';
import { Plus1 } from './Plus';
import { Eq } from './Eq';
import { TraversableWithIndex1 } from './TraversableWithIndex';
import { Unfoldable1 } from './Unfoldable';
import { Witherable1 } from './Witherable';
import { NonEmptyArray } from './NonEmptyArray2v';
import { Show } from './Show';
declare global {
    interface Array<T> {
        /** phantom property added by `fp-ts` */
        _URI: URI;
        /** phantom property added by `fp-ts` */
        _A: T;
    }
}
declare module './HKT' {
    interface URItoKind<A> {
        Array: Array<A>;
    }
}
export declare const URI = "Array";
export declare type URI = typeof URI;
/**
 * @since 1.17.0
 */
export declare const getShow: <A>(S: Show<A>) => Show<A[]>;
/**
 *
 * @example
 * import { getMonoid } from 'fp-ts/lib/Array'
 *
 * const M = getMonoid<number>()
 * assert.deepStrictEqual(M.concat([1, 2], [3, 4]), [1, 2, 3, 4])
 *
 * @since 1.0.0
 */
export declare const getMonoid: <A = never>() => Monoid<A[]>;
/**
 * Use `getEq`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const getSetoid: <A>(E: Eq<A>) => Eq<Array<A>>;
/**
 * Derives a `Eq` over the `Array` of a given element type from the `Eq` of that type. The derived eq defines two
 * arrays as equal if all elements of both arrays are compared equal pairwise with the given eq `S`. In case of
 * arrays of different lengths, the result is non equality.
 *
 * @example
 * import { eqString } from 'fp-ts/lib/Eq'
 * import { getEq } from 'fp-ts/lib/Array'
 *
 * const E = getEq(eqString)
 * assert.strictEqual(E.equals(['a', 'b'], ['a', 'b']), true)
 * assert.strictEqual(E.equals(['a'], []), false)
 *
 * @since 1.19.0
 */
export declare function getEq<A>(E: Eq<A>): Eq<Array<A>>;
/**
 * Derives an `Ord` over the Array of a given element type from the `Ord` of that type. The ordering between two such
 * arrays is equal to: the first non equal comparison of each arrays elements taken pairwise in increasing order, in
 * case of equality over all the pairwise elements; the longest array is considered the greatest, if both arrays have
 * the same length, the result is equality.
 *
 *
 * @example
 * import { getOrd } from 'fp-ts/lib/Array'
 * import { ordString } from 'fp-ts/lib/Ord'
 *
 * const O = getOrd(ordString)
 * assert.strictEqual(O.compare(['b'], ['a']), 1)
 * assert.strictEqual(O.compare(['a'], ['a']), 0)
 * assert.strictEqual(O.compare(['a'], ['b']), -1)
 *
 *
 * @since 1.2.0
 */
export declare const getOrd: <A>(O: Ord<A>) => Ord<A[]>;
/**
 * Use `array.traverse` instead
 *
 * @since 1.0.0
 * @deprecated
 */
export declare function traverse<F extends URIS3>(F: Applicative3<F>): <U, L, A, B>(ta: Array<A>, f: (a: A) => Kind3<F, U, L, B>) => Kind3<F, U, L, Array<B>>;
/** @deprecated */
export declare function traverse<F extends URIS3, U, L>(F: Applicative3C<F, U, L>): <A, B>(ta: Array<A>, f: (a: A) => Kind3<F, U, L, B>) => Kind3<F, U, L, Array<B>>;
/** @deprecated */
export declare function traverse<F extends URIS2>(F: Applicative2<F>): <L, A, B>(ta: Array<A>, f: (a: A) => Kind2<F, L, B>) => Kind2<F, L, Array<B>>;
/** @deprecated */
export declare function traverse<F extends URIS2, L>(F: Applicative2C<F, L>): <A, B>(ta: Array<A>, f: (a: A) => Kind2<F, L, B>) => Kind2<F, L, Array<B>>;
/** @deprecated */
export declare function traverse<F extends URIS>(F: Applicative1<F>): <A, B>(ta: Array<A>, f: (a: A) => Kind<F, B>) => Kind<F, Array<B>>;
/** @deprecated */
export declare function traverse<F>(F: Applicative<F>): <A, B>(ta: Array<A>, f: (a: A) => HKT<F, B>) => HKT<F, Array<B>>;
/**
 * An empty array
 *
 *
 * @since 1.9.0
 */
export declare const empty: Array<never>;
/**
 * Return a list of length `n` with element `i` initialized with `f(i)`
 *
 * @example
 * import { makeBy } from 'fp-ts/lib/Array'
 *
 * const double = (n: number): number => n * 2
 * assert.deepStrictEqual(makeBy(5, double), [0, 2, 4, 6, 8])
 *
 *
 * @since 1.10.0
 */
export declare const makeBy: <A>(n: number, f: (i: number) => A) => A[];
/**
 * Create an array containing a range of integers, including both endpoints
 *
 * @example
 * import { range } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(range(1, 5), [1, 2, 3, 4, 5])
 *
 *
 * @since 1.10.0
 */
export declare const range: (start: number, end: number) => number[];
/**
 * Create an array containing a value repeated the specified number of times
 *
 * @example
 * import { replicate } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(replicate(3, 'a'), ['a', 'a', 'a'])
 *
 *
 * @since 1.10.0
 */
export declare const replicate: <A>(n: number, a: A) => A[];
/**
 * Removes one level of nesting
 *
 * @example
 * import { flatten } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(flatten([[1], [2], [3]]), [1, 2, 3])
 *
 * @since 1.0.0
 */
export declare const flatten: <A>(ffa: A[][]) => A[];
/**
 * Use `foldLeft`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const fold: <A, B>(as: A[], onNil: B, onCons: (head: A, tail: A[]) => B) => B;
/**
 * Use `foldLeft`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const foldL: <A, B>(as: A[], onNil: () => B, onCons: (head: A, tail: A[]) => B) => B;
/**
 * Use `foldRight`
 *
 * @since 1.7.0
 * @deprecated
 */
export declare const foldr: <A, B>(as: A[], onNil: B, onCons: (init: A[], last: A) => B) => B;
/**
 * Use `foldRight`
 *
 * @since 1.7.0
 * @deprecated
 */
export declare const foldrL: <A, B>(as: A[], onNil: () => B, onCons: (init: A[], last: A) => B) => B;
/**
 * Same as `reduce` but it carries over the intermediate steps
 *
 * ```ts
 * import { scanLeft } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(scanLeft([1, 2, 3], 10, (b, a) => b - a), [ 10, 9, 7, 4 ])
 * ```
 *
 *
 * @since 1.1.0
 */
export declare const scanLeft: <A, B>(as: A[], b: B, f: (b: B, a: A) => B) => B[];
/**
 * Fold an array from the right, keeping all intermediate results instead of only the final result
 *
 * @example
 * import { scanRight } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(scanRight([1, 2, 3], 10, (a, b) => b - a), [ 4, 5, 7, 10 ])
 *
 *
 * @since 1.1.0
 */
export declare const scanRight: <A, B>(as: A[], b: B, f: (a: A, b: B) => B) => B[];
/**
 * Test whether an array is empty
 *
 * @example
 * import { isEmpty } from 'fp-ts/lib/Array'
 *
 * assert.strictEqual(isEmpty([]), true)
 *
 * @since 1.0.0
 */
export declare const isEmpty: <A>(as: A[]) => boolean;
/**
 * Test whether an array is non empty narrowing down the type to `NonEmptyArray<A>`
 *
 * @since 1.19.0
 */
export declare function isNonEmpty<A>(as: Array<A>): as is NonEmptyArray<A>;
/**
 * Test whether an array contains a particular index
 *
 * @since 1.0.0
 */
export declare const isOutOfBound: <A>(i: number, as: A[]) => boolean;
/**
 * This function provides a safe way to read a value at a particular index from an array
 *
 * @example
 * import { lookup } from 'fp-ts/lib/Array'
 * import { some, none } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(lookup(1, [1, 2, 3]), some(2))
 * assert.deepStrictEqual(lookup(3, [1, 2, 3]), none)
 *
 * @since 1.14.0
 */
export declare const lookup: <A>(i: number, as: A[]) => Option<A>;
/**
 * Use `lookup` instead
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const index: <A>(i: number, as: A[]) => Option<A>;
/**
 * Attaches an element to the front of an array, creating a new non empty array
 *
 * @example
 * import { cons } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(cons(0, [1, 2, 3]), [0, 1, 2, 3])
 *
 * @since 1.0.0
 */
export declare const cons: <A>(a: A, as: A[]) => NonEmptyArray<A>;
/**
 * Append an element to the end of an array, creating a new non empty array
 *
 * @example
 * import { snoc } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(snoc([1, 2, 3], 4), [1, 2, 3, 4])
 *
 * @since 1.0.0
 */
export declare const snoc: <A>(as: A[], a: A) => NonEmptyArray<A>;
/**
 * Get the first element in an array, or `None` if the array is empty
 *
 * @example
 * import { head } from 'fp-ts/lib/Array'
 * import { some, none } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(head([1, 2, 3]), some(1))
 * assert.deepStrictEqual(head([]), none)
 *
 * @since 1.0.0
 */
export declare const head: <A>(as: A[]) => Option<A>;
/**
 * Get the last element in an array, or `None` if the array is empty
 *
 * @example
 * import { last } from 'fp-ts/lib/Array'
 * import { some, none } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(last([1, 2, 3]), some(3))
 * assert.deepStrictEqual(last([]), none)
 *
 * @since 1.0.0
 */
export declare const last: <A>(as: A[]) => Option<A>;
/**
 * Get all but the first element of an array, creating a new array, or `None` if the array is empty
 *
 * @example
 * import { tail } from 'fp-ts/lib/Array'
 * import { some, none } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(tail([1, 2, 3]), some([2, 3]))
 * assert.deepStrictEqual(tail([]), none)
 *
 * @since 1.0.0
 */
export declare const tail: <A>(as: A[]) => Option<A[]>;
/**
 * Get all but the last element of an array, creating a new array, or `None` if the array is empty
 *
 * @example
 * import { init } from 'fp-ts/lib/Array'
 * import { some, none } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(init([1, 2, 3]), some([1, 2]))
 * assert.deepStrictEqual(init([]), none)
 *
 * @since 1.0.0
 */
export declare const init: <A>(as: A[]) => Option<A[]>;
/**
 * Use `takeLeft`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare function take<A>(n: number, as: Array<A>): Array<A>;
/**
 * Use `takeRight`
 *
 * @since 1.10.0
 * @deprecated
 */
export declare const takeEnd: <A>(n: number, as: A[]) => A[];
/**
 * Use `takeLeftWhile`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare function takeWhile<A, B extends A>(as: Array<A>, predicate: Refinement<A, B>): Array<B>;
/** @deprecated */
export declare function takeWhile<A>(as: Array<A>, predicate: Predicate<A>): Array<A>;
/**
 * Use `spanLeft`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare function span<A, B extends A>(as: Array<A>, predicate: Refinement<A, B>): {
    init: Array<B>;
    rest: Array<A>;
};
export declare function span<A>(as: Array<A>, predicate: Predicate<A>): {
    init: Array<A>;
    rest: Array<A>;
};
/**
 * Use `dropLeft`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const drop: <A>(n: number, as: A[]) => A[];
/**
 * Use `dropRight`
 *
 * @since 1.10.0
 * @deprecated
 */
export declare const dropEnd: <A>(n: number, as: A[]) => A[];
/**
 * Use `dropLeftWhile`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const dropWhile: <A>(as: A[], predicate: Predicate<A>) => A[];
/**
 * Find the first index for which a predicate holds
 *
 * @example
 * import { findIndex } from 'fp-ts/lib/Array'
 * import { some, none } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(findIndex((n: number) => n === 2)([1, 2, 3]), some(1))
 * assert.deepStrictEqual(findIndex((n: number) => n === 2)([]), none)
 *
 * @since 1.0.0
 */
export declare function findIndex<A>(predicate: Predicate<A>): (as: Array<A>) => Option<number>;
/** @deprecated */
export declare function findIndex<A>(as: Array<A>, predicate: Predicate<A>): Option<number>;
/**
 * Find the first element which satisfies a predicate (or a refinement) function
 *
 * @example
 * import { findFirst } from 'fp-ts/lib/Array'
 * import { some } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(findFirst((x: { a: number, b: number }) => x.a === 1)([{ a: 1, b: 1 }, { a: 1, b: 2 }]), some({ a: 1, b: 1 }))
 *
 * @since 1.0.0
 */
export declare function findFirst<A, B extends A>(refinement: Refinement<A, B>): (as: Array<A>) => Option<B>;
export declare function findFirst<A>(predicate: Predicate<A>): (as: Array<A>) => Option<A>;
/** @deprecated */
export declare function findFirst<A, B extends A>(as: Array<A>, predicate: Refinement<A, B>): Option<B>;
/** @deprecated */
export declare function findFirst<A>(as: Array<A>, refinement: Predicate<A>): Option<A>;
/**
 * Find the first element returned by an option based selector function
 *
 * @example
 * import { findFirstMap } from 'fp-ts/lib/Array'
 * import { some, none } from 'fp-ts/lib/Option'
 *
 * interface Person {
 *   name: string
 *   age?: number
 * }
 *
 * const persons: Array<Person> = [{ name: 'John' }, { name: 'Mary', age: 45 }, { name: 'Joey', age: 28 }]
 *
 * // returns the name of the first person that has an age
 * assert.deepStrictEqual(findFirstMap((p: Person) => (p.age === undefined ? none : some(p.name)))(persons), some('Mary'))
 *
 * @since 1.16.0
 */
export declare function findFirstMap<A, B>(f: (a: A) => Option<B>): (arr: Array<A>) => Option<B>;
/** @deprecated */
export declare function findFirstMap<A, B>(arr: Array<A>, f: (a: A) => Option<B>): Option<B>;
/**
 * Find the last element which satisfies a predicate function
 *
 * @example
 * import { findLast } from 'fp-ts/lib/Array'
 * import { some } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(findLast((x: { a: number, b: number }) => x.a === 1)([{ a: 1, b: 1 }, { a: 1, b: 2 }]), some({ a: 1, b: 2 }))
 *
 * @since 1.0.0
 */
export declare function findLast<A, B extends A>(refinement: Refinement<A, B>): (as: Array<A>) => Option<B>;
export declare function findLast<A>(predicate: Predicate<A>): (as: Array<A>) => Option<A>;
/** @deprecated */
export declare function findLast<A, B extends A>(as: Array<A>, refinement: Refinement<A, B>): Option<B>;
/** @deprecated */
export declare function findLast<A>(as: Array<A>, predicate: Predicate<A>): Option<A>;
/**
 * Find the last element returned by an option based selector function
 *
 * @example
 * import { findLastMap } from 'fp-ts/lib/Array'
 * import { some, none } from 'fp-ts/lib/Option'
 *
 * interface Person {
 *   name: string
 *   age?: number
 * }
 *
 * const persons: Array<Person> = [{ name: 'John' }, { name: 'Mary', age: 45 }, { name: 'Joey', age: 28 }]
 *
 * // returns the name of the last person that has an age
 * assert.deepStrictEqual(findLastMap((p: Person) => (p.age === undefined ? none : some(p.name)))(persons), some('Joey'))
 *
 * @since 1.16.0
 */
export declare function findLastMap<A, B>(f: (a: A) => Option<B>): (arr: Array<A>) => Option<B>;
/** @deprecated */
export declare function findLastMap<A, B>(arr: Array<A>, f: (a: A) => Option<B>): Option<B>;
/**
 * Returns the index of the last element of the list which matches the predicate
 *
 * @example
 * import { findLastIndex } from 'fp-ts/lib/Array'
 * import { some, none } from 'fp-ts/lib/Option'
 *
 * interface X {
 *   a: number
 *   b: number
 * }
 * const xs: Array<X> = [{ a: 1, b: 0 }, { a: 1, b: 1 }]
 * assert.deepStrictEqual(findLastIndex((x: { a: number }) => x.a === 1)(xs), some(1))
 * assert.deepStrictEqual(findLastIndex((x: { a: number }) => x.a === 4)(xs), none)
 *
 * @since 1.10.0
 */
export declare function findLastIndex<A>(predicate: Predicate<A>): (as: Array<A>) => Option<number>;
/** @deprecated */
export declare function findLastIndex<A>(as: Array<A>, predicate: Predicate<A>): Option<number>;
/**
 * Use `array.filter` instead
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const refine: <A, B extends A>(as: A[], refinement: Refinement<A, B>) => B[];
/**
 * @since 1.0.0
 */
export declare const copy: <A>(as: A[]) => A[];
/**
 * @since 1.0.0
 */
export declare const unsafeInsertAt: <A>(i: number, a: A, as: A[]) => A[];
/**
 * Insert an element at the specified index, creating a new array, or returning `None` if the index is out of bounds
 *
 * @example
 * import { insertAt } from 'fp-ts/lib/Array'
 * import { some } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(insertAt(2, 5)([1, 2, 3, 4]), some([1, 2, 5, 3, 4]))
 *
 * @since 1.0.0
 */
export declare function insertAt<A>(i: number, a: A): (as: Array<A>) => Option<Array<A>>;
/** @deprecated */
export declare function insertAt<A>(i: number, a: A, as: Array<A>): Option<Array<A>>;
/**
 * @since 1.0.0
 */
export declare const unsafeUpdateAt: <A>(i: number, a: A, as: A[]) => A[];
/**
 * Change the element at the specified index, creating a new array, or returning `None` if the index is out of bounds
 *
 * @example
 * import { updateAt } from 'fp-ts/lib/Array'
 * import { some, none } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(updateAt(1, 1)([1, 2, 3]), some([1, 1, 3]))
 * assert.deepStrictEqual(updateAt(1, 1)([]), none)
 *
 * @since 1.0.0
 */
export declare function updateAt<A>(i: number, a: A): (as: Array<A>) => Option<Array<A>>;
/** @deprecated */
export declare function updateAt<A>(i: number, a: A, as: Array<A>): Option<Array<A>>;
/**
 * @since 1.0.0
 */
export declare const unsafeDeleteAt: <A>(i: number, as: A[]) => A[];
/**
 * Delete the element at the specified index, creating a new array, or returning `None` if the index is out of bounds
 *
 * @example
 * import { deleteAt } from 'fp-ts/lib/Array'
 * import { some, none } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(deleteAt(0)([1, 2, 3]), some([2, 3]))
 * assert.deepStrictEqual(deleteAt(1)([]), none)
 *
 * @since 1.0.0
 */
export declare function deleteAt<A>(i: number): (as: Array<A>) => Option<Array<A>>;
/** @deprecated */
export declare function deleteAt<A>(i: number, as: Array<A>): Option<Array<A>>;
/**
 * Apply a function to the element at the specified index, creating a new array, or returning `None` if the index is out
 * of bounds
 *
 * @example
 * import { modifyAt } from 'fp-ts/lib/Array'
 * import { some, none } from 'fp-ts/lib/Option'
 *
 * const double = (x: number): number => x * 2
 * assert.deepStrictEqual(modifyAt(1, double)([1, 2, 3]), some([1, 4, 3]))
 * assert.deepStrictEqual(modifyAt(1, double)([]), none)
 *
 * @since 1.0.0
 */
export declare function modifyAt<A>(i: number, f: Endomorphism<A>): (as: Array<A>) => Option<Array<A>>;
/** @deprecated */
export declare function modifyAt<A>(as: Array<A>, i: number, f: Endomorphism<A>): Option<Array<A>>;
/**
 * Reverse an array, creating a new array
 *
 * @example
 * import { reverse } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(reverse([1, 2, 3]), [3, 2, 1])
 *
 * @since 1.0.0
 */
export declare const reverse: <A>(as: A[]) => A[];
/**
 * Extracts from an array of `Either` all the `Right` elements. All the `Right` elements are extracted in order
 *
 * @example
 * import { rights } from 'fp-ts/lib/Array'
 * import { right, left } from 'fp-ts/lib/Either'
 *
 * assert.deepStrictEqual(rights([right(1), left('foo'), right(2)]), [1, 2])
 *
 * @since 1.0.0
 */
export declare const rights: <L, A>(as: Either<L, A>[]) => A[];
/**
 * Extracts from an array of `Either` all the `Left` elements. All the `Left` elements are extracted in order
 *
 * @example
 * import { lefts } from 'fp-ts/lib/Array'
 * import { left, right } from 'fp-ts/lib/Either'
 *
 * assert.deepStrictEqual(lefts([right(1), left('foo'), right(2)]), ['foo'])
 *
 * @since 1.0.0
 */
export declare const lefts: <L, A>(as: Either<L, A>[]) => L[];
/**
 * Sort the elements of an array in increasing order, creating a new array
 *
 * @example
 * import { sort } from 'fp-ts/lib/Array'
 * import { ordNumber } from 'fp-ts/lib/Ord'
 *
 * assert.deepStrictEqual(sort(ordNumber)([3, 2, 1]), [1, 2, 3])
 *
 * @since 1.0.0
 */
export declare const sort: <A>(O: Ord<A>) => (as: A[]) => A[];
/**
 * Apply a function to pairs of elements at the same index in two arrays, collecting the results in a new array. If one
 * input array is short, excess elements of the longer array are discarded.
 *
 * @example
 * import { zipWith } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(zipWith([1, 2, 3], ['a', 'b', 'c', 'd'], (n, s) => s + n), ['a1', 'b2', 'c3'])
 *
 * @since 1.0.0
 */
export declare const zipWith: <A, B, C>(fa: A[], fb: B[], f: (a: A, b: B) => C) => C[];
/**
 * Takes two arrays and returns an array of corresponding pairs. If one input array is short, excess elements of the
 * longer array are discarded
 *
 * @example
 * import { zip } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(zip([1, 2, 3], ['a', 'b', 'c', 'd']), [[1, 'a'], [2, 'b'], [3, 'c']])
 *
 * @since 1.0.0
 */
export declare const zip: <A, B>(fa: A[], fb: B[]) => [A, B][];
/**
 * The function is reverse of `zip`. Takes an array of pairs and return two corresponding arrays
 *
 * @example
 * import { unzip } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(unzip([[1, 'a'], [2, 'b'], [3, 'c']]), [[1, 2, 3], ['a', 'b', 'c']])
 *
 *
 * @since 1.13.0
 */
export declare const unzip: <A, B>(as: [A, B][]) => [A[], B[]];
/**
 * Rotate an array to the right by `n` steps
 *
 * @example
 * import { rotate } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(rotate(2)([1, 2, 3, 4, 5]), [4, 5, 1, 2, 3])
 *
 * @since 1.0.0
 */
export declare function rotate<A>(n: number): (xs: Array<A>) => Array<A>;
/** @deprecated */
export declare function rotate<A>(n: number, xs: Array<A>): Array<A>;
/**
 * Test if a value is a member of an array. Takes a `Eq<A>` as a single
 * argument which returns the function to use to search for a value of type `A` in
 * an array of type `Array<A>`.
 *
 * @example
 * import { elem } from 'fp-ts/lib/Array'
 * import { eqNumber } from 'fp-ts/lib/Eq'
 *
 * assert.strictEqual(elem(eqNumber)(1, [1, 2, 3]), true)
 * assert.strictEqual(elem(eqNumber)(4, [1, 2, 3]), false)
 *
 * @since 1.14.0
 */
export declare const elem: <A>(E: Eq<A>) => (a: A, as: A[]) => boolean;
/**
 * Use `elem` instead
 * @since 1.3.0
 * @deprecated
 */
export declare const member: <A>(E: Eq<A>) => (as: A[], a: A) => boolean;
/**
 * Remove duplicates from an array, keeping the first occurance of an element.
 *
 * @example
 * import { uniq } from 'fp-ts/lib/Array'
 * import { eqNumber } from 'fp-ts/lib/Eq'
 *
 * assert.deepStrictEqual(uniq(eqNumber)([1, 2, 1]), [1, 2])
 *
 *
 * @since 1.3.0
 */
export declare const uniq: <A>(E: Eq<A>) => (as: A[]) => A[];
/**
 * Sort the elements of an array in increasing order, where elements are compared using first `ords[0]`, then `ords[1]`,
 * etc...
 *
 * @example
 * import { sortBy } from 'fp-ts/lib/Array'
 * import { contramap, ordString, ordNumber } from 'fp-ts/lib/Ord'
 *
 * interface Person {
 *   name: string
 *   age: number
 * }
 * const byName = contramap((p: Person) => p.name, ordString)
 * const byAge = contramap((p: Person) => p.age, ordNumber)
 *
 * const sortByNameByAge = sortBy([byName, byAge])
 *
 * if (sortByNameByAge.isSome()) {
 *   const persons = [{ name: 'a', age: 1 }, { name: 'b', age: 3 }, { name: 'c', age: 2 }, { name: 'b', age: 2 }]
 *   assert.deepStrictEqual(sortByNameByAge.value(persons), [
 *     { name: 'a', age: 1 },
 *     { name: 'b', age: 2 },
 *     { name: 'b', age: 3 },
 *     { name: 'c', age: 2 }
 *   ])
 * }
 *
 *
 * @since 1.3.0
 */
export declare const sortBy: <A>(ords: Ord<A>[]) => Option<Endomorphism<A[]>>;
/**
 * Non failing version of `sortBy`
 * @example
 * import { sortBy1 } from 'fp-ts/lib/Array'
 * import { contramap, ordString, ordNumber } from 'fp-ts/lib/Ord'
 *
 * interface Person {
 *   name: string
 *   age: number
 * }
 * const byName = contramap((p: Person) => p.name, ordString)
 * const byAge = contramap((p: Person) => p.age, ordNumber)
 *
 * const sortByNameByAge = sortBy1(byName, [byAge])
 *
 * const persons = [{ name: 'a', age: 1 }, { name: 'b', age: 3 }, { name: 'c', age: 2 }, { name: 'b', age: 2 }]
 * assert.deepStrictEqual(sortByNameByAge(persons), [
 *   { name: 'a', age: 1 },
 *   { name: 'b', age: 2 },
 *   { name: 'b', age: 3 },
 *   { name: 'c', age: 2 }
 * ])
 *
 *
 * @since 1.3.0
 */
export declare const sortBy1: <A>(head: Ord<A>, tail: Ord<A>[]) => Endomorphism<A[]>;
/**
 * Use `filterMap`
 *
 * Apply a function to each element in an array, keeping only the results which contain a value, creating a new array.
 *
 * @example
 * import { mapOption } from 'fp-ts/lib/Array'
 * import { Option, some, none } from 'fp-ts/lib/Option'
 *
 * const f = (n: number): Option<number> => (n % 2 === 0 ? none : some(n))
 * assert.deepStrictEqual(mapOption([1, 2, 3], f), [1, 3])
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const mapOption: <A, B>(as: A[], f: (a: A) => Option<B>) => B[];
/**
 * Use `compact`
 *
 * Filter an array of optional values, keeping only the elements which contain a value, creating a new array.
 *
 * @example
 * import { catOptions } from 'fp-ts/lib/Array'
 * import { some, none } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(catOptions([some(1), none, some(3)]), [1, 3])
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const catOptions: <A>(as: Option<A>[]) => A[];
/**
 * @example
 * import { array } from 'fp-ts/lib/Array'
 * import { left, right } from 'fp-ts/lib/Either'
 * import { identity } from 'fp-ts/lib/function'
 *
 * assert.deepStrictEqual(array.partitionMap([right(1), left('foo'), right(2)], identity), { left: ['foo'], right: [1, 2] })
 *
 * @since 1.0.0
 */
export declare function partitionMap<A, L, R>(f: (a: A) => Either<L, R>): (fa: Array<A>) => Separated<Array<L>, Array<R>>;
/** @deprecated */
export declare function partitionMap<A, L, R>(fa: Array<A>, f: (a: A) => Either<L, R>): Separated<Array<L>, Array<R>>;
/**
 * @since 1.0.0
 */
export declare function filter<A, B extends A>(refinement: Refinement<A, B>): (as: Array<A>) => Array<B>;
export declare function filter<A>(predicate: Predicate<A>): (as: Array<A>) => Array<A>;
/** @deprecated */
export declare function filter<A, B extends A>(as: Array<A>, refinement: Refinement<A, B>): Array<B>;
/** @deprecated */
export declare function filter<A>(as: Array<A>, predicate: Predicate<A>): Array<A>;
/**
 * Use `array.partition` instead
 *
 * @since 1.12.0
 */
export declare function partition<A, B extends A>(refinement: Refinement<A, B>): (fa: Array<A>) => Separated<Array<A>, Array<B>>;
export declare function partition<A>(predicate: Predicate<A>): (fa: Array<A>) => Separated<Array<A>, Array<A>>;
/** @deprecated */
export declare function partition<A, B extends A>(fa: Array<A>, refinement: Refinement<A, B>): Separated<Array<A>, Array<B>>;
/** @deprecated */
export declare function partition<A>(fa: Array<A>, predicate: Predicate<A>): Separated<Array<A>, Array<A>>;
/**
 * A useful recursion pattern for processing an array to produce a new array, often used for "chopping" up the input
 * array. Typically `chop` is called with some function that will consume an initial prefix of the array and produce a
 * value and the rest of the array.
 *
 * @example
 * import { Eq, eqNumber } from 'fp-ts/lib/Eq'
 * import { chop, spanLeft } from 'fp-ts/lib/Array'
 *
 * const group = <A>(S: Eq<A>): ((as: Array<A>) => Array<Array<A>>) => {
 *   return chop(as => {
 *     const { init, rest } = spanLeft((a: A) => S.equals(a, as[0]))(as)
 *     return [init, rest]
 *   })
 * }
 * assert.deepStrictEqual(group(eqNumber)([1, 1, 2, 3, 3, 4]), [[1, 1], [2], [3, 3], [4]])
 *
 *
 * @since 1.10.0
 */
export declare function chop<A, B>(f: (as: NonEmptyArray<A>) => [B, Array<A>]): (as: Array<A>) => Array<B>;
/** @deprecated */
export declare function chop<A, B>(as: Array<A>, f: (as: NonEmptyArray<A>) => [B, Array<A>]): Array<B>;
/**
 * Use `splitAt`
 *
 * @since 1.10.0
 * @deprecated
 */
export declare const split: <A>(n: number, as: A[]) => [A[], A[]];
/**
 * Splits an array into length-`n` pieces. The last piece will be shorter if `n` does not evenly divide the length of
 * the array. Note that `chunksOf(n)([])` is `[]`, not `[[]]`. This is intentional, and is consistent with a recursive
 * definition of `chunksOf`; it satisfies the property that
 *
 * ```ts
 * chunksOf(n)(xs).concat(chunksOf(n)(ys)) == chunksOf(n)(xs.concat(ys)))
 * ```
 *
 * whenever `n` evenly divides the length of `xs`.
 *
 * @example
 * import { chunksOf } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(chunksOf(2)([1, 2, 3, 4, 5]), [[1, 2], [3, 4], [5]])
 *
 * @since 1.10.0
 */
export declare function chunksOf<A>(n: number): (as: Array<A>) => Array<Array<A>>;
/** @deprecated */
export declare function chunksOf<A>(as: Array<A>, n: number): Array<Array<A>>;
/**
 * Array comprehension
 *
 * ```
 * [ g(x, y, ...) | x ← xs, y ← ys, ..., f(x, y, ...) ]
 * ```
 *
 * @example
 * import { comprehension } from 'fp-ts/lib/Array'
 * import { tuple } from 'fp-ts/lib/function'
 *
 * assert.deepStrictEqual(comprehension([[1, 2, 3], ['a', 'b']], (a, b) => (a + b.length) % 2 === 0, tuple), [
 *   [1, 'a'],
 *   [1, 'b'],
 *   [3, 'a'],
 *   [3, 'b']
 * ])
 *
 *
 * @since 1.10.0
 */
export declare function comprehension<A, B, C, D, R>(input: [Array<A>, Array<B>, Array<C>, Array<D>], f: (a: A, b: B, c: C, d: D) => boolean, g: (a: A, b: B, c: C, d: D) => R): Array<R>;
export declare function comprehension<A, B, C, R>(input: [Array<A>, Array<B>, Array<C>], f: (a: A, b: B, c: C) => boolean, g: (a: A, b: B, c: C) => R): Array<R>;
export declare function comprehension<A, R>(input: [Array<A>], f: (a: A) => boolean, g: (a: A) => R): Array<R>;
export declare function comprehension<A, B, R>(input: [Array<A>, Array<B>], f: (a: A, b: B) => boolean, g: (a: A, b: B) => R): Array<R>;
export declare function comprehension<A, R>(input: [Array<A>], f: (a: A) => boolean, g: (a: A) => R): Array<R>;
/**
 * Creates an array of unique values, in order, from all given arrays using a `Eq` for equality comparisons
 *
 * @example
 * import { union } from 'fp-ts/lib/Array'
 * import { eqNumber } from 'fp-ts/lib/Eq'
 *
 * assert.deepStrictEqual(union(eqNumber)([1, 2], [2, 3]), [1, 2, 3])
 *
 *
 * @since 1.12.0
 */
export declare const union: <A>(E: Eq<A>) => (xs: A[], ys: A[]) => A[];
/**
 * Creates an array of unique values that are included in all given arrays using a `Eq` for equality
 * comparisons. The order and references of result values are determined by the first array.
 *
 * @example
 * import { intersection } from 'fp-ts/lib/Array'
 * import { eqNumber } from 'fp-ts/lib/Eq'
 *
 * assert.deepStrictEqual(intersection(eqNumber)([1, 2], [2, 3]), [2])
 *
 *
 * @since 1.12.0
 */
export declare const intersection: <A>(E: Eq<A>) => (xs: A[], ys: A[]) => A[];
/**
 * Creates an array of array values not included in the other given array using a `Eq` for equality
 * comparisons. The order and references of result values are determined by the first array.
 *
 * @example
 * import { difference } from 'fp-ts/lib/Array'
 * import { eqNumber } from 'fp-ts/lib/Eq'
 *
 * assert.deepStrictEqual(difference(eqNumber)([1, 2], [2, 3]), [1])
 *
 *
 * @since 1.12.0
 */
export declare const difference: <A>(E: Eq<A>) => (xs: A[], ys: A[]) => A[];
/**
 * @since 1.0.0
 */
export declare const array: Monad1<URI> & Foldable2v1<URI> & Unfoldable1<URI> & TraversableWithIndex1<URI, number> & Alternative1<URI> & Plus1<URI> & Extend1<URI> & Compactable1<URI> & FilterableWithIndex1<URI, number> & Witherable1<URI> & FunctorWithIndex1<URI, number> & FoldableWithIndex1<URI, number>;
/**
 * @since 1.19.0
 */
export declare function of<A>(a: A): Array<A>;
/**
 * Break an array into its first element and remaining elements
 *
 * @example
 * import { foldLeft } from 'fp-ts/lib/Array'
 *
 * const len: <A>(as: Array<A>) => number = foldLeft(() => 0, (_, tail) => 1 + len(tail))
 * assert.strictEqual(len([1, 2, 3]), 3)
 *
 * @since 1.19.0
 */
export declare function foldLeft<A, B>(onNil: () => B, onCons: (head: A, tail: Array<A>) => B): (as: Array<A>) => B;
/**
 * Break an array into its initial elements and the last element
 *
 * @since 1.19.0
 */
export declare function foldRight<A, B>(onNil: () => B, onCons: (init: Array<A>, last: A) => B): (as: Array<A>) => B;
/**
 * Keep only a number of elements from the start of an array, creating a new array.
 * `n` must be a natural number
 *
 * @example
 * import { takeLeft } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(takeLeft(2)([1, 2, 3]), [1, 2])
 *
 * @since 1.19.0
 */
export declare function takeLeft(n: number): <A>(as: Array<A>) => Array<A>;
/**
 * Keep only a number of elements from the end of an array, creating a new array.
 * `n` must be a natural number
 *
 * @example
 * import { takeRight } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(takeRight(2)([1, 2, 3, 4, 5]), [4, 5])
 *
 * @since 1.19.0
 */
export declare function takeRight(n: number): <A>(as: Array<A>) => Array<A>;
/**
 * Calculate the longest initial subarray for which all element satisfy the specified predicate, creating a new array
 *
 * @example
 * import { takeLeftWhile } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(takeLeftWhile((n: number) => n % 2 === 0)([2, 4, 3, 6]), [2, 4])
 *
 * @since 1.19.0
 */
export declare function takeLeftWhile<A, B extends A>(refinement: Refinement<A, B>): (as: Array<A>) => Array<B>;
export declare function takeLeftWhile<A>(predicate: Predicate<A>): (as: Array<A>) => Array<A>;
/**
 * Split an array into two parts:
 * 1. the longest initial subarray for which all elements satisfy the specified predicate
 * 2. the remaining elements
 *
 * @example
 * import { spanLeft } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(spanLeft((n: number) => n % 2 === 1)([1, 3, 2, 4, 5]), { init: [1, 3], rest: [2, 4, 5] })
 *
 * @since 1.19.0
 */
export declare function spanLeft<A, B extends A>(refinement: Refinement<A, B>): (as: Array<A>) => {
    init: Array<B>;
    rest: Array<A>;
};
export declare function spanLeft<A>(predicate: Predicate<A>): (as: Array<A>) => {
    init: Array<A>;
    rest: Array<A>;
};
/**
 * Drop a number of elements from the start of an array, creating a new array
 *
 * @example
 * import { dropLeft } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(dropLeft(2)([1, 2, 3]), [3])
 *
 * @since 1.19.0
 */
export declare function dropLeft(n: number): <A>(as: Array<A>) => Array<A>;
/**
 * Drop a number of elements from the end of an array, creating a new array
 *
 * @example
 * import { dropRight } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(dropRight(2)([1, 2, 3, 4, 5]), [1, 2, 3])
 *
 * @since 1.19.0
 */
export declare function dropRight(n: number): <A>(as: Array<A>) => Array<A>;
/**
 * Remove the longest initial subarray for which all element satisfy the specified predicate, creating a new array
 *
 * @example
 * import { dropLeftWhile } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(dropLeftWhile((n: number) => n % 2 === 1)([1, 3, 2, 4, 5]), [2, 4, 5])
 *
 * @since 1.19.0
 */
export declare function dropLeftWhile<A>(predicate: Predicate<A>): (as: Array<A>) => Array<A>;
/**
 * Splits an array into two pieces, the first piece has `n` elements.
 *
 * @example
 * import { splitAt } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(splitAt(2)([1, 2, 3, 4, 5]), [[1, 2], [3, 4, 5]])
 *
 * @since 1.19.0
 */
export declare function splitAt(n: number): <A>(as: Array<A>) => [Array<A>, Array<A>];
declare const alt: <A>(that: () => A[]) => (fa: A[]) => A[], ap: <A>(fa: A[]) => <B>(fab: ((a: A) => B)[]) => B[], apFirst: <B>(fb: B[]) => <A>(fa: A[]) => A[], apSecond: <B>(fb: B[]) => <A>(fa: A[]) => B[], chain: <A, B>(f: (a: A) => B[]) => (ma: A[]) => B[], chainFirst: <A, B>(f: (a: A) => B[]) => (ma: A[]) => A[], duplicate: <A>(ma: A[]) => A[][], extend: <A, B>(f: (fa: A[]) => B) => (ma: A[]) => B[], filterMap: <A, B>(f: (a: A) => Option<B>) => (fa: A[]) => B[], filterMapWithIndex: <A, B>(f: (i: number, a: A) => Option<B>) => (fa: A[]) => B[], filterWithIndex: {
    <A, B extends A>(refinementWithIndex: import("./FilterableWithIndex").RefinementWithIndex<number, A, B>): (fa: A[]) => B[];
    <A>(predicateWithIndex: import("./FilterableWithIndex").PredicateWithIndex<number, A>): (fa: A[]) => A[];
}, foldMap: <M>(M: Monoid<M>) => <A>(f: (a: A) => M) => (fa: A[]) => M, foldMapWithIndex: <M>(M: Monoid<M>) => <A>(f: (i: number, a: A) => M) => (fa: A[]) => M, map: <A, B>(f: (a: A) => B) => (fa: A[]) => B[], mapWithIndex: <A, B>(f: (i: number, a: A) => B) => (fa: A[]) => B[], partitionMapWithIndex: <A, RL, RR>(f: (i: number, a: A) => Either<RL, RR>) => (fa: A[]) => Separated<RL[], RR[]>, partitionWithIndex: {
    <A, B extends A>(refinementWithIndex: import("./FilterableWithIndex").RefinementWithIndex<number, A, B>): (fa: A[]) => Separated<A[], B[]>;
    <A>(predicateWithIndex: import("./FilterableWithIndex").PredicateWithIndex<number, A>): (fa: A[]) => Separated<A[], A[]>;
}, reduce: <A, B>(b: B, f: (b: B, a: A) => B) => (fa: A[]) => B, reduceRight: <A, B>(b: B, f: (a: A, b: B) => B) => (fa: A[]) => B, reduceRightWithIndex: <A, B>(b: B, f: (i: number, a: A, b: B) => B) => (fa: A[]) => B, reduceWithIndex: <A, B>(b: B, f: (i: number, b: B, a: A) => B) => (fa: A[]) => B, compact: <A>(fa: Option<A>[]) => A[], separate: <A, B>(fa: Either<A, B>[]) => Separated<A[], B[]>;
export { alt, ap, apFirst, apSecond, chain, chainFirst, duplicate, extend, filterMap, filterMapWithIndex, filterWithIndex, foldMap, foldMapWithIndex, map, mapWithIndex, partitionMapWithIndex, partitionWithIndex, reduce, reduceRight, reduceRightWithIndex, reduceWithIndex, compact, separate };
