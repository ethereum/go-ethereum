import { HKT2, Kind2, Kind3, URIS2, URIS3, URIS4, Kind4 } from './HKT';
import { Semigroupoid, Semigroupoid2, Semigroupoid3, Semigroupoid3C, Semigroupoid4 } from './Semigroupoid';
/**
 * @since 1.0.0
 */
export interface Category<F> extends Semigroupoid<F> {
    readonly id: <A>() => HKT2<F, A, A>;
}
export interface Category2<F extends URIS2> extends Semigroupoid2<F> {
    readonly id: <A>() => Kind2<F, A, A>;
}
export interface Category3<F extends URIS3> extends Semigroupoid3<F> {
    readonly id: <U, A>() => Kind3<F, U, A, A>;
}
export interface Category4<F extends URIS4> extends Semigroupoid4<F> {
    readonly id: <X, U, A>() => Kind4<F, X, U, A, A>;
}
export interface Category3C<F extends URIS3, U> extends Semigroupoid3C<F, U> {
    readonly id: <A>() => Kind3<F, U, A, A>;
}
