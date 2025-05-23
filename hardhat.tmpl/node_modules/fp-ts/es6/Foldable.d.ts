import { Applicative, Applicative1, Applicative2, Applicative2C, Applicative3, Applicative3C } from './Applicative';
import { HKT, Kind, Kind2, Kind3, URIS, URIS2, URIS3 } from './HKT';
import { Monad, Monad1, Monad2, Monad2C, Monad3, Monad3C } from './Monad';
import { Monoid } from './Monoid';
import { Option } from './Option';
import { Ord } from './Ord';
import { Plus, Plus1, Plus2, Plus2C, Plus3, Plus3C } from './Plus';
import { Semiring } from './Semiring';
import { Eq } from './Eq';
import { Predicate } from './function';
/**
 * Use `Foldable2v`
 * @since 1.0.0
 * @deprecated
 */
export interface Foldable<F> {
    readonly URI: F;
    readonly reduce: <A, B>(fa: HKT<F, A>, b: B, f: (b: B, a: A) => B) => B;
}
/**
 * Use `Foldable2v`
 * @deprecated
 */
export interface Foldable1<F extends URIS> {
    readonly URI: F;
    readonly reduce: <A, B>(fa: Kind<F, A>, b: B, f: (b: B, a: A) => B) => B;
}
/**
 * Use `Foldable2v`
 * @deprecated
 */
export interface Foldable2<F extends URIS2> {
    readonly URI: F;
    readonly reduce: <L, A, B>(fa: Kind2<F, L, A>, b: B, f: (b: B, a: A) => B) => B;
}
/**
 * Use `Foldable2v`
 * @deprecated
 */
export interface Foldable3<F extends URIS3> {
    readonly URI: F;
    readonly reduce: <U, L, A, B>(fa: Kind3<F, U, L, A>, b: B, f: (b: B, a: A) => B) => B;
}
/**
 * Use `Foldable2v`
 * @deprecated
 */
export interface Foldable2C<F extends URIS2, L> {
    readonly URI: F;
    readonly _L: L;
    readonly reduce: <A, B>(fa: Kind2<F, L, A>, b: B, f: (b: B, a: A) => B) => B;
}
/**
 * Use `Foldable2v`
 * @deprecated
 */
export interface Foldable3C<F extends URIS3, U, L> {
    readonly URI: F;
    readonly _L: L;
    readonly _U: U;
    readonly reduce: <A, B>(fa: Kind3<F, U, L, A>, b: B, f: (b: B, a: A) => B) => B;
}
/**
 * Use `Foldable2v`
 * @deprecated
 */
export interface FoldableComposition<F, G> {
    readonly reduce: <A, B>(fga: HKT<F, HKT<G, A>>, b: B, f: (b: B, a: A) => B) => B;
}
/**
 * Use `Foldable2v`
 * @deprecated
 */
export interface FoldableComposition11<F extends URIS, G extends URIS> {
    readonly reduce: <A, B>(fga: Kind<F, Kind<G, A>>, b: B, f: (b: B, a: A) => B) => B;
}
/**
 * Use `Foldable2v`
 * @deprecated
 */
export interface FoldableComposition12<F extends URIS, G extends URIS2> {
    readonly reduce: <LG, A, B>(fga: Kind<F, Kind2<G, LG, A>>, b: B, f: (b: B, a: A) => B) => B;
}
/**
 * Use `Foldable2v`
 * @deprecated
 */
export interface FoldableComposition12C<F extends URIS, G extends URIS2, LG> {
    readonly reduce: <A, B>(fga: Kind<F, Kind2<G, LG, A>>, b: B, f: (b: B, a: A) => B) => B;
}
/**
 * Use `Foldable2v`
 * @deprecated
 */
export interface FoldableComposition21<F extends URIS2, G extends URIS> {
    readonly reduce: <LF, A, B>(fga: Kind2<F, LF, Kind<G, A>>, b: B, f: (b: B, a: A) => B) => B;
}
/**
 * Use `Foldable2v`
 * @deprecated
 */
export interface FoldableComposition2C1<F extends URIS2, G extends URIS, LF> {
    readonly reduce: <A, B>(fga: Kind2<F, LF, Kind<G, A>>, b: B, f: (b: B, a: A) => B) => B;
}
/**
 * Use `Foldable2v`
 * @deprecated
 */
export interface FoldableComposition22<F extends URIS2, G extends URIS2> {
    readonly reduce: <LF, LG, A, B>(fga: Kind2<F, LF, Kind2<G, LG, A>>, b: B, f: (b: B, a: A) => B) => B;
}
/**
 * Use `Foldable2v`
 * @deprecated
 */
export interface FoldableComposition22C<F extends URIS2, G extends URIS2, LG> {
    readonly reduce: <LF, A, B>(fga: Kind2<F, LF, Kind2<G, LG, A>>, b: B, f: (b: B, a: A) => B) => B;
}
/**
 * Use `Foldable2v`
 * @deprecated
 */
export interface FoldableComposition3C1<F extends URIS3, G extends URIS, UF, LF> {
    readonly reduce: <A, B>(fga: Kind3<F, UF, LF, Kind<G, A>>, b: B, f: (b: B, a: A) => B) => B;
}
/**
 * Use `Foldable2v.getFoldableComposition`
 * @since 1.0.0
 * @deprecated
 */
export declare function getFoldableComposition<F extends URIS3, G extends URIS, UF, LF>(F: Foldable3C<F, UF, LF>, G: Foldable1<G>): FoldableComposition3C1<F, G, UF, LF>;
/**
 * Use `Foldable2v.getFoldableComposition`
 * @deprecated
 */
export declare function getFoldableComposition<F extends URIS2, G extends URIS2, LG>(F: Foldable2C<F, LG>, G: Foldable2<G>): FoldableComposition22C<F, G, LG>;
/**
 * Use `Foldable2v.getFoldableComposition`
 * @deprecated
 */
export declare function getFoldableComposition<F extends URIS2, G extends URIS2>(F: Foldable2<F>, G: Foldable2<G>): FoldableComposition22<F, G>;
/**
 * Use `Foldable2v.getFoldableComposition`
 * @deprecated
 */
export declare function getFoldableComposition<F extends URIS2, G extends URIS>(F: Foldable2<F>, G: Foldable1<G>): FoldableComposition21<F, G>;
/**
 * Use `Foldable2v.getFoldableComposition`
 * @deprecated
 */
export declare function getFoldableComposition<F extends URIS, G extends URIS2>(F: Foldable1<F>, G: Foldable2<G>): FoldableComposition12<F, G>;
/**
 * Use `Foldable2v.getFoldableComposition`
 * @deprecated
 */
export declare function getFoldableComposition<F extends URIS, G extends URIS>(F: Foldable1<F>, G: Foldable1<G>): FoldableComposition11<F, G>;
/**
 * Use `Foldable2v.getFoldableComposition`
 * @deprecated
 */
export declare function getFoldableComposition<F, G>(F: Foldable<F>, G: Foldable<G>): FoldableComposition<F, G>;
/**
 * Use `Foldable2v.foldMap`
 * @since 1.0.0
 * @deprecated
 */
export declare function foldMap<F extends URIS3, M>(F: Foldable3<F>, M: Monoid<M>): <U, L, A>(fa: Kind3<F, U, L, A>, f: (a: A) => M) => M;
/**
 * Use `Foldable2v.foldMap`
 * @deprecated
 */
export declare function foldMap<F extends URIS3, M, U, L>(F: Foldable3C<F, U, L>, M: Monoid<M>): <A>(fa: Kind3<F, U, L, A>, f: (a: A) => M) => M;
/**
 * Use `Foldable2v.foldMap`
 * @deprecated
 */
export declare function foldMap<F extends URIS2, M>(F: Foldable2<F>, M: Monoid<M>): <L, A>(fa: Kind2<F, L, A>, f: (a: A) => M) => M;
/**
 * Use `Foldable2v.foldMap`
 * @deprecated
 */
export declare function foldMap<F extends URIS2, M, L>(F: Foldable2C<F, L>, M: Monoid<M>): <A>(fa: Kind2<F, L, A>, f: (a: A) => M) => M;
/**
 * Use `Foldable2v.foldMap`
 * @deprecated
 */
export declare function foldMap<F extends URIS, M>(F: Foldable1<F>, M: Monoid<M>): <A>(fa: Kind<F, A>, f: (a: A) => M) => M;
/**
 * Use `Foldable2v.foldMap`
 * @deprecated
 */
export declare function foldMap<F, M>(F: Foldable<F>, M: Monoid<M>): <A>(fa: HKT<F, A>, f: (a: A) => M) => M;
/**
 * Use `Foldable2v.foldr`
 * @since 1.0.0
 * @deprecated
 */
export declare function foldr<F extends URIS3>(F: Foldable3<F>): <U, L, A, B>(fa: Kind3<F, U, L, A>, b: B, f: (a: A, b: B) => B) => B;
/**
 * Use `Foldable2v.foldr`
 * @deprecated
 */
export declare function foldr<F extends URIS3, U, L>(F: Foldable3C<F, U, L>): <A, B>(fa: Kind3<F, U, L, A>, b: B, f: (a: A, b: B) => B) => B;
/**
 * Use `Foldable2v.foldr`
 * @deprecated
 */
export declare function foldr<F extends URIS2>(F: Foldable2<F>): <L, A, B>(fa: Kind2<F, L, A>, b: B, f: (a: A, b: B) => B) => B;
/**
 * Use `Foldable2v.foldr`
 * @deprecated
 */
export declare function foldr<F extends URIS2, L>(F: Foldable2C<F, L>): <A, B>(fa: Kind2<F, L, A>, b: B, f: (a: A, b: B) => B) => B;
/**
 * Use `Foldable2v.foldr`
 * @deprecated
 */
export declare function foldr<F extends URIS>(F: Foldable1<F>): <A, B>(fa: Kind<F, A>, b: B, f: (a: A, b: B) => B) => B;
/**
 * Use `Foldable2v.foldr`
 * @deprecated
 */
export declare function foldr<F>(F: Foldable<F>): <A, B>(fa: HKT<F, A>, b: B, f: (a: A, b: B) => B) => B;
/**
 * @since 1.0.0
 * @deprecated
 */
export declare function fold<F extends URIS3, M>(F: Foldable3<F>, M: Monoid<M>): <U, L>(fa: Kind3<F, U, L, M>) => M;
/**
 * @deprecated
 */
export declare function fold<F extends URIS3, M, U, L>(F: Foldable3C<F, U, L>, M: Monoid<M>): (fa: Kind3<F, U, L, M>) => M;
/**
 * @deprecated
 */
export declare function fold<F extends URIS2, M>(F: Foldable2<F>, M: Monoid<M>): <L>(fa: Kind2<F, L, M>) => M;
/**
 * @deprecated
 */
export declare function fold<F extends URIS2, M, L>(F: Foldable2C<F, L>, M: Monoid<M>): (fa: Kind2<F, L, M>) => M;
/**
 * @deprecated
 */
export declare function fold<F extends URIS, M>(F: Foldable1<F>, M: Monoid<M>): (fa: Kind<F, M>) => M;
/**
 * @deprecated
 */
export declare function fold<F, M>(F: Foldable<F>, M: Monoid<M>): (fa: HKT<F, M>) => M;
/**
 * Use `Foldable2v.foldM`
 * @since 1.0.0
 * @deprecated
 */
export declare function foldM<F extends URIS, M extends URIS3>(F: Foldable1<F>, M: Monad3<M>): <U, L, A, B>(f: (b: B, a: A) => Kind3<M, U, L, B>, b: B, fa: Kind<F, A>) => Kind3<M, U, L, B>;
/**
 * Use `Foldable2v.foldM`
 * @deprecated
 */
export declare function foldM<F extends URIS, M extends URIS3, U, L>(F: Foldable1<F>, M: Monad3C<M, U, L>): <A, B>(f: (b: B, a: A) => Kind3<M, U, L, B>, b: B, fa: Kind<F, A>) => Kind3<M, U, L, B>;
/**
 * Use `Foldable2v.foldM`
 * @deprecated
 */
export declare function foldM<F extends URIS, M extends URIS2>(F: Foldable1<F>, M: Monad2<M>): <L, A, B>(f: (b: B, a: A) => Kind2<M, L, B>, b: B, fa: Kind<F, A>) => Kind2<M, L, B>;
/**
 * Use `Foldable2v.foldM`
 * @deprecated
 */
export declare function foldM<F extends URIS, M extends URIS2, L>(F: Foldable1<F>, M: Monad2C<M, L>): <A, B>(f: (b: B, a: A) => Kind2<M, L, B>, b: B, fa: Kind<F, A>) => Kind2<M, L, B>;
/**
 * Use `Foldable2v.foldM`
 * @deprecated
 */
export declare function foldM<F extends URIS, M extends URIS>(F: Foldable1<F>, M: Monad1<M>): <A, B>(f: (b: B, a: A) => Kind<M, B>, b: B, fa: Kind<F, A>) => Kind<M, B>;
/**
 * Use `Foldable2v.foldM`
 * @deprecated
 */
export declare function foldM<F, M>(F: Foldable<F>, M: Monad<M>): <A, B>(f: (b: B, a: A) => HKT<M, B>, b: B, fa: HKT<F, A>) => HKT<M, B>;
/**
 * Use `Foldable2v.traverse_`
 * @since 1.0.0
 * @deprecated
 */
export declare function traverse_<M extends URIS3, F extends URIS>(M: Applicative3<M>, F: Foldable1<F>): <U, L, A, B>(f: (a: A) => Kind3<M, U, L, B>, fa: Kind<F, A>) => Kind3<M, U, L, void>;
/**
 * Use `Foldable2v.traverse_`
 * @deprecated
 */
export declare function traverse_<M extends URIS3, F extends URIS, U, L>(M: Applicative3C<M, U, L>, F: Foldable1<F>): <A, B>(f: (a: A) => Kind3<M, U, L, B>, fa: Kind<F, A>) => Kind3<M, U, L, void>;
/**
 * Use `Foldable2v.traverse_`
 * @deprecated
 */
export declare function traverse_<M extends URIS2, F extends URIS>(M: Applicative2<M>, F: Foldable1<F>): <L, A, B>(f: (a: A) => Kind2<M, L, B>, fa: Kind<F, A>) => Kind2<M, L, void>;
/**
 * Use `Foldable2v.traverse_`
 * @deprecated
 */
export declare function traverse_<M extends URIS2, F extends URIS, L>(M: Applicative2C<M, L>, F: Foldable1<F>): <A, B>(f: (a: A) => Kind2<M, L, B>, fa: Kind<F, A>) => Kind2<M, L, void>;
/**
 * Use `Foldable2v.traverse_`
 * @deprecated
 */
export declare function traverse_<M extends URIS, F extends URIS>(M: Applicative1<M>, F: Foldable1<F>): <A, B>(f: (a: A) => Kind<M, B>, fa: Kind<F, A>) => Kind<M, void>;
/**
 * Use `Foldable2v.traverse_`
 * @deprecated
 */
export declare function traverse_<M, F>(M: Applicative<M>, F: Foldable<F>): <A, B>(f: (a: A) => HKT<M, B>, fa: HKT<F, A>) => HKT<M, void>;
/**
 * Use `Foldable2v.sequence_`
 * @since 1.0.0
 */
export declare function sequence_<M extends URIS3, F extends URIS>(M: Applicative3<M>, F: Foldable1<F>): <U, L, A>(fa: Kind<F, Kind3<M, U, L, A>>) => Kind3<M, U, L, void>;
/**
 * Use `Foldable2v.sequence_`
 * @since 1.0.0
 */
export declare function sequence_<M extends URIS3, F extends URIS, U, L>(M: Applicative3C<M, U, L>, F: Foldable1<F>): <A>(fa: Kind<F, Kind3<M, U, L, A>>) => Kind3<M, U, L, void>;
/**
 * Use `Foldable2v.sequence_`
 * @since 1.0.0
 */
export declare function sequence_<M extends URIS2, F extends URIS>(M: Applicative2<M>, F: Foldable1<F>): <L, A>(fa: Kind<F, Kind2<M, L, A>>) => Kind2<M, L, void>;
/**
 * Use `Foldable2v.sequence_`
 * @since 1.0.0
 */
export declare function sequence_<M extends URIS2, F extends URIS, L>(M: Applicative2C<M, L>, F: Foldable1<F>): <A>(fa: Kind<F, Kind2<M, L, A>>) => Kind2<M, L, void>;
/**
 * Use `Foldable2v.sequence_`
 * @since 1.0.0
 */
export declare function sequence_<M extends URIS, F extends URIS>(M: Applicative1<M>, F: Foldable1<F>): <A>(fa: Kind<F, Kind<M, A>>) => Kind<M, void>;
/**
 * Use `Foldable2v.sequence_`
 * @since 1.0.0
 */
export declare function sequence_<M, F>(M: Applicative<M>, F: Foldable<F>): <A>(fa: HKT<F, HKT<M, A>>) => HKT<M, void>;
/**
 * Combines a collection of elements using the `Alt` operation
 *
 * @since 1.0.0
 * @deprecated
 */
export declare function oneOf<F extends URIS, P extends URIS3>(F: Foldable1<F>, P: Plus3<P>): <U, L, A>(fga: Kind<F, Kind3<P, U, L, A>>) => Kind3<P, U, L, A>;
/** @deprecated */
export declare function oneOf<F extends URIS, P extends URIS3, U, L>(F: Foldable1<F>, P: Plus3C<P, U, L>): <A>(fga: Kind<F, Kind3<P, U, L, A>>) => Kind3<P, U, L, A>;
/** @deprecated */
export declare function oneOf<F extends URIS, P extends URIS2>(F: Foldable1<F>, P: Plus2<P>): <L, A>(fga: Kind<F, Kind2<P, L, A>>) => Kind2<P, L, A>;
/** @deprecated */
export declare function oneOf<F extends URIS, P extends URIS2, L>(F: Foldable1<F>, P: Plus2C<P, L>): <A>(fga: Kind<F, Kind2<P, L, A>>) => Kind2<P, L, A>;
/** @deprecated */
export declare function oneOf<F extends URIS, P extends URIS>(F: Foldable1<F>, P: Plus1<P>): <A>(fga: Kind<F, Kind<P, A>>) => Kind<P, A>;
/** @deprecated */
export declare function oneOf<F, P>(F: Foldable<F>, P: Plus<P>): <A>(fga: HKT<F, HKT<P, A>>) => HKT<P, A>;
/**
 * Use `Foldable2v.intercalate`
 * @since 1.0.0
 */
export declare function intercalate<F extends URIS3, M>(F: Foldable3<F>, M: Monoid<M>): (sep: M) => <U, L>(fm: Kind3<F, U, L, M>) => M;
/**
 * Use `Foldable2v.intercalate`
 * @since 1.0.0
 */
export declare function intercalate<F extends URIS3, M, U, L>(F: Foldable3C<F, U, L>, M: Monoid<M>): (sep: M) => (fm: Kind3<F, U, L, M>) => M;
/**
 * Use `Foldable2v.intercalate`
 * @since 1.0.0
 */
export declare function intercalate<F extends URIS2, M>(F: Foldable2<F>, M: Monoid<M>): (sep: M) => <L>(fm: Kind2<F, L, M>) => M;
/**
 * Use `Foldable2v.intercalate`
 * @since 1.0.0
 */
export declare function intercalate<F extends URIS2, M, L>(F: Foldable2C<F, L>, M: Monoid<M>): (sep: M) => (fm: Kind2<F, L, M>) => M;
/**
 * Use `Foldable2v.intercalate`
 * @since 1.0.0
 */
export declare function intercalate<F extends URIS, M>(F: Foldable1<F>, M: Monoid<M>): (sep: M) => (fm: Kind<F, M>) => M;
/**
 * Use `Foldable2v.intercalate`
 * @since 1.0.0
 */
export declare function intercalate<F, M>(F: Foldable<F>, M: Monoid<M>): (sep: M) => (fm: HKT<F, M>) => M;
/**
 * Find the sum of the numeric values in a data structure
 *
 * @since 1.0.0
 * @deprecated
 */
export declare function sum<F extends URIS3, A>(F: Foldable3<F>, S: Semiring<A>): <U, L>(fa: Kind3<F, U, L, A>) => A;
/** @deprecated */
export declare function sum<F extends URIS3, A, U, L>(F: Foldable3C<F, U, L>, S: Semiring<A>): (fa: Kind3<F, U, L, A>) => A;
/** @deprecated */
export declare function sum<F extends URIS2, A>(F: Foldable2<F>, S: Semiring<A>): <L>(fa: Kind2<F, L, A>) => A;
/** @deprecated */
export declare function sum<F extends URIS2, A, L>(F: Foldable2C<F, L>, S: Semiring<A>): (fa: Kind2<F, L, A>) => A;
/** @deprecated */
export declare function sum<F extends URIS, A>(F: Foldable1<F>, S: Semiring<A>): (fa: Kind<F, A>) => A;
/** @deprecated */
export declare function sum<F, A>(F: Foldable<F>, S: Semiring<A>): (fa: HKT<F, A>) => A;
/**
 * Find the product of the numeric values in a data structure
 *
 * @since 1.0.0
 * @deprecated
 */
export declare function product<F extends URIS3, A>(F: Foldable3<F>, S: Semiring<A>): <U, L>(fa: Kind3<F, U, L, A>) => A;
/** @deprecated */
export declare function product<F extends URIS3, A, U, L>(F: Foldable3C<F, U, L>, S: Semiring<A>): (fa: Kind3<F, U, L, A>) => A;
/** @deprecated */
export declare function product<F extends URIS2, A>(F: Foldable2<F>, S: Semiring<A>): <L>(fa: Kind2<F, L, A>) => A;
/** @deprecated */
export declare function product<F extends URIS2, A, L>(F: Foldable2C<F, L>, S: Semiring<A>): (fa: Kind2<F, L, A>) => A;
/** @deprecated */
export declare function product<F extends URIS, A>(F: Foldable1<F>, S: Semiring<A>): (fa: Kind<F, A>) => A;
/** @deprecated */
export declare function product<F, A>(F: Foldable<F>, S: Semiring<A>): (fa: HKT<F, A>) => A;
/**
 * Use `Foldable2v.elem`
 * @since 1.0.0
 */
export declare function elem<F extends URIS3, A>(F: Foldable3<F>, E: Eq<A>): <U, L>(a: A, fa: Kind3<F, U, L, A>) => boolean;
/**
 * Use `Foldable2v.elem`
 * @since 1.0.0
 */
export declare function elem<F extends URIS3, A, U, L>(F: Foldable3C<F, U, L>, E: Eq<A>): (a: A, fa: Kind3<F, U, L, A>) => boolean;
/**
 * Use `Foldable2v.elem`
 * @since 1.0.0
 */
export declare function elem<F extends URIS2, A>(F: Foldable2<F>, E: Eq<A>): <L>(a: A, fa: Kind2<F, L, A>) => boolean;
/**
 * Use `Foldable2v.elem`
 * @since 1.0.0
 */
export declare function elem<F extends URIS2, A, L>(F: Foldable2C<F, L>, E: Eq<A>): (a: A, fa: Kind2<F, L, A>) => boolean;
/**
 * Use `Foldable2v.elem`
 * @since 1.0.0
 */
export declare function elem<F extends URIS, A>(F: Foldable1<F>, E: Eq<A>): (a: A, fa: Kind<F, A>) => boolean;
/**
 * Use `Foldable2v.elem`
 * @since 1.0.0
 */
export declare function elem<F, A>(F: Foldable<F>, E: Eq<A>): (a: A, fa: HKT<F, A>) => boolean;
/**
 * Try to find an element in a data structure which satisfies a predicate
 *
 * @since 1.0.0
 * @deprecated
 */
export declare function find<F extends URIS3>(F: Foldable3<F>): <U, L, A>(fa: Kind3<F, U, L, A>, p: Predicate<A>) => Option<A>;
/** @deprecated */
export declare function find<F extends URIS3, U, L>(F: Foldable3C<F, U, L>): <A>(fa: Kind3<F, U, L, A>, p: Predicate<A>) => Option<A>;
/** @deprecated */
export declare function find<F extends URIS2>(F: Foldable2<F>): <L, A>(fa: Kind2<F, L, A>, p: Predicate<A>) => Option<A>;
/** @deprecated */
export declare function find<F extends URIS2, L>(F: Foldable2C<F, L>): <A>(fa: Kind2<F, L, A>, p: Predicate<A>) => Option<A>;
/** @deprecated */
export declare function find<F extends URIS>(F: Foldable1<F>): <A>(fa: Kind<F, A>, p: Predicate<A>) => Option<A>;
/** @deprecated */
export declare function find<F>(F: Foldable<F>): <A>(fa: HKT<F, A>, p: Predicate<A>) => Option<A>;
/**
 * Find the smallest element of a structure, according to its `Ord` instance
 *
 * @since 1.0.0
 * @deprecated
 */
export declare function minimum<F extends URIS3, A>(F: Foldable3<F>, O: Ord<A>): <U, L>(fa: Kind3<F, U, L, A>) => Option<A>;
/** @deprecated */
export declare function minimum<F extends URIS3, A, U, L>(F: Foldable3C<F, U, L>, O: Ord<A>): (fa: Kind3<F, U, L, A>) => Option<A>;
/** @deprecated */
export declare function minimum<F extends URIS2, A>(F: Foldable2<F>, O: Ord<A>): <L>(fa: Kind2<F, L, A>) => Option<A>;
/** @deprecated */
export declare function minimum<F extends URIS2, A, L>(F: Foldable2C<F, L>, O: Ord<A>): (fa: Kind2<F, L, A>) => Option<A>;
/** @deprecated */
export declare function minimum<F extends URIS, A>(F: Foldable1<F>, O: Ord<A>): (fa: Kind<F, A>) => Option<A>;
/** @deprecated */
export declare function minimum<F, A>(F: Foldable<F>, O: Ord<A>): (fa: HKT<F, A>) => Option<A>;
/**
 * Find the largest element of a structure, according to its `Ord` instance
 *
 * @since 1.0.0
 * @deprecated
 */
export declare function maximum<F extends URIS3, A>(F: Foldable3<F>, O: Ord<A>): <U, L>(fa: Kind3<F, U, L, A>) => Option<A>;
/** @deprecated */
export declare function maximum<F extends URIS3, A, U, L>(F: Foldable3C<F, U, L>, O: Ord<A>): (fa: Kind3<F, U, L, A>) => Option<A>;
/** @deprecated */
export declare function maximum<F extends URIS2, A>(F: Foldable2<F>, O: Ord<A>): <L>(fa: Kind2<F, L, A>) => Option<A>;
/** @deprecated */
export declare function maximum<F extends URIS2, A, L>(F: Foldable2C<F, L>, O: Ord<A>): (fa: Kind2<F, L, A>) => Option<A>;
/** @deprecated */
export declare function maximum<F extends URIS, A>(F: Foldable1<F>, O: Ord<A>): (fa: Kind<F, A>) => Option<A>;
/** @deprecated */
export declare function maximum<F, A>(F: Foldable<F>, O: Ord<A>): (fa: HKT<F, A>) => Option<A>;
/**
 * Use `Foldable2v`'s `toArray`
 * @since 1.0.0
 */
export declare function toArray<F extends URIS3>(F: Foldable3<F>): <U, L, A>(fa: Kind3<F, U, L, A>) => Array<A>;
/**
 * Use `Foldable2v`'s `toArray`
 * @since 1.0.0
 */
export declare function toArray<F extends URIS3, U, L>(F: Foldable3C<F, U, L>): <A>(fa: Kind3<F, U, L, A>) => Array<A>;
/**
 * Use `Foldable2v`'s `toArray`
 * @since 1.0.0
 */
export declare function toArray<F extends URIS2>(F: Foldable2<F>): <L, A>(fa: Kind2<F, L, A>) => Array<A>;
/**
 * Use `Foldable2v`'s `toArray`
 * @since 1.0.0
 */
export declare function toArray<F extends URIS2, L>(F: Foldable2C<F, L>): <A>(fa: Kind2<F, L, A>) => Array<A>;
/**
 * Use `Foldable2v`'s `toArray`
 * @since 1.0.0
 */
export declare function toArray<F extends URIS>(F: Foldable1<F>): <A>(fa: Kind<F, A>) => Array<A>;
/**
 * Use `Foldable2v`'s `toArray`
 * @since 1.0.0
 */
export declare function toArray<F>(F: Foldable<F>): <A>(fa: HKT<F, A>) => Array<A>;
/**
 * Traverse a data structure, performing some effects encoded by an `Applicative` functor at each value, ignoring the
 * final result.
 *
 * @since 1.7.0
 * @deprecated
 */
export declare function traverse<M extends URIS3, F extends URIS>(M: Applicative3<M>, F: Foldable1<F>): <U, L, A, B>(fa: Kind<F, A>, f: (a: A) => Kind3<M, U, L, B>) => Kind3<M, U, L, void>;
/** @deprecated */
export declare function traverse<M extends URIS3, F extends URIS, U, L>(M: Applicative3C<M, U, L>, F: Foldable1<F>): <A, B>(fa: Kind<F, A>, f: (a: A) => Kind3<M, U, L, B>) => Kind3<M, U, L, void>;
/** @deprecated */
export declare function traverse<M extends URIS2, F extends URIS>(M: Applicative2<M>, F: Foldable1<F>): <L, A, B>(fa: Kind<F, A>, f: (a: A) => Kind2<M, L, B>) => Kind2<M, L, void>;
/** @deprecated */
export declare function traverse<M extends URIS2, F extends URIS, L>(M: Applicative2C<M, L>, F: Foldable1<F>): <A, B>(fa: Kind<F, A>, f: (a: A) => Kind2<M, L, B>) => Kind2<M, L, void>;
/** @deprecated */
export declare function traverse<M extends URIS, F extends URIS>(M: Applicative1<M>, F: Foldable1<F>): <A, B>(fa: Kind<F, A>, f: (a: A) => Kind<M, B>) => Kind<M, void>;
/** @deprecated */
export declare function traverse<M, F>(M: Applicative<M>, F: Foldable<F>): <A, B>(fa: HKT<F, A>, f: (a: A) => HKT<M, B>) => HKT<M, void>;
