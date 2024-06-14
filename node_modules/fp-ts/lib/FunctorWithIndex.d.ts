/**
 * @file A `FunctorWithIndex` is a type constructor which supports a mapping operation `mapWithIndex`.
 *
 * `mapWithIndex` can be used to turn functions `i -> a -> b` into functions `f a -> f b` whose argument and return types use the type
 * constructor `f` to represent some computational context.
 *
 * Instances must satisfy the following laws:
 *
 * 1. Identity: `F.mapWithIndex(fa, (_i, a) => a) = fa`
 * 2. Composition: `F.mapWithIndex(fa, (_i, a) => bc(ab(a))) = F.mapWithIndex(F.mapWithIndex(fa, ab), bc)`
 */
import { HKT, Kind, Kind2, Kind3, Kind4, URIS, URIS2, URIS3, URIS4 } from './HKT';
import { Functor, Functor1, Functor2, Functor3, Functor4, Functor2C, Functor3C, FunctorComposition, FunctorComposition11, FunctorComposition12, FunctorComposition12C, FunctorComposition21, FunctorComposition2C1, FunctorComposition22, FunctorComposition22C, FunctorComposition3C1 } from './Functor';
/**
 * @since 1.12.0
 */
export interface FunctorWithIndex<F, I> extends Functor<F> {
    readonly mapWithIndex: <A, B>(fa: HKT<F, A>, f: (i: I, a: A) => B) => HKT<F, B>;
}
export interface FunctorWithIndex1<F extends URIS, I> extends Functor1<F> {
    readonly mapWithIndex: <A, B>(fa: Kind<F, A>, f: (i: I, a: A) => B) => Kind<F, B>;
}
export interface FunctorWithIndex2<F extends URIS2, I> extends Functor2<F> {
    readonly mapWithIndex: <L, A, B>(fa: Kind2<F, L, A>, f: (i: I, a: A) => B) => Kind2<F, L, B>;
}
export interface FunctorWithIndex3<F extends URIS3, I> extends Functor3<F> {
    readonly mapWithIndex: <U, L, A, B>(fa: Kind3<F, U, L, A>, f: (i: I, a: A) => B) => Kind3<F, U, L, B>;
}
export interface FunctorWithIndex4<F extends URIS4, I> extends Functor4<F> {
    readonly mapWithIndex: <X, U, L, A, B>(fa: Kind4<F, X, U, L, A>, f: (i: I, a: A) => B) => Kind4<F, X, U, L, B>;
}
export interface FunctorWithIndex2C<F extends URIS2, I, L> extends Functor2C<F, L> {
    readonly mapWithIndex: <A, B>(fa: Kind2<F, L, A>, f: (i: I, a: A) => B) => Kind2<F, L, B>;
}
export interface FunctorWithIndex3C<F extends URIS3, I, U, L> extends Functor3C<F, U, L> {
    readonly mapWithIndex: <A, B>(fa: Kind3<F, U, L, A>, f: (i: I, a: A) => B) => Kind3<F, U, L, B>;
}
export interface FunctorWithIndexComposition<F, FI, G, GI> extends FunctorComposition<F, G> {
    readonly mapWithIndex: <A, B>(fga: HKT<F, HKT<G, A>>, f: (i: [FI, GI], a: A) => B) => HKT<F, HKT<G, B>>;
}
export interface FunctorWithIndexComposition11<F extends URIS, FI, G extends URIS, GI> extends FunctorComposition11<F, G> {
    readonly mapWithIndex: <A, B>(fa: Kind<F, Kind<G, A>>, f: (i: [FI, GI], a: A) => B) => Kind<F, Kind<G, B>>;
}
export interface FunctorWithIndexComposition12<F extends URIS, FI, G extends URIS2, GI> extends FunctorComposition12<F, G> {
    readonly mapWithIndex: <L, A, B>(fa: Kind<F, Kind2<G, L, A>>, f: (i: [FI, GI], a: A) => B) => Kind<F, Kind2<G, L, B>>;
}
export interface FunctorWithIndexComposition12C<F extends URIS, FI, G extends URIS2, GI, L> extends FunctorComposition12C<F, G, L> {
    readonly mapWithIndex: <A, B>(fa: Kind<F, Kind2<G, L, A>>, f: (i: [FI, GI], a: A) => B) => Kind<F, Kind2<G, L, B>>;
}
export interface FunctorWithIndexComposition21<F extends URIS2, FI, G extends URIS, GI> extends FunctorComposition21<F, G> {
    readonly mapWithIndex: <L, A, B>(fa: Kind2<F, L, Kind<G, A>>, f: (i: [FI, GI], a: A) => B) => Kind2<F, L, Kind<G, B>>;
}
export interface FunctorWithIndexComposition2C1<F extends URIS2, FI, G extends URIS, GI, L> extends FunctorComposition2C1<F, G, L> {
    readonly mapWithIndex: <A, B>(fa: Kind2<F, L, Kind<G, A>>, f: (i: [FI, GI], a: A) => B) => Kind2<F, L, Kind<G, B>>;
}
export interface FunctorWithIndexComposition22<F extends URIS2, FI, G extends URIS2, GI> extends FunctorComposition22<F, G> {
    readonly mapWithIndex: <L, M, A, B>(fa: Kind2<F, L, Kind2<G, M, A>>, f: (i: [FI, GI], a: A) => B) => Kind2<F, L, Kind2<G, M, B>>;
}
export interface FunctorWithIndexComposition22C<F extends URIS2, FI, G extends URIS2, GI, LG> extends FunctorComposition22C<F, G, LG> {
    readonly mapWithIndex: <L, A, B>(fa: Kind2<F, L, Kind2<G, LG, A>>, f: (i: [FI, GI], a: A) => B) => Kind2<F, L, Kind2<G, LG, B>>;
}
export interface FunctorWithIndexComposition3C1<F extends URIS3, FI, G extends URIS, GI, UF, LF> extends FunctorComposition3C1<F, G, UF, LF> {
    readonly mapWithIndex: <A, B>(fa: Kind3<F, UF, LF, Kind<G, A>>, f: (i: [FI, GI], a: A) => B) => Kind3<F, UF, LF, Kind<G, B>>;
}
/**
 * @since 1.12.0
 */
export declare function getFunctorWithIndexComposition<F extends URIS3, FI, G extends URIS, GI, U, L>(F: FunctorWithIndex3C<F, FI, U, L>, G: FunctorWithIndex1<G, FI>): FunctorWithIndexComposition3C1<F, FI, G, GI, U, L>;
export declare function getFunctorWithIndexComposition<F extends URIS2, FI, G extends URIS2, GI, L>(F: FunctorWithIndex2<F, FI>, G: FunctorWithIndex2C<G, FI, L>): FunctorWithIndexComposition22C<F, FI, G, GI, L>;
export declare function getFunctorWithIndexComposition<F extends URIS2, FI, G extends URIS2, GI>(F: FunctorWithIndex2<F, FI>, G: FunctorWithIndex2<G, FI>): FunctorWithIndexComposition22<F, FI, G, GI>;
export declare function getFunctorWithIndexComposition<F extends URIS2, FI, G extends URIS, GI, L>(F: FunctorWithIndex2C<F, FI, L>, G: FunctorWithIndex1<G, GI>): FunctorWithIndexComposition2C1<F, FI, G, GI, L>;
export declare function getFunctorWithIndexComposition<F extends URIS2, FI, G extends URIS, GI>(F: FunctorWithIndex2<F, FI>, G: FunctorWithIndex1<G, GI>): FunctorWithIndexComposition21<F, FI, G, GI>;
export declare function getFunctorWithIndexComposition<F extends URIS, FI, G extends URIS2, GI, L>(F: FunctorWithIndex1<F, FI>, G: FunctorWithIndex2C<G, GI, L>): FunctorWithIndexComposition12C<F, FI, G, GI, L>;
export declare function getFunctorWithIndexComposition<F extends URIS, FI, G extends URIS2, GI>(F: FunctorWithIndex1<F, FI>, G: FunctorWithIndex2<G, GI>): FunctorWithIndexComposition12<F, FI, G, GI>;
export declare function getFunctorWithIndexComposition<F extends URIS, FI, G extends URIS, GI>(F: FunctorWithIndex1<F, FI>, G: FunctorWithIndex1<G, GI>): FunctorWithIndexComposition11<F, FI, G, GI>;
export declare function getFunctorWithIndexComposition<F, FI, G, GI>(F: FunctorWithIndex<F, FI>, G: FunctorWithIndex<G, GI>): FunctorWithIndexComposition<F, FI, G, GI>;
