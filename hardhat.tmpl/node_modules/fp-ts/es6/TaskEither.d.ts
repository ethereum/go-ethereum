/**
 * @file `TaskEither<L, A>` represents an asynchronous computation that either yields a value of type `A` or fails yielding an
 * error of type `L`. If you want to represent an asynchronous computation that never fails, please see `Task`.
 */
import { Alt2 } from './Alt';
import { Bifunctor2 } from './Bifunctor';
import { Either } from './Either';
import { Lazy, Predicate, Refinement } from './function';
import { IO } from './IO';
import { IOEither } from './IOEither';
import { Monad2 } from './Monad';
import { MonadIO2 } from './MonadIO';
import { MonadTask2 } from './MonadTask';
import { MonadThrow2 } from './MonadThrow';
import { Monoid } from './Monoid';
import { Semigroup } from './Semigroup';
import { Task } from './Task';
declare module './HKT' {
    interface URItoKind2<L, A> {
        TaskEither: TaskEither<L, A>;
    }
}
export declare const URI = "TaskEither";
export declare type URI = typeof URI;
/**
 * @since 1.0.0
 */
export declare class TaskEither<L, A> {
    readonly value: Task<Either<L, A>>;
    readonly _A: A;
    readonly _L: L;
    readonly _URI: URI;
    constructor(value: Task<Either<L, A>>);
    /** Runs the inner `Task` */
    run(): Promise<Either<L, A>>;
    /** @obsolete */
    map<B>(f: (a: A) => B): TaskEither<L, B>;
    /** @obsolete */
    ap<B>(fab: TaskEither<L, (a: A) => B>): TaskEither<L, B>;
    /**
     * Flipped version of `ap`
     * @obsolete
     */
    ap_<B, C>(this: TaskEither<L, (b: B) => C>, fb: TaskEither<L, B>): TaskEither<L, C>;
    /**
     * Combine two (parallel) effectful actions, keeping only the result of the first
     * @since 1.6.0
     * @obsolete
     */
    applyFirst<B>(fb: TaskEither<L, B>): TaskEither<L, A>;
    /**
     * Combine two (parallel) effectful actions, keeping only the result of the second
     * @since 1.5.0
     * @obsolete
     */
    applySecond<B>(fb: TaskEither<L, B>): TaskEither<L, B>;
    /**
     * Combine two (sequential) effectful actions, keeping only the result of the first
     * @since 1.12.0
     * @obsolete
     */
    chainFirst<B>(fb: TaskEither<L, B>): TaskEither<L, A>;
    /**
     * Combine two (sequential) effectful actions, keeping only the result of the second
     * @since 1.12.0
     * @obsolete
     */
    chainSecond<B>(fb: TaskEither<L, B>): TaskEither<L, B>;
    /** @obsolete */
    chain<B>(f: (a: A) => TaskEither<L, B>): TaskEither<L, B>;
    /** @obsolete */
    fold<R>(onLeft: (l: L) => R, onRight: (a: A) => R): Task<R>;
    /**
     * Similar to `fold`, but the result is flattened.
     * @since 1.10.0
     * @obsolete
     */
    foldTask<R>(onLeft: (l: L) => Task<R>, onRight: (a: A) => Task<R>): Task<R>;
    /**
     * Similar to `fold`, but the result is flattened.
     * @since 1.10.0
     * @obsolete
     */
    foldTaskEither<M, B>(onLeft: (l: L) => TaskEither<M, B>, onRight: (a: A) => TaskEither<M, B>): TaskEither<M, B>;
    /**
     * Similar to `fold`, return the value from Right or the given argument if Left.
     * @since 1.17.0
     * @obsolete
     */
    getOrElse(a: A): Task<A>;
    /**
     * @since 1.17.0
     * @obsolete
     */
    getOrElseL(f: (l: L) => A): Task<A>;
    /** @obsolete */
    mapLeft<M>(f: (l: L) => M): TaskEither<M, A>;
    /**
     * Transforms the failure value of the `TaskEither` into a new `TaskEither`
     * @obsolete
     */
    orElse<M>(f: (l: L) => TaskEither<M, A>): TaskEither<M, A>;
    /**
     * @since 1.6.0
     * @obsolete
     */
    alt(fy: TaskEither<L, A>): TaskEither<L, A>;
    /**
     * @since 1.2.0
     * @obsolete
     */
    bimap<V, B>(f: (l: L) => V, g: (a: A) => B): TaskEither<V, B>;
    /**
     * Return `Right` if the given action succeeds, `Left` if it throws
     * @since 1.10.0
     * @obsolete
     */
    attempt<M = L>(): TaskEither<M, Either<L, A>>;
    /**
     * @since 1.11.0
     * @obsolete
     */
    filterOrElse<B extends A>(p: Refinement<A, B>, zero: L): TaskEither<L, B>;
    filterOrElse(p: Predicate<A>, zero: L): TaskEither<L, A>;
    /**
     * @since 1.11.0
     * @obsolete
     */
    filterOrElseL<B extends A>(p: Refinement<A, B>, zero: (a: A) => L): TaskEither<L, B>;
    filterOrElseL(p: Predicate<A>, zero: (a: A) => L): TaskEither<L, A>;
}
/**
 * Use `rightTask`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const right: <L, A>(fa: Task<A>) => TaskEither<L, A>;
/**
 * Use `leftTask`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const left: <L, A>(fl: Task<L>) => TaskEither<L, A>;
/**
 * @since 1.0.0
 */
export declare const fromEither: <L, A>(fa: Either<L, A>) => TaskEither<L, A>;
/**
 * Use `rightIO`
 *
 * @since 1.5.0
 * @deprecated
 */
export declare const fromIO: <L, A>(fa: IO<A>) => TaskEither<L, A>;
/**
 * Use `left2v`
 *
 * @since 1.3.0
 * @deprecated
 */
export declare const fromLeft: <L, A>(l: L) => TaskEither<L, A>;
/**
 * @since 1.6.0
 */
export declare const fromIOEither: <L, A>(fa: IOEither<L, A>) => TaskEither<L, A>;
/**
 * @since 1.9.0
 */
export declare const getSemigroup: <L, A>(S: Semigroup<A>) => Semigroup<TaskEither<L, A>>;
/**
 * @since 1.9.0
 */
export declare const getApplySemigroup: <L, A>(S: Semigroup<A>) => Semigroup<TaskEither<L, A>>;
/**
 * @since 1.9.0
 */
export declare const getApplyMonoid: <L, A>(M: Monoid<A>) => Monoid<TaskEither<L, A>>;
/**
 * Transforms a `Promise` into a `TaskEither`, catching the possible error.
 *
 * @example
 * import { createHash } from 'crypto'
 * import { TaskEither, tryCatch } from 'fp-ts/lib/TaskEither'
 * import { createReadStream } from 'fs'
 * import { left } from 'fp-ts/lib/Either'
 *
 * const md5 = (path: string): TaskEither<string, string> => {
 *   const mkHash = (p: string) =>
 *     new Promise<string>((resolve, reject) => {
 *       const hash = createHash('md5')
 *       const rs = createReadStream(p)
 *       rs.on('error', (error: Error) => reject(error.message))
 *       rs.on('data', (chunk: string) => hash.update(chunk))
 *       rs.on('end', () => {
 *         return resolve(hash.digest('hex'))
 *       })
 *     })
 *   return tryCatch(() => mkHash(path), message => `cannot create md5 hash: ${String(message)}`)
 * }
 *
 * md5('foo')
 *   .run()
 *   .then(x => {
 *     assert.deepStrictEqual(x, left(`cannot create md5 hash: ENOENT: no such file or directory, open 'foo'`))
 *   })
 *
 *
 * @since 1.0.0
 */
export declare const tryCatch: <L, A>(f: Lazy<Promise<A>>, onrejected: (reason: unknown) => L) => TaskEither<L, A>;
/**
 * Convert a node style callback function to one returning a `TaskEither`
 *
 * **Note**. If the function `f` admits multiple overloadings, `taskify` will pick last one. If you want a different
 * behaviour, add an explicit type annotation
 *
 * ```ts
 * // readFile admits multiple overloadings
 *
 * // const readFile: (a: string) => TaskEither<NodeJS.ErrnoException, Buffer>
 * const readFile = taskify(fs.readFile)
 *
 * const readFile2: (filename: string, encoding: string) => TaskEither<NodeJS.ErrnoException, Buffer> = taskify(
 *   fs.readFile
 * )
 * ```
 *
 * @example
 * import { taskify } from 'fp-ts/lib/TaskEither'
 * import * as fs from 'fs'
 *
 * // const stat: (a: string | Buffer) => TaskEither<NodeJS.ErrnoException, fs.Stats>
 * const stat = taskify(fs.stat)
 * assert.strictEqual(stat.length, 0)
 *
 *
 * @since 1.5.0
 */
export declare function taskify<L, R>(f: (cb: (e: L | null | undefined, r?: R) => void) => void): () => TaskEither<L, R>;
export declare function taskify<A, L, R>(f: (a: A, cb: (e: L | null | undefined, r?: R) => void) => void): (a: A) => TaskEither<L, R>;
export declare function taskify<A, B, L, R>(f: (a: A, b: B, cb: (e: L | null | undefined, r?: R) => void) => void): (a: A, b: B) => TaskEither<L, R>;
export declare function taskify<A, B, C, L, R>(f: (a: A, b: B, c: C, cb: (e: L | null | undefined, r?: R) => void) => void): (a: A, b: B, c: C) => TaskEither<L, R>;
export declare function taskify<A, B, C, D, L, R>(f: (a: A, b: B, c: C, d: D, cb: (e: L | null | undefined, r?: R) => void) => void): (a: A, b: B, c: C, d: D) => TaskEither<L, R>;
export declare function taskify<A, B, C, D, E, L, R>(f: (a: A, b: B, c: C, d: D, e: E, cb: (e: L | null | undefined, r?: R) => void) => void): (a: A, b: B, c: C, d: D, e: E) => TaskEither<L, R>;
/**
 * Make sure that a resource is cleaned up in the event of an exception. The
 * release action is called regardless of whether the body action throws or
 * returns.
 *
 * @since 1.10.0
 */
export declare const bracket: <L, A, B>(acquire: TaskEither<L, A>, use: (a: A) => TaskEither<L, B>, release: (a: A, e: Either<L, B>) => TaskEither<L, void>) => TaskEither<L, B>;
/**
 * @since 1.0.0
 */
export declare const taskEither: Monad2<URI> & Bifunctor2<URI> & Alt2<URI> & MonadIO2<URI> & MonadTask2<URI> & MonadThrow2<URI>;
/**
 * Like `TaskEither` but `ap` is sequential
 *
 * @since 1.10.0
 */
export declare const taskEitherSeq: typeof taskEither;
/**
 * @since 1.19.0
 */
export declare function right2v<E = never, A = never>(a: A): TaskEither<E, A>;
/**
 * @since 1.19.0
 */
export declare function left2v<E = never, A = never>(e: E): TaskEither<E, A>;
/**
 * @since 1.19.0
 */
export declare function rightIO<E = never, A = never>(ma: IO<A>): TaskEither<E, A>;
/**
 * @since 1.19.0
 */
export declare function leftIO<E = never, A = never>(me: IO<E>): TaskEither<E, A>;
/**
 * @since 1.19.0
 */
export declare function rightTask<E = never, A = never>(ma: Task<A>): TaskEither<E, A>;
/**
 * @since 1.19.0
 */
export declare function leftTask<E = never, A = never>(me: Task<E>): TaskEither<E, A>;
/**
 * @since 1.19.0
 */
export declare function fold<E, A, R>(onLeft: (e: E) => Task<R>, onRight: (a: A) => Task<R>): (ma: TaskEither<E, A>) => Task<R>;
/**
 * @since 1.19.0
 */
export declare function getOrElse<E, A>(f: (e: E) => Task<A>): (ma: TaskEither<E, A>) => Task<A>;
/**
 * @since 1.19.0
 */
export declare function orElse<E, A, M>(f: (e: E) => TaskEither<M, A>): (ma: TaskEither<E, A>) => TaskEither<M, A>;
declare const alt: <L, A>(that: () => TaskEither<L, A>) => (fa: TaskEither<L, A>) => TaskEither<L, A>, ap: <L, A>(fa: TaskEither<L, A>) => <B>(fab: TaskEither<L, (a: A) => B>) => TaskEither<L, B>, apFirst: <L, B>(fb: TaskEither<L, B>) => <A>(fa: TaskEither<L, A>) => TaskEither<L, A>, apSecond: <L, B>(fb: TaskEither<L, B>) => <A>(fa: TaskEither<L, A>) => TaskEither<L, B>, bimap: <L, A, M, B>(f: (l: L) => M, g: (a: A) => B) => (fa: TaskEither<L, A>) => TaskEither<M, B>, chain: <L, A, B>(f: (a: A) => TaskEither<L, B>) => (ma: TaskEither<L, A>) => TaskEither<L, B>, chainFirst: <L, A, B>(f: (a: A) => TaskEither<L, B>) => (ma: TaskEither<L, A>) => TaskEither<L, A>, flatten: <L, A>(mma: TaskEither<L, TaskEither<L, A>>) => TaskEither<L, A>, map: <A, B>(f: (a: A) => B) => <L>(fa: TaskEither<L, A>) => TaskEither<L, B>, mapLeft: <L, A, M>(f: (l: L) => M) => (fa: TaskEither<L, A>) => TaskEither<M, A>, fromOption: <E>(onNone: () => E) => <A>(ma: import("./Option").Option<A>) => TaskEither<E, A>, fromPredicate: {
    <E, A, B extends A>(refinement: Refinement<A, B>, onFalse: (a: A) => E): (a: A) => TaskEither<E, B>;
    <E, A>(predicate: Predicate<A>, onFalse: (a: A) => E): (a: A) => TaskEither<E, A>;
}, filterOrElse: {
    <E, A, B extends A>(refinement: Refinement<A, B>, onFalse: (a: A) => E): (ma: TaskEither<E, A>) => TaskEither<E, B>;
    <E, A>(predicate: Predicate<A>, onFalse: (a: A) => E): (ma: TaskEither<E, A>) => TaskEither<E, A>;
};
export { alt, ap, apFirst, apSecond, bimap, chain, chainFirst, flatten, map, mapLeft, fromOption, fromPredicate, filterOrElse };
