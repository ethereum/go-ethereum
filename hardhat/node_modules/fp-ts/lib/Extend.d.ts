import { Functor, Functor1, Functor2, Functor2C, Functor3, Functor3C, Functor4 } from './Functor';
import { HKT, Kind, Kind2, Kind3, URIS, URIS2, URIS3, URIS4, Kind4 } from './HKT';
/**
 * @since 1.0.0
 */
export interface Extend<F> extends Functor<F> {
    readonly extend: <A, B>(ea: HKT<F, A>, f: (fa: HKT<F, A>) => B) => HKT<F, B>;
}
export interface Extend1<F extends URIS> extends Functor1<F> {
    readonly extend: <A, B>(ea: Kind<F, A>, f: (fa: Kind<F, A>) => B) => Kind<F, B>;
}
export interface Extend2<F extends URIS2> extends Functor2<F> {
    readonly extend: <L, A, B>(ea: Kind2<F, L, A>, f: (fa: Kind2<F, L, A>) => B) => Kind2<F, L, B>;
}
export interface Extend3<F extends URIS3> extends Functor3<F> {
    readonly extend: <U, L, A, B>(ea: Kind3<F, U, L, A>, f: (fa: Kind3<F, U, L, A>) => B) => Kind3<F, U, L, B>;
}
export interface Extend2C<F extends URIS2, L> extends Functor2C<F, L> {
    readonly extend: <A, B>(ea: Kind2<F, L, A>, f: (fa: Kind2<F, L, A>) => B) => Kind2<F, L, B>;
}
export interface Extend3C<F extends URIS3, U, L> extends Functor3C<F, U, L> {
    readonly extend: <A, B>(ea: Kind3<F, U, L, A>, f: (fa: Kind3<F, U, L, A>) => B) => Kind3<F, U, L, B>;
}
export interface Extend4<W extends URIS4> extends Functor4<W> {
    readonly extend: <X, U, L, A, B>(wa: Kind4<W, X, U, L, A>, f: (wa: Kind4<W, X, U, L, A>) => B) => Kind4<W, X, U, L, B>;
}
/**
 * Use `pipeable`'s `duplicate`
 * @since 1.0.0
 * @deprecated
 */
export declare function duplicate<F extends URIS3>(E: Extend3<F>): <U, L, A>(ma: Kind3<F, U, L, A>) => Kind3<F, U, L, Kind3<F, U, L, A>>;
/**
 * Use `pipeable`'s `duplicate`
 * @deprecated
 */
export declare function duplicate<F extends URIS3, U, L>(E: Extend3C<F, U, L>): <A>(ma: Kind3<F, U, L, A>) => Kind3<F, U, L, Kind3<F, U, L, A>>;
/**
 * Use `pipeable`'s `duplicate`
 * @deprecated
 */
export declare function duplicate<F extends URIS2>(E: Extend2<F>): <L, A>(ma: Kind2<F, L, A>) => Kind2<F, L, Kind2<F, L, A>>;
/**
 * Use `pipeable`'s `duplicate`
 * @deprecated
 */
export declare function duplicate<F extends URIS2, L>(E: Extend2C<F, L>): <A>(ma: Kind2<F, L, A>) => Kind2<F, L, Kind2<F, L, A>>;
/**
 * Use `pipeable`'s `duplicate`
 * @deprecated
 */
export declare function duplicate<F extends URIS>(E: Extend1<F>): <A>(ma: Kind<F, A>) => Kind<F, Kind<F, A>>;
/**
 * Use `pipeable`'s `duplicate`
 * @deprecated
 */
export declare function duplicate<F>(E: Extend<F>): <A>(ma: HKT<F, A>) => HKT<F, HKT<F, A>>;
