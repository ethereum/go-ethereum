import { IO } from './IO';
import { IxMonad3 } from './IxMonad';
import { Monad3C } from './Monad';
declare module './HKT' {
    interface URItoKind3<U, L, A> {
        IxIO: IxIO<U, L, A>;
    }
}
export declare const URI = "IxIO";
export declare type URI = typeof URI;
/**
 * @since 1.0.0
 */
export declare class IxIO<I, O, A> {
    readonly value: IO<A>;
    readonly _A: A;
    readonly _L: O;
    readonly _U: I;
    readonly _URI: URI;
    constructor(value: IO<A>);
    run(): A;
    ichain<Z, B>(f: (a: A) => IxIO<O, Z, B>): IxIO<I, Z, B>;
    map<B>(f: (a: A) => B): IxIO<I, O, B>;
    ap<B>(fab: IxIO<I, I, (a: A) => B>): IxIO<I, I, B>;
    chain<B>(f: (a: A) => IxIO<I, I, B>): IxIO<I, I, B>;
}
/**
 * @since 1.0.0
 */
export declare const iof: <I, A>(a: A) => IxIO<I, I, A>;
/**
 * @since 1.0.0
 */
export declare const getMonad: <I = never>() => Monad3C<"IxIO", I, I>;
/**
 * @since 1.0.0
 */
export declare const ixIO: IxMonad3<URI>;
