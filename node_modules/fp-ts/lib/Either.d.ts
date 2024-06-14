/**
 * @file Represents a value of one of two possible types (a disjoint union).
 *
 * An instance of `Either` is either an instance of `Left` or `Right`.
 *
 * A common use of `Either` is as an alternative to `Option` for dealing with possible missing values. In this usage,
 * `None` is replaced with a `Left` which can contain useful information. `Right` takes the place of `Some`. Convention
 * dictates that `Left` is used for failure and `Right` is used for success.
 *
 * For example, you could use `Either<string, number>` to detect whether a received input is a `string` or a `number`.
 *
 * ```ts
 * const parse = (errorMessage: string) => (input: string): Either<string, number> => {
 *   const n = parseInt(input, 10)
 *   return isNaN(n) ? left(errorMessage) : right(n)
 * }
 * ```
 *
 * `Either` is right-biased, which means that `Right` is assumed to be the default case to operate on. If it is `Left`,
 * operations like `map`, `chain`, ... return the `Left` value unchanged:
 *
 * ```ts
 * right(12).map(double) // right(24)
 * left(23).map(double)  // left(23)
 * ```
 */
import { Alt2, Alt2C } from './Alt';
import { Bifunctor2 } from './Bifunctor';
import { ChainRec2 } from './ChainRec';
import { Compactable2C } from './Compactable';
import { Extend2 } from './Extend';
import { Filterable2C } from './Filterable';
import { Foldable2v2 } from './Foldable2v';
import { Lazy, Predicate, Refinement } from './function';
import { Monad2, Monad2C } from './Monad';
import { Monoid } from './Monoid';
import { Option } from './Option';
import { Semigroup } from './Semigroup';
import { Eq } from './Eq';
import { Traversable2v2 } from './Traversable2v';
import { Validation } from './Validation';
import { Witherable2C } from './Witherable';
import { MonadThrow2 } from './MonadThrow';
import { Show } from './Show';
declare module './HKT' {
    interface URItoKind2<L, A> {
        Either: Either<L, A>;
    }
}
export declare const URI = "Either";
export declare type URI = typeof URI;
/**
 * @since 1.0.0
 */
export declare type Either<L, A> = Left<L, A> | Right<L, A>;
/**
 * Left side of `Either`
 */
export declare class Left<L, A> {
    readonly value: L;
    readonly _tag: 'Left';
    readonly _A: A;
    readonly _L: L;
    readonly _URI: URI;
    constructor(value: L);
    /**
     * The given function is applied if this is a `Right`
     * @obsolete
     */
    map<B>(f: (a: A) => B): Either<L, B>;
    /** @obsolete */
    ap<B>(fab: Either<L, (a: A) => B>): Either<L, B>;
    /**
     * Flipped version of `ap`
     * @obsolete
     */
    ap_<B, C>(this: Either<L, (b: B) => C>, fb: Either<L, B>): Either<L, C>;
    /**
     * Binds the given function across `Right`
     * @obsolete
     */
    chain<B>(f: (a: A) => Either<L, B>): Either<L, B>;
    /** @obsolete */
    bimap<V, B>(f: (l: L) => V, g: (a: A) => B): Either<V, B>;
    /** @obsolete */
    alt(fy: Either<L, A>): Either<L, A>;
    /**
     * Lazy version of `alt`
     *
     * @example
     * import { right } from 'fp-ts/lib/Either'
     *
     * assert.deepStrictEqual(right(1).orElse(() => right(2)), right(1))
     *
     * @since 1.6.0
     * @obsolete
     */
    orElse<M>(fy: (l: L) => Either<M, A>): Either<M, A>;
    /** @obsolete */
    extend<B>(f: (ea: Either<L, A>) => B): Either<L, B>;
    /** @obsolete */
    reduce<B>(b: B, f: (b: B, a: A) => B): B;
    /**
     * Applies a function to each case in the data structure
     * @obsolete
     */
    fold<B>(onLeft: (l: L) => B, onRight: (a: A) => B): B;
    /**
     * Returns the value from this `Right` or the given argument if this is a `Left`
     * @obsolete
     */
    getOrElse(a: A): A;
    /**
     * Returns the value from this `Right` or the result of given argument if this is a `Left`
     * @obsolete
     */
    getOrElseL(f: (l: L) => A): A;
    /**
     * Maps the left side of the disjunction
     * @obsolete
     */
    mapLeft<M>(f: (l: L) => M): Either<M, A>;
    inspect(): string;
    toString(): string;
    /**
     * Returns `true` if the either is an instance of `Left`, `false` otherwise
     * @obsolete
     */
    isLeft(): this is Left<L, A>;
    /**
     * Returns `true` if the either is an instance of `Right`, `false` otherwise
     * @obsolete
     */
    isRight(): this is Right<L, A>;
    /**
     * Swaps the disjunction values
     * @obsolete
     */
    swap(): Either<A, L>;
    /**
     * Returns `Right` with the existing value of `Right` if this is a `Right` and the given predicate `p` holds for the
     * right value, returns `Left(zero)` if this is a `Right` and the given predicate `p` does not hold for the right
     * value, returns `Left` with the existing value of `Left` if this is a `Left`.
     *
     * @example
     * import { right, left } from 'fp-ts/lib/Either'
     *
     * assert.deepStrictEqual(right(12).filterOrElse(n => n > 10, -1), right(12))
     * assert.deepStrictEqual(right(7).filterOrElse(n => n > 10, -1), left(-1))
     * assert.deepStrictEqual(left<number, number>(12).filterOrElse(n => n > 10, -1), left(12))
     *
     * @since 1.3.0
     * @obsolete
     */
    filterOrElse<B extends A>(p: Refinement<A, B>, zero: L): Either<L, B>;
    filterOrElse(p: Predicate<A>, zero: L): Either<L, A>;
    /**
     * Lazy version of `filterOrElse`
     * @since 1.3.0
     * @obsolete
     */
    filterOrElseL<B extends A>(p: Refinement<A, B>, zero: (a: A) => L): Either<L, B>;
    filterOrElseL(p: Predicate<A>, zero: (a: A) => L): Either<L, A>;
    /**
     * Use `filterOrElse` instead
     * @since 1.6.0
     * @deprecated
     */
    refineOrElse<B extends A>(p: Refinement<A, B>, zero: L): Either<L, B>;
    /**
     * Lazy version of `refineOrElse`
     * Use `filterOrElseL` instead
     * @since 1.6.0
     * @deprecated
     */
    refineOrElseL<B extends A>(p: Refinement<A, B>, zero: (a: A) => L): Either<L, B>;
}
/**
 * Right side of `Either`
 */
export declare class Right<L, A> {
    readonly value: A;
    readonly _tag: 'Right';
    readonly _A: A;
    readonly _L: L;
    readonly _URI: URI;
    constructor(value: A);
    map<B>(f: (a: A) => B): Either<L, B>;
    ap<B>(fab: Either<L, (a: A) => B>): Either<L, B>;
    ap_<B, C>(this: Either<L, (b: B) => C>, fb: Either<L, B>): Either<L, C>;
    chain<B>(f: (a: A) => Either<L, B>): Either<L, B>;
    bimap<V, B>(f: (l: L) => V, g: (a: A) => B): Either<V, B>;
    alt(fy: Either<L, A>): Either<L, A>;
    orElse<M>(fy: (l: L) => Either<M, A>): Either<M, A>;
    extend<B>(f: (ea: Either<L, A>) => B): Either<L, B>;
    reduce<B>(b: B, f: (b: B, a: A) => B): B;
    fold<B>(onLeft: (l: L) => B, onRight: (a: A) => B): B;
    getOrElse(a: A): A;
    getOrElseL(f: (l: L) => A): A;
    mapLeft<M>(f: (l: L) => M): Either<M, A>;
    inspect(): string;
    toString(): string;
    isLeft(): this is Left<L, A>;
    isRight(): this is Right<L, A>;
    swap(): Either<A, L>;
    filterOrElse<B extends A>(p: Refinement<A, B>, zero: L): Either<L, B>;
    filterOrElse(p: Predicate<A>, zero: L): Either<L, A>;
    filterOrElseL<B extends A>(p: Refinement<A, B>, zero: (a: A) => L): Either<L, B>;
    filterOrElseL(p: Predicate<A>, zero: (a: A) => L): Either<L, A>;
    refineOrElse<B extends A>(p: Refinement<A, B>, zero: L): Either<L, B>;
    refineOrElseL<B extends A>(p: Refinement<A, B>, zero: (a: A) => L): Either<L, B>;
}
/**
 * @since 1.17.0
 */
export declare const getShow: <L, A>(SL: Show<L>, SA: Show<A>) => Show<Either<L, A>>;
/**
 * Use `getEq`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const getSetoid: <L, A>(EL: Eq<L>, EA: Eq<A>) => Eq<Either<L, A>>;
/**
 * @since 1.19.0
 */
export declare function getEq<L, A>(EL: Eq<L>, EA: Eq<A>): Eq<Either<L, A>>;
/**
 * Semigroup returning the left-most non-`Left` value. If both operands are `Right`s then the inner values are
 * appended using the provided `Semigroup`
 *
 * @example
 * import { getSemigroup, left, right } from 'fp-ts/lib/Either'
 * import { semigroupSum } from 'fp-ts/lib/Semigroup'
 *
 * const S = getSemigroup<string, number>(semigroupSum)
 * assert.deepStrictEqual(S.concat(left('a'), left('b')), left('a'))
 * assert.deepStrictEqual(S.concat(left('a'), right(2)), right(2))
 * assert.deepStrictEqual(S.concat(right(1), left('b')), right(1))
 * assert.deepStrictEqual(S.concat(right(1), right(2)), right(3))
 *
 *
 * @since 1.7.0
 */
export declare const getSemigroup: <L, A>(S: Semigroup<A>) => Semigroup<Either<L, A>>;
/**
 * `Apply` semigroup
 *
 * @example
 * import { getApplySemigroup, left, right } from 'fp-ts/lib/Either'
 * import { semigroupSum } from 'fp-ts/lib/Semigroup'
 *
 * const S = getApplySemigroup<string, number>(semigroupSum)
 * assert.deepStrictEqual(S.concat(left('a'), left('b')), left('a'))
 * assert.deepStrictEqual(S.concat(left('a'), right(2)), left('a'))
 * assert.deepStrictEqual(S.concat(right(1), left('b')), left('b'))
 * assert.deepStrictEqual(S.concat(right(1), right(2)), right(3))
 *
 *
 * @since 1.7.0
 */
export declare const getApplySemigroup: <L, A>(S: Semigroup<A>) => Semigroup<Either<L, A>>;
/**
 * @since 1.7.0
 */
export declare const getApplyMonoid: <L, A>(M: Monoid<A>) => Monoid<Either<L, A>>;
/**
 * Constructs a new `Either` holding a `Left` value. This usually represents a failure, due to the right-bias of this
 * structure
 *
 * @since 1.0.0
 */
export declare const left: <L, A>(l: L) => Either<L, A>;
/**
 * Constructs a new `Either` holding a `Right` value. This usually represents a successful value due to the right bias
 * of this structure
 *
 * @since 1.0.0
 */
export declare const right: <L, A>(a: A) => Either<L, A>;
/**
 * Use `fromPredicate` instead
 *
 * @since 1.6.0
 * @deprecated
 */
export declare const fromRefinement: <L, A, B extends A>(refinement: Refinement<A, B>, onFalse: (a: A) => L) => (a: A) => Either<L, B>;
/**
 * Takes a default and a `Option` value, if the value is a `Some`, turn it into a `Right`, if the value is a `None` use
 * the provided default as a `Left`
 *
 * @since 1.0.0
 */
export declare const fromOption: <L>(onNone: L) => <A>(fa: Option<A>) => Either<L, A>;
/**
 * Takes a default and a nullable value, if the value is not nully, turn it into a `Right`, if the value is nully use
 * the provided default as a `Left`
 *
 * @since 1.0.0
 */
export declare const fromNullable: <L>(defaultValue: L) => <A>(a: A | null | undefined) => Either<L, A>;
/**
 * Default value for the optional `onerror` argument of `tryCatch`
 *
 * @since 1.0.0
 */
export declare const toError: (e: unknown) => Error;
/**
 * Use `tryCatch2v` instead
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const tryCatch: <A>(f: Lazy<A>, onerror?: (e: unknown) => Error) => Either<Error, A>;
/**
 * Constructs a new `Either` from a function that might throw
 *
 * @example
 * import { Either, left, right, tryCatch2v } from 'fp-ts/lib/Either'
 *
 * const unsafeHead = <A>(as: Array<A>): A => {
 *   if (as.length > 0) {
 *     return as[0]
 *   } else {
 *     throw new Error('empty array')
 *   }
 * }
 *
 * const head = <A>(as: Array<A>): Either<Error, A> => {
 *   return tryCatch2v(() => unsafeHead(as), e => (e instanceof Error ? e : new Error('unknown error')))
 * }
 *
 * assert.deepStrictEqual(head([]), left(new Error('empty array')))
 * assert.deepStrictEqual(head([1, 2, 3]), right(1))
 *
 * @since 1.11.0
 */
export declare const tryCatch2v: <L, A>(f: Lazy<A>, onerror: (e: unknown) => L) => Either<L, A>;
/**
 * @since 1.0.0
 */
export declare const fromValidation: <L, A>(fa: Validation<L, A>) => Either<L, A>;
/**
 * Returns `true` if the either is an instance of `Left`, `false` otherwise
 *
 * @since 1.0.0
 */
export declare const isLeft: <L, A>(fa: Either<L, A>) => fa is Left<L, A>;
/**
 * Returns `true` if the either is an instance of `Right`, `false` otherwise
 *
 * @since 1.0.0
 */
export declare const isRight: <L, A>(fa: Either<L, A>) => fa is Right<L, A>;
/**
 * Use `getWitherable`
 *
 * @since 1.7.0
 * @deprecated
 */
export declare function getCompactable<L>(ML: Monoid<L>): Compactable2C<URI, L>;
/**
 * Use `getWitherable`
 *
 * @since 1.7.0
 * @deprecated
 */
export declare function getFilterable<L>(ML: Monoid<L>): Filterable2C<URI, L>;
/**
 * Builds `Witherable` instance for `Either` given `Monoid` for the left side
 *
 * @since 1.7.0
 */
export declare function getWitherable<L>(ML: Monoid<L>): Witherable2C<URI, L>;
/**
 * Converts a JavaScript Object Notation (JSON) string into an object.
 *
 * @example
 * import { parseJSON, toError } from 'fp-ts/lib/Either'
 *
 * assert.deepStrictEqual(parseJSON('{"a":1}', toError).value, { a: 1 })
 * assert.deepStrictEqual(parseJSON('{"a":}', toError).value, new SyntaxError('Unexpected token } in JSON at position 5'))
 *
 * @since 1.16.0
 */
export declare const parseJSON: <L>(s: string, onError: (reason: unknown) => L) => Either<L, unknown>;
/**
 * Converts a JavaScript value to a JavaScript Object Notation (JSON) string.
 *
 * @example
 * import { stringifyJSON, toError } from 'fp-ts/lib/Either'
 *
 * assert.deepStrictEqual(stringifyJSON({ a: 1 }, toError).value, '{"a":1}')
 * const circular: any = { ref: null }
 * circular.ref = circular
 * assert.deepStrictEqual(stringifyJSON(circular, toError).value, new TypeError('Converting circular structure to JSON'))
 *
 * @since 1.16.0
 */
export declare const stringifyJSON: <L>(u: unknown, onError: (reason: unknown) => L) => Either<L, string>;
/**
 * @since 1.0.0
 */
export declare const either: Monad2<URI> & Foldable2v2<URI> & Traversable2v2<URI> & Bifunctor2<URI> & Alt2<URI> & Extend2<URI> & ChainRec2<URI> & MonadThrow2<URI>;
/**
 * @since 1.19.0
 */
export declare function fold<E, A, R>(onLeft: (e: E) => R, onRight: (a: A) => R): (ma: Either<E, A>) => R;
/**
 * @since 1.19.0
 */
export declare function orElse<E, A, M>(f: (e: E) => Either<M, A>): (ma: Either<E, A>) => Either<M, A>;
/**
 * @since 1.19.0
 */
export declare function getOrElse<E, A>(f: (e: E) => A): (ma: Either<E, A>) => A;
/**
 * @since 1.19.0
 */
export declare function elem<A>(E: Eq<A>): (a: A) => <E>(ma: Either<E, A>) => boolean;
/**
 * @since 1.19.0
 */
export declare function getValidation<E>(S: Semigroup<E>): Monad2C<URI, E> & Alt2C<URI, E>;
/**
 * @since 1.19.0
 */
export declare function getValidationSemigroup<E, A>(SE: Semigroup<E>, SA: Semigroup<A>): Semigroup<Either<E, A>>;
/**
 * @since 1.19.0
 */
export declare function getValidationMonoid<E, A>(SE: Semigroup<E>, SA: Monoid<A>): Monoid<Either<E, A>>;
declare const alt: <L, A>(that: () => Either<L, A>) => (fa: Either<L, A>) => Either<L, A>, ap: <L, A>(fa: Either<L, A>) => <B>(fab: Either<L, (a: A) => B>) => Either<L, B>, apFirst: <L, B>(fb: Either<L, B>) => <A>(fa: Either<L, A>) => Either<L, A>, apSecond: <L, B>(fb: Either<L, B>) => <A>(fa: Either<L, A>) => Either<L, B>, bimap: <L, A, M, B>(f: (l: L) => M, g: (a: A) => B) => (fa: Either<L, A>) => Either<M, B>, chain: <L, A, B>(f: (a: A) => Either<L, B>) => (ma: Either<L, A>) => Either<L, B>, chainFirst: <L, A, B>(f: (a: A) => Either<L, B>) => (ma: Either<L, A>) => Either<L, A>, duplicate: <L, A>(ma: Either<L, A>) => Either<L, Either<L, A>>, extend: <L, A, B>(f: (fa: Either<L, A>) => B) => (ma: Either<L, A>) => Either<L, B>, flatten: <L, A>(mma: Either<L, Either<L, A>>) => Either<L, A>, foldMap: <M>(M: Monoid<M>) => <A>(f: (a: A) => M) => <L>(fa: Either<L, A>) => M, map: <A, B>(f: (a: A) => B) => <L>(fa: Either<L, A>) => Either<L, B>, mapLeft: <L, A, M>(f: (l: L) => M) => (fa: Either<L, A>) => Either<M, A>, reduce: <A, B>(b: B, f: (b: B, a: A) => B) => <L>(fa: Either<L, A>) => B, reduceRight: <A, B>(b: B, f: (a: A, b: B) => B) => <L>(fa: Either<L, A>) => B, fromPredicate: {
    <E, A, B extends A>(refinement: Refinement<A, B>, onFalse: (a: A) => E): (a: A) => Either<E, B>;
    <E, A>(predicate: Predicate<A>, onFalse: (a: A) => E): (a: A) => Either<E, A>;
}, filterOrElse: {
    <E, A, B extends A>(refinement: Refinement<A, B>, onFalse: (a: A) => E): (ma: Either<E, A>) => Either<E, B>;
    <E, A>(predicate: Predicate<A>, onFalse: (a: A) => E): (ma: Either<E, A>) => Either<E, A>;
};
export { alt, ap, apFirst, apSecond, bimap, chain, chainFirst, duplicate, extend, flatten, foldMap, map, mapLeft, reduce, reduceRight, fromPredicate, filterOrElse };
/**
 * Lazy version of `fromOption`
 *
 * @since 1.3.0
 */
export declare const fromOptionL: <L>(onNone: Lazy<L>) => <A>(fa: Option<A>) => Either<L, A>;
