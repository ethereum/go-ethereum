/**
 * @file The `Plus` type class extends the `alt` type class with a value that should be the left and right identity for `alt`.
 *
 * It is similar to `Monoid`, except that it applies to types of kind `* -> *`, like `Array` or `Option`, rather than
 * concrete types like `string` or `number`.
 *
 * `Plus` instances should satisfy the following laws:
 *
 * 1. Left identity: `A.alt(zero, fa) == fa`
 * 2. Right identity: `A.alt(fa, zero) == fa`
 * 3. Annihilation: `A.map(zero, fa) == zero`
 */
import { Alt, Alt1, Alt2, Alt2C, Alt3, Alt3C } from './Alt';
import { HKT, Kind, Kind2, Kind3, URIS, URIS2, URIS3 } from './HKT';
/**
 * @since 1.0.0
 */
export interface Plus<F> extends Alt<F> {
    readonly zero: <A>() => HKT<F, A>;
}
export interface Plus1<F extends URIS> extends Alt1<F> {
    readonly zero: <A>() => Kind<F, A>;
}
export interface Plus2<F extends URIS2> extends Alt2<F> {
    readonly zero: <L, A>() => Kind2<F, L, A>;
}
export interface Plus3<F extends URIS3> extends Alt3<F> {
    readonly zero: <U, L, A>() => Kind3<F, U, L, A>;
}
export interface Plus2C<F extends URIS2, L> extends Alt2C<F, L> {
    readonly zero: <A>() => Kind2<F, L, A>;
}
export interface Plus3C<F extends URIS3, U, L> extends Alt3C<F, U, L> {
    readonly zero: <A>() => Kind3<F, U, L, A>;
}
