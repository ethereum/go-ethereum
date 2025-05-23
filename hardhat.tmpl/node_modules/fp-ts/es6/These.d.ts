import { Bifunctor2 } from './Bifunctor';
import { Either } from './Either';
import { Foldable2v2 } from './Foldable2v';
import { Functor2 } from './Functor';
import { Monad2C } from './Monad';
import { Option } from './Option';
import { Semigroup } from './Semigroup';
import { Eq } from './Eq';
import { Show } from './Show';
import { Traversable2v2 } from './Traversable2v';
declare module './HKT' {
    interface URItoKind2<L, A> {
        These: These<L, A>;
    }
}
export declare const URI = "These";
export declare type URI = typeof URI;
/**
 * @since 1.0.0
 */
export declare type These<L, A> = This<L, A> | That<L, A> | Both<L, A>;
export declare class This<L, A> {
    readonly value: L;
    readonly _tag: 'This';
    readonly _A: A;
    readonly _L: L;
    readonly _URI: URI;
    constructor(value: L);
    /** @obsolete */
    map<B>(f: (a: A) => B): These<L, B>;
    /** @obsolete */
    bimap<M, B>(f: (l: L) => M, g: (a: A) => B): These<M, B>;
    /** @obsolete */
    reduce<B>(b: B, f: (b: B, a: A) => B): B;
    /**
     * Applies a function to each case in the data structure
     * @obsolete
     */
    fold<B>(onLeft: (l: L) => B, onRight: (a: A) => B, onBoth: (l: L, a: A) => B): B;
    inspect(): string;
    toString(): string;
    /**
     * Returns `true` if the these is `This`, `false` otherwise
     * @obsolete
     */
    isThis(): this is This<L, A>;
    /**
     * Returns `true` if the these is `That`, `false` otherwise
     * @obsolete
     */
    isThat(): this is That<L, A>;
    /**
     * Returns `true` if the these is `Both`, `false` otherwise
     * @obsolete
     */
    isBoth(): this is Both<L, A>;
}
export declare class That<L, A> {
    readonly value: A;
    readonly _tag: 'That';
    readonly _A: A;
    readonly _L: L;
    readonly _URI: URI;
    constructor(value: A);
    map<B>(f: (a: A) => B): These<L, B>;
    bimap<M, B>(f: (l: L) => M, g: (a: A) => B): These<M, B>;
    reduce<B>(b: B, f: (b: B, a: A) => B): B;
    fold<B>(onLeft: (l: L) => B, onRight: (a: A) => B, onBoth: (l: L, a: A) => B): B;
    inspect(): string;
    toString(): string;
    isThis(): this is This<L, A>;
    isThat(): this is That<L, A>;
    isBoth(): this is Both<L, A>;
}
export declare class Both<L, A> {
    readonly l: L;
    readonly a: A;
    readonly _tag: 'Both';
    readonly _A: A;
    readonly _L: L;
    readonly _URI: URI;
    constructor(l: L, a: A);
    map<B>(f: (a: A) => B): These<L, B>;
    bimap<M, B>(f: (l: L) => M, g: (a: A) => B): These<M, B>;
    reduce<B>(b: B, f: (b: B, a: A) => B): B;
    fold<B>(onLeft: (l: L) => B, onRight: (a: A) => B, onBoth: (l: L, a: A) => B): B;
    inspect(): string;
    toString(): string;
    isThis(): this is This<L, A>;
    isThat(): this is That<L, A>;
    isBoth(): this is Both<L, A>;
}
/**
 * @since 1.17.0
 */
export declare const getShow: <L, A>(SL: Show<L>, SA: Show<A>) => Show<These<L, A>>;
/**
 * Use `getEq`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const getSetoid: <L, A>(EL: Eq<L>, EA: Eq<A>) => Eq<These<L, A>>;
/**
 * @since 1.19.0
 */
export declare function getEq<L, A>(EL: Eq<L>, EA: Eq<A>): Eq<These<L, A>>;
/**
 * @since 1.0.0
 */
export declare const getSemigroup: <L, A>(SL: Semigroup<L>, SA: Semigroup<A>) => Semigroup<These<L, A>>;
/**
 * Use `right`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const that: <L, A>(a: A) => These<L, A>;
/**
 * @since 1.0.0
 */
export declare const getMonad: <L>(S: Semigroup<L>) => Monad2C<"These", L>;
/**
 * Use `left`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const this_: <L, A>(l: L) => These<L, A>;
/**
 * @since 1.0.0
 */
export declare const both: <L, A>(l: L, a: A) => These<L, A>;
/**
 * Use `toTuple`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const fromThese: <L, A>(defaultThis: L, defaultThat: A) => (fa: These<L, A>) => [L, A];
/**
 * Use `getLeft`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const theseLeft: <L, A>(fa: These<L, A>) => Option<L>;
/**
 * Use `getRight`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const theseRight: <L, A>(fa: These<L, A>) => Option<A>;
/**
 * Use `isLeft`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const isThis: <L, A>(fa: These<L, A>) => fa is This<L, A>;
/**
 * Use `isRight`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const isThat: <L, A>(fa: These<L, A>) => fa is That<L, A>;
/**
 * Returns `true` if the these is an instance of `Both`, `false` otherwise
 *
 * @since 1.0.0
 */
export declare const isBoth: <L, A>(fa: These<L, A>) => fa is Both<L, A>;
/**
 * Use `leftOrBoth`
 *
 * @since 1.13.0
 * @deprecated
 */
export declare const thisOrBoth: <L, A>(defaultThis: L, ma: Option<A>) => These<L, A>;
/**
 * Use `rightOrBoth`
 *
 * @since 1.13.0
 * @deprecated
 */
export declare const thatOrBoth: <L, A>(defaultThat: A, ml: Option<L>) => These<L, A>;
/**
 * Use `getLeftOnly`
 *
 * @since 1.13.0
 * @deprecated
 */
export declare const theseThis: <L, A>(fa: These<L, A>) => Option<L>;
/**
 * Use `getRightOnly`
 *
 * @since 1.13.0
 * @deprecated
 */
export declare const theseThat: <L, A>(fa: These<L, A>) => Option<A>;
/**
 * Takes a pair of `Option`s and attempts to create a `These` from them
 *
 * @example
 * import { fromOptions, left, right, both } from 'fp-ts/lib/These'
 * import { none, some } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(fromOptions(none, none), none)
 * assert.deepStrictEqual(fromOptions(some('a'), none), some(left('a')))
 * assert.deepStrictEqual(fromOptions(none, some(1)), some(right(1)))
 * assert.deepStrictEqual(fromOptions(some('a'), some(1)), some(both('a', 1)))
 *
 * @since 1.13.0
 */
export declare const fromOptions: <L, A>(fl: Option<L>, fa: Option<A>) => Option<These<L, A>>;
/**
 * @example
 * import { fromEither, left, right } from 'fp-ts/lib/These'
 * import * as E from 'fp-ts/lib/Either'
 *
 * assert.deepStrictEqual(fromEither(E.left('a')), left('a'))
 * assert.deepStrictEqual(fromEither(E.right(1)), right(1))
 *
 * @since 1.13.0
 */
export declare const fromEither: <L, A>(fa: Either<L, A>) => These<L, A>;
/**
 * @since 1.0.0
 */
export declare const these: Functor2<URI> & Bifunctor2<URI> & Foldable2v2<URI> & Traversable2v2<URI>;
/**
 * @since 1.19.0
 */
export declare const left: <E = never, A = never>(left: E) => These<E, A>;
/**
 * @since 1.19.0
 */
export declare const right: <E = never, A = never>(right: A) => These<E, A>;
/**
 * Returns `true` if the these is an instance of `Left`, `false` otherwise
 *
 * @since 1.19.0
 */
export declare const isLeft: <E, A>(fa: These<E, A>) => fa is This<E, A>;
/**
 * Returns `true` if the these is an instance of `Right`, `false` otherwise
 *
 * @since 1.19.0
 */
export declare const isRight: <E, A>(fa: These<E, A>) => fa is That<E, A>;
/**
 * @example
 * import { toTuple, left, right, both } from 'fp-ts/lib/These'
 *
 * assert.deepStrictEqual(toTuple('a', 1)(left('b')), ['b', 1])
 * assert.deepStrictEqual(toTuple('a', 1)(right(2)), ['a', 2])
 * assert.deepStrictEqual(toTuple('a', 1)(both('b', 2)), ['b', 2])
 *
 * @since 1.19.0
 */
export declare const toTuple: <E, A>(e: E, a: A) => (fa: These<E, A>) => [E, A];
/**
 * Returns an `L` value if possible
 *
 * @example
 * import { getLeft, left, right, both } from 'fp-ts/lib/These'
 * import { none, some } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(getLeft(left('a')), some('a'))
 * assert.deepStrictEqual(getLeft(right(1)), none)
 * assert.deepStrictEqual(getLeft(both('a', 1)), some('a'))
 *
 * @since 1.19.0
 */
export declare const getLeft: <E, A>(fa: These<E, A>) => Option<E>;
/**
 * Returns an `A` value if possible
 *
 * @example
 * import { getRight, left, right, both } from 'fp-ts/lib/These'
 * import { none, some } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(getRight(left('a')), none)
 * assert.deepStrictEqual(getRight(right(1)), some(1))
 * assert.deepStrictEqual(getRight(both('a', 1)), some(1))
 *
 * @since 1.19.0
 */
export declare const getRight: <E, A>(fa: These<E, A>) => Option<A>;
/**
 * @example
 * import { leftOrBoth, left, both } from 'fp-ts/lib/These'
 * import { none, some } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(leftOrBoth('a')(none), left('a'))
 * assert.deepStrictEqual(leftOrBoth('a')(some(1)), both('a', 1))
 *
 * @since 1.19.0
 */
export declare function leftOrBoth<E>(defaultLeft: E): <A>(ma: Option<A>) => These<E, A>;
/**
 * @example
 * import { rightOrBoth, right, both } from 'fp-ts/lib/These'
 * import { none, some } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(rightOrBoth(1)(none), right(1))
 * assert.deepStrictEqual(rightOrBoth(1)(some('a')), both('a', 1))
 *
 * @since 1.19.0
 */
export declare function rightOrBoth<A>(defaultRight: A): <E>(me: Option<E>) => These<E, A>;
/**
 * Returns the `L` value if and only if the value is constructed with `Left`
 *
 * @example
 * import { getLeftOnly, left, right, both } from 'fp-ts/lib/These'
 * import { none, some } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(getLeftOnly(left('a')), some('a'))
 * assert.deepStrictEqual(getLeftOnly(right(1)), none)
 * assert.deepStrictEqual(getLeftOnly(both('a', 1)), none)
 *
 * @since 1.19.0
 */
export declare const getLeftOnly: <E, A>(fa: These<E, A>) => Option<E>;
/**
 * Returns the `A` value if and only if the value is constructed with `Right`
 *
 * @example
 * import { getRightOnly, left, right, both } from 'fp-ts/lib/These'
 * import { none, some } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(getRightOnly(left('a')), none)
 * assert.deepStrictEqual(getRightOnly(right(1)), some(1))
 * assert.deepStrictEqual(getRightOnly(both('a', 1)), none)
 *
 *
 * @since 1.19.0
 */
export declare const getRightOnly: <E, A>(fa: These<E, A>) => Option<A>;
/**
 * @since 1.19.0
 */
export declare function fold<E, A, R>(onLeft: (e: E) => R, onRight: (a: A) => R, onBoth: (e: E, a: A) => R): (fa: These<E, A>) => R;
declare const bimap: <L, A, M, B>(f: (l: L) => M, g: (a: A) => B) => (fa: These<L, A>) => These<M, B>, foldMap: <M>(M: import("./Monoid").Monoid<M>) => <A>(f: (a: A) => M) => <L>(fa: These<L, A>) => M, map: <A, B>(f: (a: A) => B) => <L>(fa: These<L, A>) => These<L, B>, mapLeft: <L, A, M>(f: (l: L) => M) => (fa: These<L, A>) => These<M, A>, reduce: <A, B>(b: B, f: (b: B, a: A) => B) => <L>(fa: These<L, A>) => B, reduceRight: <A, B>(b: B, f: (a: A, b: B) => B) => <L>(fa: These<L, A>) => B;
export { bimap, foldMap, map, mapLeft, reduce, reduceRight };
