import { Comonad2C } from './Comonad';
import { Monoid } from './Monoid';
import { Functor2 } from './Functor';
declare module './HKT' {
    interface URItoKind2<L, A> {
        Traced: Traced<L, A>;
    }
}
export declare const URI = "Traced";
export declare type URI = typeof URI;
/**
 * @since 1.16.0
 */
export declare class Traced<P, A> {
    readonly run: (p: P) => A;
    readonly _A: A;
    readonly _L: P;
    readonly _URI: URI;
    constructor(run: (p: P) => A);
    /** @obsolete */
    map<B>(f: (a: A) => B): Traced<P, B>;
}
/**
 * Extracts a value at a relative position which depends on the current value.
 * @since 1.16.0
 */
export declare const tracks: <P, A>(M: Monoid<P>, f: (a: A) => P) => (wa: Traced<P, A>) => A;
/**
 * Get the current position
 * @since 1.16.0
 */
export declare const listen: <P, A>(wa: Traced<P, A>) => Traced<P, [A, P]>;
/**
 * Get a value which depends on the current position
 * @since 1.16.0
 */
export declare const listens: <P, A, B>(wa: Traced<P, A>, f: (p: P) => B) => Traced<P, [A, B]>;
/**
 * Apply a function to the current position
 * @since 1.16.0
 */
export declare const censor: <P, A>(wa: Traced<P, A>, f: (p: P) => P) => Traced<P, A>;
/**
 * @since 1.16.0
 */
export declare function getComonad<P>(monoid: Monoid<P>): Comonad2C<URI, P>;
/**
 * @since 1.16.0
 */
export declare const traced: Functor2<URI>;
