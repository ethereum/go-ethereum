import { Category2 } from './Category';
import { Monad2 } from './Monad';
import { Profunctor2 } from './Profunctor';
import { Strong2 } from './Strong';
import { Choice2 } from './Choice';
import { Semigroup } from './Semigroup';
import { Monoid } from './Monoid';
declare module './HKT' {
    interface URItoKind2<L, A> {
        Reader: Reader<L, A>;
    }
}
export declare const URI = "Reader";
export declare type URI = typeof URI;
/**
 * @since 1.0.0
 */
export declare class Reader<E, A> {
    readonly run: (e: E) => A;
    readonly _A: A;
    readonly _L: E;
    readonly _URI: URI;
    constructor(run: (e: E) => A);
    /** @obsolete */
    map<B>(f: (a: A) => B): Reader<E, B>;
    /** @obsolete */
    ap<B>(fab: Reader<E, (a: A) => B>): Reader<E, B>;
    /**
     * Flipped version of `ap`
     * @obsolete
     */
    ap_<B, C>(this: Reader<E, (b: B) => C>, fb: Reader<E, B>): Reader<E, C>;
    /** @obsolete */
    chain<B>(f: (a: A) => Reader<E, B>): Reader<E, B>;
    /**
     * @since 1.6.1
     * @obsolete
     */
    local<E2 = E>(f: (e: E2) => E): Reader<E2, A>;
}
/**
 * reads the current context
 *
 * @since 1.0.0
 */
export declare const ask: <E>() => Reader<E, E>;
/**
 * Projects a value from the global context in a Reader
 *
 * @since 1.0.0
 */
export declare const asks: <E, A>(f: (e: E) => A) => Reader<E, A>;
/**
 * changes the value of the local context during the execution of the action `fa`
 *
 * @since 1.0.0
 */
export declare const local: <E, E2 = E>(f: (e: E2) => E) => <A>(fa: Reader<E, A>) => Reader<E2, A>;
/**
 * @since 1.14.0
 */
export declare const getSemigroup: <E, A>(S: Semigroup<A>) => Semigroup<Reader<E, A>>;
/**
 * @since 1.14.0
 */
export declare const getMonoid: <E, A>(M: Monoid<A>) => Monoid<Reader<E, A>>;
/**
 * @since 1.0.0
 */
export declare const reader: Monad2<URI> & Profunctor2<URI> & Category2<URI> & Strong2<URI> & Choice2<URI>;
/**
 * @since 1.19.0
 */
export declare const of: <R, A>(a: A) => Reader<R, A>;
declare const ap: <L, A>(fa: Reader<L, A>) => <B>(fab: Reader<L, (a: A) => B>) => Reader<L, B>, apFirst: <L, B>(fb: Reader<L, B>) => <A>(fa: Reader<L, A>) => Reader<L, A>, apSecond: <L, B>(fb: Reader<L, B>) => <A>(fa: Reader<L, A>) => Reader<L, B>, chain: <L, A, B>(f: (a: A) => Reader<L, B>) => (ma: Reader<L, A>) => Reader<L, B>, chainFirst: <L, A, B>(f: (a: A) => Reader<L, B>) => (ma: Reader<L, A>) => Reader<L, A>, compose: <L, A>(la: Reader<L, A>) => <B>(ab: Reader<A, B>) => Reader<L, B>, flatten: <L, A>(mma: Reader<L, Reader<L, A>>) => Reader<L, A>, map: <A, B>(f: (a: A) => B) => <L>(fa: Reader<L, A>) => Reader<L, B>, promap: <A, B, C, D>(f: (a: A) => B, g: (c: C) => D) => (fbc: Reader<B, C>) => Reader<A, D>;
export { ap, apFirst, apSecond, chain, chainFirst, compose, flatten, map, promap };
