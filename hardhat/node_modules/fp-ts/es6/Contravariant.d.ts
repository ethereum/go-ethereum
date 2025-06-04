import { HKT, Kind, Kind2, Kind3, URIS, URIS2, URIS3, URIS4, Kind4 } from './HKT';
/**
 * @since 1.0.0
 */
export interface Contravariant<F> {
    readonly URI: F;
    readonly contramap: <A, B>(fa: HKT<F, A>, f: (b: B) => A) => HKT<F, B>;
}
export interface Contravariant1<F extends URIS> {
    readonly URI: F;
    readonly contramap: <A, B>(fa: Kind<F, A>, f: (b: B) => A) => Kind<F, B>;
}
export interface Contravariant2<F extends URIS2> {
    readonly URI: F;
    readonly contramap: <L, A, B>(fa: Kind2<F, L, A>, f: (b: B) => A) => Kind2<F, L, B>;
}
export interface Contravariant3<F extends URIS3> {
    readonly URI: F;
    readonly contramap: <U, L, A, B>(fa: Kind3<F, U, L, A>, f: (b: B) => A) => Kind3<F, U, L, B>;
}
export interface Contravariant2C<F extends URIS2, L> {
    readonly URI: F;
    readonly _L: L;
    readonly contramap: <A, B>(fa: Kind2<F, L, A>, f: (b: B) => A) => Kind2<F, L, B>;
}
export interface Contravariant3C<F extends URIS3, U, L> {
    readonly URI: F;
    readonly _L: L;
    readonly _U: U;
    readonly contramap: <A, B>(fa: Kind3<F, U, L, A>, f: (b: B) => A) => Kind3<F, U, L, B>;
}
export interface Contravariant4<F extends URIS4> {
    readonly URI: F;
    readonly contramap: <X, U, L, A, B>(fa: Kind4<F, X, U, L, A>, f: (b: B) => A) => Kind4<F, X, U, L, B>;
}
/**
 * Use `pipeable`'s `contramap`
 * @since 1.0.0
 * @deprecated
 */
export declare function lift<F extends URIS3>(contravariant: Contravariant3<F>): <A, B>(f: (b: B) => A) => <U, L>(fa: Kind3<F, U, L, A>) => Kind3<F, U, L, B>;
/**
 * Use `pipeable`'s `contramap`
 * @deprecated
 */
export declare function lift<F extends URIS3, U, L>(contravariant: Contravariant3C<F, U, L>): <A, B>(f: (b: B) => A) => (fa: Kind3<F, U, L, A>) => Kind3<F, U, L, B>;
/**
 * Use `pipeable`'s `contramap`
 * @deprecated
 */
export declare function lift<F extends URIS2>(contravariant: Contravariant2<F>): <A, B>(f: (b: B) => A) => <L>(fa: Kind2<F, L, A>) => Kind2<F, L, B>;
/**
 * Use `pipeable`'s `contramap`
 * @deprecated
 */
export declare function lift<F extends URIS2, L>(contravariant: Contravariant2C<F, L>): <A, B>(f: (b: B) => A) => (fa: Kind2<F, L, A>) => Kind2<F, L, B>;
/**
 * Use `pipeable`'s `contramap`
 * @deprecated
 */
export declare function lift<F extends URIS>(contravariant: Contravariant1<F>): <A, B>(f: (b: B) => A) => (fa: Kind<F, A>) => Kind<F, B>;
/**
 * Use `pipeable`'s `contramap`
 * @deprecated
 */
export declare function lift<F>(contravariant: Contravariant<F>): <A, B>(f: (b: B) => A) => (fa: HKT<F, A>) => HKT<F, B>;
