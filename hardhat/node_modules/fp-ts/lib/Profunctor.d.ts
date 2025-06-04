import { Functor2, Functor2C, Functor3, Functor4 } from './Functor';
import { HKT2, Kind2, Kind3, Kind4, URIS2, URIS3, URIS4 } from './HKT';
/**
 * @since 1.0.0
 */
export interface Profunctor<F> {
    readonly URI: F;
    readonly map: <L, A, B>(fa: HKT2<F, L, A>, f: (a: A) => B) => HKT2<F, L, B>;
    readonly promap: <A, B, C, D>(fbc: HKT2<F, B, C>, f: (a: A) => B, g: (c: C) => D) => HKT2<F, A, D>;
}
export interface Profunctor2<F extends URIS2> extends Functor2<F> {
    readonly promap: <A, B, C, D>(fbc: Kind2<F, B, C>, f: (a: A) => B, g: (c: C) => D) => Kind2<F, A, D>;
}
export interface Profunctor2C<F extends URIS2, L> extends Functor2C<F, L> {
    readonly promap: <A, C, D>(flc: Kind2<F, L, C>, f: (a: A) => L, g: (c: C) => D) => Kind2<F, A, D>;
}
export interface Profunctor3<F extends URIS3> extends Functor3<F> {
    readonly promap: <U, A, B, C, D>(fbc: Kind3<F, U, B, C>, f: (a: A) => B, g: (c: C) => D) => Kind3<F, U, A, D>;
}
export interface Profunctor4<F extends URIS4> extends Functor4<F> {
    readonly promap: <X, U, A, B, C, D>(fbc: Kind4<F, X, U, B, C>, f: (a: A) => B, g: (c: C) => D) => Kind4<F, X, U, A, D>;
}
/**
 * @since 1.0.0
 * @deprecated
 */
export declare function lmap<F extends URIS3>(profunctor: Profunctor3<F>): <U, A, B, C>(fbc: Kind3<F, U, B, C>, f: (a: A) => B) => Kind3<F, U, A, C>;
/** @deprecated */
export declare function lmap<F extends URIS2>(profunctor: Profunctor2<F>): <A, B, C>(fbc: Kind2<F, B, C>, f: (a: A) => B) => Kind2<F, A, C>;
/** @deprecated */
export declare function lmap<F>(profunctor: Profunctor<F>): <A, B, C>(fbc: HKT2<F, B, C>, f: (a: A) => B) => HKT2<F, A, C>;
/**
 * @since 1.0.0
 * @deprecated
 */
export declare function rmap<F extends URIS3>(profunctor: Profunctor3<F>): <U, B, C, D>(fbc: Kind3<F, U, B, C>, g: (c: C) => D) => Kind3<F, U, B, D>;
/** @deprecated */
export declare function rmap<F extends URIS2>(profunctor: Profunctor2<F>): <B, C, D>(fbc: Kind2<F, B, C>, g: (c: C) => D) => Kind2<F, B, D>;
/** @deprecated */
export declare function rmap<F>(profunctor: Profunctor<F>): <B, C, D>(fbc: HKT2<F, B, C>, g: (c: C) => D) => HKT2<F, B, D>;
