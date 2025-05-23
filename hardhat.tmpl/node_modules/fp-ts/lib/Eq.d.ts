/**
 * @file The `Eq` type class represents types which support decidable equality.
 *
 * Instances must satisfy the following laws:
 *
 * 1. Reflexivity: `E.equals(a, a) === true`
 * 2. Symmetry: `E.equals(a, b) === E.equals(b, a)`
 * 3. Transitivity: if `E.equals(a, b) === true` and `E.equals(b, c) === true`, then `E.equals(a, c) === true`
 *
 * See [Getting started with fp-ts: Eq](https://dev.to/gcanti/getting-started-with-fp-ts-setoid-39f3)
 */
import { Contravariant1 } from './Contravariant';
declare module './HKT' {
    interface URItoKind<A> {
        Eq: Eq<A>;
    }
}
/**
 * @since 1.19.0
 */
export declare const URI = "Eq";
/**
 * @since 1.19.0
 */
export declare type URI = typeof URI;
/**
 * @file The `Eq` type class represents types which support decidable equality.
 *
 * Instances must satisfy the following laws:
 *
 * 1. Reflexivity: `E.equals(a, a) === true`
 * 2. Symmetry: `E.equals(a, b) === E.equals(b, a)`
 * 3. Transitivity: if `E.equals(a, b) === true` and `E.equals(b, c) === true`, then `E.equals(a, c) === true`
 *
 * See [Getting started with fp-ts: Eq](https://dev.to/gcanti/getting-started-with-fp-ts-eq-39f3)
 */
/**
 * @since 1.19.0
 */
export interface Eq<A> {
    readonly equals: (x: A, y: A) => boolean;
}
/**
 * @since 1.19.0
 */
export declare function fromEquals<A>(equals: (x: A, y: A) => boolean): Eq<A>;
/**
 * @since 1.19.0
 */
export declare function strictEqual<A>(a: A, b: A): boolean;
/**
 * @since 1.19.0
 */
export declare const eqString: Eq<string>;
/**
 * @since 1.19.0
 */
export declare const eqNumber: Eq<number>;
/**
 * @since 1.19.0
 */
export declare const eqBoolean: Eq<boolean>;
/**
 * @since 1.19.0
 */
export declare function getStructEq<O extends {
    [key: string]: any;
}>(eqs: {
    [K in keyof O]: Eq<O[K]>;
}): Eq<O>;
/**
 * Given a tuple of `Eq`s returns a `Eq` for the tuple
 *
 * @example
 * import { getTupleEq, eqString, eqNumber, eqBoolean } from 'fp-ts/lib/Eq'
 *
 * const E = getTupleEq(eqString, eqNumber, eqBoolean)
 * assert.strictEqual(E.equals(['a', 1, true], ['a', 1, true]), true)
 * assert.strictEqual(E.equals(['a', 1, true], ['b', 1, true]), false)
 * assert.strictEqual(E.equals(['a', 1, true], ['a', 2, true]), false)
 * assert.strictEqual(E.equals(['a', 1, true], ['a', 1, false]), false)
 *
 * @since 1.19.0
 */
export declare function getTupleEq<T extends Array<Eq<any>>>(...eqs: T): Eq<{
    [K in keyof T]: T[K] extends Eq<infer A> ? A : never;
}>;
/**
 * @since 1.19.0
 */
export declare const eq: Contravariant1<URI>;
declare const contramap: <A, B>(f: (b: B) => A) => (fa: Eq<A>) => Eq<B>;
export { contramap };
/**
 * @since 1.19.0
 */
export declare const eqDate: Eq<Date>;
