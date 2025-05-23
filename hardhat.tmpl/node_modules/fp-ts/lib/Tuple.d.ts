/**
 * @file Adapted from https://github.com/purescript/purescript-tuples
 */
import { Applicative2C } from './Applicative';
import { Apply2C } from './Apply';
import { Bifunctor2 } from './Bifunctor';
import { Chain2C } from './Chain';
import { ChainRec2C } from './ChainRec';
import { Comonad2 } from './Comonad';
import { Foldable2v2 } from './Foldable2v';
import { Monad2C } from './Monad';
import { Monoid } from './Monoid';
import { Ord } from './Ord';
import { Semigroup } from './Semigroup';
import { Semigroupoid2 } from './Semigroupoid';
import { Eq } from './Eq';
import { Show } from './Show';
import { Traversable2v2 } from './Traversable2v';
declare module './HKT' {
    interface URItoKind2<L, A> {
        Tuple: Tuple<L, A>;
    }
}
export declare const URI = "Tuple";
export declare type URI = typeof URI;
/**
 * @since 1.0.0
 */
export declare class Tuple<L, A> {
    readonly fst: L;
    readonly snd: A;
    readonly _A: A;
    readonly _L: L;
    readonly _URI: URI;
    constructor(fst: L, snd: A);
    /** @obsolete */
    compose<B>(ab: Tuple<A, B>): Tuple<L, B>;
    /** @obsolete */
    map<B>(f: (a: A) => B): Tuple<L, B>;
    /** @obsolete */
    bimap<M, B>(f: (l: L) => M, g: (a: A) => B): Tuple<M, B>;
    /** @obsolete */
    extract(): A;
    /** @obsolete */
    extend<B>(f: (fa: Tuple<L, A>) => B): Tuple<L, B>;
    /** @obsolete */
    reduce<B>(b: B, f: (b: B, a: A) => B): B;
    /**
     * Exchange the first and second components of a tuple
     * @obsolete
     */
    swap(): Tuple<A, L>;
    inspect(): string;
    toString(): string;
    /** @obsolete */
    toTuple(): [L, A];
}
/**
 * @since 1.17.0
 */
export declare const getShow: <L, A>(SL: Show<L>, SA: Show<A>) => Show<Tuple<L, A>>;
/**
 * Use `getEq`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const getSetoid: <L, A>(EL: Eq<L>, EA: Eq<A>) => Eq<Tuple<L, A>>;
/**
 * @since 1.19.0
 */
export declare function getEq<L, A>(EL: Eq<L>, EA: Eq<A>): Eq<Tuple<L, A>>;
/**
 * To obtain the result, the `fst`s are `compare`d, and if they are `EQ`ual, the
 * `snd`s are `compare`d.
 *
 * @since 1.0.0
 */
export declare const getOrd: <L, A>(OL: Ord<L>, OA: Ord<A>) => Ord<Tuple<L, A>>;
/**
 * @since 1.0.0
 */
export declare const getSemigroup: <L, A>(SL: Semigroup<L>, SA: Semigroup<A>) => Semigroup<Tuple<L, A>>;
/**
 * @since 1.0.0
 */
export declare const getMonoid: <L, A>(ML: Monoid<L>, MA: Monoid<A>) => Monoid<Tuple<L, A>>;
/**
 * @since 1.0.0
 */
export declare const getApply: <L>(S: Semigroup<L>) => Apply2C<"Tuple", L>;
/**
 * @since 1.0.0
 */
export declare const getApplicative: <L>(M: Monoid<L>) => Applicative2C<"Tuple", L>;
/**
 * @since 1.0.0
 */
export declare const getChain: <L>(S: Semigroup<L>) => Chain2C<"Tuple", L>;
/**
 * @since 1.0.0
 */
export declare const getMonad: <L>(M: Monoid<L>) => Monad2C<"Tuple", L>;
/**
 * @since 1.0.0
 */
export declare const getChainRec: <L>(M: Monoid<L>) => ChainRec2C<"Tuple", L>;
/**
 * @since 1.0.0
 */
export declare const tuple: Semigroupoid2<URI> & Bifunctor2<URI> & Comonad2<URI> & Foldable2v2<URI> & Traversable2v2<URI>;
/**
 * @since 1.19.0
 */
export declare function swap<L, A>(sa: Tuple<L, A>): Tuple<A, L>;
/**
 * @since 1.19.0
 */
export declare function fst<L, A>(fa: Tuple<L, A>): L;
/**
 * @since 1.19.0
 */
export declare function snd<L, A>(fa: Tuple<L, A>): A;
declare const bimap: <L, A, M, B>(f: (l: L) => M, g: (a: A) => B) => (fa: Tuple<L, A>) => Tuple<M, B>, compose: <L, A>(la: Tuple<L, A>) => <B>(ab: Tuple<A, B>) => Tuple<L, B>, duplicate: <L, A>(ma: Tuple<L, A>) => Tuple<L, Tuple<L, A>>, extend: <L, A, B>(f: (fa: Tuple<L, A>) => B) => (ma: Tuple<L, A>) => Tuple<L, B>, foldMap: <M>(M: Monoid<M>) => <A>(f: (a: A) => M) => <L>(fa: Tuple<L, A>) => M, map: <A, B>(f: (a: A) => B) => <L>(fa: Tuple<L, A>) => Tuple<L, B>, mapLeft: <L, A, M>(f: (l: L) => M) => (fa: Tuple<L, A>) => Tuple<M, A>, reduce: <A, B>(b: B, f: (b: B, a: A) => B) => <L>(fa: Tuple<L, A>) => B, reduceRight: <A, B>(b: B, f: (a: A, b: B) => B) => <L>(fa: Tuple<L, A>) => B;
export { bimap, compose, duplicate, extend, foldMap, map, mapLeft, reduce, reduceRight };
