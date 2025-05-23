/**
 * @file Adapted from http://okmij.org/ftp/Computation/free-monad.html and https://github.com/purescript/purescript-free
 */
import { HKT, Kind, Kind2, Kind3, URIS, URIS2, URIS3 } from './HKT';
import { Monad, Monad1, Monad2, Monad2C, Monad3, Monad3C } from './Monad';
export declare const URI = "Free";
export declare type URI = typeof URI;
declare module './HKT' {
    interface URItoKind2<L, A> {
        Free: Free<L, A>;
    }
}
/**
 * @data
 * @constructor Pure
 * @constructor Impure
 * @since 1.0.0
 */
export declare type Free<F, A> = Pure<F, A> | Impure<F, A, any>;
export declare class Pure<F, A> {
    readonly value: A;
    readonly _tag: 'Pure';
    readonly _A: A;
    readonly _L: F;
    readonly _URI: URI;
    constructor(value: A);
    map<B>(f: (a: A) => B): Free<F, B>;
    ap<B>(fab: Free<F, (a: A) => B>): Free<F, B>;
    /**
     * Flipped version of `ap`
     */
    ap_<B, C>(this: Free<F, (b: B) => C>, fb: Free<F, B>): Free<F, C>;
    chain<B>(f: (a: A) => Free<F, B>): Free<F, B>;
    inspect(): string;
    toString(): string;
    isPure(): this is Pure<F, A>;
    isImpure(): this is Impure<F, A, any>;
}
export declare class Impure<F, A, X> {
    readonly fx: HKT<F, X>;
    readonly f: (x: X) => Free<F, A>;
    readonly _tag: 'Impure';
    readonly _A: A;
    readonly _L: F;
    readonly _URI: URI;
    constructor(fx: HKT<F, X>, f: (x: X) => Free<F, A>);
    map<B>(f: (a: A) => B): Free<F, B>;
    ap<B>(fab: Free<F, (a: A) => B>): Free<F, B>;
    ap_<B, C>(this: Free<F, (b: B) => C>, fb: Free<F, B>): Free<F, C>;
    chain<B>(f: (a: A) => Free<F, B>): Free<F, B>;
    inspect(): string;
    toString(): string;
    isPure(): this is Pure<F, A>;
    isImpure(): this is Impure<F, A, X>;
}
/**
 * @since 1.0.0
 */
export declare const of: <F, A>(a: A) => Free<F, A>;
/**
 * Lift an impure value described by the generating type constructor `F` into the free monad
 *
 * @since 1.0.0
 */
export declare const liftF: <F, A>(fa: HKT<F, A>) => Free<F, A>;
/**
 * Use a natural transformation to change the generating type constructor of a free monad
 *
 * @since 1.0.0
 */
export declare function hoistFree<F extends URIS3 = never, G extends URIS3 = never>(nt: <U, L, A>(fa: Kind3<F, U, L, A>) => Kind3<G, U, L, A>): <A>(fa: Free<F, A>) => Free<G, A>;
export declare function hoistFree<F extends URIS2 = never, G extends URIS2 = never>(nt: <L, A>(fa: Kind2<F, L, A>) => Kind2<G, L, A>): <A>(fa: Free<F, A>) => Free<G, A>;
export declare function hoistFree<F extends URIS = never, G extends URIS = never>(nt: <A>(fa: Kind<F, A>) => Kind<G, A>): <A>(fa: Free<F, A>) => Free<G, A>;
export declare function hoistFree<F, G>(nt: <A>(fa: HKT<F, A>) => HKT<G, A>): <A>(fa: Free<F, A>) => Free<G, A>;
export interface FoldFree3<M extends URIS3> {
    <F extends URIS3, U, L, A>(nt: <X>(fa: Kind3<F, U, L, X>) => Kind3<M, U, L, X>, fa: Free<F, A>): Kind3<M, U, L, A>;
    <F extends URIS2, U, L, A>(nt: <X>(fa: Kind2<F, L, X>) => Kind3<M, U, L, X>, fa: Free<F, A>): Kind3<M, U, L, A>;
    <F extends URIS, U, L, A>(nt: <X>(fa: Kind<F, X>) => Kind3<M, U, L, X>, fa: Free<F, A>): Kind3<M, U, L, A>;
}
export interface FoldFree3C<M extends URIS3, U, L> {
    <F extends URIS3, A>(nt: <X>(fa: Kind3<F, U, L, X>) => Kind3<M, U, L, X>, fa: Free<F, A>): Kind3<M, U, L, A>;
    <F extends URIS2, A>(nt: <X>(fa: Kind2<F, L, X>) => Kind3<M, U, L, X>, fa: Free<F, A>): Kind3<M, U, L, A>;
    <F extends URIS, A>(nt: <X>(fa: Kind<F, X>) => Kind3<M, U, L, X>, fa: Free<F, A>): Kind3<M, U, L, A>;
}
export interface FoldFree2<M extends URIS2> {
    <F extends URIS2, L, A>(nt: <X>(fa: Kind2<F, L, X>) => Kind2<M, L, X>, fa: Free<F, A>): Kind2<M, L, A>;
    <F extends URIS, L, A>(nt: <X>(fa: Kind<F, X>) => Kind2<M, L, X>, fa: Free<F, A>): Kind2<M, L, A>;
}
export interface FoldFree2C<M extends URIS2, L> {
    <F extends URIS2, A>(nt: <X>(fa: Kind2<F, L, X>) => Kind2<M, L, X>, fa: Free<F, A>): Kind2<M, L, A>;
    <F extends URIS, A>(nt: <X>(fa: Kind<F, X>) => Kind2<M, L, X>, fa: Free<F, A>): Kind2<M, L, A>;
}
/**
 * @since 1.0.0
 */
export declare function foldFree<M extends URIS3>(M: Monad3<M>): FoldFree3<M>;
export declare function foldFree<M extends URIS3, U, L>(M: Monad3C<M, U, L>): FoldFree3C<M, U, L>;
export declare function foldFree<M extends URIS2>(M: Monad2<M>): FoldFree2<M>;
export declare function foldFree<M extends URIS2, L>(M: Monad2C<M, L>): FoldFree2C<M, L>;
export declare function foldFree<M extends URIS>(M: Monad1<M>): <F extends URIS, A>(nt: <X>(fa: Kind<F, X>) => Kind<M, X>, fa: Free<F, A>) => Kind<M, A>;
export declare function foldFree<M>(M: Monad<M>): <F, A>(nt: <X>(fa: HKT<F, X>) => HKT<M, X>, fa: Free<F, A>) => HKT<M, A>;
