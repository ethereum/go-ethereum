import { HKT2, Kind2, Kind3, URIS2, URIS3, URIS4, Kind4 } from './HKT';
/**
 * @since 1.0.0
 */
export interface Bifunctor<F> {
    readonly URI: F;
    readonly bimap: <L, A, M, B>(fla: HKT2<F, L, A>, f: (l: L) => M, g: (a: A) => B) => HKT2<F, M, B>;
}
export interface Bifunctor2<F extends URIS2> {
    readonly URI: F;
    readonly bimap: <L, A, M, B>(fla: Kind2<F, L, A>, f: (l: L) => M, g: (a: A) => B) => Kind2<F, M, B>;
}
export interface Bifunctor2C<F extends URIS2, L> {
    readonly URI: F;
    readonly _L: L;
    readonly bimap: <A, M, B>(fla: Kind2<F, L, A>, f: (l: L) => M, g: (a: A) => B) => Kind2<F, M, B>;
}
export interface Bifunctor3<F extends URIS3> {
    readonly URI: F;
    readonly bimap: <U, L, A, M, B>(fla: Kind3<F, U, L, A>, f: (l: L) => M, g: (a: A) => B) => Kind3<F, U, M, B>;
}
export interface Bifunctor3C<F extends URIS3, U> {
    readonly URI: F;
    readonly _U: U;
    readonly bimap: <L, A, M, B>(fla: Kind3<F, U, L, A>, f: (l: L) => M, g: (a: A) => B) => Kind3<F, U, M, B>;
}
export interface Bifunctor4<F extends URIS4> {
    readonly URI: F;
    readonly bimap: <X, U, L, A, M, B>(fla: Kind4<F, X, U, L, A>, f: (l: L) => M, g: (a: A) => B) => Kind4<F, X, U, M, B>;
    readonly mapLeft: <X, U, L, A, M>(fla: Kind4<F, X, U, L, A>, f: (l: L) => M) => Kind4<F, X, U, M, A>;
}
