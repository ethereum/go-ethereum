import { Monad2 } from './Monad';
declare module './HKT' {
    interface URItoKind2<L, A> {
        State: State<L, A>;
    }
}
export declare const URI = "State";
export declare type URI = typeof URI;
/**
 * @since 1.0.0
 */
export declare class State<S, A> {
    readonly run: (s: S) => [A, S];
    readonly _A: A;
    readonly _L: S;
    readonly _URI: URI;
    constructor(run: (s: S) => [A, S]);
    /** @obsolete */
    eval(s: S): A;
    /** @obsolete */
    exec(s: S): S;
    /** @obsolete */
    map<B>(f: (a: A) => B): State<S, B>;
    /** @obsolete */
    ap<B>(fab: State<S, (a: A) => B>): State<S, B>;
    /**
     * Flipped version of `ap`
     * @obsolete
     */
    ap_<B, C>(this: State<S, (b: B) => C>, fb: State<S, B>): State<S, C>;
    /**
     * Combine two effectful actions, keeping only the result of the first
     * @since 1.7.0
     * @obsolete
     */
    applyFirst<B>(fb: State<S, B>): State<S, A>;
    /**
     * Combine two effectful actions, keeping only the result of the second
     * @since 1.7.0
     * @obsolete
     */
    applySecond<B>(fb: State<S, B>): State<S, B>;
    /** @obsolete */
    chain<B>(f: (a: A) => State<S, B>): State<S, B>;
}
/**
 * Get the current state
 *
 * @since 1.0.0
 */
export declare const get: <S>() => State<S, S>;
/**
 * Set the state
 *
 * @since 1.0.0
 */
export declare const put: <S>(s: S) => State<S, void>;
/**
 * Modify the state by applying a function to the current state
 *
 * @since 1.0.0
 */
export declare const modify: <S>(f: (s: S) => S) => State<S, undefined>;
/**
 * Get a value which depends on the current state
 *
 * @since 1.0.0
 */
export declare const gets: <S, A>(f: (s: S) => A) => State<S, A>;
/**
 * @since 1.0.0
 */
export declare const state: Monad2<URI>;
/**
 * @since 1.19.0
 */
export declare function of<S, A>(a: A): State<S, A>;
/**
 * Run a computation in the `State` monad, discarding the final state
 *
 * @since 1.19.0
 */
export declare function evalState<S, A>(ma: State<S, A>, s: S): A;
/**
 * Run a computation in the `State` monad discarding the result
 *
 * @since 1.19.0
 */
export declare function execState<S, A>(ma: State<S, A>, s: S): S;
declare const ap: <L, A>(fa: State<L, A>) => <B>(fab: State<L, (a: A) => B>) => State<L, B>, apFirst: <L, B>(fb: State<L, B>) => <A>(fa: State<L, A>) => State<L, A>, apSecond: <L, B>(fb: State<L, B>) => <A>(fa: State<L, A>) => State<L, B>, chain: <L, A, B>(f: (a: A) => State<L, B>) => (ma: State<L, A>) => State<L, B>, chainFirst: <L, A, B>(f: (a: A) => State<L, B>) => (ma: State<L, A>) => State<L, A>, flatten: <L, A>(mma: State<L, State<L, A>>) => State<L, A>, map: <A, B>(f: (a: A) => B) => <L>(fa: State<L, A>) => State<L, B>;
export { ap, apFirst, apSecond, chain, chainFirst, flatten, map };
