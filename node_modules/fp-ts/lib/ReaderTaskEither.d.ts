import { Alt3 } from './Alt';
import { Bifunctor3 } from './Bifunctor';
import { Either } from './Either';
import { IO } from './IO';
import { IOEither } from './IOEither';
import { Monad3 } from './Monad';
import { MonadIO3 } from './MonadIO';
import { MonadTask3 } from './MonadTask';
import { MonadThrow3 } from './MonadThrow';
import { Reader } from './Reader';
import { Task } from './Task';
import * as taskEither from './TaskEither';
import TaskEither = taskEither.TaskEither;
declare module './HKT' {
    interface URItoKind3<U, L, A> {
        ReaderTaskEither: ReaderTaskEither<U, L, A>;
    }
}
export declare const URI = "ReaderTaskEither";
export declare type URI = typeof URI;
/**
 * @since 1.6.0
 */
export declare class ReaderTaskEither<E, L, A> {
    readonly value: (e: E) => TaskEither<L, A>;
    readonly _A: A;
    readonly _L: L;
    readonly _U: E;
    readonly _URI: URI;
    constructor(value: (e: E) => TaskEither<L, A>);
    /** Runs the inner `TaskEither` */
    run(e: E): Promise<Either<L, A>>;
    /** @obsolete */
    map<B>(f: (a: A) => B): ReaderTaskEither<E, L, B>;
    /** @obsolete */
    ap<B>(fab: ReaderTaskEither<E, L, (a: A) => B>): ReaderTaskEither<E, L, B>;
    /**
     * Flipped version of `ap`
     * @obsolete
     */
    ap_<B, C>(this: ReaderTaskEither<E, L, (b: B) => C>, fb: ReaderTaskEither<E, L, B>): ReaderTaskEither<E, L, C>;
    /**
     * Combine two effectful actions, keeping only the result of the first
     * @obsolete
     */
    applyFirst<B>(fb: ReaderTaskEither<E, L, B>): ReaderTaskEither<E, L, A>;
    /**
     * Combine two effectful actions, keeping only the result of the second
     * @obsolete
     */
    applySecond<B>(fb: ReaderTaskEither<E, L, B>): ReaderTaskEither<E, L, B>;
    /** @obsolete */
    chain<B>(f: (a: A) => ReaderTaskEither<E, L, B>): ReaderTaskEither<E, L, B>;
    /** @obsolete */
    fold<R>(left: (l: L) => R, right: (a: A) => R): Reader<E, Task<R>>;
    /** @obsolete */
    mapLeft<M>(f: (l: L) => M): ReaderTaskEither<E, M, A>;
    /**
     * Transforms the failure value of the `ReaderTaskEither` into a new `ReaderTaskEither`
     * @obsolete
     */
    orElse<M>(f: (l: L) => ReaderTaskEither<E, M, A>): ReaderTaskEither<E, M, A>;
    /** @obsolete */
    alt(fy: ReaderTaskEither<E, L, A>): ReaderTaskEither<E, L, A>;
    /** @obsolete */
    bimap<V, B>(f: (l: L) => V, g: (a: A) => B): ReaderTaskEither<E, V, B>;
    /**
     * @since 1.6.1
     * @obsolete
     */
    local<E2 = E>(f: (e: E2) => E): ReaderTaskEither<E2, L, A>;
}
/**
 * @since 1.6.0
 */
export declare const ask: <E, L>() => ReaderTaskEither<E, L, E>;
/**
 * @since 1.6.0
 */
export declare const asks: <E, L, A>(f: (e: E) => A) => ReaderTaskEither<E, L, A>;
/**
 * @since 1.6.0
 */
export declare const local: <E, E2 = E>(f: (e: E2) => E) => <L, A>(fa: ReaderTaskEither<E, L, A>) => ReaderTaskEither<E2, L, A>;
/**
 * Use `rightTask`
 *
 * @since 1.6.0
 * @deprecated
 */
export declare const right: <E, L, A>(fa: Task<A>) => ReaderTaskEither<E, L, A>;
/**
 * Use `leftTask`
 *
 * @since 1.6.0
 * @deprecated
 */
export declare const left: <E, L, A>(fa: Task<L>) => ReaderTaskEither<E, L, A>;
/**
 * @since 1.6.0
 */
export declare const fromTaskEither: <E, L, A>(fa: taskEither.TaskEither<L, A>) => ReaderTaskEither<E, L, A>;
/**
 * Use `rightReader`
 *
 * @since 1.6.0
 * @deprecated
 */
export declare const fromReader: <E, L, A>(fa: Reader<E, A>) => ReaderTaskEither<E, L, A>;
/**
 * @since 1.6.0
 */
export declare const fromEither: <E, L, A>(fa: Either<L, A>) => ReaderTaskEither<E, L, A>;
/**
 * Use `rightIO`
 *
 * @since 1.6.0
 * @deprecated
 */
export declare const fromIO: <E, L, A>(fa: IO<A>) => ReaderTaskEither<E, L, A>;
/**
 * Use `left2v`
 *
 * @since 1.6.0
 * @deprecated
 */
export declare const fromLeft: <E, L, A>(l: L) => ReaderTaskEither<E, L, A>;
/**
 * @since 1.6.0
 */
export declare const fromIOEither: <E, L, A>(fa: IOEither<L, A>) => ReaderTaskEither<E, L, A>;
/**
 * @since 1.6.0
 */
export declare const tryCatch: <E, L, A>(f: (e: E) => Promise<A>, onrejected: (reason: unknown, e: E) => L) => ReaderTaskEither<E, L, A>;
/**
 * @since 1.6.0
 */
export declare const readerTaskEither: Monad3<URI> & Bifunctor3<URI> & Alt3<URI> & MonadIO3<URI> & MonadTask3<URI> & MonadThrow3<URI>;
/**
 * Like `readerTaskEither` but `ap` is sequential
 * @since 1.10.0
 */
export declare const readerTaskEitherSeq: typeof readerTaskEither;
/**
 * @since 1.19.0
 */
export declare const left2v: <R, E = never, A = never>(e: E) => ReaderTaskEither<R, E, A>;
/**
 * @since 1.19.0
 */
export declare const right2v: <R, E = never, A = never>(a: A) => ReaderTaskEither<R, E, A>;
/**
 * @since 1.19.0
 */
export declare const rightReader: <R, E = never, A = never>(ma: Reader<R, A>) => ReaderTaskEither<R, E, A>;
/**
 * @since 1.19.0
 */
export declare const rightIO: <R, E = never, A = never>(ma: IO<A>) => ReaderTaskEither<R, E, A>;
/**
 * @since 1.19.0
 */
export declare const rightTask: <R, E = never, A = never>(fa: Task<A>) => ReaderTaskEither<R, E, A>;
/**
 * @since 1.19.0
 */
export declare const leftTask: <R, E = never, A = never>(fa: Task<E>) => ReaderTaskEither<R, E, A>;
declare const alt: <U, L, A>(that: () => ReaderTaskEither<U, L, A>) => (fa: ReaderTaskEither<U, L, A>) => ReaderTaskEither<U, L, A>, ap: <U, L, A>(fa: ReaderTaskEither<U, L, A>) => <B>(fab: ReaderTaskEither<U, L, (a: A) => B>) => ReaderTaskEither<U, L, B>, apFirst: <U, L, B>(fb: ReaderTaskEither<U, L, B>) => <A>(fa: ReaderTaskEither<U, L, A>) => ReaderTaskEither<U, L, A>, apSecond: <U, L, B>(fb: ReaderTaskEither<U, L, B>) => <A>(fa: ReaderTaskEither<U, L, A>) => ReaderTaskEither<U, L, B>, bimap: <L, A, M, B>(f: (l: L) => M, g: (a: A) => B) => <U>(fa: ReaderTaskEither<U, L, A>) => ReaderTaskEither<U, M, B>, chain: <U, L, A, B>(f: (a: A) => ReaderTaskEither<U, L, B>) => (ma: ReaderTaskEither<U, L, A>) => ReaderTaskEither<U, L, B>, chainFirst: <U, L, A, B>(f: (a: A) => ReaderTaskEither<U, L, B>) => (ma: ReaderTaskEither<U, L, A>) => ReaderTaskEither<U, L, A>, flatten: <U, L, A>(mma: ReaderTaskEither<U, L, ReaderTaskEither<U, L, A>>) => ReaderTaskEither<U, L, A>, map: <A, B>(f: (a: A) => B) => <U, L>(fa: ReaderTaskEither<U, L, A>) => ReaderTaskEither<U, L, B>, mapLeft: <L, A, M>(f: (l: L) => M) => <U>(fa: ReaderTaskEither<U, L, A>) => ReaderTaskEither<U, M, A>, fromOption: <E>(onNone: () => E) => <U, A>(ma: import("./Option").Option<A>) => ReaderTaskEither<U, E, A>, fromPredicate: {
    <E, A, B extends A>(refinement: import("./function").Refinement<A, B>, onFalse: (a: A) => E): <U>(a: A) => ReaderTaskEither<U, E, B>;
    <E, A>(predicate: import("./function").Predicate<A>, onFalse: (a: A) => E): <U>(a: A) => ReaderTaskEither<U, E, A>;
}, filterOrElse: {
    <E, A, B extends A>(refinement: import("./function").Refinement<A, B>, onFalse: (a: A) => E): <U>(ma: ReaderTaskEither<U, E, A>) => ReaderTaskEither<U, E, B>;
    <E, A>(predicate: import("./function").Predicate<A>, onFalse: (a: A) => E): <U>(ma: ReaderTaskEither<U, E, A>) => ReaderTaskEither<U, E, A>;
};
export { alt, ap, apFirst, apSecond, bimap, chain, chainFirst, flatten, map, mapLeft, fromOption, fromPredicate, filterOrElse };
