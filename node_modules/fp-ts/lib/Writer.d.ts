import { Functor2 } from './Functor';
import { Monad2C } from './Monad';
import { Monoid } from './Monoid';
declare module './HKT' {
    interface URItoKind2<L, A> {
        Writer: Writer<L, A>;
    }
}
export declare const URI = "Writer";
export declare type URI = typeof URI;
/**
 * @since 1.0.0
 */
export declare class Writer<W, A> {
    readonly run: () => [A, W];
    readonly _A: A;
    readonly _L: W;
    readonly _URI: URI;
    constructor(run: () => [A, W]);
    /** @obsolete */
    eval(): A;
    /** @obsolete */
    exec(): W;
    /** @obsolete */
    map<B>(f: (a: A) => B): Writer<W, B>;
}
/**
 * Appends a value to the accumulator
 *
 * @since 1.0.0
 */
export declare const tell: <W>(w: W) => Writer<W, void>;
/**
 * Modifies the result to include the changes to the accumulator
 *
 * @since 1.3.0
 */
export declare const listen: <W, A>(fa: Writer<W, A>) => Writer<W, [A, W]>;
/**
 * Applies the returned function to the accumulator
 *
 * @since 1.3.0
 */
export declare const pass: <W, A>(fa: Writer<W, [A, (w: W) => W]>) => Writer<W, A>;
/**
 * Use `listens2v`
 *
 * @since 1.3.0
 * @deprecated
 */
export declare const listens: <W, A, B>(fa: Writer<W, A>, f: (w: W) => B) => Writer<W, [A, B]>;
/**
 * Use `censor2v`
 *
 * @since 1.3.0
 * @deprecated
 */
export declare const censor: <W, A>(fa: Writer<W, A>, f: (w: W) => W) => Writer<W, A>;
/**
 *
 * @since 1.0.0
 */
export declare const getMonad: <W>(M: Monoid<W>) => Monad2C<"Writer", W>;
/**
 * @since 1.0.0
 */
export declare const writer: Functor2<URI>;
/**
 * @since 1.19.0
 */
export declare function evalWriter<W, A>(fa: Writer<W, A>): A;
/**
 * @since 1.19.0
 */
export declare function execWriter<W, A>(fa: Writer<W, A>): W;
/**
 * Projects a value from modifications made to the accumulator during an action
 *
 * @since 1.19.0
 */
export declare function listens2v<W, B>(f: (w: W) => B): <A>(fa: Writer<W, A>) => Writer<W, [A, B]>;
/**
 * Modify the final accumulator value by applying a function
 *
 * @since 1.19.0
 */
export declare function censor2v<W>(f: (w: W) => W): <A>(fa: Writer<W, A>) => Writer<W, A>;
declare const map: <A, B>(f: (a: A) => B) => <L>(fa: Writer<L, A>) => Writer<L, B>;
export { map };
