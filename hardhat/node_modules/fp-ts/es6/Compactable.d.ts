/**
 * @file `Compactable` represents data structures which can be _compacted_/_filtered_. This is a generalization of
 * `catOptions` as a new function `compact`. `compact` has relations with `Functor`, `Applicative`,
 * `Monad`, `Plus`, and `Traversable` in that we can use these classes to provide the ability to
 * operate on a data type by eliminating intermediate `None`s. This is useful for representing the filtering out of
 * values, or failure.
 *
 * Adapted from https://github.com/LiamGoodacre/purescript-filterable/blob/master/src/Data/Compactable.purs
 */
import { Either } from './Either';
import { Functor, Functor1, Functor2, Functor2C, Functor3C, FunctorComposition, FunctorComposition11, FunctorComposition12, FunctorComposition12C, FunctorComposition21, FunctorComposition22, FunctorComposition22C, FunctorComposition2C1, FunctorComposition3C1 } from './Functor';
import { HKT, Kind, Kind2, Kind3, URIS, URIS2, URIS3, URIS4, Kind4 } from './HKT';
import { Option } from './Option';
/**
 * A `Separated` type which holds `left` and `right` parts.
 *
 * @since 1.7.0
 */
export interface Separated<A, B> {
    readonly left: A;
    readonly right: B;
}
/**
 * @since 1.7.0
 */
export interface Compactable<F> {
    readonly URI: F;
    /**
     * Compacts a data structure unwrapping inner Option
     */
    readonly compact: <A>(fa: HKT<F, Option<A>>) => HKT<F, A>;
    /**
     * Separates a data structure moving inner Left to the left side and inner Right to the right side of Separated
     */
    readonly separate: <A, B>(fa: HKT<F, Either<A, B>>) => Separated<HKT<F, A>, HKT<F, B>>;
}
export interface Compactable1<F extends URIS> {
    readonly URI: F;
    readonly compact: <A>(fa: Kind<F, Option<A>>) => Kind<F, A>;
    readonly separate: <A, B>(fa: Kind<F, Either<A, B>>) => Separated<Kind<F, A>, Kind<F, B>>;
}
export interface Compactable2<F extends URIS2> {
    readonly URI: F;
    readonly compact: <L, A>(fa: Kind2<F, L, Option<A>>) => Kind2<F, L, A>;
    readonly separate: <L, A, B>(fa: Kind2<F, L, Either<A, B>>) => Separated<Kind2<F, L, A>, Kind2<F, L, B>>;
}
export interface Compactable2C<F extends URIS2, L> {
    readonly URI: F;
    readonly _L: L;
    readonly compact: <A>(fa: Kind2<F, L, Option<A>>) => Kind2<F, L, A>;
    readonly separate: <A, B>(fa: Kind2<F, L, Either<A, B>>) => Separated<Kind2<F, L, A>, Kind2<F, L, B>>;
}
export interface Compactable3<F extends URIS3> {
    readonly URI: F;
    readonly compact: <U, L, A>(fa: Kind3<F, U, L, Option<A>>) => Kind3<F, U, L, A>;
    readonly separate: <U, L, A, B>(fa: Kind3<F, U, L, Either<A, B>>) => Separated<Kind3<F, U, L, A>, Kind3<F, U, L, B>>;
}
export interface Compactable3C<F extends URIS3, U, L> {
    readonly URI: F;
    readonly _L: L;
    readonly _U: U;
    readonly compact: <A>(fa: Kind3<F, U, L, Option<A>>) => Kind3<F, U, L, A>;
    readonly separate: <A, B>(fa: Kind3<F, U, L, Either<A, B>>) => Separated<Kind3<F, U, L, A>, Kind3<F, U, L, B>>;
}
export interface Compactable4<F extends URIS4> {
    readonly URI: F;
    readonly compact: <X, U, L, A>(fa: Kind4<F, X, U, L, Option<A>>) => Kind4<F, X, U, L, A>;
    readonly separate: <X, U, L, A, B>(fa: Kind4<F, X, U, L, Either<A, B>>) => Separated<Kind4<F, X, U, L, A>, Kind4<F, X, U, L, B>>;
}
export interface CompactableComposition<F, G> extends FunctorComposition<F, G> {
    readonly compact: <A>(fga: HKT<F, HKT<G, Option<A>>>) => HKT<F, HKT<G, A>>;
    readonly separate: <A, B>(fge: HKT<F, HKT<G, Either<A, B>>>) => Separated<HKT<F, HKT<G, A>>, HKT<F, HKT<G, B>>>;
}
export interface CompactableComposition11<F extends URIS, G extends URIS> extends FunctorComposition11<F, G> {
    readonly compact: <A>(fga: Kind<F, Kind<G, Option<A>>>) => Kind<F, Kind<G, A>>;
    readonly separate: <A, B>(fge: Kind<F, Kind<G, Either<A, B>>>) => Separated<Kind<F, Kind<G, A>>, Kind<F, Kind<G, B>>>;
}
export interface CompactableComposition12<F extends URIS, G extends URIS2> extends FunctorComposition12<F, G> {
    readonly compact: <LG, A>(fga: Kind<F, Kind2<G, LG, Option<A>>>) => Kind<F, Kind2<G, LG, A>>;
    readonly separate: <LG, A, B>(fge: Kind<F, Kind2<G, LG, Either<A, B>>>) => Separated<Kind<F, Kind2<G, LG, A>>, Kind<F, Kind2<G, LG, B>>>;
}
export interface CompactableComposition12C<F extends URIS, G extends URIS2, LG> extends FunctorComposition12C<F, G, LG> {
    readonly compact: <A>(fga: Kind<F, Kind2<G, LG, Option<A>>>) => Kind<F, Kind2<G, LG, A>>;
    readonly separate: <A, B>(fge: Kind<F, Kind2<G, LG, Either<A, B>>>) => Separated<Kind<F, Kind2<G, LG, A>>, Kind<F, Kind2<G, LG, B>>>;
}
export interface CompactableComposition21<F extends URIS2, G extends URIS> extends FunctorComposition21<F, G> {
    readonly compact: <LF, A>(fga: Kind2<F, LF, Kind<G, Option<A>>>) => Kind2<F, LF, Kind<G, A>>;
    readonly separate: <LF, A, B>(fge: Kind2<F, LF, Kind<G, Either<A, B>>>) => Separated<Kind2<F, LF, Kind<G, A>>, Kind2<F, LF, Kind<G, B>>>;
}
export interface CompactableComposition2C1<F extends URIS2, G extends URIS, LF> extends FunctorComposition2C1<F, G, LF> {
    readonly compact: <A>(fga: Kind2<F, LF, Kind<G, Option<A>>>) => Kind2<F, LF, Kind<G, A>>;
    readonly separate: <A, B>(fge: Kind2<F, LF, Kind<G, Either<A, B>>>) => Separated<Kind2<F, LF, Kind<G, A>>, Kind2<F, LF, Kind<G, B>>>;
}
export interface CompactableComposition22<F extends URIS2, G extends URIS2> extends FunctorComposition22<F, G> {
    readonly compact: <LF, LG, A>(fga: Kind2<F, LF, Kind2<G, LG, Option<A>>>) => Kind2<F, LF, Kind2<G, LG, A>>;
    readonly separate: <LF, LG, A, B>(fge: Kind2<F, LF, Kind2<G, LG, Either<A, B>>>) => Separated<Kind2<F, LF, Kind2<G, LG, A>>, Kind2<F, LF, Kind2<G, LG, B>>>;
}
export interface CompactableComposition22C<F extends URIS2, G extends URIS2, LG> extends FunctorComposition22C<F, G, LG> {
    readonly compact: <LF, A>(fga: Kind2<F, LF, Kind2<G, LG, Option<A>>>) => Kind2<F, LF, Kind2<G, LG, A>>;
    readonly separate: <LF, A, B>(fge: Kind2<F, LF, Kind2<G, LG, Either<A, B>>>) => Separated<Kind2<F, LF, Kind2<G, LG, A>>, Kind2<F, LF, Kind2<G, LG, B>>>;
}
export interface CompactableComposition3C1<F extends URIS3, G extends URIS, UF, LF> extends FunctorComposition3C1<F, G, UF, LF> {
    readonly compact: <A>(fga: Kind3<F, UF, LF, Kind<G, Option<A>>>) => Kind3<F, UF, LF, Kind<G, A>>;
    readonly separate: <A, B>(fge: Kind3<F, UF, LF, Kind<G, Either<A, B>>>) => Separated<Kind3<F, UF, LF, Kind<G, A>>, Kind3<F, UF, LF, Kind<G, B>>>;
}
/**
 * @since 1.12.0
 */
export declare function getCompactableComposition<F extends URIS3, G extends URIS, UF, LF>(F: Functor3C<F, UF, LF>, G: Compactable1<G> & Functor1<G>): CompactableComposition3C1<F, G, UF, LF>;
export declare function getCompactableComposition<F extends URIS2, G extends URIS2, LG>(F: Functor2<F>, G: Compactable2C<G, LG> & Functor2C<G, LG>): CompactableComposition22C<F, G, LG>;
export declare function getCompactableComposition<F extends URIS2, G extends URIS2>(F: Functor2<F>, G: Compactable2<G> & Functor2<G>): CompactableComposition22<F, G>;
export declare function getCompactableComposition<F extends URIS2, G extends URIS, LF>(F: Functor2C<F, LF>, G: Compactable1<G> & Functor1<G>): CompactableComposition2C1<F, G, LF>;
export declare function getCompactableComposition<F extends URIS2, G extends URIS>(F: Functor2<F>, G: Compactable1<G> & Functor1<G>): CompactableComposition21<F, G>;
export declare function getCompactableComposition<F extends URIS, G extends URIS2, LG>(F: Functor1<F>, G: Compactable2C<G, LG> & Functor2C<G, LG>): CompactableComposition12<F, G>;
export declare function getCompactableComposition<F extends URIS, G extends URIS2>(F: Functor1<F>, G: Compactable2<G> & Functor2<G>): CompactableComposition12<F, G>;
export declare function getCompactableComposition<F extends URIS, G extends URIS>(F: Functor1<F>, G: Compactable1<G> & Functor1<G>): CompactableComposition11<F, G>;
export declare function getCompactableComposition<F, G>(F: Functor<F>, G: Compactable<G> & Functor<G>): CompactableComposition<F, G>;
