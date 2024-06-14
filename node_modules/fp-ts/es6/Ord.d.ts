/**
 * @file The `Ord` type class represents types which support comparisons with a _total order_.
 *
 * Instances should satisfy the laws of total orderings:
 *
 * 1. Reflexivity: `S.compare(a, a) <= 0`
 * 2. Antisymmetry: if `S.compare(a, b) <= 0` and `S.compare(b, a) <= 0` then `a <-> b`
 * 3. Transitivity: if `S.compare(a, b) <= 0` and `S.compare(b, c) <= 0` then `S.compare(a, c) <= 0`
 *
 * See [Getting started with fp-ts: Ord](https://dev.to/gcanti/getting-started-with-fp-ts-ord-5f1e)
 */
import { Contravariant1 } from './Contravariant';
import { Eq } from './Eq';
import { Ordering } from './Ordering';
import { Semigroup } from './Semigroup';
declare module './HKT' {
    interface URItoKind<A> {
        Ord: Ord<A>;
    }
}
/**
 * @since 1.19.0
 */
export declare const URI = "Ord";
/**
 * @since 1.19.0
 */
export declare type URI = typeof URI;
/**
 * @since 1.0.0
 */
export interface Ord<A> extends Eq<A> {
    readonly compare: (x: A, y: A) => Ordering;
}
/**
 * @since 1.0.0
 * @deprecated
 */
export declare const unsafeCompare: (x: any, y: any) => Ordering;
/**
 * @since 1.0.0
 */
export declare const ordString: Ord<string>;
/**
 * @since 1.0.0
 */
export declare const ordNumber: Ord<number>;
/**
 * @since 1.0.0
 */
export declare const ordBoolean: Ord<boolean>;
/**
 * Test whether one value is _strictly less than_ another
 *
 * @since 1.19.0
 */
export declare const lt: <A>(O: Ord<A>) => (x: A, y: A) => boolean;
/**
 * Use `lt`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const lessThan: <A>(O: Ord<A>) => (x: A, y: A) => boolean;
/**
 * Test whether one value is _strictly greater than_ another
 *
 * @since 1.19.0
 */
export declare const gt: <A>(O: Ord<A>) => (x: A, y: A) => boolean;
/**
 * Use `gt`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const greaterThan: <A>(O: Ord<A>) => (x: A, y: A) => boolean;
/**
 * Test whether one value is _non-strictly less than_ another
 *
 * @since 1.19.0
 */
export declare const leq: <A>(O: Ord<A>) => (x: A, y: A) => boolean;
/**
 * Use `leq`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const lessThanOrEq: <A>(O: Ord<A>) => (x: A, y: A) => boolean;
/**
 * Test whether one value is _non-strictly greater than_ another
 *
 * @since 1.19.0
 */
export declare const geq: <A>(O: Ord<A>) => (x: A, y: A) => boolean;
/**
 * Use `geq`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const greaterThanOrEq: <A>(O: Ord<A>) => (x: A, y: A) => boolean;
/**
 * Take the minimum of two values. If they are considered equal, the first argument is chosen
 *
 * @since 1.0.0
 */
export declare const min: <A>(O: Ord<A>) => (x: A, y: A) => A;
/**
 * Take the maximum of two values. If they are considered equal, the first argument is chosen
 *
 * @since 1.0.0
 */
export declare const max: <A>(O: Ord<A>) => (x: A, y: A) => A;
/**
 * Clamp a value between a minimum and a maximum
 *
 * @since 1.0.0
 */
export declare const clamp: <A>(O: Ord<A>) => (low: A, hi: A) => (x: A) => A;
/**
 * Test whether a value is between a minimum and a maximum (inclusive)
 *
 * @since 1.0.0
 */
export declare const between: <A>(O: Ord<A>) => (low: A, hi: A) => (x: A) => boolean;
/**
 * @since 1.0.0
 */
export declare const fromCompare: <A>(compare: (x: A, y: A) => Ordering) => Ord<A>;
/**
 * @since 1.0.0
 */
export declare function contramap<A, B>(f: (b: B) => A): (O: Ord<A>) => Ord<B>;
/** @deprecated */
export declare function contramap<A, B>(f: (b: B) => A, O: Ord<A>): Ord<B>;
/**
 * @since 1.0.0
 */
export declare const getSemigroup: <A = never>() => Semigroup<Ord<A>>;
/**
 * Given a tuple of `Ord`s returns an `Ord` for the tuple
 *
 * @example
 * import { getTupleOrd, ordString, ordNumber, ordBoolean } from 'fp-ts/lib/Ord'
 *
 * const O = getTupleOrd(ordString, ordNumber, ordBoolean)
 * assert.strictEqual(O.compare(['a', 1, true], ['b', 2, true]), -1)
 * assert.strictEqual(O.compare(['a', 1, true], ['a', 2, true]), -1)
 * assert.strictEqual(O.compare(['a', 1, true], ['a', 1, false]), 1)
 *
 * @since 1.14.3
 */
export declare const getTupleOrd: <T extends Ord<any>[]>(...ords: T) => Ord<{ [K in keyof T]: T[K] extends Ord<infer A> ? A : never; }>;
/**
 * Use `getTupleOrd` instead
 * @since 1.0.0
 * @deprecated
 */
export declare const getProductOrd: <A, B>(OA: Ord<A>, OB: Ord<B>) => Ord<[A, B]>;
/**
 * @since 1.3.0
 */
export declare const getDualOrd: <A>(O: Ord<A>) => Ord<A>;
/**
 * @since 1.19.0
 */
export declare const ord: Contravariant1<URI>;
/**
 * @since 1.4.0
 */
export declare const ordDate: Ord<Date>;
