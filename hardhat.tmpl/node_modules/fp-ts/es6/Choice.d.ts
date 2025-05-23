/**
 * @file The `Choice` class extends `Profunctor` with combinators for working with
 * sum types.
 *
 * `left` and `right` lift values in a `Profunctor` to act on the `Left` and
 * `Right` components of a sum, respectively.
 *
 * Looking at `Choice` through the intuition of inputs and outputs
 * yields the following type signature:
 *
 * ```purescript
 * left ::  forall input output a. p input output -> p (Either input a) (Either output a)
 * right :: forall input output a. p input output -> p (Either a input) (Either a output)
 * ```
 *
 * If we specialize the profunctor `p` to the `function` arrow, we get the following type
 * signatures:
 *
 * ```purescript
 * left ::  forall input output a. (input -> output) -> (Either input a) -> (Either output a)
 * right :: forall input output a. (input -> output) -> (Either a input) -> (Either a output)
 * ```
 *
 * When the `profunctor` is `Function` application, `left` allows you to map a function over the
 * left side of an `Either`, and `right` maps it over the right side (same as `map` would do).
 *
 * Adapted from https://github.com/purescript/purescript-profunctor/blob/master/src/Data/Profunctor/Choice.purs
 */
import { Either } from './Either';
import { HKT2, Kind2, Kind3, URIS2, URIS3, URIS4, Kind4 } from './HKT';
import { Profunctor, Profunctor2, Profunctor3, Profunctor4 } from './Profunctor';
import { Category, Category2, Category3 } from './Category';
/**
 * @since 1.11.0
 */
export interface Choice<F> extends Profunctor<F> {
    readonly left: <A, B, C>(pab: HKT2<F, A, B>) => HKT2<F, Either<A, C>, Either<B, C>>;
    readonly right: <A, B, C>(pbc: HKT2<F, B, C>) => HKT2<F, Either<A, B>, Either<A, C>>;
}
export interface Choice2<F extends URIS2> extends Profunctor2<F> {
    readonly left: <A, B, C>(pab: Kind2<F, A, B>) => Kind2<F, Either<A, C>, Either<B, C>>;
    readonly right: <A, B, C>(pbc: Kind2<F, B, C>) => Kind2<F, Either<A, B>, Either<A, C>>;
}
export interface Choice3<F extends URIS3> extends Profunctor3<F> {
    readonly left: <U, A, B, C>(pab: Kind3<F, U, A, B>) => Kind3<F, U, Either<A, C>, Either<B, C>>;
    readonly right: <U, A, B, C>(pbc: Kind3<F, U, B, C>) => Kind3<F, U, Either<A, B>, Either<A, C>>;
}
export interface Choice4<F extends URIS4> extends Profunctor4<F> {
    readonly left: <X, U, A, B, C>(pab: Kind4<F, X, U, A, B>) => Kind4<F, X, U, Either<A, C>, Either<B, C>>;
    readonly right: <X, U, A, B, C>(pbc: Kind4<F, X, U, B, C>) => Kind4<F, X, U, Either<A, B>, Either<A, C>>;
}
/**
 * Compose a value acting on a sum from two values, each acting on one of
 * the components of the sum.
 *
 * Specializing `(+++)` to function application would look like this:
 *
 * ```purescript
 * (+++) :: forall a b c d. (a -> b) -> (c -> d) -> (Either a c) -> (Either b d)
 * ```
 *
 * We take two functions, `f` and `g`, and we transform them into a single function which
 * takes an `Either`and maps `f` over the left side and `g` over the right side.  Just like
 * `bi-map` would do for the `bi-functor` instance of `Either`.
 *
 * @since 1.11.0
 */
export declare function splitChoice<F extends URIS3>(F: Category3<F> & Choice3<F>): <U, A, B, C, D>(pab: Kind3<F, U, A, B>, pcd: Kind3<F, U, C, D>) => Kind3<F, U, Either<A, C>, Either<B, D>>;
export declare function splitChoice<F extends URIS2>(F: Category2<F> & Choice2<F>): <A, B, C, D>(pab: Kind2<F, A, B>, pcd: Kind2<F, C, D>) => Kind2<F, Either<A, C>, Either<B, D>>;
export declare function splitChoice<F>(F: Category<F> & Choice<F>): <A, B, C, D>(pab: HKT2<F, A, B>, pcd: HKT2<F, C, D>) => HKT2<F, Either<A, C>, Either<B, D>>;
/**
 * Compose a value which eliminates a sum from two values, each eliminating
 * one side of the sum.
 *
 * This combinator is useful when assembling values from smaller components,
 * because it provides a way to support two different types of input.
 *
 * Specializing `(|||)` to function application would look like this:
 *
 * ```purescript
 * (|||) :: forall a b c d. (a -> c) -> (b -> c) -> Either a b -> c
 * ```
 *
 * We take two functions, `f` and `g`, which both return the same type `c` and we transform them into a
 * single function which takes an `Either` value with the parameter type of `f` on the left side and
 * the parameter type of `g` on the right side. The function then runs either `f` or `g`, depending on
 * whether the `Either` value is a `Left` or a `Right`.
 * This allows us to bundle two different computations which both have the same result type into one
 * function which will run the approriate computation based on the parameter supplied in the `Either` value.
 *
 * @since 1.11.0
 */
export declare function fanin<F extends URIS3>(F: Category3<F> & Choice3<F>): <U, A, B, C>(pac: Kind3<F, U, A, C>, pbc: Kind3<F, U, B, C>) => Kind3<F, U, Either<A, B>, C>;
export declare function fanin<F extends URIS2>(F: Category2<F> & Choice2<F>): <A, B, C>(pac: Kind2<F, A, C>, pbc: Kind2<F, B, C>) => Kind2<F, Either<A, B>, C>;
export declare function fanin<F>(F: Category<F> & Choice<F>): <A, B, C>(pac: HKT2<F, A, C>, pbc: HKT2<F, B, C>) => HKT2<F, Either<A, B>, C>;
