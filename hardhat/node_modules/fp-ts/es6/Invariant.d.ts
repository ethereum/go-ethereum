import { HKT, HKT2, HKT3, Kind, Kind2, Kind3, URIS, URIS2, URIS3 } from './HKT';
/**
 * @since 1.0.0
 */
export interface Invariant<F> {
    readonly URI: F;
    readonly imap: <A, B>(fa: HKT<F, A>, f: (a: A) => B, g: (b: B) => A) => HKT<F, B>;
}
export interface Invariant1<F extends URIS> {
    readonly URI: F;
    readonly imap: <A, B>(fa: HKT<F, A>, f: (a: A) => B, g: (b: B) => A) => Kind<F, B>;
}
export interface Invariant2<F extends URIS2> {
    readonly URI: F;
    readonly imap: <L, A, B>(fa: HKT2<F, L, A>, f: (a: A) => B, g: (b: B) => A) => Kind2<F, L, B>;
}
export interface Invariant3<F extends URIS3> {
    readonly URI: F;
    readonly imap: <U, L, A, B>(fa: HKT3<F, U, L, A>, f: (a: A) => B, g: (b: B) => A) => Kind3<F, U, L, B>;
}
export interface Invariant2C<F extends URIS2, L> {
    readonly URI: F;
    readonly _L: L;
    readonly imap: <A, B>(fa: HKT2<F, L, A>, f: (a: A) => B, g: (b: B) => A) => Kind2<F, L, B>;
}
export interface Invariant3C<F extends URIS3, U, L> {
    readonly URI: F;
    readonly _L: L;
    readonly _U: U;
    readonly imap: <A, B>(fa: HKT3<F, U, L, A>, f: (a: A) => B, g: (b: B) => A) => Kind3<F, U, L, B>;
}
