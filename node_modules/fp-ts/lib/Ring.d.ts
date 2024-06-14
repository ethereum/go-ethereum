/**
 * @file The `Ring` class is for types that support addition, multiplication, and subtraction operations.
 *
 * Instances must satisfy the following law in addition to the `Semiring` laws:
 *
 * - Additive inverse: `a - a = (zero - a) + a = zero`
 *
 * Adapted from https://github.com/purescript/purescript-prelude/blob/master/src/Data/Ring.purs
 */
import { Semiring } from './Semiring';
/**
 * @since 1.0.0
 */
export interface Ring<A> extends Semiring<A> {
    readonly sub: (x: A, y: A) => A;
}
/**
 * @since 1.0.0
 */
export declare const getFunctionRing: <A, B>(ring: Ring<B>) => Ring<(a: A) => B>;
/**
 * `negate x` can be used as a shorthand for `zero - x`
 *
 * @since 1.0.0
 */
export declare const negate: <A>(ring: Ring<A>) => (a: A) => A;
/**
 * Given a tuple of `Ring`s returns a `Ring` for the tuple
 *
 * @example
 * import { getTupleRing } from 'fp-ts/lib/Ring'
 * import { fieldNumber } from 'fp-ts/lib/Field'
 *
 * const R = getTupleRing(fieldNumber, fieldNumber, fieldNumber)
 * assert.deepStrictEqual(R.add([1, 2, 3], [4, 5, 6]), [5, 7, 9])
 * assert.deepStrictEqual(R.mul([1, 2, 3], [4, 5, 6]), [4, 10, 18])
 * assert.deepStrictEqual(R.one, [1, 1, 1])
 * assert.deepStrictEqual(R.sub([1, 2, 3], [4, 5, 6]), [-3, -3, -3])
 * assert.deepStrictEqual(R.zero, [0, 0, 0])
 *
 * @since 1.14.3
 */
export declare const getTupleRing: <T extends Ring<any>[]>(...rings: T) => Ring<{ [K in keyof T]: T[K] extends Ring<infer A> ? A : never; }>;
/**
 * Use `getTupleRing` instead
 * @since 1.0.0
 * @deprecated
 */
export declare const getProductRing: <A, B>(RA: Ring<A>, RB: Ring<B>) => Ring<[A, B]>;
