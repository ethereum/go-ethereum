import { HKT, Kind, Kind2, Kind3, Kind4, URIS, URIS2, URIS3, URIS4 } from './HKT';
/**
 * @since 1.0.0
 */
export interface Functor<F> {
    readonly URI: F;
    readonly map: <A, B>(fa: HKT<F, A>, f: (a: A) => B) => HKT<F, B>;
}
export interface Functor1<F extends URIS> {
    readonly URI: F;
    readonly map: <A, B>(fa: Kind<F, A>, f: (a: A) => B) => Kind<F, B>;
}
export interface Functor2<F extends URIS2> {
    readonly URI: F;
    readonly map: <L, A, B>(fa: Kind2<F, L, A>, f: (a: A) => B) => Kind2<F, L, B>;
}
export interface Functor3<F extends URIS3> {
    readonly URI: F;
    readonly map: <U, L, A, B>(fa: Kind3<F, U, L, A>, f: (a: A) => B) => Kind3<F, U, L, B>;
}
export interface Functor4<F extends URIS4> {
    readonly URI: F;
    readonly map: <X, U, L, A, B>(fa: Kind4<F, X, U, L, A>, f: (a: A) => B) => Kind4<F, X, U, L, B>;
}
export interface Functor2C<F extends URIS2, L> {
    readonly URI: F;
    readonly _L: L;
    readonly map: <A, B>(fa: Kind2<F, L, A>, f: (a: A) => B) => Kind2<F, L, B>;
}
export interface Functor3C<F extends URIS3, U, L> {
    readonly URI: F;
    readonly _L: L;
    readonly _U: U;
    readonly map: <A, B>(fa: Kind3<F, U, L, A>, f: (a: A) => B) => Kind3<F, U, L, B>;
}
export interface Functor4<F extends URIS4> {
    readonly URI: F;
    readonly map: <X, U, L, A, B>(fa: Kind4<F, X, U, L, A>, f: (a: A) => B) => Kind4<F, X, U, L, B>;
}
export interface FunctorComposition<F, G> {
    readonly map: <A, B>(fa: HKT<F, HKT<G, A>>, f: (a: A) => B) => HKT<F, HKT<G, B>>;
}
export interface FunctorComposition11<F extends URIS, G extends URIS> {
    readonly map: <A, B>(fa: Kind<F, Kind<G, A>>, f: (a: A) => B) => Kind<F, Kind<G, B>>;
}
export interface FunctorComposition12<F extends URIS, G extends URIS2> {
    readonly map: <LG, A, B>(fa: Kind<F, Kind2<G, LG, A>>, f: (a: A) => B) => Kind<F, Kind2<G, LG, B>>;
}
export interface FunctorComposition12C<F extends URIS, G extends URIS2, LG> {
    readonly map: <A, B>(fa: Kind<F, Kind2<G, LG, A>>, f: (a: A) => B) => Kind<F, Kind2<G, LG, B>>;
}
export interface FunctorComposition21<F extends URIS2, G extends URIS> {
    readonly map: <LF, A, B>(fa: Kind2<F, LF, Kind<G, A>>, f: (a: A) => B) => Kind2<F, LF, Kind<G, B>>;
}
export interface FunctorComposition2C1<F extends URIS2, G extends URIS, LF> {
    readonly map: <A, B>(fa: Kind2<F, LF, Kind<G, A>>, f: (a: A) => B) => Kind2<F, LF, Kind<G, B>>;
}
export interface FunctorComposition22<F extends URIS2, G extends URIS2> {
    readonly map: <LF, LG, A, B>(fa: Kind2<F, LF, Kind2<G, LG, A>>, f: (a: A) => B) => Kind2<F, LF, Kind2<G, LG, B>>;
}
export interface FunctorComposition22C<F extends URIS2, G extends URIS2, LG> {
    readonly map: <LF, A, B>(fa: Kind2<F, LF, Kind2<G, LG, A>>, f: (a: A) => B) => Kind2<F, LF, Kind2<G, LG, B>>;
}
export interface FunctorComposition3C1<F extends URIS3, G extends URIS, UF, LF> {
    readonly map: <A, B>(fa: Kind3<F, UF, LF, Kind<G, A>>, f: (a: A) => B) => Kind3<F, UF, LF, Kind<G, B>>;
}
/**
 * Use `pipeable`'s `map`
 * @since 1.0.0
 * @deprecated
 */
export declare function lift<F extends URIS3>(F: Functor3<F>): <A, B>(f: (a: A) => B) => <U, L>(fa: Kind3<F, U, L, A>) => Kind3<F, U, L, B>;
/**
 * Use `pipeable`'s `map`
 * @deprecated
 */
export declare function lift<F extends URIS3, U, L>(F: Functor3C<F, U, L>): <A, B>(f: (a: A) => B) => (fa: Kind3<F, U, L, A>) => Kind3<F, U, L, B>;
/**
 * Use `pipeable`'s `map`
 * @deprecated
 */
export declare function lift<F extends URIS2>(F: Functor2<F>): <A, B>(f: (a: A) => B) => <L>(fa: Kind2<F, L, A>) => Kind2<F, L, B>;
/**
 * Use `pipeable`'s `map`
 * @deprecated
 */
export declare function lift<F extends URIS2, L>(F: Functor2C<F, L>): <A, B>(f: (a: A) => B) => (fa: Kind2<F, L, A>) => Kind2<F, L, B>;
/**
 * Use `pipeable`'s `map`
 * @deprecated
 */
export declare function lift<F extends URIS>(F: Functor1<F>): <A, B>(f: (a: A) => B) => (fa: Kind<F, A>) => Kind<F, B>;
/**
 * Use `pipeable`'s `map`
 * @deprecated
 */
export declare function lift<F>(F: Functor<F>): <A, B>(f: (a: A) => B) => (fa: HKT<F, A>) => HKT<F, B>;
/**
 * Ignore the return value of a computation, using the specified return value instead (`<$`)
 *
 * @since 1.0.0
 * @deprecated
 */
export declare function voidRight<F extends URIS3>(F: Functor3<F>): <U, L, A, B>(a: A, fb: Kind3<F, U, L, B>) => Kind3<F, U, L, A>;
/** @deprecated */
export declare function voidRight<F extends URIS3, U, L>(F: Functor3C<F, U, L>): <A, B>(a: A, fb: Kind3<F, U, L, B>) => Kind3<F, U, L, A>;
/** @deprecated */
export declare function voidRight<F extends URIS2>(F: Functor2<F>): <L, A, B>(a: A, fb: Kind2<F, L, B>) => Kind2<F, L, A>;
/** @deprecated */
export declare function voidRight<F extends URIS2, L>(F: Functor2C<F, L>): <A, B>(a: A, fb: Kind2<F, L, B>) => Kind2<F, L, A>;
/** @deprecated */
export declare function voidRight<F extends URIS>(F: Functor1<F>): <A, B>(a: A, fb: Kind<F, B>) => Kind<F, A>;
/** @deprecated */
export declare function voidRight<F>(F: Functor<F>): <A, B>(a: A, fb: HKT<F, B>) => HKT<F, A>;
/**
 * A version of `voidRight` with its arguments flipped (`$>`)
 *
 * @since 1.0.0
 * @deprecated
 */
export declare function voidLeft<F extends URIS3>(F: Functor3<F>): <U, L, A, B>(fa: Kind3<F, U, L, A>, b: B) => Kind3<F, U, L, B>;
/** @deprecated */
export declare function voidLeft<F extends URIS3, U, L>(F: Functor3C<F, U, L>): <A, B>(fa: Kind3<F, U, L, A>, b: B) => Kind3<F, U, L, B>;
/** @deprecated */
export declare function voidLeft<F extends URIS2>(F: Functor2<F>): <L, A, B>(fa: Kind2<F, L, A>, b: B) => Kind2<F, L, B>;
/** @deprecated */
export declare function voidLeft<F extends URIS2, L>(F: Functor2C<F, L>): <A, B>(fa: Kind2<F, L, A>, b: B) => Kind2<F, L, B>;
/** @deprecated */
export declare function voidLeft<F extends URIS>(F: Functor1<F>): <A, B>(fa: Kind<F, A>, b: B) => Kind<F, B>;
/** @deprecated */
export declare function voidLeft<F>(F: Functor<F>): <A, B>(fa: HKT<F, A>, b: B) => HKT<F, B>;
/**
 * Apply a value in a computational context to a value in no context. Generalizes `flip`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare function flap<F extends URIS3>(functor: Functor3<F>): <U, L, A, B>(a: A, ff: Kind3<F, U, L, (a: A) => B>) => Kind3<F, U, L, B>;
/** @deprecated */
export declare function flap<F extends URIS3, U, L>(functor: Functor3C<F, U, L>): <A, B>(a: A, ff: Kind3<F, U, L, (a: A) => B>) => Kind3<F, U, L, B>;
/** @deprecated */
export declare function flap<F extends URIS2>(functor: Functor2<F>): <L, A, B>(a: A, ff: Kind2<F, L, (a: A) => B>) => Kind2<F, L, B>;
/** @deprecated */
export declare function flap<F extends URIS2, L>(functor: Functor2C<F, L>): <A, B>(a: A, ff: Kind2<F, L, (a: A) => B>) => Kind2<F, L, B>;
/** @deprecated */
export declare function flap<F extends URIS>(functor: Functor1<F>): <A, B>(a: A, ff: Kind<F, (a: A) => B>) => Kind<F, B>;
/** @deprecated */
export declare function flap<F>(functor: Functor<F>): <A, B>(a: A, ff: HKT<F, (a: A) => B>) => HKT<F, B>;
/**
 * @since 1.0.0
 */
export declare function getFunctorComposition<F extends URIS3, G extends URIS, UF, LF>(F: Functor3C<F, UF, LF>, G: Functor1<G>): FunctorComposition3C1<F, G, UF, LF>;
export declare function getFunctorComposition<F extends URIS2, G extends URIS2, LG>(F: Functor2<F>, G: Functor2C<G, LG>): FunctorComposition22C<F, G, LG>;
export declare function getFunctorComposition<F extends URIS2, G extends URIS2>(F: Functor2<F>, G: Functor2<G>): FunctorComposition22<F, G>;
export declare function getFunctorComposition<F extends URIS2, G extends URIS, LF>(F: Functor2C<F, LF>, G: Functor1<G>): FunctorComposition2C1<F, G, LF>;
export declare function getFunctorComposition<F extends URIS2, G extends URIS>(F: Functor2<F>, G: Functor1<G>): FunctorComposition21<F, G>;
export declare function getFunctorComposition<F extends URIS, G extends URIS2, LG>(F: Functor1<F>, G: Functor2C<G, LG>): FunctorComposition12C<F, G, LG>;
export declare function getFunctorComposition<F extends URIS, G extends URIS2>(F: Functor1<F>, G: Functor2<G>): FunctorComposition12<F, G>;
export declare function getFunctorComposition<F extends URIS, G extends URIS>(F: Functor1<F>, G: Functor1<G>): FunctorComposition11<F, G>;
export declare function getFunctorComposition<F, G>(F: Functor<F>, G: Functor<G>): FunctorComposition<F, G>;
