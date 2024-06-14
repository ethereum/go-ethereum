/**
 * @file The `Chain` type class extends the `Apply` type class with a `chain` operation which composes computations in
 * sequence, using the return value of one computation to determine the next computation.
 *
 * Instances must satisfy the following law in addition to the `Apply` laws:
 *
 * 1. Associativity: `F.chain(F.chain(fa, afb), bfc) <-> F.chain(fa, a => F.chain(afb(a), bfc))`
 *
 * Note. `Apply`'s `ap` can be derived: `(fab, fa) => F.chain(fab, f => F.map(f, fa))`
 */
import { Apply, Apply1, Apply2, Apply2C, Apply3, Apply3C, Apply4 } from './Apply';
import { HKT, Kind, Kind2, Kind3, URIS, URIS2, URIS3, URIS4, Kind4 } from './HKT';
/**
 * @since 1.0.0
 */
export interface Chain<F> extends Apply<F> {
    readonly chain: <A, B>(fa: HKT<F, A>, f: (a: A) => HKT<F, B>) => HKT<F, B>;
}
export interface Chain1<F extends URIS> extends Apply1<F> {
    readonly chain: <A, B>(fa: Kind<F, A>, f: (a: A) => Kind<F, B>) => Kind<F, B>;
}
export interface Chain2<F extends URIS2> extends Apply2<F> {
    readonly chain: <L, A, B>(fa: Kind2<F, L, A>, f: (a: A) => Kind2<F, L, B>) => Kind2<F, L, B>;
}
export interface Chain3<F extends URIS3> extends Apply3<F> {
    readonly chain: <U, L, A, B>(fa: Kind3<F, U, L, A>, f: (a: A) => Kind3<F, U, L, B>) => Kind3<F, U, L, B>;
}
export interface Chain2C<F extends URIS2, L> extends Apply2C<F, L> {
    readonly chain: <A, B>(fa: Kind2<F, L, A>, f: (a: A) => Kind2<F, L, B>) => Kind2<F, L, B>;
}
export interface Chain3C<F extends URIS3, U, L> extends Apply3C<F, U, L> {
    readonly chain: <A, B>(fa: Kind3<F, U, L, A>, f: (a: A) => Kind3<F, U, L, B>) => Kind3<F, U, L, B>;
}
export interface Chain4<F extends URIS4> extends Apply4<F> {
    readonly chain: <X, U, L, A, B>(fa: Kind4<F, X, U, L, A>, f: (a: A) => Kind4<F, X, U, L, B>) => Kind4<F, X, U, L, B>;
}
/**
 * Use `pipeable`'s `flatten`
 * @since 1.0.0
 * @deprecated
 */
export declare function flatten<F extends URIS3>(chain: Chain3<F>): <U, L, A>(mma: Kind3<F, U, L, Kind3<F, U, L, A>>) => Kind3<F, U, L, A>;
/**
 * Use `pipeable`'s `flatten`
 * @deprecated
 */
export declare function flatten<F extends URIS3, U, L>(chain: Chain3C<F, U, L>): <A>(mma: Kind3<F, U, L, Kind3<F, U, L, A>>) => Kind3<F, U, L, A>;
/**
 * Use `pipeable`'s `flatten`
 * @deprecated
 */
export declare function flatten<F extends URIS2>(chain: Chain2<F>): <L, A>(mma: Kind2<F, L, Kind2<F, L, A>>) => Kind2<F, L, A>;
/**
 * Use `pipeable`'s `flatten`
 * @deprecated
 */
export declare function flatten<F extends URIS2, L>(chain: Chain2C<F, L>): <A>(mma: Kind2<F, L, Kind2<F, L, A>>) => Kind2<F, L, A>;
/**
 * Use `pipeable`'s `flatten`
 * @deprecated
 */
export declare function flatten<F extends URIS>(chain: Chain1<F>): <A>(mma: Kind<F, Kind<F, A>>) => Kind<F, A>;
/**
 * Use `pipeable`'s `flatten`
 * @deprecated
 */
export declare function flatten<F>(chain: Chain<F>): <A>(mma: HKT<F, HKT<F, A>>) => HKT<F, A>;
