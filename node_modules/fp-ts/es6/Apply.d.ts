/**
 * @file The `Apply` class provides the `ap` which is used to apply a function to an argument under a type constructor.
 *
 * `Apply` can be used to lift functions of two or more arguments to work on values wrapped with the type constructor
 * `f`.
 *
 * Instances must satisfy the following law in addition to the `Functor` laws:
 *
 * 1. Associative composition: `F.ap(F.ap(F.map(fbc, bc => ab => a => bc(ab(a))), fab), fa) = F.ap(fbc, F.ap(fab, fa))`
 *
 * Formally, `Apply` represents a strong lax semi-monoidal endofunctor.
 */
import { Functor, Functor1, Functor2, Functor2C, Functor3, Functor3C, Functor4 } from './Functor';
import { HKT, Kind, Kind2, Kind3, URIS, URIS2, URIS3, URIS4, Kind4 } from './HKT';
import { Semigroup } from './Semigroup';
import { Curried2, Curried3, Curried4 } from './function';
/**
 * @since 1.0.0
 */
export interface Apply<F> extends Functor<F> {
    readonly ap: <A, B>(fab: HKT<F, (a: A) => B>, fa: HKT<F, A>) => HKT<F, B>;
}
export interface Apply1<F extends URIS> extends Functor1<F> {
    readonly ap: <A, B>(fab: Kind<F, (a: A) => B>, fa: Kind<F, A>) => Kind<F, B>;
}
export interface Apply2<F extends URIS2> extends Functor2<F> {
    readonly ap: <L, A, B>(fab: Kind2<F, L, (a: A) => B>, fa: Kind2<F, L, A>) => Kind2<F, L, B>;
}
export interface Apply3<F extends URIS3> extends Functor3<F> {
    readonly ap: <U, L, A, B>(fab: Kind3<F, U, L, (a: A) => B>, fa: Kind3<F, U, L, A>) => Kind3<F, U, L, B>;
}
export interface Apply2C<F extends URIS2, L> extends Functor2C<F, L> {
    readonly ap: <A, B>(fab: Kind2<F, L, (a: A) => B>, fa: Kind2<F, L, A>) => Kind2<F, L, B>;
}
export interface Apply3C<F extends URIS3, U, L> extends Functor3C<F, U, L> {
    readonly ap: <A, B>(fab: Kind3<F, U, L, (a: A) => B>, fa: Kind3<F, U, L, A>) => Kind3<F, U, L, B>;
}
export interface Apply4<F extends URIS4> extends Functor4<F> {
    readonly ap: <X, U, L, A, B>(fab: Kind4<F, X, U, L, (a: A) => B>, fa: Kind4<F, X, U, L, A>) => Kind4<F, X, U, L, B>;
}
/**
 * Use `pipeable`'s `apFirst`
 * @since 1.0.0
 * @deprecated
 */
export declare function applyFirst<F extends URIS3>(F: Apply3<F>): <U, L, A, B>(fa: Kind3<F, U, L, A>, fb: Kind3<F, U, L, B>) => Kind3<F, U, L, A>;
/**
 * Use `pipeable`'s `apFirst`
 * @deprecated
 */
export declare function applyFirst<F extends URIS3, U, L>(F: Apply3C<F, U, L>): <A, B>(fa: Kind3<F, U, L, A>, fb: Kind3<F, U, L, B>) => Kind3<F, U, L, A>;
/**
 * Use `pipeable`'s `apFirst`
 * @deprecated
 */
export declare function applyFirst<F extends URIS2>(F: Apply2<F>): <L, A, B>(fa: Kind2<F, L, A>, fb: Kind2<F, L, B>) => Kind2<F, L, A>;
/**
 * Use `pipeable`'s `apFirst`
 * @deprecated
 */
export declare function applyFirst<F extends URIS2, L>(F: Apply2C<F, L>): <A, B>(fa: Kind2<F, L, A>, fb: Kind2<F, L, B>) => Kind2<F, L, A>;
/**
 * Use `pipeable`'s `apFirst`
 * @deprecated
 */
export declare function applyFirst<F extends URIS>(F: Apply1<F>): <A, B>(fa: Kind<F, A>, fb: Kind<F, B>) => Kind<F, A>;
/**
 * Use `pipeable`'s `apFirst`
 * @deprecated
 */
export declare function applyFirst<F>(F: Apply<F>): <A, B>(fa: HKT<F, A>, fb: HKT<F, B>) => HKT<F, A>;
/**
 * Use `pipeable`'s `apSecond`
 * @since 1.0.0
 * @deprecated
 */
export declare function applySecond<F extends URIS3>(F: Apply3<F>): <U, L, A, B>(fa: Kind3<F, U, L, A>, fb: Kind3<F, U, L, B>) => Kind3<F, U, L, B>;
/**
 * Use `pipeable`'s `apSecond`
 * @deprecated
 */
export declare function applySecond<F extends URIS3, U, L>(F: Apply3C<F, U, L>): <A, B>(fa: Kind3<F, U, L, A>, fb: Kind3<F, U, L, B>) => Kind3<F, U, L, B>;
/**
 * Use `pipeable`'s `apSecond`
 * @deprecated
 */
export declare function applySecond<F extends URIS2>(F: Apply2<F>): <L, A, B>(fa: Kind2<F, L, A>, fb: Kind2<F, L, B>) => Kind2<F, L, B>;
/**
 * Use `pipeable`'s `apSecond`
 * @deprecated
 */
export declare function applySecond<F extends URIS2, L>(F: Apply2C<F, L>): <A, B>(fa: Kind2<F, L, A>, fb: Kind2<F, L, B>) => Kind2<F, L, B>;
/**
 * Use `pipeable`'s `apSecond`
 * @deprecated
 */
export declare function applySecond<F extends URIS>(F: Apply1<F>): <A, B>(fa: Kind<F, A>, fb: Kind<F, B>) => Kind<F, B>;
/**
 * Use `pipeable`'s `apSecond`
 * @deprecated
 */
export declare function applySecond<F>(F: Apply<F>): <A, B>(fa: HKT<F, A>, fb: HKT<F, B>) => HKT<F, B>;
/**
 * Lift a function of two arguments to a function which accepts and returns values wrapped with the type constructor `F`
 *
 * Use `sequenceT` / `sequenceS` instead.
 *
 * @since 1.0.0
 * @deprecated
 */
export declare function liftA2<F extends URIS3>(F: Apply3<F>): <A, B, C>(f: Curried2<A, B, C>) => <U, L>(fa: Kind3<F, U, L, A>) => (fb: Kind3<F, U, L, B>) => Kind3<F, U, L, C>;
/** @deprecated */
export declare function liftA2<F extends URIS3, U, L>(F: Apply3C<F, U, L>): <A, B, C>(f: Curried2<A, B, C>) => (fa: Kind3<F, U, L, A>) => (fb: Kind3<F, U, L, B>) => Kind3<F, U, L, C>;
/** @deprecated */
export declare function liftA2<F extends URIS2>(F: Apply2<F>): <A, B, C>(f: Curried2<A, B, C>) => <L>(fa: Kind2<F, L, A>) => (fb: Kind2<F, L, B>) => Kind2<F, L, C>;
/** @deprecated */
export declare function liftA2<F extends URIS2, L>(F: Apply2C<F, L>): <A, B, C>(f: Curried2<A, B, C>) => (fa: Kind2<F, L, A>) => (fb: Kind2<F, L, B>) => Kind2<F, L, C>;
/** @deprecated */
export declare function liftA2<F extends URIS>(F: Apply1<F>): <A, B, C>(f: Curried2<A, B, C>) => Curried2<Kind<F, A>, Kind<F, B>, Kind<F, C>>;
/** @deprecated */
export declare function liftA2<F>(F: Apply<F>): <A, B, C>(f: Curried2<A, B, C>) => Curried2<HKT<F, A>, HKT<F, B>, HKT<F, C>>;
/**
 * Lift a function of three arguments to a function which accepts and returns values wrapped with the type constructor
 * `F`
 *
 * Use `sequenceT` / `sequenceS` instead.
 *
 * @since 1.0.0
 * @deprecated
 */
export declare function liftA3<F extends URIS3>(F: Apply3<F>): <A, B, C, D>(f: Curried3<A, B, C, D>) => <U, L>(fa: Kind3<F, U, L, A>) => (fb: Kind3<F, U, L, B>) => (fc: Kind3<F, U, L, C>) => Kind3<F, U, L, D>;
/** @deprecated */
export declare function liftA3<F extends URIS3, U, L>(F: Apply3C<F, U, L>): <A, B, C, D>(f: Curried3<A, B, C, D>) => (fa: Kind3<F, U, L, A>) => (fb: Kind3<F, U, L, B>) => (fc: Kind3<F, U, L, C>) => Kind3<F, U, L, D>;
/** @deprecated */
export declare function liftA3<F extends URIS2>(F: Apply2<F>): <A, B, C, D>(f: Curried3<A, B, C, D>) => <L>(fa: Kind2<F, L, A>) => (fb: Kind2<F, L, B>) => (fc: Kind2<F, L, C>) => Kind2<F, L, D>;
/** @deprecated */
export declare function liftA3<F extends URIS2, L>(F: Apply2C<F, L>): <A, B, C, D>(f: Curried3<A, B, C, D>) => (fa: Kind2<F, L, A>) => (fb: Kind2<F, L, B>) => (fc: Kind2<F, L, C>) => Kind2<F, L, D>;
/** @deprecated */
export declare function liftA3<F extends URIS>(F: Apply1<F>): <A, B, C, D>(f: Curried3<A, B, C, D>) => Curried3<Kind<F, A>, Kind<F, B>, Kind<F, C>, Kind<F, D>>;
/** @deprecated */
export declare function liftA3<F>(F: Apply<F>): <A, B, C, D>(f: Curried3<A, B, C, D>) => Curried3<HKT<F, A>, HKT<F, B>, HKT<F, C>, HKT<F, D>>;
/**
 * Lift a function of four arguments to a function which accepts and returns values wrapped with the type constructor
 * `F`
 *
 * Use `sequenceT` / `sequenceS` instead.
 *
 * @since 1.0.0
 * @deprecated
 */
export declare function liftA4<F extends URIS3>(F: Apply3<F>): <A, B, C, D, E>(f: Curried4<A, B, C, D, E>) => <U, L>(fa: Kind3<F, U, L, A>) => (fb: Kind3<F, U, L, B>) => (fc: Kind3<F, U, L, C>) => (fd: Kind3<F, U, L, D>) => Kind3<F, U, L, E>;
/** @deprecated */
export declare function liftA4<F extends URIS3, U, L>(F: Apply3C<F, U, L>): <A, B, C, D, E>(f: Curried4<A, B, C, D, E>) => (fa: Kind3<F, U, L, A>) => (fb: Kind3<F, U, L, B>) => (fc: Kind3<F, U, L, C>) => (fd: Kind3<F, U, L, D>) => Kind3<F, U, L, E>;
/** @deprecated */
export declare function liftA4<F extends URIS2>(F: Apply2<F>): <A, B, C, D, E>(f: Curried4<A, B, C, D, E>) => <L>(fa: Kind2<F, L, A>) => (fb: Kind2<F, L, B>) => (fc: Kind2<F, L, C>) => (fd: Kind2<F, L, D>) => Kind2<F, L, E>;
/** @deprecated */
export declare function liftA4<F extends URIS2, L>(F: Apply2C<F, L>): <A, B, C, D, E>(f: Curried4<A, B, C, D, E>) => (fa: Kind2<F, L, A>) => (fb: Kind2<F, L, B>) => (fc: Kind2<F, L, C>) => (fd: Kind2<F, L, D>) => Kind2<F, L, E>;
/** @deprecated */
export declare function liftA4<F extends URIS>(F: Apply1<F>): <A, B, C, D, E>(f: Curried4<A, B, C, D, E>) => Curried4<Kind<F, A>, Kind<F, B>, Kind<F, C>, Kind<F, D>, Kind<F, E>>;
/** @deprecated */
export declare function liftA4<F>(F: Apply<F>): <A, B, C, D, E>(f: Curried4<A, B, C, D, E>) => Curried4<HKT<F, A>, HKT<F, B>, HKT<F, C>, HKT<F, D>, HKT<F, E>>;
/**
 * If `F` is a `Apply` and `S` is a `Semigroup` over `A` then `HKT<F, A>` is a `Semigroup` over `A` as well
 *
 * @example
 * import { getSemigroup } from 'fp-ts/lib/Apply'
 * import { option, some, none } from 'fp-ts/lib/Option'
 * import { monoidSum } from 'fp-ts/lib/Monoid'
 *
 * const S = getSemigroup(option, monoidSum)()
 * assert.deepStrictEqual(S.concat(none, none), none)
 * assert.deepStrictEqual(S.concat(some(1), none), none)
 * assert.deepStrictEqual(S.concat(none, some(2)), none)
 * assert.deepStrictEqual(S.concat(some(1), some(2)), some(3))
 *
 * @since 1.4.0
 * @deprecated
 */
export declare function getSemigroup<F extends URIS3, A>(F: Apply3<F>, S: Semigroup<A>): <U = never, L = never>() => Semigroup<Kind3<F, U, L, A>>;
/** @deprecated */
export declare function getSemigroup<F extends URIS3, U, L, A>(F: Apply3C<F, U, L>, S: Semigroup<A>): () => Semigroup<Kind3<F, U, L, A>>;
/** @deprecated */
export declare function getSemigroup<F extends URIS2, A>(F: Apply2<F>, S: Semigroup<A>): <L = never>() => Semigroup<Kind2<F, L, A>>;
/** @deprecated */
export declare function getSemigroup<F extends URIS2, L, A>(F: Apply2C<F, L>, S: Semigroup<A>): () => Semigroup<Kind2<F, L, A>>;
/** @deprecated */
export declare function getSemigroup<F extends URIS, A>(F: Apply1<F>, S: Semigroup<A>): () => Semigroup<Kind<F, A>>;
/** @deprecated */
export declare function getSemigroup<F, A>(F: Apply<F>, S: Semigroup<A>): () => Semigroup<HKT<F, A>>;
export interface SequenceT3<F extends URIS3> {
    <U, L, T extends Array<Kind3<F, U, L, any>>>(...t: T & {
        0: Kind3<F, U, L, any>;
    }): Kind3<F, U, L, {
        [K in keyof T]: [T[K]] extends [Kind3<F, U, L, infer A>] ? A : never;
    }>;
}
export interface SequenceT3C<F extends URIS3, U, L> {
    <T extends Array<Kind3<F, U, L, any>>>(...t: T & {
        0: Kind3<F, U, L, any>;
    }): Kind3<F, U, L, {
        [K in keyof T]: [T[K]] extends [Kind3<F, U, L, infer A>] ? A : never;
    }>;
}
export interface SequenceT2<F extends URIS2> {
    <L, T extends Array<Kind2<F, L, any>>>(...t: T & {
        0: Kind2<F, L, any>;
    }): Kind2<F, L, {
        [K in keyof T]: [T[K]] extends [Kind2<F, L, infer A>] ? A : never;
    }>;
}
export interface SequenceT2C<F extends URIS2, L> {
    <T extends Array<Kind2<F, L, any>>>(...t: T & {
        0: Kind2<F, L, any>;
    }): Kind2<F, L, {
        [K in keyof T]: [T[K]] extends [Kind2<F, L, infer A>] ? A : never;
    }>;
}
export interface SequenceT1<F extends URIS> {
    <T extends Array<Kind<F, any>>>(...t: T & {
        0: Kind<F, any>;
    }): Kind<F, {
        [K in keyof T]: [T[K]] extends [Kind<F, infer A>] ? A : never;
    }>;
}
export interface SequenceT<F> {
    <T extends Array<HKT<F, any>>>(...t: T & {
        0: HKT<F, any>;
    }): HKT<F, {
        [K in keyof T]: [T[K]] extends [HKT<F, infer A>] ? A : never;
    }>;
}
/**
 * Tuple sequencing, i.e., take a tuple of monadic actions and does them from left-to-right, returning the resulting tuple.
 *
 * @example
 * import { sequenceT } from 'fp-ts/lib/Apply'
 * import { option, some, none } from 'fp-ts/lib/Option'
 *
 * const sequenceTOption = sequenceT(option)
 * assert.deepStrictEqual(sequenceTOption(some(1)), some([1]))
 * assert.deepStrictEqual(sequenceTOption(some(1), some('2')), some([1, '2']))
 * assert.deepStrictEqual(sequenceTOption(some(1), some('2'), none), none)
 *
 * @since 1.5.0
 */
export declare function sequenceT<F extends URIS3>(F: Apply3<F>): SequenceT3<F>;
export declare function sequenceT<F extends URIS3, U, L>(F: Apply3C<F, U, L>): SequenceT3C<F, U, L>;
export declare function sequenceT<F extends URIS2>(F: Apply2<F>): SequenceT2<F>;
export declare function sequenceT<F extends URIS2, L>(F: Apply2C<F, L>): SequenceT2C<F, L>;
export declare function sequenceT<F extends URIS>(F: Apply1<F>): SequenceT1<F>;
export declare function sequenceT<F>(F: Apply<F>): SequenceT<F>;
declare type EnforceNonEmptyRecord<R> = keyof R extends never ? never : R;
/**
 * Like `Apply.sequenceT` but works with structs instead of tuples.
 *
 * @example
 * import { either, right, left } from 'fp-ts/lib/Either'
 * import { sequenceS } from 'fp-ts/lib/Apply'
 *
 * const ado = sequenceS(either)
 *
 * assert.deepStrictEqual(
 *   ado({
 *     a: right<string, number>(1),
 *     b: right<string, boolean>(true)
 *   }),
 *   right({ a: 1, b: true })
 * )
 * assert.deepStrictEqual(
 *   ado({
 *     a: right<string, number>(1),
 *     b: left<string, number>('error')
 *   }),
 *   left('error')
 * )
 *
 * @since 1.15.0
 */
export declare function sequenceS<F extends URIS3>(F: Apply3<F>): <U, L, R extends Record<string, Kind3<F, U, L, any>>>(r: EnforceNonEmptyRecord<R> & Record<string, Kind3<F, U, L, any>>) => Kind3<F, U, L, {
    [K in keyof R]: [R[K]] extends [Kind3<F, any, any, infer A>] ? A : never;
}>;
export declare function sequenceS<F extends URIS3, U, L>(F: Apply3C<F, U, L>): <R extends Record<string, Kind3<F, U, L, any>>>(r: EnforceNonEmptyRecord<R>) => Kind3<F, U, L, {
    [K in keyof R]: [R[K]] extends [Kind3<F, any, any, infer A>] ? A : never;
}>;
export declare function sequenceS<F extends URIS2>(F: Apply2<F>): <L, R extends Record<string, Kind2<F, L, any>>>(r: EnforceNonEmptyRecord<R> & Record<string, Kind2<F, L, any>>) => Kind2<F, L, {
    [K in keyof R]: [R[K]] extends [Kind2<F, any, infer A>] ? A : never;
}>;
export declare function sequenceS<F extends URIS2, L>(F: Apply2C<F, L>): <R extends Record<string, Kind2<F, L, any>>>(r: EnforceNonEmptyRecord<R>) => Kind2<F, L, {
    [K in keyof R]: [R[K]] extends [Kind2<F, any, infer A>] ? A : never;
}>;
export declare function sequenceS<F extends URIS>(F: Apply1<F>): <R extends Record<string, Kind<F, any>>>(r: EnforceNonEmptyRecord<R>) => Kind<F, {
    [K in keyof R]: [R[K]] extends [Kind<F, infer A>] ? A : never;
}>;
export declare function sequenceS<F>(F: Apply<F>): <R extends Record<string, HKT<F, any>>>(r: EnforceNonEmptyRecord<R>) => HKT<F, {
    [K in keyof R]: [R[K]] extends [HKT<F, infer A>] ? A : never;
}>;
export {};
