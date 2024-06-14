import { Separated } from './Compactable';
import { Either } from './Either';
import { Filterable, Filterable1, Filterable2, Filterable2C, Filterable3, Filterable3C, Filterable4 } from './Filterable';
import { FunctorWithIndex, FunctorWithIndex1, FunctorWithIndex2, FunctorWithIndex2C, FunctorWithIndex3, FunctorWithIndex3C, FunctorWithIndex4 } from './FunctorWithIndex';
import { HKT, Kind, Kind2, Kind3, URIS, URIS2, URIS3, URIS4, Kind4 } from './HKT';
import { Option } from './Option';
export declare type RefinementWithIndex<I, A, B extends A> = (i: I, a: A) => a is B;
export declare type PredicateWithIndex<I, A> = (i: I, a: A) => boolean;
interface FilterWithIndex<F, I> {
    <A, B extends A>(fa: HKT<F, A>, refinementWithIndex: RefinementWithIndex<I, A, B>): HKT<F, B>;
    <A>(fa: HKT<F, A>, predicateWithIndex: PredicateWithIndex<I, A>): HKT<F, A>;
}
interface PartitionWithIndex<F, I> {
    <A, B extends A>(fa: HKT<F, A>, refinementWithIndex: RefinementWithIndex<I, A, B>): Separated<HKT<F, A>, HKT<F, B>>;
    <A>(fa: HKT<F, A>, predicateWithIndex: PredicateWithIndex<I, A>): Separated<HKT<F, A>, HKT<F, A>>;
}
/**
 * @since 1.12.0
 */
export interface FilterableWithIndex<F, I> extends FunctorWithIndex<F, I>, Filterable<F> {
    readonly partitionMapWithIndex: <RL, RR, A>(fa: HKT<F, A>, f: (i: I, a: A) => Either<RL, RR>) => Separated<HKT<F, RL>, HKT<F, RR>>;
    readonly partitionWithIndex: PartitionWithIndex<F, I>;
    readonly filterMapWithIndex: <A, B>(fa: HKT<F, A>, f: (i: I, a: A) => Option<B>) => HKT<F, B>;
    readonly filterWithIndex: FilterWithIndex<F, I>;
}
interface FilterWithIndex1<F extends URIS, I> {
    <A, B extends A>(fa: Kind<F, A>, refinementWithIndex: RefinementWithIndex<I, A, B>): Kind<F, B>;
    <A>(fa: Kind<F, A>, predicateWithIndex: PredicateWithIndex<I, A>): Kind<F, A>;
}
interface PartitionWithIndex1<F extends URIS, I> {
    <A, B extends A>(fa: Kind<F, A>, refinementWithIndex: RefinementWithIndex<I, A, B>): Separated<Kind<F, A>, Kind<F, B>>;
    <A>(fa: Kind<F, A>, predicateWithIndex: PredicateWithIndex<I, A>): Separated<Kind<F, A>, Kind<F, A>>;
}
export interface FilterableWithIndex1<F extends URIS, I> extends FunctorWithIndex1<F, I>, Filterable1<F> {
    readonly partitionMapWithIndex: <RL, RR, A>(fa: Kind<F, A>, f: (i: I, a: A) => Either<RL, RR>) => Separated<Kind<F, RL>, Kind<F, RR>>;
    readonly partitionWithIndex: PartitionWithIndex1<F, I>;
    readonly filterMapWithIndex: <A, B>(fa: Kind<F, A>, f: (i: I, a: A) => Option<B>) => Kind<F, B>;
    readonly filterWithIndex: FilterWithIndex1<F, I>;
}
interface FilterWithIndex2<F extends URIS2, I> {
    <L, A, B extends A>(fa: Kind2<F, L, A>, refinementWithIndex: RefinementWithIndex<I, A, B>): Kind2<F, L, B>;
    <L, A>(fa: Kind2<F, L, A>, predicateWithIndex: PredicateWithIndex<I, A>): Kind2<F, L, A>;
}
interface PartitionWithIndex2<F extends URIS2, I> {
    <L, A, B extends A>(fa: Kind2<F, L, A>, refinementWithIndex: RefinementWithIndex<I, A, B>): Separated<Kind2<F, L, A>, Kind2<F, L, B>>;
    <L, A>(fa: Kind2<F, L, A>, predicateWithIndex: PredicateWithIndex<I, A>): Separated<Kind2<F, L, A>, Kind2<F, L, A>>;
}
export interface FilterableWithIndex2<F extends URIS2, I> extends FunctorWithIndex2<F, I>, Filterable2<F> {
    readonly partitionMapWithIndex: <RL, RR, L, A>(fa: Kind2<F, L, A>, f: (i: I, a: A) => Either<RL, RR>) => Separated<Kind2<F, L, RL>, Kind2<F, L, RR>>;
    readonly partitionWithIndex: PartitionWithIndex2<F, I>;
    readonly filterMapWithIndex: <L, A, B>(fa: Kind2<F, L, A>, f: (i: I, a: A) => Option<B>) => Kind2<F, L, B>;
    readonly filterWithIndex: FilterWithIndex2<F, I>;
}
interface FilterWithIndex2C<F extends URIS2, I, L> {
    <A, B extends A>(fa: Kind2<F, L, A>, refinementWithIndex: RefinementWithIndex<I, A, B>): Kind2<F, L, B>;
    <A>(fa: Kind2<F, L, A>, predicateWithIndex: PredicateWithIndex<I, A>): Kind2<F, L, A>;
}
interface PartitionWithIndex2C<F extends URIS2, I, L> {
    <A, B extends A>(fa: Kind2<F, L, A>, refinementWithIndex: RefinementWithIndex<I, A, B>): Separated<Kind2<F, L, A>, Kind2<F, L, B>>;
    <A>(fa: Kind2<F, L, A>, predicateWithIndex: PredicateWithIndex<I, A>): Separated<Kind2<F, L, A>, Kind2<F, L, A>>;
}
export interface FilterableWithIndex2C<F extends URIS2, I, L> extends FunctorWithIndex2C<F, I, L>, Filterable2C<F, L> {
    readonly partitionMapWithIndex: <RL, RR, A>(fa: Kind2<F, L, A>, f: (i: I, a: A) => Either<RL, RR>) => Separated<Kind2<F, L, RL>, Kind2<F, L, RR>>;
    readonly partitionWithIndex: PartitionWithIndex2C<F, I, L>;
    readonly filterMapWithIndex: <A, B>(fa: Kind2<F, L, A>, f: (i: I, a: A) => Option<B>) => Kind2<F, L, B>;
    readonly filterWithIndex: FilterWithIndex2C<F, I, L>;
}
interface FilterWithIndex3<F extends URIS3, I> {
    <U, L, A, B extends A>(fa: Kind3<F, U, L, A>, refinementWithIndex: RefinementWithIndex<I, A, B>): Kind3<F, U, L, B>;
    <U, L, A>(fa: Kind3<F, U, L, A>, predicateWithIndex: PredicateWithIndex<I, A>): Kind3<F, U, L, A>;
}
interface PartitionWithIndex3<F extends URIS3, I> {
    <U, L, A, B extends A>(fa: Kind3<F, U, L, A>, refinementWithIndex: RefinementWithIndex<I, A, B>): Separated<Kind3<F, U, L, A>, Kind3<F, U, L, B>>;
    <U, L, A>(fa: Kind3<F, U, L, A>, predicateWithIndex: PredicateWithIndex<I, A>): Separated<Kind3<F, U, L, A>, Kind3<F, U, L, A>>;
}
export interface FilterableWithIndex3<F extends URIS3, I> extends FunctorWithIndex3<F, I>, Filterable3<F> {
    readonly partitionMapWithIndex: <RL, RR, U, L, A>(fa: Kind3<F, U, L, A>, f: (i: I, a: A) => Either<RL, RR>) => Separated<Kind3<F, U, L, RL>, Kind3<F, U, L, RR>>;
    readonly partitionWithIndex: PartitionWithIndex3<F, I>;
    readonly filterMapWithIndex: <U, L, A, B>(fa: Kind3<F, U, L, A>, f: (i: I, a: A) => Option<B>) => Kind3<F, U, L, B>;
    readonly filterWithIndex: FilterWithIndex3<F, I>;
}
interface FilterWithIndex3C<F extends URIS3, I, U, L> {
    <A, B extends A>(fa: Kind3<F, U, L, A>, refinementWithIndex: RefinementWithIndex<I, A, B>): Kind3<F, U, L, B>;
    <A>(fa: Kind3<F, U, L, A>, predicateWithIndex: PredicateWithIndex<I, A>): Kind3<F, U, L, A>;
}
interface PartitionWithIndex3C<F extends URIS3, I, U, L> {
    <A, B extends A>(fa: Kind3<F, U, L, A>, refinementWithIndex: RefinementWithIndex<I, A, B>): Separated<Kind3<F, U, L, A>, Kind3<F, U, L, B>>;
    <A>(fa: Kind3<F, U, L, A>, predicateWithIndex: PredicateWithIndex<I, A>): Separated<Kind3<F, U, L, A>, Kind3<F, U, L, A>>;
}
export interface FilterableWithIndex3C<F extends URIS3, I, U, L> extends FunctorWithIndex3C<F, I, U, L>, Filterable3C<F, U, L> {
    readonly partitionMapWithIndex: <RL, RR, A>(fa: Kind3<F, U, L, A>, f: (i: I, a: A) => Either<RL, RR>) => Separated<Kind3<F, U, L, RL>, Kind3<F, U, L, RR>>;
    readonly partitionWithIndex: PartitionWithIndex3C<F, I, U, L>;
    readonly filterMapWithIndex: <A, B>(fa: Kind3<F, U, L, A>, f: (i: I, a: A) => Option<B>) => Kind3<F, U, L, B>;
    readonly filterWithIndex: FilterWithIndex3C<F, I, U, L>;
}
export interface FilterWithIndex4<F extends URIS4, I> {
    <X, U, L, A, B extends A>(fa: Kind4<F, X, U, L, A>, refinementWithIndex: RefinementWithIndex<I, A, B>): Kind4<F, X, U, L, B>;
    <X, U, L, A>(fa: Kind4<F, X, U, L, A>, predicateWithIndex: PredicateWithIndex<I, A>): Kind4<F, X, U, L, A>;
}
export interface PartitionWithIndex4<F extends URIS4, I> {
    <X, U, L, A, B extends A>(fa: Kind4<F, X, U, L, A>, refinementWithIndex: RefinementWithIndex<I, A, B>): Separated<Kind4<F, X, U, L, A>, Kind4<F, X, U, L, B>>;
    <X, U, L, A>(fa: Kind4<F, X, U, L, A>, predicateWithIndex: PredicateWithIndex<I, A>): Separated<Kind4<F, X, U, L, A>, Kind4<F, X, U, L, A>>;
}
export interface FilterableWithIndex4<F extends URIS4, I> extends FunctorWithIndex4<F, I>, Filterable4<F> {
    readonly partitionMapWithIndex: <RL, RR, X, U, L, A>(fa: Kind4<F, X, U, L, A>, f: (i: I, a: A) => Either<RL, RR>) => Separated<Kind4<F, X, U, L, RL>, Kind4<F, X, U, L, RR>>;
    readonly partitionWithIndex: PartitionWithIndex4<F, I>;
    readonly filterMapWithIndex: <X, U, L, A, B>(fa: Kind4<F, X, U, L, A>, f: (i: I, a: A) => Option<B>) => Kind4<F, X, U, L, B>;
    readonly filterWithIndex: FilterWithIndex4<F, I>;
}
export {};
