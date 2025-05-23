/**
 * @file Mutable references in the `IO` monad
 */
import { IO } from './IO';
/**
 * @example
 * import { newIORef } from 'fp-ts/lib/IORef'
 *
 * assert.strictEqual(
 *   newIORef(1)
 *     .chain(ref => ref.write(2).chain(() => ref.read))
 *     .run(),
 *   2
 * )
 * @since 1.8.0
 */
export declare class IORef<A> {
    private value;
    read: IO<A>;
    constructor(value: A);
    /**
     * @since 1.8.0
     */
    write(a: A): IO<void>;
    /**
     * @since 1.8.0
     */
    modify(f: (a: A) => A): IO<void>;
}
/**
 * @since 1.8.0
 */
export declare const newIORef: <A>(a: A) => IO<IORef<A>>;
