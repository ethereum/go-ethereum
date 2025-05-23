import { Applicative, Applicative1, Applicative2, Applicative2C, Applicative3, Applicative3C } from './Applicative';
import { Foldable, Foldable1, Foldable2, Foldable2C, Foldable3, Foldable3C, FoldableComposition, FoldableComposition11 } from './Foldable';
import { Functor, Functor1, Functor2, Functor2C, Functor3, Functor3C, FunctorComposition, FunctorComposition11 } from './Functor';
import { HKT, Kind, Kind2, Kind3, URIS, URIS2, URIS3 } from './HKT';
/**
 * Use `Traversable2v` instead
 * @deprecated
 */
export interface Traversable<T> extends Functor<T>, Foldable<T> {
    /**
     * Runs an action for every element in a data structure and accumulates the results
     */
    readonly traverse: Traverse<T>;
}
/**
 * Use `Traversable2v` instead
 * @deprecated
 */
export interface Traversable1<T extends URIS> extends Functor1<T>, Foldable1<T> {
    readonly traverse: Traverse1<T>;
}
/**
 * Use `Traversable2v` instead
 * @deprecated
 */
export interface Traversable2<T extends URIS2> extends Functor2<T>, Foldable2<T> {
    readonly traverse: Traverse2<T>;
}
/**
 * Use `Traversable2v` instead
 * @deprecated
 */
export interface Traversable2C<T extends URIS2, TL> extends Functor2C<T, TL>, Foldable2C<T, TL> {
    readonly traverse: Traverse2C<T, TL>;
}
/**
 * Use `Traversable2v` instead
 * @deprecated
 */
export interface Traversable3<T extends URIS3> extends Functor3<T>, Foldable3<T> {
    readonly traverse: Traverse3<T>;
}
/**
 * Use `Traversable2v` instead
 * @deprecated
 */
export interface Traversable3C<T extends URIS3, TU, TL> extends Functor3C<T, TU, TL>, Foldable3C<T, TU, TL> {
    readonly traverse: Traverse3C<T, TU, TL>;
}
/**
 * @since 1.7.0
 */
export interface Traverse<T> {
    <F extends URIS3>(F: Applicative3<F>): <FU, FL, A, B>(ta: HKT<T, A>, f: (a: A) => Kind3<F, FU, FL, B>) => Kind3<F, FU, FL, HKT<T, B>>;
    <F extends URIS3, FU, FL>(F: Applicative3C<F, FU, FL>): <A, B>(ta: HKT<T, A>, f: (a: A) => Kind3<F, FU, FL, B>) => Kind3<F, FU, FL, HKT<T, B>>;
    <F extends URIS2>(F: Applicative2<F>): <FL, A, B>(ta: HKT<T, A>, f: (a: A) => Kind2<F, FL, B>) => Kind2<F, FL, HKT<T, B>>;
    <F extends URIS2, FL>(F: Applicative2C<F, FL>): <A, B>(ta: HKT<T, A>, f: (a: A) => Kind2<F, FL, B>) => Kind2<F, FL, HKT<T, B>>;
    <F extends URIS>(F: Applicative1<F>): <A, B>(ta: HKT<T, A>, f: (a: A) => Kind<F, B>) => Kind<F, HKT<T, B>>;
    <F>(F: Applicative<F>): <A, B>(ta: HKT<T, A>, f: (a: A) => HKT<F, B>) => HKT<F, HKT<T, B>>;
}
/**
 * @since 1.7.0
 */
export interface Traverse1<T extends URIS> {
    <F extends URIS3>(F: Applicative3<F>): <FU, FL, A, B>(ta: Kind<T, A>, f: (a: A) => Kind3<F, FU, FL, B>) => Kind3<F, FU, FL, Kind<T, B>>;
    <F extends URIS3, FU, FL>(F: Applicative3C<F, FU, FL>): <A, B>(ta: Kind<T, A>, f: (a: A) => Kind3<F, FU, FL, B>) => Kind3<F, FU, FL, Kind<T, B>>;
    <F extends URIS2>(F: Applicative2<F>): <FL, A, B>(ta: Kind<T, A>, f: (a: A) => Kind2<F, FL, B>) => Kind2<F, FL, Kind<T, B>>;
    <F extends URIS2, FL>(F: Applicative2C<F, FL>): <A, B>(ta: Kind<T, A>, f: (a: A) => Kind2<F, FL, B>) => Kind2<F, FL, Kind<T, B>>;
    <F extends URIS>(F: Applicative1<F>): <A, B>(ta: Kind<T, A>, f: (a: A) => Kind<F, B>) => Kind<F, Kind<T, B>>;
    <F>(F: Applicative<F>): <A, B>(ta: Kind<T, A>, f: (a: A) => HKT<F, B>) => HKT<F, Kind<T, B>>;
}
/**
 * @since 1.7.0
 */
export interface Traverse2<T extends URIS2> {
    <F extends URIS3>(F: Applicative3<F>): <TL, FU, FL, A, B>(ta: Kind2<T, TL, A>, f: (a: A) => Kind3<F, FU, FL, B>) => Kind3<F, FU, FL, Kind2<T, TL, B>>;
    <F extends URIS3, FU, FL>(F: Applicative3C<F, FU, FL>): <TL, A, B>(ta: Kind2<T, TL, A>, f: (a: A) => Kind3<F, FU, FL, B>) => Kind3<F, FU, FL, Kind2<T, TL, B>>;
    <F extends URIS2>(F: Applicative2<F>): <TL, FL, A, B>(ta: Kind2<T, TL, A>, f: (a: A) => Kind2<F, FL, B>) => Kind2<F, FL, Kind2<T, TL, B>>;
    <F extends URIS2, FL>(F: Applicative2C<F, FL>): <TL, A, B>(ta: Kind2<T, TL, A>, f: (a: A) => Kind2<F, FL, B>) => Kind2<F, FL, Kind2<T, TL, B>>;
    <F extends URIS>(F: Applicative1<F>): <TL, A, B>(ta: Kind2<T, TL, A>, f: (a: A) => Kind<F, B>) => Kind<F, Kind2<T, TL, B>>;
    <F>(F: Applicative<F>): <TL, A, B>(ta: Kind2<T, TL, A>, f: (a: A) => HKT<F, B>) => HKT<F, Kind2<T, TL, B>>;
}
/**
 * @since 1.7.0
 */
export interface Traverse2C<T extends URIS2, TL> {
    <F extends URIS3>(F: Applicative3<F>): <FU, FL, A, B>(ta: Kind2<T, TL, A>, f: (a: A) => Kind3<F, FU, FL, B>) => Kind3<F, FU, FL, Kind2<T, TL, B>>;
    <F extends URIS3, FU, FL>(F: Applicative3C<F, FU, FL>): <A, B>(ta: Kind2<T, TL, A>, f: (a: A) => Kind3<F, FU, FL, B>) => Kind3<F, FU, FL, Kind2<T, TL, B>>;
    <F extends URIS2>(F: Applicative2<F>): <FL, A, B>(ta: Kind2<T, TL, A>, f: (a: A) => Kind2<F, FL, B>) => Kind2<F, FL, Kind2<T, TL, B>>;
    <F extends URIS2, FL>(F: Applicative2C<F, FL>): <A, B>(ta: Kind2<T, TL, A>, f: (a: A) => Kind2<F, FL, B>) => Kind2<F, FL, Kind2<T, TL, B>>;
    <F extends URIS>(F: Applicative1<F>): <A, B>(ta: Kind2<T, TL, A>, f: (a: A) => Kind<F, B>) => Kind<F, Kind2<T, TL, B>>;
    <F>(F: Applicative<F>): <A, B>(ta: Kind2<T, TL, A>, f: (a: A) => HKT<F, B>) => HKT<F, Kind2<T, TL, B>>;
}
/**
 * @since 1.7.0
 */
export interface Traverse3<T extends URIS3> {
    <F extends URIS3>(F: Applicative3<F>): <TU, TL, FU, FL, A, B>(ta: Kind3<T, TU, TL, A>, f: (a: A) => Kind3<F, FU, FL, B>) => Kind3<F, FU, FL, Kind3<T, TU, TL, B>>;
    <F extends URIS3, FU, FL>(F: Applicative3C<F, FU, FL>): <TU, TL, A, B>(ta: Kind3<T, TU, TL, A>, f: (a: A) => Kind3<F, FU, FL, B>) => Kind3<F, FU, FL, Kind3<T, TU, TL, B>>;
    <F extends URIS2>(F: Applicative2<F>): <TU, TL, FL, A, B>(ta: Kind3<T, TU, TL, A>, f: (a: A) => Kind2<F, FL, B>) => Kind2<F, FL, Kind3<T, TU, TL, B>>;
    <F extends URIS2, FL>(F: Applicative2C<F, FL>): <TU, TL, A, B>(ta: Kind3<T, TU, TL, A>, f: (a: A) => Kind2<F, FL, B>) => Kind2<F, FL, Kind3<T, TU, TL, B>>;
    <F extends URIS>(F: Applicative1<F>): <TU, TL, A, B>(ta: Kind3<T, TU, TL, A>, f: (a: A) => Kind<F, B>) => Kind<F, Kind3<T, TU, TL, B>>;
    <F>(F: Applicative<F>): <TU, TL, A, B>(ta: Kind3<T, TU, TL, A>, f: (a: A) => HKT<F, B>) => HKT<F, Kind3<T, TU, TL, B>>;
}
/**
 * @since 1.7.0
 */
export interface Traverse3C<T extends URIS3, TU, TL> {
    <F extends URIS3>(F: Applicative3<F>): <FU, FL, A, B>(ta: Kind3<T, TU, TL, A>, f: (a: A) => Kind3<F, FU, FL, B>) => Kind3<F, FU, FL, Kind3<T, TU, TL, B>>;
    <F extends URIS3, FU, FL>(F: Applicative3C<F, FU, FL>): <A, B>(ta: Kind3<T, TU, TL, A>, f: (a: A) => Kind3<F, FU, FL, B>) => Kind3<F, FU, FL, Kind3<T, TU, TL, B>>;
    <F extends URIS2>(F: Applicative2<F>): <FL, A, B>(ta: Kind3<T, TU, TL, A>, f: (a: A) => Kind2<F, FL, B>) => Kind2<F, FL, Kind3<T, TU, TL, B>>;
    <F extends URIS2, FL>(F: Applicative2C<F, FL>): <A, B>(ta: Kind3<T, TU, TL, A>, f: (a: A) => Kind2<F, FL, B>) => Kind2<F, FL, Kind3<T, TU, TL, B>>;
    <F extends URIS>(F: Applicative1<F>): <A, B>(ta: Kind3<T, TU, TL, A>, f: (a: A) => Kind<F, B>) => Kind<F, Kind3<T, TU, TL, B>>;
    <F>(F: Applicative<F>): <A, B>(ta: Kind3<T, TU, TL, A>, f: (a: A) => HKT<F, B>) => HKT<F, Kind3<T, TU, TL, B>>;
}
export interface TraversableComposition<F, G> extends FoldableComposition<F, G>, FunctorComposition<F, G> {
    readonly traverse: <H>(H: Applicative<H>) => <A, B>(fga: HKT<F, HKT<G, A>>, f: (a: A) => HKT<H, B>) => HKT<H, HKT<F, HKT<G, B>>>;
}
export interface TraverseComposition11<F extends URIS, G extends URIS> {
    <H extends URIS3>(H: Applicative3<H>): <HU, HL, A, B>(fga: Kind<F, Kind<G, A>>, f: (a: A) => Kind3<H, HU, HL, B>) => Kind3<H, HU, HL, Kind<F, Kind<G, B>>>;
    <H extends URIS3, HU, HL>(H: Applicative3C<H, HU, HL>): <A, B>(fga: Kind<F, Kind<G, A>>, f: (a: A) => Kind3<H, HU, HL, B>) => Kind3<H, HU, HL, Kind<F, Kind<G, B>>>;
    <H extends URIS2>(H: Applicative2<H>): <HL, A, B>(fga: Kind<F, Kind<G, A>>, f: (a: A) => Kind2<H, HL, B>) => Kind2<H, HL, Kind<F, Kind<G, B>>>;
    <H extends URIS2, HL>(H: Applicative2C<H, HL>): <A, B>(fga: Kind<F, Kind<G, A>>, f: (a: A) => Kind2<H, HL, B>) => Kind2<H, HL, Kind<F, Kind<G, B>>>;
    <H extends URIS>(H: Applicative1<H>): <A, B>(fga: Kind<F, Kind<G, A>>, f: (a: A) => Kind<H, B>) => Kind<H, Kind<F, Kind<G, B>>>;
    <H>(H: Applicative<H>): <A, B>(fga: Kind<F, Kind<G, A>>, f: (a: A) => HKT<H, B>) => HKT<H, Kind<F, Kind<G, B>>>;
}
export interface TraversableComposition11<F extends URIS, G extends URIS>// tslint:disable-next-line: deprecation
 extends FoldableComposition11<F, G>, FunctorComposition11<F, G> {
    readonly traverse: TraverseComposition11<F, G>;
}
/**
 * Use `traverse` contained in each traversable data structure instead.
 *
 *
 * @example
 * import { array } from 'fp-ts/lib/Array'
 * import { none, option, some } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(array.traverse(option)([1, 2, 3], n => (n >= 0 ? some(n) : none)), some([1, 2, 3]))
 * assert.deepStrictEqual(array.traverse(option)([-1, 2, 3], n => (n >= 0 ? some(n) : none)), none)
 *
 * @since 1.0.0
 * @deprecated
 */
export declare function traverse<F extends URIS3, T extends URIS2>(F: Applicative3<F>, T: Traversable2<T>): <UF, LF, LT, A, B>(ta: Kind2<T, LT, A>, f: (a: A) => Kind3<F, UF, LF, B>) => Kind3<F, UF, LF, Kind2<T, LT, B>>;
export declare function traverse<F extends URIS2, T extends URIS2>(F: Applicative2<F>, T: Traversable2<T>): <LF, LT, A, B>(ta: Kind2<T, LT, A>, f: (a: A) => Kind2<F, LF, B>) => Kind2<F, LF, Kind2<T, LT, B>>;
export declare function traverse<F extends URIS2, T extends URIS2, LF>(F: Applicative2C<F, LF>, T: Traversable2<T>): <LT, A, B>(ta: Kind2<T, LT, A>, f: (a: A) => Kind2<F, LF, B>) => Kind2<F, LF, Kind2<T, LT, B>>;
export declare function traverse<F extends URIS, T extends URIS2>(F: Applicative1<F>, T: Traversable2<T>): <LT, A, B>(ta: Kind2<T, LT, A>, f: (a: A) => Kind<F, B>) => Kind<F, Kind2<T, LT, B>>;
export declare function traverse<F extends URIS3, T extends URIS>(F: Applicative3<F>, T: Traversable1<T>): <U, L, A, B>(ta: Kind<T, A>, f: (a: A) => Kind3<F, U, L, B>) => Kind3<F, U, L, Kind<T, B>>;
export declare function traverse<F extends URIS2, T extends URIS>(F: Applicative2<F>, T: Traversable1<T>): <L, A, B>(ta: Kind<T, A>, f: (a: A) => Kind2<F, L, B>) => Kind2<F, L, Kind<T, B>>;
export declare function traverse<F extends URIS2, T extends URIS, L>(F: Applicative2C<F, L>, T: Traversable1<T>): <A, B>(ta: Kind<T, A>, f: (a: A) => Kind2<F, L, B>) => Kind2<F, L, Kind<T, B>>;
export declare function traverse<F extends URIS, T extends URIS>(F: Applicative1<F>, T: Traversable1<T>): <A, B>(ta: Kind<T, A>, f: (a: A) => Kind<F, B>) => Kind<F, Kind<T, B>>;
export declare function traverse<F, T>(F: Applicative<F>, T: Traversable<T>): <A, B>(ta: HKT<T, A>, f: (a: A) => HKT<F, B>) => HKT<F, HKT<T, B>>;
/**
 * Use `sequence` contained in each traversable data structure instead.
 *
 * @example
 * import { array } from 'fp-ts/lib/Array'
 * import { none, option, some } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(array.sequence(option)([some(1), some(2), some(3)]), some([1, 2, 3]))
 * assert.deepStrictEqual(array.sequence(option)([none, some(2), some(3)]), none)
 *
 * @since 1.0.0
 * @deprecated
 */
export declare function sequence<F extends URIS2, T extends URIS2>(F: Applicative2<F>, T: Traversable2<T>): <LF, LT, A>(tfa: Kind2<T, LT, Kind2<F, LF, A>>) => Kind2<F, LF, Kind2<T, LT, A>>;
export declare function sequence<F extends URIS2, T extends URIS2, LF>(F: Applicative2C<F, LF>, T: Traversable2<T>): <LT, A>(tfa: Kind2<T, LT, Kind2<F, LF, A>>) => Kind2<F, LF, Kind2<T, LT, A>>;
export declare function sequence<F extends URIS, T extends URIS2>(F: Applicative1<F>, T: Traversable2<T>): <L, A>(tfa: Kind2<T, L, Kind<F, A>>) => Kind<F, Kind2<T, L, A>>;
export declare function sequence<F extends URIS3, T extends URIS>(F: Applicative3<F>, T: Traversable1<T>): <U, L, A>(tfa: Kind<T, Kind3<F, U, L, A>>) => Kind3<F, U, L, Kind<T, A>>;
export declare function sequence<F extends URIS3, T extends URIS, U, L>(F: Applicative3C<F, U, L>, T: Traversable1<T>): <A>(tfa: Kind<T, Kind3<F, U, L, A>>) => Kind3<F, U, L, Kind<T, A>>;
export declare function sequence<F extends URIS2, T extends URIS>(F: Applicative2<F>, T: Traversable1<T>): <L, A>(tfa: Kind<T, Kind2<F, L, A>>) => Kind2<F, L, Kind<T, A>>;
export declare function sequence<F extends URIS2, T extends URIS, L>(F: Applicative2C<F, L>, T: Traversable1<T>): <A>(tfa: Kind<T, Kind2<F, L, A>>) => Kind2<F, L, Kind<T, A>>;
export declare function sequence<F extends URIS, T extends URIS>(F: Applicative1<F>, T: Traversable1<T>): <A>(tfa: Kind<T, Kind<F, A>>) => Kind<F, Kind<T, A>>;
export declare function sequence<F, T extends URIS>(F: Applicative<F>, T: Traversable1<T>): <A>(tfa: Kind<T, HKT<F, A>>) => HKT<F, Kind<T, A>>;
export declare function sequence<F, T>(F: Applicative<F>, T: Traversable<T>): <A>(tfa: HKT<T, HKT<F, A>>) => HKT<F, HKT<T, A>>;
/**
 * Use `Traversable2v`'s `getTraversableComposition` instead.
 *
 * @since 1.0.0
 * @deprecated
 */
export declare function getTraversableComposition<F extends URIS, G extends URIS>(F: Traversable1<F>, G: Traversable1<G>): TraversableComposition11<F, G>;
export declare function getTraversableComposition<F, G>(F: Traversable<F>, G: Traversable<G>): TraversableComposition<F, G>;
