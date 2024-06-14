/**
 * @file Adapted from https://github.com/parsonsmatt/purescript-pair
 */
import { Applicative1 } from './Applicative';
import { Comonad1 } from './Comonad';
import { Foldable2v1 } from './Foldable2v';
import { Endomorphism } from './function';
import { Monoid } from './Monoid';
import { Ord } from './Ord';
import { Semigroup } from './Semigroup';
import { Eq } from './Eq';
import { Traversable2v1 } from './Traversable2v';
import { Show } from './Show';
declare module './HKT' {
    interface URItoKind<A> {
        Pair: Pair<A>;
    }
}
export declare const URI = "Pair";
export declare type URI = typeof URI;
/**
 * @data
 * @constructor Pair
 * @since 1.0.0
 */
export declare class Pair<A> {
    readonly fst: A;
    readonly snd: A;
    readonly _A: A;
    readonly _URI: URI;
    constructor(fst: A, snd: A);
    /** Map a function over the first field of a pair */
    first(f: Endomorphism<A>): Pair<A>;
    /** Map a function over the second field of a pair */
    second(f: Endomorphism<A>): Pair<A>;
    /** Swaps the elements in a pair */
    swap(): Pair<A>;
    map<B>(f: (a: A) => B): Pair<B>;
    ap<B>(fab: Pair<(a: A) => B>): Pair<B>;
    /**
     * Flipped version of `ap`
     */
    ap_<B, C>(this: Pair<(b: B) => C>, fb: Pair<B>): Pair<C>;
    reduce<B>(b: B, f: (b: B, a: A) => B): B;
    extract(): A;
    extend<B>(f: (fb: Pair<A>) => B): Pair<B>;
}
/**
 * @since 1.17.0
 */
export declare const getShow: <L, A>(S: Show<A>) => Show<Pair<A>>;
/**
 * Use `getEq`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const getSetoid: <A>(S: Eq<A>) => Eq<Pair<A>>;
/**
 * @since 1.19.0
 */
export declare function getEq<A>(S: Eq<A>): Eq<Pair<A>>;
/**
 * @since 1.0.0
 */
export declare const getOrd: <A>(O: Ord<A>) => Ord<Pair<A>>;
/**
 * @since 1.0.0
 */
export declare const getSemigroup: <A>(S: Semigroup<A>) => Semigroup<Pair<A>>;
/**
 * @since 1.0.0
 */
export declare const getMonoid: <A>(M: Monoid<A>) => Monoid<Pair<A>>;
/**
 * @since 1.0.0
 */
export declare const pair: Applicative1<URI> & Foldable2v1<URI> & Traversable2v1<URI> & Comonad1<URI>;
