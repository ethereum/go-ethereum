/**
 * @file `Filterable` represents data structures which can be _partitioned_/_filtered_.
 *
 * Adapted from https://github.com/LiamGoodacre/purescript-filterable/blob/master/src/Data/Filterable.purs
 */
import { Compactable, Compactable1, Compactable2, Compactable2C, Compactable3, Compactable3C, CompactableComposition, CompactableComposition11, CompactableComposition12, CompactableComposition12C, CompactableComposition21, CompactableComposition22, CompactableComposition3C1, Separated, Compactable4 } from './Compactable';
import { Either } from './Either';
import { Predicate, Refinement } from './function';
import { Functor, Functor1, Functor2, Functor2C, Functor3, Functor3C, FunctorComposition, FunctorComposition11, FunctorComposition12, FunctorComposition12C, FunctorComposition21, FunctorComposition22, FunctorComposition3C1, Functor4 } from './Functor';
import { HKT, Kind, Kind2, Kind3, URIS, URIS2, URIS3, URIS4, Kind4 } from './HKT';
import { Option } from './Option';
interface Filter<F> {
    <A, B extends A>(fa: HKT<F, A>, refinement: Refinement<A, B>): HKT<F, B>;
    <A>(fa: HKT<F, A>, predicate: Predicate<A>): HKT<F, A>;
}
interface Partition<F> {
    <A, B extends A>(fa: HKT<F, A>, refinement: Refinement<A, B>): Separated<HKT<F, A>, HKT<F, B>>;
    <A>(fa: HKT<F, A>, predicate: Predicate<A>): Separated<HKT<F, A>, HKT<F, A>>;
}
/**
 * @since 1.7.0
 */
export interface Filterable<F> extends Functor<F>, Compactable<F> {
    /**
     * Partition a data structure based on an either predicate.
     */
    readonly partitionMap: <RL, RR, A>(fa: HKT<F, A>, f: (a: A) => Either<RL, RR>) => Separated<HKT<F, RL>, HKT<F, RR>>;
    /**
     * Partition a data structure based on a boolean predicate.
     */
    readonly partition: Partition<F>;
    /**
     * Map over a data structure and filter based on an option predicate.
     */
    readonly filterMap: <A, B>(fa: HKT<F, A>, f: (a: A) => Option<B>) => HKT<F, B>;
    /**
     * Filter a data structure based on a boolean predicate.
     */
    readonly filter: Filter<F>;
}
interface Filter1<F extends URIS> {
    <A, B extends A>(fa: Kind<F, A>, refinement: Refinement<A, B>): Kind<F, B>;
    <A>(fa: Kind<F, A>, predicate: Predicate<A>): Kind<F, A>;
}
interface Partition1<F extends URIS> {
    <A, B extends A>(fa: Kind<F, A>, refinement: Refinement<A, B>): Separated<Kind<F, A>, Kind<F, B>>;
    <A>(fa: Kind<F, A>, predicate: Predicate<A>): Separated<Kind<F, A>, Kind<F, A>>;
}
/**
 * @since 1.7.0
 */
export interface Filterable1<F extends URIS> extends Functor1<F>, Compactable1<F> {
    readonly partitionMap: <RL, RR, A>(fa: Kind<F, A>, f: (a: A) => Either<RL, RR>) => Separated<Kind<F, RL>, Kind<F, RR>>;
    readonly partition: Partition1<F>;
    readonly filterMap: <A, B>(fa: Kind<F, A>, f: (a: A) => Option<B>) => Kind<F, B>;
    readonly filter: Filter1<F>;
}
interface Filter2<F extends URIS2> {
    <L, A, B extends A>(fa: Kind2<F, L, A>, refinement: Refinement<A, B>): Kind2<F, L, B>;
    <L, A>(fa: Kind2<F, L, A>, predicate: Predicate<A>): Kind2<F, L, A>;
}
interface Partition2<F extends URIS2> {
    <L, A, B extends A>(fa: Kind2<F, L, A>, refinement: Refinement<A, B>): Separated<Kind2<F, L, A>, Kind2<F, L, B>>;
    <L, A>(fa: Kind2<F, L, A>, predicate: Predicate<A>): Separated<Kind2<F, L, A>, Kind2<F, L, A>>;
}
/**
 * @since 1.7.0
 */
export interface Filterable2<F extends URIS2> extends Functor2<F>, Compactable2<F> {
    readonly partitionMap: <RL, RR, L, A>(fa: Kind2<F, L, A>, f: (a: A) => Either<RL, RR>) => Separated<Kind2<F, L, RL>, Kind2<F, L, RR>>;
    readonly partition: Partition2<F>;
    readonly filterMap: <L, A, B>(fa: Kind2<F, L, A>, f: (a: A) => Option<B>) => Kind2<F, L, B>;
    readonly filter: Filter2<F>;
}
interface Filter2C<F extends URIS2, L> {
    <A, B extends A>(fa: Kind2<F, L, A>, refinement: Refinement<A, B>): Kind2<F, L, B>;
    <A>(fa: Kind2<F, L, A>, predicate: Predicate<A>): Kind2<F, L, A>;
}
interface Partition2C<F extends URIS2, L> {
    <A, B extends A>(fa: Kind2<F, L, A>, refinement: Refinement<A, B>): Separated<Kind2<F, L, A>, Kind2<F, L, B>>;
    <A>(fa: Kind2<F, L, A>, predicate: Predicate<A>): Separated<Kind2<F, L, A>, Kind2<F, L, A>>;
}
/**
 * @since 1.7.0
 */
export interface Filterable2C<F extends URIS2, L> extends Functor2C<F, L>, Compactable2C<F, L> {
    readonly partitionMap: <RL, RR, A>(fa: Kind2<F, L, A>, f: (a: A) => Either<RL, RR>) => Separated<Kind2<F, L, RL>, Kind2<F, L, RR>>;
    readonly partition: Partition2C<F, L>;
    readonly filterMap: <A, B>(fa: Kind2<F, L, A>, f: (a: A) => Option<B>) => Kind2<F, L, B>;
    readonly filter: Filter2C<F, L>;
}
interface Filter3<F extends URIS3> {
    <U, L, A, B extends A>(fa: Kind3<F, U, L, A>, refinement: Refinement<A, B>): Kind3<F, U, L, B>;
    <U, L, A>(fa: Kind3<F, U, L, A>, predicate: Predicate<A>): Kind3<F, U, L, A>;
}
interface Partition3<F extends URIS3> {
    <U, L, A, B extends A>(fa: Kind3<F, U, L, A>, refinement: Refinement<A, B>): Separated<Kind3<F, U, L, A>, Kind3<F, U, L, B>>;
    <U, L, A>(fa: Kind3<F, U, L, A>, predicate: Predicate<A>): Separated<Kind3<F, U, L, A>, Kind3<F, U, L, A>>;
}
/**
 * @since 1.7.0
 */
export interface Filterable3<F extends URIS3> extends Functor3<F>, Compactable3<F> {
    readonly partitionMap: <RL, RR, U, L, A>(fa: Kind3<F, U, L, A>, f: (a: A) => Either<RL, RR>) => Separated<Kind3<F, U, L, RL>, Kind3<F, U, L, RR>>;
    readonly partition: Partition3<F>;
    readonly filterMap: <U, L, A, B>(fa: Kind3<F, U, L, A>, f: (a: A) => Option<B>) => Kind3<F, U, L, B>;
    readonly filter: Filter3<F>;
}
interface Filter3C<F extends URIS3, U, L> {
    <A, B extends A>(fa: Kind3<F, U, L, A>, refinement: Refinement<A, B>): Kind3<F, U, L, B>;
    <A>(fa: Kind3<F, U, L, A>, predicate: Predicate<A>): Kind3<F, U, L, A>;
}
interface Partition3C<F extends URIS3, U, L> {
    <A, B extends A>(fa: Kind3<F, U, L, A>, refinement: Refinement<A, B>): Separated<Kind3<F, U, L, A>, Kind3<F, U, L, B>>;
    <A>(fa: Kind3<F, U, L, A>, predicate: Predicate<A>): Separated<Kind3<F, U, L, A>, Kind3<F, U, L, A>>;
}
/**
 * @since 1.7.0
 */
export interface Filterable3C<F extends URIS3, U, L> extends Functor3C<F, U, L>, Compactable3C<F, U, L> {
    readonly partitionMap: <RL, RR, A>(fa: Kind3<F, U, L, A>, f: (a: A) => Either<RL, RR>) => Separated<Kind3<F, U, L, RL>, Kind3<F, U, L, RR>>;
    readonly partition: Partition3C<F, U, L>;
    readonly filterMap: <A, B>(fa: Kind3<F, U, L, A>, f: (a: A) => Option<B>) => Kind3<F, U, L, B>;
    readonly filter: Filter3C<F, U, L>;
}
export interface Filter4<F extends URIS4> {
    <X, U, L, A, B extends A>(fa: Kind4<F, X, U, L, A>, refinement: Refinement<A, B>): Kind4<F, X, U, L, B>;
    <X, U, L, A>(fa: Kind4<F, X, U, L, A>, predicate: Predicate<A>): Kind4<F, X, U, L, A>;
}
export interface Partition4<F extends URIS4> {
    <X, U, L, A, B extends A>(fa: Kind4<F, X, U, L, A>, refinement: Refinement<A, B>): Separated<Kind4<F, X, U, L, A>, Kind4<F, X, U, L, B>>;
    <X, U, L, A>(fa: Kind4<F, X, U, L, A>, predicate: Predicate<A>): Separated<Kind4<F, X, U, L, A>, Kind4<F, X, U, L, A>>;
}
export interface Filterable4<F extends URIS4> extends Functor4<F>, Compactable4<F> {
    readonly partitionMap: <RL, RR, X, U, L, A>(fa: Kind4<F, X, U, L, A>, f: (a: A) => Either<RL, RR>) => Separated<Kind4<F, X, U, L, RL>, Kind4<F, X, U, L, RR>>;
    readonly partition: Partition4<F>;
    readonly filterMap: <X, U, L, A, B>(fa: Kind4<F, X, U, L, A>, f: (a: A) => Option<B>) => Kind4<F, X, U, L, B>;
    readonly filter: Filter4<F>;
}
export interface FilterableComposition<F, G> extends FunctorComposition<F, G>, CompactableComposition<F, G> {
    readonly partitionMap: <RL, RR, A>(fa: HKT<F, HKT<G, A>>, f: (a: A) => Either<RL, RR>) => Separated<HKT<F, HKT<G, RL>>, HKT<F, HKT<G, RR>>>;
    readonly partition: <A>(fa: HKT<F, HKT<G, A>>, predicate: Predicate<A>) => Separated<HKT<F, HKT<G, A>>, HKT<F, HKT<G, A>>>;
    readonly filterMap: <A, B>(fa: HKT<F, HKT<G, A>>, f: (a: A) => Option<B>) => HKT<F, HKT<G, B>>;
    readonly filter: <A>(fa: HKT<F, HKT<G, A>>, predicate: Predicate<A>) => HKT<F, HKT<G, A>>;
}
export interface FilterableComposition11<F extends URIS, G extends URIS> extends FunctorComposition11<F, G>, CompactableComposition11<F, G> {
    readonly partitionMap: <RL, RR, A>(fa: Kind<F, Kind<G, A>>, f: (a: A) => Either<RL, RR>) => Separated<Kind<F, Kind<G, RL>>, Kind<F, Kind<G, RR>>>;
    readonly partition: <A>(fa: Kind<F, Kind<G, A>>, predicate: Predicate<A>) => Separated<Kind<F, Kind<G, A>>, Kind<F, Kind<G, A>>>;
    readonly filterMap: <A, B>(fa: Kind<F, Kind<G, A>>, f: (a: A) => Option<B>) => Kind<F, Kind<G, B>>;
    readonly filter: <A>(fa: Kind<F, Kind<G, A>>, predicate: Predicate<A>) => Kind<F, Kind<G, A>>;
}
export interface FilterableComposition12<F extends URIS, G extends URIS2> extends FunctorComposition12<F, G>, CompactableComposition12<F, G> {
    readonly partitionMap: <LG, RL, RR, A>(fa: Kind<F, Kind2<G, LG, A>>, f: (a: A) => Either<RL, RR>) => Separated<Kind<F, Kind2<G, LG, RL>>, Kind<F, Kind2<G, LG, RR>>>;
    readonly partition: <LG, A>(fa: Kind<F, Kind2<G, LG, A>>, predicate: Predicate<A>) => Separated<Kind<F, Kind2<G, LG, A>>, Kind<F, Kind2<G, LG, A>>>;
    readonly filterMap: <LG, A, B>(fa: Kind<F, Kind2<G, LG, A>>, f: (a: A) => Option<B>) => Kind<F, Kind2<G, LG, B>>;
    readonly filter: <LG, A>(fa: Kind<F, Kind2<G, LG, A>>, predicate: Predicate<A>) => Kind<F, Kind2<G, LG, A>>;
}
export interface FilterableComposition12C<F extends URIS, G extends URIS2, LG> extends FunctorComposition12C<F, G, LG>, CompactableComposition12C<F, G, LG> {
    readonly partitionMap: <RL, RR, A>(fa: Kind<F, Kind2<G, LG, A>>, f: (a: A) => Either<RL, RR>) => Separated<Kind<F, Kind2<G, LG, RL>>, Kind<F, Kind2<G, LG, RR>>>;
    readonly partition: <A>(fa: Kind<F, Kind2<G, LG, A>>, predicate: Predicate<A>) => Separated<Kind<F, Kind2<G, LG, A>>, Kind<F, Kind2<G, LG, A>>>;
    readonly filterMap: <A, B>(fa: Kind<F, Kind2<G, LG, A>>, f: (a: A) => Option<B>) => Kind<F, Kind2<G, LG, B>>;
    readonly filter: <A>(fa: Kind<F, Kind2<G, LG, A>>, predicate: Predicate<A>) => Kind<F, Kind2<G, LG, A>>;
}
export interface FilterableComposition21<F extends URIS2, G extends URIS> extends FunctorComposition21<F, G>, CompactableComposition21<F, G> {
    readonly partitionMap: <LF, RL, RR, A>(fa: Kind2<F, LF, Kind<G, A>>, f: (a: A) => Either<RL, RR>) => Separated<Kind2<F, LF, Kind<G, RL>>, Kind2<F, LF, Kind<G, RR>>>;
    readonly partition: <LF, A>(fa: Kind2<F, LF, Kind<G, A>>, predicate: Predicate<A>) => Separated<Kind2<F, LF, Kind<G, A>>, Kind2<F, LF, Kind<G, A>>>;
    readonly filterMap: <LF, A, B>(fa: Kind2<F, LF, Kind<G, A>>, f: (a: A) => Option<B>) => Kind2<F, LF, Kind<G, B>>;
    readonly filter: <LF, A>(fa: Kind2<F, LF, Kind<G, A>>, predicate: Predicate<A>) => Kind2<F, LF, Kind<G, A>>;
}
export interface FilterableComposition2C1<F extends URIS2, G extends URIS, LF> extends FunctorComposition21<F, G>, CompactableComposition21<F, G> {
    readonly partitionMap: <RL, RR, A>(fa: Kind2<F, LF, Kind<G, A>>, f: (a: A) => Either<RL, RR>) => Separated<Kind2<F, LF, Kind<G, RL>>, Kind2<F, LF, Kind<G, RR>>>;
    readonly partition: <A>(fa: Kind2<F, LF, Kind<G, A>>, predicate: Predicate<A>) => Separated<Kind2<F, LF, Kind<G, A>>, Kind2<F, LF, Kind<G, A>>>;
    readonly filterMap: <A, B>(fa: Kind2<F, LF, Kind<G, A>>, f: (a: A) => Option<B>) => Kind2<F, LF, Kind<G, B>>;
    readonly filter: <A>(fa: Kind2<F, LF, Kind<G, A>>, predicate: Predicate<A>) => Kind2<F, LF, Kind<G, A>>;
}
export interface FilterableComposition22<F extends URIS2, G extends URIS2> extends FunctorComposition22<F, G>, CompactableComposition22<F, G> {
    readonly partitionMap: <LF, LG, RL, RR, A>(fa: Kind2<F, LF, Kind2<G, LG, A>>, f: (a: A) => Either<RL, RR>) => Separated<Kind2<F, LF, Kind2<G, LG, RL>>, Kind2<F, LF, Kind2<G, LG, RR>>>;
    readonly partition: <LF, LG, A>(fa: Kind2<F, LF, Kind2<G, LG, A>>, predicate: Predicate<A>) => Separated<Kind2<F, LF, Kind2<G, LG, A>>, Kind2<F, LF, Kind2<G, LG, A>>>;
    readonly filterMap: <LF, LG, A, B>(fa: Kind2<F, LF, Kind2<G, LG, A>>, f: (a: A) => Option<B>) => Kind2<F, LF, Kind2<G, LG, B>>;
    readonly filter: <LF, LG, A>(fa: Kind2<F, LF, Kind2<G, LG, A>>, predicate: Predicate<A>) => Kind2<F, LF, Kind2<G, LG, A>>;
}
export interface FilterableComposition22C<F extends URIS2, G extends URIS2, LG> extends FunctorComposition22<F, G>, CompactableComposition22<F, G> {
    readonly partitionMap: <LF, RL, RR, A>(fa: Kind2<F, LF, Kind2<G, LG, A>>, f: (a: A) => Either<RL, RR>) => Separated<Kind2<F, LF, Kind2<G, LG, RL>>, Kind2<F, LF, Kind2<G, LG, RR>>>;
    readonly partition: <LF, A>(fa: Kind2<F, LF, Kind2<G, LG, A>>, predicate: Predicate<A>) => Separated<Kind2<F, LF, Kind2<G, LG, A>>, Kind2<F, LF, Kind2<G, LG, A>>>;
    readonly filterMap: <LF, A, B>(fa: Kind2<F, LF, Kind2<G, LG, A>>, f: (a: A) => Option<B>) => Kind2<F, LF, Kind2<G, LG, B>>;
    readonly filter: <LF, A>(fa: Kind2<F, LF, Kind2<G, LG, A>>, predicate: Predicate<A>) => Kind2<F, LF, Kind2<G, LG, A>>;
}
export interface FilterableComposition3C1<F extends URIS3, G extends URIS, UF, LF> extends FunctorComposition3C1<F, G, UF, LF>, CompactableComposition3C1<F, G, UF, LF> {
    readonly partitionMap: <RL, RR, A>(fa: Kind3<F, UF, LF, Kind<G, A>>, f: (a: A) => Either<RL, RR>) => Separated<Kind3<F, UF, LF, Kind<G, RL>>, Kind3<F, UF, LF, Kind<G, RR>>>;
    readonly partition: <A>(fa: Kind3<F, UF, LF, Kind<G, A>>, predicate: Predicate<A>) => Separated<Kind3<F, UF, LF, Kind<G, A>>, Kind3<F, UF, LF, Kind<G, A>>>;
    readonly filterMap: <A, B>(fa: Kind3<F, UF, LF, Kind<G, A>>, f: (a: A) => Option<B>) => Kind3<F, UF, LF, Kind<G, B>>;
    readonly filter: <A>(fa: Kind3<F, UF, LF, Kind<G, A>>, predicate: Predicate<A>) => Kind3<F, UF, LF, Kind<G, A>>;
}
/**
 * @since 1.12.0
 */
export declare function getFilterableComposition<F extends URIS3, G extends URIS, UF, LF>(F: Functor3C<F, UF, LF>, G: Filterable1<G>): FilterableComposition3C1<F, G, UF, LF>;
export declare function getFilterableComposition<F extends URIS2, G extends URIS2, LG>(F: Functor2<F>, G: Filterable2C<G, LG>): FilterableComposition22C<F, G, LG>;
export declare function getFilterableComposition<F extends URIS2, G extends URIS2>(F: Functor2<F>, G: Filterable2<G>): FilterableComposition22<F, G>;
export declare function getFilterableComposition<F extends URIS2, G extends URIS, LF>(F: Functor2C<F, LF>, G: Filterable1<G>): FilterableComposition2C1<F, G, LF>;
export declare function getFilterableComposition<F extends URIS2, G extends URIS>(F: Functor2<F>, G: Filterable1<G>): FilterableComposition21<F, G>;
export declare function getFilterableComposition<F extends URIS, G extends URIS2, LG>(F: Functor1<F>, G: Filterable2C<G, LG>): FilterableComposition12C<F, G, LG>;
export declare function getFilterableComposition<F extends URIS, G extends URIS2>(F: Functor1<F>, G: Filterable2<G>): FilterableComposition12<F, G>;
export declare function getFilterableComposition<F extends URIS, G extends URIS>(F: Functor1<F>, G: Filterable1<G>): FilterableComposition11<F, G>;
export declare function getFilterableComposition<F, G>(F: Functor<F>, G: Filterable<G>): FilterableComposition<F, G>;
export {};
