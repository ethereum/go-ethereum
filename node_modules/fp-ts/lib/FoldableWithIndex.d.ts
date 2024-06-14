/**
 * @file A `Foldable` with an additional index.
 * A `FoldableWithIndex` instance must be compatible with its `Foldable` instance
 *
 * ```ts
 * reduce(fa, b, f) = reduceWithIndex(fa, b, (_, b, a) => f(b, a))
 * foldMap(M)(fa, f) = foldMapWithIndex(M)(fa, (_, a) => f(a))
 * foldr(fa, b, f) = foldrWithIndex(fa, b, (_, a, b) => f(a, b))
 * ```
 */
import { Foldable2v, Foldable2v1, Foldable2v2, Foldable2v2C, Foldable2v3, Foldable2v3C, Foldable2vComposition, Foldable2vComposition11, Foldable2vComposition12, Foldable2vComposition12C, Foldable2vComposition21, Foldable2vComposition2C1, Foldable2vComposition22, Foldable2vComposition22C, Foldable2vComposition3C1, Foldable2v4 } from './Foldable2v';
import { HKT, Kind, Kind2, Kind3, URIS, URIS2, URIS3, URIS4, Kind4 } from './HKT';
import { Monoid } from './Monoid';
/**
 * @since 1.12.0
 */
export interface FoldableWithIndex<F, I> extends Foldable2v<F> {
    readonly reduceWithIndex: <A, B>(fa: HKT<F, A>, b: B, f: (i: I, b: B, a: A) => B) => B;
    readonly foldMapWithIndex: <M>(M: Monoid<M>) => <A>(fa: HKT<F, A>, f: (i: I, a: A) => M) => M;
    readonly foldrWithIndex: <A, B>(fa: HKT<F, A>, b: B, f: (i: I, a: A, b: B) => B) => B;
}
export interface FoldableWithIndex1<F extends URIS, I> extends Foldable2v1<F> {
    readonly reduceWithIndex: <A, B>(fa: Kind<F, A>, b: B, f: (i: I, b: B, a: A) => B) => B;
    readonly foldMapWithIndex: <M>(M: Monoid<M>) => <A>(fa: Kind<F, A>, f: (i: I, a: A) => M) => M;
    readonly foldrWithIndex: <A, B>(fa: Kind<F, A>, b: B, f: (i: I, a: A, b: B) => B) => B;
}
export interface FoldableWithIndex2<F extends URIS2, I> extends Foldable2v2<F> {
    readonly reduceWithIndex: <L, A, B>(fa: Kind2<F, L, A>, b: B, f: (i: I, b: B, a: A) => B) => B;
    readonly foldMapWithIndex: <M>(M: Monoid<M>) => <L, A>(fa: Kind2<F, L, A>, f: (i: I, a: A) => M) => M;
    readonly foldrWithIndex: <L, A, B>(fa: Kind2<F, L, A>, b: B, f: (i: I, a: A, b: B) => B) => B;
}
export interface FoldableWithIndex3<F extends URIS3, I> extends Foldable2v3<F> {
    readonly reduceWithIndex: <U, L, A, B>(fa: Kind3<F, U, L, A>, b: B, f: (i: I, b: B, a: A) => B) => B;
    readonly foldMapWithIndex: <M>(M: Monoid<M>) => <U, L, A>(fa: Kind3<F, U, L, A>, f: (i: I, a: A) => M) => M;
    readonly foldrWithIndex: <U, L, A, B>(fa: Kind3<F, U, L, A>, b: B, f: (i: I, a: A, b: B) => B) => B;
}
export interface FoldableWithIndex2C<F extends URIS2, I, L> extends Foldable2v2C<F, L> {
    readonly reduceWithIndex: <A, B>(fa: Kind2<F, L, A>, b: B, f: (i: I, b: B, a: A) => B) => B;
    readonly foldMapWithIndex: <M>(M: Monoid<M>) => <A>(fa: Kind2<F, L, A>, f: (i: I, a: A) => M) => M;
    readonly foldrWithIndex: <A, B>(fa: Kind2<F, L, A>, b: B, f: (i: I, a: A, b: B) => B) => B;
}
export interface FoldableWithIndex3C<F extends URIS3, I, U, L> extends Foldable2v3C<F, U, L> {
    readonly reduceWithIndex: <A, B>(fa: Kind3<F, U, L, A>, b: B, f: (i: I, b: B, a: A) => B) => B;
    readonly foldMapWithIndex: <M>(M: Monoid<M>) => <A>(fa: Kind3<F, U, L, A>, f: (i: I, a: A) => M) => M;
    readonly foldrWithIndex: <A, B>(fa: Kind3<F, U, L, A>, b: B, f: (i: I, a: A, b: B) => B) => B;
}
export interface FoldableWithIndex4<F extends URIS4, I> extends Foldable2v4<F> {
    readonly reduceWithIndex: <X, U, L, A, B>(fa: Kind4<F, X, U, L, A>, b: B, f: (i: I, b: B, a: A) => B) => B;
    readonly foldMapWithIndex: <M>(M: Monoid<M>) => <X, U, L, A>(fa: Kind4<F, X, U, L, A>, f: (i: I, a: A) => M) => M;
    readonly reduceRightWithIndex: <X, U, L, A, B>(fa: Kind4<F, X, U, L, A>, b: B, f: (i: I, a: A, b: B) => B) => B;
}
export interface FoldableWithIndexComposition<F, FI, G, GI> extends Foldable2vComposition<F, G> {
    readonly reduceWithIndex: <A, B>(fga: HKT<F, HKT<G, A>>, b: B, f: (i: [FI, GI], b: B, a: A) => B) => B;
    readonly foldMapWithIndex: <M>(M: Monoid<M>) => <A>(fga: HKT<F, HKT<G, A>>, f: (i: [FI, GI], a: A) => M) => M;
    readonly foldrWithIndex: <A, B>(fga: HKT<F, HKT<G, A>>, b: B, f: (i: [FI, GI], a: A, b: B) => B) => B;
}
export interface FoldableWithIndexComposition11<F extends URIS, FI, G extends URIS, GI> extends Foldable2vComposition11<F, G> {
    readonly reduceWithIndex: <A, B>(fga: Kind<F, Kind<G, A>>, b: B, f: (i: [FI, GI], b: B, a: A) => B) => B;
    readonly foldMapWithIndex: <M>(M: Monoid<M>) => <A>(fga: Kind<F, Kind<G, A>>, f: (i: [FI, GI], a: A) => M) => M;
    readonly foldrWithIndex: <A, B>(fga: Kind<F, Kind<G, A>>, b: B, f: (i: [FI, GI], a: A, b: B) => B) => B;
}
export interface FoldableWithIndexComposition12<F extends URIS, FI, G extends URIS2, GI> extends Foldable2vComposition12<F, G> {
    readonly reduceWithIndex: <LG, A, B>(fga: Kind<F, Kind2<G, LG, A>>, b: B, f: (i: [FI, GI], b: B, a: A) => B) => B;
    readonly foldMapWithIndex: <M>(M: Monoid<M>) => <LG, A>(fga: Kind<F, Kind2<G, LG, A>>, f: (i: [FI, GI], a: A) => M) => M;
    readonly foldrWithIndex: <LG, A, B>(fga: Kind<F, Kind2<G, LG, A>>, b: B, f: (i: [FI, GI], a: A, b: B) => B) => B;
}
export interface FoldableWithIndexComposition12C<F extends URIS, FI, G extends URIS2, GI, LG> extends Foldable2vComposition12C<F, G, LG> {
    readonly reduceWithIndex: <A, B>(fga: Kind<F, Kind2<G, LG, A>>, b: B, f: (i: [FI, GI], b: B, a: A) => B) => B;
    readonly foldMapWithIndex: <M>(M: Monoid<M>) => <A>(fga: Kind<F, Kind2<G, LG, A>>, f: (i: [FI, GI], a: A) => M) => M;
    readonly foldrWithIndex: <A, B>(fga: Kind<F, Kind2<G, LG, A>>, b: B, f: (i: [FI, GI], a: A, b: B) => B) => B;
}
export interface FoldableWithIndexComposition21<F extends URIS2, FI, G extends URIS, GI> extends Foldable2vComposition21<F, G> {
    readonly reduceWithIndex: <LF, A, B>(fga: Kind2<F, LF, Kind<G, A>>, b: B, f: (i: [FI, GI], b: B, a: A) => B) => B;
    readonly foldMapWithIndex: <M>(M: Monoid<M>) => <LF, A>(fga: Kind2<F, LF, Kind<G, A>>, f: (i: [FI, GI], a: A) => M) => M;
    readonly foldrWithIndex: <LF, A, B>(fga: Kind2<F, LF, Kind<G, A>>, b: B, f: (i: [FI, GI], a: A, b: B) => B) => B;
}
export interface FoldableWithIndexComposition2C1<F extends URIS2, FI, G extends URIS, GI, LF> extends Foldable2vComposition2C1<F, G, LF> {
    readonly reduceWithIndex: <A, B>(fga: Kind2<F, LF, Kind<G, A>>, b: B, f: (i: [FI, GI], b: B, a: A) => B) => B;
    readonly foldMapWithIndex: <M>(M: Monoid<M>) => <A>(fga: Kind2<F, LF, Kind<G, A>>, f: (i: [FI, GI], a: A) => M) => M;
    readonly foldrWithIndex: <A, B>(fga: Kind2<F, LF, Kind<G, A>>, b: B, f: (i: [FI, GI], a: A, b: B) => B) => B;
}
export interface FoldableWithIndexComposition22<F extends URIS2, FI, G extends URIS2, GI> extends Foldable2vComposition22<F, G> {
    readonly reduceWithIndex: <LF, LG, A, B>(fga: Kind2<F, LF, Kind2<G, LG, A>>, b: B, f: (i: [FI, GI], b: B, a: A) => B) => B;
    readonly foldMapWithIndex: <M>(M: Monoid<M>) => <LF, LG, A>(fga: Kind2<F, LF, Kind2<G, LG, A>>, f: (i: [FI, GI], a: A) => M) => M;
    readonly foldrWithIndex: <LF, LG, A, B>(fga: Kind2<F, LF, Kind2<G, LG, A>>, b: B, f: (i: [FI, GI], a: A, b: B) => B) => B;
}
export interface FoldableWithIndexComposition22C<F extends URIS2, FI, G extends URIS2, GI, LG> extends Foldable2vComposition22C<F, G, LG> {
    readonly reduceWithIndex: <LF, A, B>(fga: Kind2<F, LF, Kind2<G, LG, A>>, b: B, f: (i: [FI, GI], b: B, a: A) => B) => B;
    readonly foldMapWithIndex: <M>(M: Monoid<M>) => <LF, A>(fga: Kind2<F, LF, Kind2<G, LG, A>>, f: (i: [FI, GI], a: A) => M) => M;
    readonly foldrWithIndex: <LF, A, B>(fga: Kind2<F, LF, Kind2<G, LG, A>>, b: B, f: (i: [FI, GI], a: A, b: B) => B) => B;
}
export interface FoldableWithIndexComposition3C1<F extends URIS3, FI, G extends URIS, GI, UF, LF> extends Foldable2vComposition3C1<F, G, UF, LF> {
    readonly reduceWithIndex: <A, B>(fga: Kind3<F, UF, LF, Kind<G, A>>, b: B, f: (i: [FI, GI], b: B, a: A) => B) => B;
    readonly foldMapWithIndex: <M>(M: Monoid<M>) => <A>(fga: Kind3<F, UF, LF, Kind<G, A>>, f: (i: [FI, GI], a: A) => M) => M;
    readonly foldrWithIndex: <A, B>(fga: Kind3<F, UF, LF, Kind<G, A>>, b: B, f: (i: [FI, GI], a: A, b: B) => B) => B;
}
/**
 * @since 1.12.0
 */
export declare function getFoldableWithIndexComposition<F extends URIS3, FI, G extends URIS, GI, UF, LF>(F: FoldableWithIndex3C<F, FI, UF, LF>, G: FoldableWithIndex1<G, GI>): FoldableWithIndexComposition3C1<F, FI, G, GI, UF, LF>;
export declare function getFoldableWithIndexComposition<F extends URIS2, FI, G extends URIS2, GI, LG>(F: FoldableWithIndex2<F, FI>, G: FoldableWithIndex2C<G, GI, LG>): FoldableWithIndexComposition22C<F, FI, G, GI, LG>;
export declare function getFoldableWithIndexComposition<F extends URIS2, FI, G extends URIS2, GI>(F: FoldableWithIndex2<F, FI>, G: FoldableWithIndex2<G, GI>): FoldableWithIndexComposition22<F, FI, G, GI>;
export declare function getFoldableWithIndexComposition<F extends URIS2, FI, G extends URIS, GI, LF>(F: FoldableWithIndex2C<F, FI, LF>, G: FoldableWithIndex1<G, GI>): FoldableWithIndexComposition2C1<F, FI, G, GI, LF>;
export declare function getFoldableWithIndexComposition<F extends URIS2, FI, G extends URIS, GI>(F: FoldableWithIndex2<F, FI>, G: FoldableWithIndex1<G, GI>): FoldableWithIndexComposition21<F, FI, G, GI>;
export declare function getFoldableWithIndexComposition<F extends URIS, FI, G extends URIS2, GI>(F: FoldableWithIndex1<F, FI>, G: FoldableWithIndex2<G, GI>): FoldableWithIndexComposition12<F, FI, G, GI>;
export declare function getFoldableWithIndexComposition<F extends URIS, FI, G extends URIS2, GI>(F: FoldableWithIndex1<F, FI>, G: FoldableWithIndex2<G, GI>): FoldableWithIndexComposition12<F, FI, G, GI>;
export declare function getFoldableWithIndexComposition<F extends URIS, FI, G extends URIS, GI>(F: FoldableWithIndex1<F, FI>, G: FoldableWithIndex1<G, GI>): FoldableWithIndexComposition11<F, FI, G, GI>;
export declare function getFoldableWithIndexComposition<F, FI, G, GI>(F: FoldableWithIndex<F, FI>, G: FoldableWithIndex<G, GI>): FoldableWithIndexComposition<F, FI, G, GI>;
