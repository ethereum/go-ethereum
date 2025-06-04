import { Either } from './Either';
import { Group } from './Group';
import { Eq } from './Eq';
import { Monad1 } from './Monad';
declare module './HKT' {
    interface URItoKind<A> {
        FreeGroup: FreeGroup<A>;
    }
}
export declare const URI = "FreeGroup";
export declare type URI = typeof URI;
/**
 * @since 1.13.0
 */
export declare class FreeGroup<A> {
    readonly value: Array<Either<A, A>>;
    readonly _A: A;
    readonly _URI: URI;
    constructor(value: Array<Either<A, A>>);
    map<B>(f: (a: A) => B): FreeGroup<B>;
    ap<B>(fab: FreeGroup<(a: A) => B>): FreeGroup<B>;
    ap_<B, C>(this: FreeGroup<(b: B) => C>, fb: FreeGroup<B>): FreeGroup<C>;
    chain<B>(f: (a: A) => FreeGroup<B>): FreeGroup<B>;
}
/**
 * Smart constructor which normalizes an array
 *
 * @since 1.13.0
 */
export declare const fromArray: <A>(E: Eq<A>) => (as: Either<A, A>[]) => FreeGroup<A>;
/**
 * Reduce a term of a free group to canonical form, i.e. cancelling adjacent inverses.
 *
 * @since 1.13.0
 */
export declare const normalize: <A>(E: Eq<A>) => (g: Either<A, A>[]) => Either<A, A>[];
/**
 * Use `getEq`
 *
 * @since 1.13.0
 * @deprecated
 */
export declare const getSetoid: <A>(S: Eq<A>) => Eq<FreeGroup<A>>;
/**
 * @since 1.19.0
 */
export declare function getEq<A>(S: Eq<A>): Eq<FreeGroup<A>>;
/**
 * @since 1.13.0
 */
export declare const empty: FreeGroup<never>;
/**
 * @since 1.13.0
 */
export declare const getGroup: <A>(E: Eq<A>) => Group<FreeGroup<A>>;
/**
 * @since 1.13.0
 */
export declare const freeGroup: Monad1<URI>;
