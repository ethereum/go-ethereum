/**
 * @file `Task<A>` represents an asynchronous computation that yields a value of type `A` and **never fails**.
 * If you want to represent an asynchronous computation that may fail, please see `TaskEither`.
 */
import { Either } from './Either';
import { Lazy } from './function';
import { IO } from './IO';
import { Monad1 } from './Monad';
import { MonadIO1 } from './MonadIO';
import { MonadTask1 } from './MonadTask';
import { Monoid } from './Monoid';
import { Semigroup } from './Semigroup';
declare module './HKT' {
    interface URItoKind<A> {
        Task: Task<A>;
    }
}
export declare const URI = "Task";
export declare type URI = typeof URI;
/**
 * @since 1.0.0
 */
export declare class Task<A> {
    readonly run: Lazy<Promise<A>>;
    readonly _A: A;
    readonly _URI: URI;
    constructor(run: Lazy<Promise<A>>);
    /** @obsolete */
    map<B>(f: (a: A) => B): Task<B>;
    /** @obsolete */
    ap<B>(fab: Task<(a: A) => B>): Task<B>;
    /**
     * Flipped version of `ap`
     * @obsolete
     */
    ap_<B, C>(this: Task<(b: B) => C>, fb: Task<B>): Task<C>;
    /**
     * Combine two effectful actions, keeping only the result of the first
     * @since 1.6.0
     * @obsolete
     */
    applyFirst<B>(fb: Task<B>): Task<A>;
    /**
     * Combine two effectful actions, keeping only the result of the second
     * @since 1.5.0
     * @obsolete
     */
    applySecond<B>(fb: Task<B>): Task<B>;
    /** @obsolete */
    chain<B>(f: (a: A) => Task<B>): Task<B>;
    inspect(): string;
    toString(): string;
}
/**
 * @since 1.0.0
 */
export declare const getRaceMonoid: <A = never>() => Monoid<Task<A>>;
/**
 * @since 1.0.0
 */
export declare const getSemigroup: <A>(S: Semigroup<A>) => Semigroup<Task<A>>;
/**
 * @since 1.0.0
 */
export declare const getMonoid: <A>(M: Monoid<A>) => Monoid<Task<A>>;
/**
 * @since 1.0.0
 */
export declare const tryCatch: <L, A>(f: Lazy<Promise<A>>, onrejected: (reason: unknown) => L) => Task<Either<L, A>>;
/**
 * Lifts an IO action into a Task
 *
 * @since 1.0.0
 */
export declare const fromIO: <A>(io: IO<A>) => Task<A>;
/**
 * Use `delay2v`
 *
 * @since 1.7.0
 * @deprecated
 */
export declare const delay: <A>(millis: number, a: A) => Task<A>;
/**
 * @since 1.0.0
 */
export declare const task: Monad1<URI> & MonadIO1<URI> & MonadTask1<URI>;
/**
 * Like `Task` but `ap` is sequential
 *
 * @since 1.10.0
 */
export declare const taskSeq: typeof task;
/**
 * @since 1.19.0
 */
export declare function of<A>(a: A): Task<A>;
/**
 * @since 1.19.0
 */
export declare const never: Task<never>;
/**
 * @since 1.19.0
 */
export declare function delay2v(millis: number): <A>(ma: Task<A>) => Task<A>;
declare const ap: <A>(fa: Task<A>) => <B>(fab: Task<(a: A) => B>) => Task<B>, apFirst: <B>(fb: Task<B>) => <A>(fa: Task<A>) => Task<A>, apSecond: <B>(fb: Task<B>) => <A>(fa: Task<A>) => Task<B>, chain: <A, B>(f: (a: A) => Task<B>) => (ma: Task<A>) => Task<B>, chainFirst: <A, B>(f: (a: A) => Task<B>) => (ma: Task<A>) => Task<A>, flatten: <A>(mma: Task<Task<A>>) => Task<A>, map: <A, B>(f: (a: A) => B) => (fa: Task<A>) => Task<B>;
export { ap, apFirst, apSecond, chain, chainFirst, flatten, map };
