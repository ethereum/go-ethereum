import { Comonad1 } from './Comonad';
import { FoldableWithIndex1 } from './FoldableWithIndex';
import { Predicate, Refinement } from './function';
import { FunctorWithIndex1 } from './FunctorWithIndex';
import { Monad1 } from './Monad';
import { Option } from './Option';
import { Ord } from './Ord';
import { Semigroup } from './Semigroup';
import { Eq } from './Eq';
import { TraversableWithIndex1 } from './TraversableWithIndex';
declare module './HKT' {
    interface URItoKind<A> {
        NonEmptyArray: NonEmptyArray<A>;
    }
}
export declare const URI = "NonEmptyArray";
export declare type URI = typeof URI;
/**
 * @since 1.0.0
 */
export declare class NonEmptyArray<A> {
    readonly head: A;
    readonly tail: Array<A>;
    readonly _A: A;
    readonly _URI: URI;
    constructor(head: A, tail: Array<A>);
    /**
     * Converts this `NonEmptyArray` to a plain `Array`
     *
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     *
     * assert.deepStrictEqual(new NonEmptyArray(1, [2, 3]).toArray(), [1, 2, 3])
     */
    toArray(): Array<A>;
    /**
     * Converts this `NonEmptyArray` to a plain `Array` using the given map function
     *
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     *
     * assert.deepStrictEqual(new NonEmptyArray('a', ['bb', 'ccc']).toArrayMap(s => s.length), [1, 2, 3])
     *
     * @since 1.14.0
     */
    toArrayMap<B>(f: (a: A) => B): Array<B>;
    /**
     * Concatenates this `NonEmptyArray` and passed `Array`
     *
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     *
     * assert.deepStrictEqual(new NonEmptyArray<number>(1, []).concatArray([2]), new NonEmptyArray(1, [2]))
     */
    concatArray(as: Array<A>): NonEmptyArray<A>;
    /**
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     *
     * const double = (n: number): number => n * 2
     * assert.deepStrictEqual(new NonEmptyArray(1, [2]).map(double), new NonEmptyArray(2, [4]))
     */
    map<B>(f: (a: A) => B): NonEmptyArray<B>;
    mapWithIndex<B>(f: (i: number, a: A) => B): NonEmptyArray<B>;
    /**
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     *
     * const x = new NonEmptyArray(1, [2])
     * const double = (n: number): number => n * 2
     * assert.deepStrictEqual(x.ap(new NonEmptyArray(double, [double])).toArray(), [2, 4, 2, 4])
     */
    ap<B>(fab: NonEmptyArray<(a: A) => B>): NonEmptyArray<B>;
    /**
     * Flipped version of `ap`
     *
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     *
     * const x = new NonEmptyArray(1, [2])
     * const double = (n: number) => n * 2
     * assert.deepStrictEqual(new NonEmptyArray(double, [double]).ap_(x).toArray(), [2, 4, 2, 4])
     */
    ap_<B, C>(this: NonEmptyArray<(b: B) => C>, fb: NonEmptyArray<B>): NonEmptyArray<C>;
    /**
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     *
     * const x = new NonEmptyArray(1, [2])
     * const f = (a: number) => new NonEmptyArray(a, [4])
     * assert.deepStrictEqual(x.chain(f).toArray(), [1, 4, 2, 4])
     */
    chain<B>(f: (a: A) => NonEmptyArray<B>): NonEmptyArray<B>;
    /**
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     *
     * const x = new NonEmptyArray(1, [2])
     * const y = new NonEmptyArray(3, [4])
     * assert.deepStrictEqual(x.concat(y).toArray(), [1, 2, 3, 4])
     */
    concat(y: NonEmptyArray<A>): NonEmptyArray<A>;
    /**
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     *
     * const x = new NonEmptyArray('a', ['b'])
     * assert.strictEqual(x.reduce('', (b, a) => b + a), 'ab')
     */
    reduce<B>(b: B, f: (b: B, a: A) => B): B;
    /**
     * @since 1.12.0
     */
    reduceWithIndex<B>(b: B, f: (i: number, b: B, a: A) => B): B;
    /**
     * @since 1.12.0
     */
    foldr<B>(b: B, f: (a: A, b: B) => B): B;
    /**
     * @since 1.12.0
     */
    foldrWithIndex<B>(b: B, f: (i: number, a: A, b: B) => B): B;
    /**
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     * import { fold, monoidSum } from 'fp-ts/lib/Monoid'
     *
     * const sum = (as: NonEmptyArray<number>) => fold(monoidSum)(as.toArray())
     * assert.deepStrictEqual(new NonEmptyArray(1, [2, 3, 4]).extend(sum), new NonEmptyArray(10, [9, 7, 4]))
     */
    extend<B>(f: (fa: NonEmptyArray<A>) => B): NonEmptyArray<B>;
    /**
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     *
     * assert.strictEqual(new NonEmptyArray(1, [2, 3]).extract(), 1)
     */
    extract(): A;
    /**
     * Same as `toString`
     */
    inspect(): string;
    /**
     * Return stringified representation of this `NonEmptyArray`
     */
    toString(): string;
    /**
     * Gets minimum of this `NonEmptyArray` using specified `Ord` instance
     *
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     * import { ordNumber } from 'fp-ts/lib/Ord'
     *
     * assert.strictEqual(new NonEmptyArray(1, [2, 3]).min(ordNumber), 1)
     *
     * @since 1.3.0
     */
    min(ord: Ord<A>): A;
    /**
     * Gets maximum of this `NonEmptyArray` using specified `Ord` instance
     *
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     * import { ordNumber } from 'fp-ts/lib/Ord'
     *
     * assert.strictEqual(new NonEmptyArray(1, [2, 3]).max(ordNumber), 3)
     *
     * @since 1.3.0
     */
    max(ord: Ord<A>): A;
    /**
     * Gets last element of this `NonEmptyArray`
     *
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     *
     * assert.strictEqual(new NonEmptyArray(1, [2, 3]).last(), 3)
     * assert.strictEqual(new NonEmptyArray(1, []).last(), 1)
     *
     * @since 1.6.0
     */
    last(): A;
    /**
     * Sorts this `NonEmptyArray` using specified `Ord` instance
     *
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     * import { ordNumber } from 'fp-ts/lib/Ord'
     *
     * assert.deepStrictEqual(new NonEmptyArray(3, [2, 1]).sort(ordNumber), new NonEmptyArray(1, [2, 3]))
     *
     * @since 1.6.0
     */
    sort(ord: Ord<A>): NonEmptyArray<A>;
    /**
     * Reverts this `NonEmptyArray`
     *
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     *
     * assert.deepStrictEqual(new NonEmptyArray(1, [2, 3]).reverse(), new NonEmptyArray(3, [2, 1]))
     *
     * @since 1.6.0
     */
    reverse(): NonEmptyArray<A>;
    /**
     * @since 1.10.0
     */
    length(): number;
    /**
     * This function provides a safe way to read a value at a particular index from an NonEmptyArray
     *
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     * import { some, none } from 'fp-ts/lib/Option'
     *
     * assert.deepStrictEqual(new NonEmptyArray(1, [2, 3]).lookup(1), some(2))
     * assert.deepStrictEqual(new NonEmptyArray(1, [2, 3]).lookup(3), none)
     *
     * @since 1.14.0
     */
    lookup(i: number): Option<A>;
    /**
     * Use `lookup` instead
     * @since 1.11.0
     * @deprecated
     */
    index(i: number): Option<A>;
    /**
     * Find the first element which satisfies a predicate (or a refinement) function
     *
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     * import { some } from 'fp-ts/lib/Option'
     *
     * assert.deepStrictEqual(new NonEmptyArray({ a: 1, b: 1 }, [{ a: 1, b: 2 }]).findFirst(x => x.a === 1), some({ a: 1, b: 1 }))
     *
     * @since 1.11.0
     */
    findFirst<B extends A>(refinement: Refinement<A, B>): Option<B>;
    findFirst(predicate: Predicate<A>): Option<A>;
    /**
     * Find the last element which satisfies a predicate function
     *
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     * import { some } from 'fp-ts/lib/Option'
     *
     * assert.deepStrictEqual(new NonEmptyArray({ a: 1, b: 1 }, [{ a: 1, b: 2 }]).findLast(x => x.a === 1), some({ a: 1, b: 2 }))
     *
     * @since 1.11.0
     */
    findLast<B extends A>(predicate: Refinement<A, B>): Option<B>;
    findLast(predicate: Predicate<A>): Option<A>;
    /**
     * Find the first index for which a predicate holds
     *
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     * import { some, none } from 'fp-ts/lib/Option'
     *
     * assert.deepStrictEqual(new NonEmptyArray(1, [2, 3]).findIndex(x => x === 2), some(1))
     * assert.deepStrictEqual(new NonEmptyArray<number>(1, []).findIndex(x => x === 2), none)
     *
     * @since 1.11.0
     */
    findIndex(predicate: Predicate<A>): Option<number>;
    /**
     * Returns the index of the last element of the list which matches the predicate
     *
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     * import { some, none } from 'fp-ts/lib/Option'
     *
     * interface X {
     *   a: number
     *   b: number
     * }
     * const xs: NonEmptyArray<X> = new NonEmptyArray({ a: 1, b: 0 }, [{ a: 1, b: 1 }])
     * assert.deepStrictEqual(xs.findLastIndex(x => x.a === 1), some(1))
     * assert.deepStrictEqual(xs.findLastIndex(x => x.a === 4), none)
     *
     * @since 1.11.0
     */
    findLastIndex(predicate: Predicate<A>): Option<number>;
    /**
     * Insert an element at the specified index, creating a new NonEmptyArray, or returning `None` if the index is out of bounds
     *
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     * import { some } from 'fp-ts/lib/Option'
     *
     * assert.deepStrictEqual(new NonEmptyArray(1, [2, 3, 4]).insertAt(2, 5), some(new NonEmptyArray(1, [2, 5, 3, 4])))
     *
     * @since 1.11.0
     */
    insertAt(i: number, a: A): Option<NonEmptyArray<A>>;
    /**
     * Change the element at the specified index, creating a new NonEmptyArray, or returning `None` if the index is out of bounds
     *
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     * import { some, none } from 'fp-ts/lib/Option'
     *
     * assert.deepStrictEqual(new NonEmptyArray(1, [2, 3]).updateAt(1, 1), some(new NonEmptyArray(1, [1, 3])))
     * assert.deepStrictEqual(new NonEmptyArray(1, []).updateAt(1, 1), none)
     *
     * @since 1.11.0
     */
    updateAt(i: number, a: A): Option<NonEmptyArray<A>>;
    /**
     * Filter an NonEmptyArray, keeping the elements which satisfy a predicate function, creating a new NonEmptyArray or returning `None` in case the resulting NonEmptyArray would have no remaining elements.
     *
     * @since 1.11.0
     */
    filter<B extends A>(predicate: Refinement<A, B>): Option<NonEmptyArray<B>>;
    filter(predicate: Predicate<A>): Option<NonEmptyArray<A>>;
    /**
     * @since 1.12.0
     */
    filterWithIndex(predicate: (i: number, a: A) => boolean): Option<NonEmptyArray<A>>;
    /**
     * @since 1.14.0
     */
    some(predicate: Predicate<A>): boolean;
    /**
     * @since 1.14.0
     */
    every(predicate: Predicate<A>): boolean;
}
/**
 * Builds a `NonEmptyArray` from an `Array` returning `none` if `as` is an empty array
 *
 * @since 1.0.0
 */
export declare const fromArray: <A>(as: A[]) => Option<NonEmptyArray<A>>;
/**
 * Builds a `Semigroup` instance for `NonEmptyArray`
 *
 * @since 1.0.0
 */
export declare const getSemigroup: <A = never>() => Semigroup<NonEmptyArray<A>>;
/**
 * Use `getEq`
 *
 * @since 1.14.0
 * @deprecated
 */
export declare const getSetoid: <A>(S: Eq<A>) => Eq<NonEmptyArray<A>>;
/**
 * @example
 * import { NonEmptyArray, getEq } from 'fp-ts/lib/NonEmptyArray'
 * import { eqNumber } from 'fp-ts/lib/Eq'
 *
 * const E = getEq(eqNumber)
 * assert.strictEqual(E.equals(new NonEmptyArray(1, []), new NonEmptyArray(1, [])), true)
 * assert.strictEqual(E.equals(new NonEmptyArray(1, []), new NonEmptyArray(1, [2])), false)
 *
 * @since 1.19.0
 */
export declare function getEq<A>(S: Eq<A>): Eq<NonEmptyArray<A>>;
/**
 * Group equal, consecutive elements of an array into non empty arrays.
 *
 * @example
 * import { NonEmptyArray, group } from 'fp-ts/lib/NonEmptyArray'
 * import { ordNumber } from 'fp-ts/lib/Ord'
 *
 * assert.deepStrictEqual(group(ordNumber)([1, 2, 1, 1]), [
 *   new NonEmptyArray(1, []),
 *   new NonEmptyArray(2, []),
 *   new NonEmptyArray(1, [1])
 * ])
 *
 * @since 1.7.0
 */
export declare const group: <A>(S: Eq<A>) => (as: A[]) => NonEmptyArray<A>[];
/**
 * Sort and then group the elements of an array into non empty arrays.
 *
 * @example
 * import { NonEmptyArray, groupSort } from 'fp-ts/lib/NonEmptyArray'
 * import { ordNumber } from 'fp-ts/lib/Ord'
 *
 * assert.deepStrictEqual(groupSort(ordNumber)([1, 2, 1, 1]), [new NonEmptyArray(1, [1, 1]), new NonEmptyArray(2, [])])
 *
 * @since 1.7.0
 */
export declare const groupSort: <A>(O: Ord<A>) => (as: A[]) => NonEmptyArray<A>[];
/**
 * Splits an array into sub-non-empty-arrays stored in an object, based on the result of calling a `string`-returning
 * function on each element, and grouping the results according to values returned
 *
 * @example
 * import { NonEmptyArray, groupBy } from 'fp-ts/lib/NonEmptyArray'
 *
 * assert.deepStrictEqual(groupBy(['foo', 'bar', 'foobar'], a => String(a.length)), {
 *   '3': new NonEmptyArray('foo', ['bar']),
 *   '6': new NonEmptyArray('foobar', [])
 * })
 *
 * @since 1.10.0
 */
export declare const groupBy: <A>(as: A[], f: (a: A) => string) => {
    [key: string]: NonEmptyArray<A>;
};
/**
 * @since 1.0.0
 */
export declare const nonEmptyArray: Monad1<URI> & Comonad1<URI> & TraversableWithIndex1<URI, number> & FunctorWithIndex1<URI, number> & FoldableWithIndex1<URI, number>;
