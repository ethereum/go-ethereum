import { Chain, Chain1, Chain2, Chain2C, Chain3, Chain3C } from './Chain';
import { Either } from './Either';
import { HKT, Kind, Kind2, Kind3, URIS, URIS2, URIS3 } from './HKT';
/**
 * @since 1.0.0
 */
export interface ChainRec<F> extends Chain<F> {
    readonly chainRec: <A, B>(a: A, f: (a: A) => HKT<F, Either<A, B>>) => HKT<F, B>;
}
export interface ChainRec1<F extends URIS> extends Chain1<F> {
    readonly chainRec: <A, B>(a: A, f: (a: A) => Kind<F, Either<A, B>>) => Kind<F, B>;
}
export interface ChainRec2<F extends URIS2> extends Chain2<F> {
    readonly chainRec: <L, A, B>(a: A, f: (a: A) => Kind2<F, L, Either<A, B>>) => Kind2<F, L, B>;
}
export interface ChainRec3<F extends URIS3> extends Chain3<F> {
    readonly chainRec: <U, L, A, B>(a: A, f: (a: A) => Kind3<F, U, L, Either<A, B>>) => Kind3<F, U, L, B>;
}
export interface ChainRec2C<F extends URIS2, L> extends Chain2C<F, L> {
    readonly chainRec: <A, B>(a: A, f: (a: A) => Kind2<F, L, Either<A, B>>) => Kind2<F, L, B>;
}
export interface ChainRec3C<F extends URIS3, U, L> extends Chain3C<F, U, L> {
    readonly chainRec: <A, B>(a: A, f: (a: A) => Kind3<F, U, L, Either<A, B>>) => Kind3<F, U, L, B>;
}
/**
 * @since 1.0.0
 */
export declare const tailRec: <A, B>(f: (a: A) => Either<A, B>, a: A) => B;
