/**
 * @file `IO<A>` represents a non-deterministic synchronous computation that can cause side effects, yields a value of
 * type `A` and **never fails**. If you want to represent a synchronous computation that may fail, please see
 * `IOEither`.
 *
 * IO actions are terminated by calling their `run()` method that executes the computation and returns the result.
 * Ideally each application should call `run()` only once for a root value of type `Task` or `IO` that represents the entire
 * application. However, this might vary a bit depending on how you construct your application.  An application
 * framework with fp-ts types might take care of calling `run()` for you, while another application framework without
 * fp-ts typing might force you to call `run()` at multiple locations whenever the framework demands less strictly typed
 * values as inputs for its method calls.
 *
 * Below are some basic examples of how you can wrap values and function calls with `IO`.
 *
 * ```ts
 * import { IO, io } from 'fp-ts/lib/IO'
 *
 * const constant: IO<number> = io.of(123);
 * constant.run()  // returns 123
 *
 * const random: IO<number> = new IO(() => Math.random())
 * random.run()  // returns a random number
 * random.run()  // returns another random number
 *
 * const log = (...args): IO<void> => new IO(() => console.log(...args));
 * log('hello world').run()  // returns undefined and outputs "hello world" to console
 * ```
 *
 * In the example above we implemented type safe alternatives for `Math.random()` and `console.log()`. The main
 * motivation was to explain how you can wrap values. However, fp-ts provides type safe alternatives for such basic
 * tools through `Console` and `Random` modules. So you don't need to constantly redefine them.
 *
 * The next code snippet below is an example of a case where type safety affects the end result. Using `console.log()`
 * directly would break the code, resulting in both logging actions being executed when the value is not `null`.  You
 * can confirm this by removing `.run()` from the end of the example code and replacing calls to `log()` with
 * standard`console.log()`.
 *
 * ```ts
 * import { IO } from 'fp-ts/lib/IO'
 * import { fromNullable } from 'fp-ts/lib/Option'
 * import { log } from 'fp-ts/lib/Console'
 *
 * const logger = (input: number|null) => fromNullable(input).fold(
 *   log('Received null'),
 *   (value) => log(`Received ${value}`),
 * );
 *
 * logger(123).run();  // returns undefined and outputs "Received 123" to console
 * ```
 *
 * In addition to creating IO actions we need a way to combine them to build the application.  For example we might have
 * several `IO<void>` actions that only cause side effects without returning a result.  We  can combine several `IO<void>`
 * actions into one for sequential execution with `sequence_(io, array)` as follows. This is useful when you care about
 * the execution order but do not care about the results.
 *
 * ```ts
 * import { IO, io } from 'fp-ts/lib/IO'
 * import { array } from 'fp-ts/lib/Array'
 * import { sequence_ } from 'fp-ts/lib/Foldable2v'
 * import { log } from 'fp-ts/lib/Console'
 *
 * const logGiraffe: IO<void> = log('giraffe');
 * const logZebra: IO<void> = log('zebra');
 *
 * const logGiraffeThenZebra: IO<void> = sequence_(io, array)([ logGiraffe, logZebra ])
 *
 * logGiraffeThenZebra.run();  // returns undefined and outputs words "giraffe" and "zebra" to console
 * ```
 *
 * We might also have several IO actions that yield some values that we want to capture. We can combine them by
 * using `sequenceS(io)` over an object matching the structure of the expected result. This is useful when you care
 * about the results but do not care about the execution order.
 *
 * ```ts
 * import { IO, io } from 'fp-ts/lib/IO'
 * import { sequenceS } from 'fp-ts/lib/Apply'
 *
 * interface Result {
 *   name: string,
 *   age: number,
 * }
 *
 * const computations: { [K in keyof Result]: IO<Result[K]> } = {
 *   name: io.of('Aristotle'),
 *   age: io.of(60),
 * }
 *
 * const computation: IO<Result> = sequenceS(io)(computations)
 *
 * computation.run()  // returns { name: 'Aristotle', age: 60 }
 * ```
 */
import { Monad1 } from './Monad';
import { Monoid } from './Monoid';
import { Semigroup } from './Semigroup';
import { Lazy } from './function';
import { MonadIO1 } from './MonadIO';
declare module './HKT' {
    interface URItoKind<A> {
        IO: IO<A>;
    }
}
export declare const URI = "IO";
export declare type URI = typeof URI;
/**
 * @since 1.0.0
 */
export declare class IO<A> {
    readonly run: Lazy<A>;
    readonly _A: A;
    readonly _URI: URI;
    constructor(run: Lazy<A>);
    /** @obsolete */
    map<B>(f: (a: A) => B): IO<B>;
    /** @obsolete */
    ap<B>(fab: IO<(a: A) => B>): IO<B>;
    /**
     * Flipped version of `ap`
     * @obsolete
     */
    ap_<B, C>(this: IO<(b: B) => C>, fb: IO<B>): IO<C>;
    /**
     * Combine two effectful actions, keeping only the result of the first
     * @since 1.6.0
     * @obsolete
     */
    applyFirst<B>(fb: IO<B>): IO<A>;
    /**
     * Combine two effectful actions, keeping only the result of the second
     * @since 1.5.0
     * @obsolete
     */
    applySecond<B>(fb: IO<B>): IO<B>;
    /** @obsolete */
    chain<B>(f: (a: A) => IO<B>): IO<B>;
    inspect(): string;
    toString(): string;
}
/**
 * @since 1.0.0
 */
export declare const getSemigroup: <A>(S: Semigroup<A>) => Semigroup<IO<A>>;
/**
 * @since 1.0.0
 */
export declare const getMonoid: <A>(M: Monoid<A>) => Monoid<IO<A>>;
/**
 * @since 1.0.0
 */
export declare const io: Monad1<URI> & MonadIO1<URI>;
declare const ap: <A>(fa: IO<A>) => <B>(fab: IO<(a: A) => B>) => IO<B>, apFirst: <B>(fb: IO<B>) => <A>(fa: IO<A>) => IO<A>, apSecond: <B>(fb: IO<B>) => <A>(fa: IO<A>) => IO<B>, chain: <A, B>(f: (a: A) => IO<B>) => (ma: IO<A>) => IO<B>, chainFirst: <A, B>(f: (a: A) => IO<B>) => (ma: IO<A>) => IO<A>, flatten: <A>(mma: IO<IO<A>>) => IO<A>, map: <A, B>(f: (a: A) => B) => (fa: IO<A>) => IO<B>;
export { ap, apFirst, apSecond, chain, chainFirst, flatten, map };
