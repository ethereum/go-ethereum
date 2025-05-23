/**
 * @file The `Applicative` type class extends the `Apply` type class with a `of` function, which can be used to create values
 * of type `f a` from values of type `a`.
 *
 * Where `Apply` provides the ability to lift functions of two or more arguments to functions whose arguments are
 * wrapped using `f`, and `Functor` provides the ability to lift functions of one argument, `pure` can be seen as the
 * function which lifts functions of _zero_ arguments. That is, `Applicative` functors support a lifting operation for
 * any number of function arguments.
 *
 * Instances must satisfy the following laws in addition to the `Apply` laws:
 *
 * 1. Identity: `A.ap(A.of(a => a), fa) = fa`
 * 2. Homomorphism: `A.ap(A.of(ab), A.of(a)) = A.of(ab(a))`
 * 3. Interchange: `A.ap(fab, A.of(a)) = A.ap(A.of(ab => ab(a)), fab)`
 *
 * Note. `Functor`'s `map` can be derived: `A.map(x, f) = A.ap(A.of(f), x)`
 */
import { Apply, Apply1, Apply2, Apply2C, Apply3, Apply3C, Apply4 } from './Apply';
import { FunctorComposition, FunctorComposition11, FunctorComposition12, FunctorComposition12C, FunctorComposition21, FunctorComposition22, FunctorComposition22C, FunctorComposition2C1, FunctorComposition3C1 } from './Functor';
import { HKT, Kind, Kind2, Kind3, URIS, URIS2, URIS3, URIS4, Kind4 } from './HKT';
import { Monoid } from './Monoid';
/**
 * @since 1.0.0
 */
export interface Applicative<F> extends Apply<F> {
    readonly of: <A>(a: A) => HKT<F, A>;
}
export interface Applicative1<F extends URIS> extends Apply1<F> {
    readonly of: <A>(a: A) => Kind<F, A>;
}
export interface Applicative2<F extends URIS2> extends Apply2<F> {
    readonly of: <L, A>(a: A) => Kind2<F, L, A>;
}
export interface Applicative3<F extends URIS3> extends Apply3<F> {
    readonly of: <U, L, A>(a: A) => Kind3<F, U, L, A>;
}
export interface Applicative2C<F extends URIS2, L> extends Apply2C<F, L> {
    readonly of: <A>(a: A) => Kind2<F, L, A>;
}
export interface Applicative3C<F extends URIS3, U, L> extends Apply3C<F, U, L> {
    readonly of: <A>(a: A) => Kind3<F, U, L, A>;
}
export interface Applicative4<F extends URIS4> extends Apply4<F> {
    readonly of: <X, U, L, A>(a: A) => Kind4<F, X, U, L, A>;
}
export interface ApplicativeComposition<F, G> extends FunctorComposition<F, G> {
    readonly of: <A>(a: A) => HKT<F, HKT<G, A>>;
    readonly ap: <A, B>(fgab: HKT<F, HKT<G, (a: A) => B>>, fga: HKT<F, HKT<G, A>>) => HKT<F, HKT<G, B>>;
}
export interface ApplicativeComposition11<F extends URIS, G extends URIS> extends FunctorComposition11<F, G> {
    readonly of: <A>(a: A) => Kind<F, Kind<G, A>>;
    readonly ap: <A, B>(fgab: Kind<F, Kind<G, (a: A) => B>>, fga: Kind<F, Kind<G, A>>) => Kind<F, Kind<G, B>>;
}
export interface ApplicativeComposition12<F extends URIS, G extends URIS2> extends FunctorComposition12<F, G> {
    readonly of: <LG, A>(a: A) => Kind<F, Kind2<G, LG, A>>;
    readonly ap: <LG, A, B>(fgab: Kind<F, Kind2<G, LG, (a: A) => B>>, fga: Kind<F, Kind2<G, LG, A>>) => Kind<F, Kind2<G, LG, B>>;
}
export interface ApplicativeComposition12C<F extends URIS, G extends URIS2, LG> extends FunctorComposition12C<F, G, LG> {
    readonly of: <A>(a: A) => Kind<F, Kind2<G, LG, A>>;
    readonly ap: <A, B>(fgab: Kind<F, Kind2<G, LG, (a: A) => B>>, fga: Kind<F, Kind2<G, LG, A>>) => Kind<F, Kind2<G, LG, B>>;
}
export interface ApplicativeComposition21<F extends URIS2, G extends URIS> extends FunctorComposition21<F, G> {
    readonly of: <LF, A>(a: A) => Kind2<F, LF, Kind<G, A>>;
    readonly ap: <LF, A, B>(fgab: Kind2<F, LF, Kind<G, (a: A) => B>>, fga: Kind2<F, LF, Kind<G, A>>) => Kind2<F, LF, Kind<G, B>>;
}
export interface ApplicativeComposition2C1<F extends URIS2, G extends URIS, LF> extends FunctorComposition2C1<F, G, LF> {
    readonly of: <A>(a: A) => Kind2<F, LF, Kind<G, A>>;
    readonly ap: <A, B>(fgab: Kind2<F, LF, Kind<G, (a: A) => B>>, fga: Kind2<F, LF, Kind<G, A>>) => Kind2<F, LF, Kind<G, B>>;
}
export interface ApplicativeComposition22<F extends URIS2, G extends URIS2> extends FunctorComposition22<F, G> {
    readonly of: <LF, LG, A>(a: A) => Kind2<F, LF, Kind2<G, LG, A>>;
    readonly ap: <L, M, A, B>(fgab: Kind2<F, L, Kind2<G, M, (a: A) => B>>, fga: Kind2<F, L, Kind2<G, M, A>>) => Kind2<F, L, Kind2<G, M, B>>;
}
export interface ApplicativeComposition22C<F extends URIS2, G extends URIS2, LG> extends FunctorComposition22C<F, G, LG> {
    readonly of: <LF, A>(a: A) => Kind2<F, LF, Kind2<G, LG, A>>;
    readonly ap: <LF, A, B>(fgab: Kind2<F, LF, Kind2<G, LG, (a: A) => B>>, fga: Kind2<F, LF, Kind2<G, LG, A>>) => Kind2<F, LF, Kind2<G, LG, B>>;
}
export interface ApplicativeComposition3C1<F extends URIS3, G extends URIS, UF, LF> extends FunctorComposition3C1<F, G, UF, LF> {
    readonly of: <A>(a: A) => Kind3<F, UF, LF, Kind<G, A>>;
    readonly ap: <A, B>(fgab: Kind3<F, UF, LF, Kind<G, (a: A) => B>>, fga: Kind3<F, UF, LF, Kind<G, A>>) => Kind3<F, UF, LF, Kind<G, B>>;
}
/**
 * Perform a applicative action when a condition is true
 *
 * @example
 * import { IO, io } from 'fp-ts/lib/IO'
 * import { when } from 'fp-ts/lib/Applicative'
 *
 * const log: Array<string> = []
 * const action = new IO(() => {
 *   log.push('action called')
 * })
 * when(io)(false, action).run()
 * assert.deepStrictEqual(log, [])
 * when(io)(true, action).run()
 * assert.deepStrictEqual(log, ['action called'])
 *
 * @since 1.0.0
 * @deprecated
 */
export declare function when<F extends URIS3>(F: Applicative3<F>): <U, L>(condition: boolean, fu: Kind3<F, U, L, void>) => Kind3<F, U, L, void>;
/** @deprecated */
export declare function when<F extends URIS3, U, L>(F: Applicative3C<F, U, L>): (condition: boolean, fu: Kind3<F, U, L, void>) => Kind3<F, U, L, void>;
/** @deprecated */
export declare function when<F extends URIS2>(F: Applicative2<F>): <L>(condition: boolean, fu: Kind2<F, L, void>) => Kind2<F, L, void>;
/** @deprecated */
export declare function when<F extends URIS2, L>(F: Applicative2C<F, L>): (condition: boolean, fu: Kind2<F, L, void>) => Kind2<F, L, void>;
/** @deprecated */
export declare function when<F extends URIS>(F: Applicative1<F>): (condition: boolean, fu: Kind<F, void>) => Kind<F, void>;
/** @deprecated */
export declare function when<F>(F: Applicative<F>): (condition: boolean, fu: HKT<F, void>) => HKT<F, void>;
/**
 * Like `Functor`, `Applicative`s compose. If `F` and `G` have `Applicative` instances, then so does `F<G<_>>`
 *
 * @example
 * import { getApplicativeComposition } from 'fp-ts/lib/Applicative'
 * import { option, Option, some } from 'fp-ts/lib/Option'
 * import { task, Task } from 'fp-ts/lib/Task'
 *
 * const x: Task<Option<number>> = task.of(some(1))
 * const y: Task<Option<number>> = task.of(some(2))
 *
 * const A = getApplicativeComposition(task, option)
 *
 * const sum = (a: number) => (b: number): number => a + b
 * A.ap(A.map(x, sum), y)
 *   .run()
 *   .then(result => assert.deepStrictEqual(result, some(3)))
 *
 * @since 1.0.0
 */
export declare function getApplicativeComposition<F extends URIS3, G extends URIS, UF, LF>(F: Applicative3C<F, UF, LF>, G: Applicative1<G>): ApplicativeComposition3C1<F, G, UF, LF>;
export declare function getApplicativeComposition<F extends URIS2, G extends URIS2, LG>(F: Applicative2<F>, G: Applicative2C<G, LG>): ApplicativeComposition22C<F, G, LG>;
export declare function getApplicativeComposition<F extends URIS2, G extends URIS2>(F: Applicative2<F>, G: Applicative2<G>): ApplicativeComposition22<F, G>;
export declare function getApplicativeComposition<F extends URIS2, G extends URIS2, LG>(F: Applicative2<F>, G: Applicative2C<G, LG>): ApplicativeComposition22C<F, G, LG>;
export declare function getApplicativeComposition<F extends URIS2, G extends URIS>(F: Applicative2<F>, G: Applicative1<G>): ApplicativeComposition21<F, G>;
export declare function getApplicativeComposition<F extends URIS, G extends URIS2>(F: Applicative1<F>, G: Applicative2<G>): ApplicativeComposition12<F, G>;
export declare function getApplicativeComposition<F extends URIS, G extends URIS2, LG>(F: Applicative1<F>, G: Applicative2C<G, LG>): ApplicativeComposition12C<F, G, LG>;
export declare function getApplicativeComposition<F extends URIS, G extends URIS>(F: Applicative1<F>, G: Applicative1<G>): ApplicativeComposition11<F, G>;
export declare function getApplicativeComposition<F, G extends URIS2>(F: Applicative<F>, G: Applicative2<G>): ApplicativeComposition<F, G>;
export declare function getApplicativeComposition<F, G extends URIS>(F: Applicative<F>, G: Applicative1<G>): ApplicativeComposition<F, G>;
export declare function getApplicativeComposition<F, G>(F: Applicative<F>, G: Applicative<G>): ApplicativeComposition<F, G>;
/**
 * If `F` is a `Applicative` and `M` is a `Monoid` over `A` then `HKT<F, A>` is a `Monoid` over `A` as well.
 * Adapted from http://hackage.haskell.org/package/monoids-0.2.0.2/docs/Data-Monoid-Applicative.html
 *
 * @example
 * import { getMonoid } from 'fp-ts/lib/Applicative'
 * import { option, some, none } from 'fp-ts/lib/Option'
 * import { monoidSum } from 'fp-ts/lib/Monoid'
 *
 * const M = getMonoid(option, monoidSum)()
 * assert.deepStrictEqual(M.concat(none, none), none)
 * assert.deepStrictEqual(M.concat(some(1), none), none)
 * assert.deepStrictEqual(M.concat(none, some(2)), none)
 * assert.deepStrictEqual(M.concat(some(1), some(2)), some(3))
 *
 * @since 1.4.0
 * @deprecated
 */
export declare function getMonoid<F extends URIS3, A>(F: Applicative3<F>, M: Monoid<A>): <U = never, L = never>() => Monoid<Kind3<F, U, L, A>>;
/** @deprecated */
export declare function getMonoid<F extends URIS3, U, L, A>(F: Applicative3C<F, U, L>, M: Monoid<A>): () => Monoid<Kind3<F, U, L, A>>;
/** @deprecated */
export declare function getMonoid<F extends URIS2, A>(F: Applicative2<F>, M: Monoid<A>): <L = never>() => Monoid<Kind2<F, L, A>>;
/** @deprecated */
export declare function getMonoid<F extends URIS2, L, A>(F: Applicative2C<F, L>, M: Monoid<A>): () => Monoid<Kind2<F, L, A>>;
/** @deprecated */
export declare function getMonoid<F extends URIS, A>(F: Applicative1<F>, M: Monoid<A>): () => Monoid<Kind<F, A>>;
/** @deprecated */
export declare function getMonoid<F, A>(F: Applicative<F>, M: Monoid<A>): () => Monoid<HKT<F, A>>;
