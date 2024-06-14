import { HKT2, Kind2, Kind3, Kind4, URIS2, URIS3, URIS4 } from './HKT';
/**
 * @since 1.0.0
 */
export interface Semigroupoid<F> {
    readonly URI: F;
    readonly compose: <L, A, B>(ab: HKT2<F, A, B>, la: HKT2<F, L, A>) => HKT2<F, L, B>;
}
export interface Semigroupoid2<F extends URIS2> {
    readonly URI: F;
    readonly compose: <L, A, B>(ab: Kind2<F, A, B>, la: Kind2<F, L, A>) => Kind2<F, L, B>;
}
export interface Semigroupoid2C<F extends URIS2, L> {
    readonly URI: F;
    readonly compose: <A, B>(ab: Kind2<F, A, B>, la: Kind2<F, L, A>) => Kind2<F, L, B>;
}
export interface Semigroupoid3<F extends URIS3> {
    readonly URI: F;
    readonly compose: <U, L, A, B>(ab: Kind3<F, U, A, B>, la: Kind3<F, U, L, A>) => Kind3<F, U, L, B>;
}
export interface Semigroupoid4<F extends URIS4> {
    readonly URI: F;
    readonly compose: <X, U, L, A, B>(ab: Kind4<F, X, U, A, B>, la: Kind4<F, X, U, L, A>) => Kind4<F, X, U, L, B>;
}
export interface Semigroupoid3C<F extends URIS3, U> {
    readonly URI: F;
    readonly _U: U;
    readonly compose: <L, A, B>(ab: Kind3<F, U, A, B>, la: Kind3<F, U, L, A>) => Kind3<F, U, L, B>;
}
