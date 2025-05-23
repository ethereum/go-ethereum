/**
 * @file The `Validation` functor, used for applicative validation
 *
 * The `Applicative` instance collects multiple failures in an arbitrary `Semigroup`.
 *
 * Adapted from https://github.com/purescript/purescript-validation
 */
import { Alt2C } from './Alt';
import { Applicative2C } from './Applicative';
import { Bifunctor2 } from './Bifunctor';
import { Compactable2C } from './Compactable';
import { Either } from './Either';
import { Filterable2C } from './Filterable';
import { Foldable2v2 } from './Foldable2v';
import { Predicate, Refinement, Lazy } from './function';
import { Functor2 } from './Functor';
import { Monad2C } from './Monad';
import { Monoid } from './Monoid';
import { Semigroup } from './Semigroup';
import { Eq } from './Eq';
import { Traversable2v2 } from './Traversable2v';
import { Witherable2C } from './Witherable';
import { MonadThrow2C } from './MonadThrow';
import { Show } from './Show';
declare module './HKT' {
    interface URItoKind2<L, A> {
        Validation: Validation<L, A>;
    }
}
export declare const URI = "Validation";
export declare type URI = typeof URI;
/**
 * @example
 * import { Validation, getApplicative, success, failure } from 'fp-ts/lib/Validation'
 * import { NonEmptyArray, getSemigroup } from 'fp-ts/lib/NonEmptyArray'
 *
 * interface Person {
 *   readonly name: string
 *   readonly age: number
 * }
 *
 * // curried constructor
 * const person = (name: string) => (age: number): Person => ({ name, age })
 *
 * // validators
 * function validateName(input: string): Validation<NonEmptyArray<string>, string> {
 *   return input.length === 0 ? failure(new NonEmptyArray('Invalid name: empty string', [])) : success(input)
 * }
 * function validateAge(input: string): Validation<NonEmptyArray<string>, number> {
 *   const n = parseFloat(input)
 *   if (isNaN(n)) {
 *     return failure(new NonEmptyArray(`Invalid age: not a number ${input}`, []))
 *   }
 *   return n % 1 !== 0 ? failure(new NonEmptyArray(`Invalid age: not an integer ${n}`, [])) : success(n)
 * }
 *
 * // get an `Applicative` instance for Validation<NonEmptyArray<string>, ?>
 * const A = getApplicative(getSemigroup<string>())
 *
 * function validatePerson(input: Record<string, string>): Validation<NonEmptyArray<string>, Person> {
 *   return A.ap(validateName(input['name']).map(person), validateAge(input['age']))
 * }
 *
 * assert.deepStrictEqual(validatePerson({ name: '', age: '1.2' }), failure(new NonEmptyArray("Invalid name: empty string", ["Invalid age: not an integer 1.2"])))
 *
 * assert.deepStrictEqual(validatePerson({ name: 'Giulio', age: '44' }), success({ "name": "Giulio", "age": 44 }))
 *
 * @since 1.0.0
 */
export declare type Validation<L, A> = Failure<L, A> | Success<L, A>;
export declare class Failure<L, A> {
    readonly value: L;
    readonly _tag: 'Failure';
    readonly _A: A;
    readonly _L: L;
    readonly _URI: URI;
    constructor(value: L);
    /** @obsolete */
    map<B>(f: (a: A) => B): Validation<L, B>;
    /** @obsolete */
    bimap<V, B>(f: (l: L) => V, g: (a: A) => B): Validation<V, B>;
    /** @obsolete */
    reduce<B>(b: B, f: (b: B, a: A) => B): B;
    /** @obsolete */
    fold<B>(failure: (l: L) => B, success: (a: A) => B): B;
    /**
     * Returns the value from this `Success` or the given argument if this is a `Failure`
     * @obsolete
     */
    getOrElse(a: A): A;
    /**
     * Returns the value from this `Success` or the result of given argument if this is a `Failure`
     * @obsolete
     */
    getOrElseL(f: (l: L) => A): A;
    /** @obsolete */
    mapFailure<M>(f: (l: L) => M): Validation<M, A>;
    /** @obsolete */
    swap(): Validation<A, L>;
    inspect(): string;
    toString(): string;
    /**
     * Returns `true` if the validation is an instance of `Failure`, `false` otherwise
     * @obsolete
     */
    isFailure(): this is Failure<L, A>;
    /**
     * Returns `true` if the validation is an instance of `Success`, `false` otherwise
     * @obsolete
     */
    isSuccess(): this is Success<L, A>;
}
export declare class Success<L, A> {
    readonly value: A;
    readonly _tag: 'Success';
    readonly _A: A;
    readonly _L: L;
    readonly _URI: URI;
    constructor(value: A);
    map<B>(f: (a: A) => B): Validation<L, B>;
    bimap<V, B>(f: (l: L) => V, g: (a: A) => B): Validation<V, B>;
    reduce<B>(b: B, f: (b: B, a: A) => B): B;
    fold<B>(failure: (l: L) => B, success: (a: A) => B): B;
    getOrElse(a: A): A;
    getOrElseL(f: (l: L) => A): A;
    mapFailure<M>(f: (l: L) => M): Validation<M, A>;
    swap(): Validation<A, L>;
    inspect(): string;
    toString(): string;
    isFailure(): this is Failure<L, A>;
    isSuccess(): this is Success<L, A>;
}
/**
 * @since 1.17.0
 */
export declare const getShow: <L, A>(SL: Show<L>, SA: Show<A>) => Show<Validation<L, A>>;
/**
 * Use `getEq`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const getSetoid: <L, A>(EL: Eq<L>, EA: Eq<A>) => Eq<Validation<L, A>>;
/**
 * @since 1.19.0
 */
export declare function getEq<L, A>(EL: Eq<L>, EA: Eq<A>): Eq<Validation<L, A>>;
/**
 * @since 1.0.0
 */
export declare const success: <L, A>(a: A) => Validation<L, A>;
/**
 * @example
 * import { Validation, success, failure, getApplicative } from 'fp-ts/lib/Validation'
 * import { getArraySemigroup } from 'fp-ts/lib/Semigroup'
 *
 * interface Person {
 *   name: string
 *   age: number
 * }
 *
 * const person = (name: string) => (age: number): Person => ({ name, age })
 *
 * const validateName = (name: string): Validation<string[], string> =>
 *   name.length === 0 ? failure(['invalid name']) : success(name)
 *
 * const validateAge = (age: number): Validation<string[], number> =>
 *   age > 0 && age % 1 === 0 ? success(age) : failure(['invalid age'])
 *
 * const A = getApplicative(getArraySemigroup<string>())
 *
 * const validatePerson = (name: string, age: number): Validation<string[], Person> =>
 *   A.ap(A.map(validateName(name), person), validateAge(age))
 *
 * assert.deepStrictEqual(validatePerson('Nicolas Bourbaki', 45), success({ "name": "Nicolas Bourbaki", "age": 45 }))
 * assert.deepStrictEqual(validatePerson('Nicolas Bourbaki', -1), failure(["invalid age"]))
 * assert.deepStrictEqual(validatePerson('', 0), failure(["invalid name", "invalid age"]))
 *
 * @since 1.0.0
 */
export declare const getApplicative: <L>(S: Semigroup<L>) => Applicative2C<"Validation", L>;
/**
 * **Note**: This function is here just to avoid switching to / from `Either`
 *
 * @since 1.0.0
 */
export declare const getMonad: <L>(S: Semigroup<L>) => Monad2C<"Validation", L>;
/**
 * @since 1.0.0
 */
export declare const failure: <L, A>(l: L) => Validation<L, A>;
/**
 * @since 1.0.0
 */
export declare function fromPredicate<L, A, B extends A>(predicate: Refinement<A, B>, f: (a: A) => L): (a: A) => Validation<L, B>;
export declare function fromPredicate<L, A>(predicate: Predicate<A>, f: (a: A) => L): (a: A) => Validation<L, A>;
/**
 * @since 1.0.0
 */
export declare const fromEither: <L, A>(e: Either<L, A>) => Validation<L, A>;
/**
 * Constructs a new `Validation` from a function that might throw
 *
 * @example
 * import { Validation, failure, success, tryCatch } from 'fp-ts/lib/Validation'
 *
 * const unsafeHead = <A>(as: Array<A>): A => {
 *   if (as.length > 0) {
 *     return as[0]
 *   } else {
 *     throw new Error('empty array')
 *   }
 * }
 *
 * const head = <A>(as: Array<A>): Validation<Error, A> => {
 *   return tryCatch(() => unsafeHead(as), e => (e instanceof Error ? e : new Error('unknown error')))
 * }
 *
 * assert.deepStrictEqual(head([]), failure(new Error('empty array')))
 * assert.deepStrictEqual(head([1, 2, 3]), success(1))
 *
 * @since 1.16.0
 */
export declare const tryCatch: <L, A>(f: Lazy<A>, onError: (e: unknown) => L) => Validation<L, A>;
/**
 * @since 1.0.0
 */
export declare const getSemigroup: <L, A>(SL: Semigroup<L>, SA: Semigroup<A>) => Semigroup<Validation<L, A>>;
/**
 * @since 1.0.0
 */
export declare const getMonoid: <L, A>(SL: Semigroup<L>, SA: Monoid<A>) => Monoid<Validation<L, A>>;
/**
 * @since 1.0.0
 */
export declare const getAlt: <L>(S: Semigroup<L>) => Alt2C<"Validation", L>;
/**
 * Returns `true` if the validation is an instance of `Failure`, `false` otherwise
 *
 * @since 1.0.0
 */
export declare const isFailure: <L, A>(fa: Validation<L, A>) => fa is Failure<L, A>;
/**
 * Returns `true` if the validation is an instance of `Success`, `false` otherwise
 *
 * @since 1.0.0
 */
export declare const isSuccess: <L, A>(fa: Validation<L, A>) => fa is Success<L, A>;
/**
 * Builds `Compactable` instance for `Validation` given `Monoid` for the failure side
 *
 * @since 1.7.0
 */
export declare function getCompactable<L>(ML: Monoid<L>): Compactable2C<URI, L>;
/**
 * Builds `Filterable` instance for `Validation` given `Monoid` for the left side
 *
 * @since 1.7.0
 */
export declare function getFilterable<L>(ML: Monoid<L>): Filterable2C<URI, L>;
/**
 * Builds `Witherable` instance for `Validation` given `Monoid` for the left side
 *
 * @since 1.7.0
 */
export declare function getWitherable<L>(ML: Monoid<L>): Witherable2C<URI, L>;
/**
 * @since 1.16.0
 */
export declare const getMonadThrow: <L>(S: Semigroup<L>) => MonadThrow2C<"Validation", L>;
/**
 * @since 1.0.0
 */
export declare const validation: Functor2<URI> & Bifunctor2<URI> & Foldable2v2<URI> & Traversable2v2<URI>;
