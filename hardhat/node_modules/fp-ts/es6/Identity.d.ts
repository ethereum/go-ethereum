import { Alt1 } from './Alt';
import { ChainRec1 } from './ChainRec';
import { Comonad1 } from './Comonad';
import { Foldable2v1 } from './Foldable2v';
import { Lazy } from './function';
import { Monad1 } from './Monad';
import { Eq } from './Eq';
import { Show } from './Show';
import { Traversable2v1 } from './Traversable2v';
declare module './HKT' {
    interface URItoKind<A> {
        Identity: Identity<A>;
    }
}
export declare const URI = "Identity";
export declare type URI = typeof URI;
/**
 * @since 1.0.0
 */
export declare class Identity<A> {
    readonly value: A;
    readonly _A: A;
    readonly _URI: URI;
    constructor(value: A);
    /** @obsolete */
    map<B>(f: (a: A) => B): Identity<B>;
    /** @obsolete */
    ap<B>(fab: Identity<(a: A) => B>): Identity<B>;
    /**
     * Flipped version of `ap`
     * @obsolete
     */
    ap_<B, C>(this: Identity<(b: B) => C>, fb: Identity<B>): Identity<C>;
    /** @obsolete */
    chain<B>(f: (a: A) => Identity<B>): Identity<B>;
    /** @obsolete */
    reduce<B>(b: B, f: (b: B, a: A) => B): B;
    /** @obsolete */
    alt(fx: Identity<A>): Identity<A>;
    /**
     * Lazy version of `alt`
     *
     * @example
     * import { Identity } from 'fp-ts/lib/Identity'
     *
     * const a = new Identity(1)
     * assert.deepStrictEqual(a.orElse(() => new Identity(2)), a)
     *
     * @since 1.6.0
     * @obsolete
     */
    orElse(fx: Lazy<Identity<A>>): Identity<A>;
    /** @obsolete */
    extract(): A;
    /** @obsolete */
    extend<B>(f: (ea: Identity<A>) => B): Identity<B>;
    /** @obsolete */
    fold<B>(f: (a: A) => B): B;
    inspect(): string;
    toString(): string;
}
/**
 * @since 1.17.0
 */
export declare const getShow: <A>(S: Show<A>) => Show<Identity<A>>;
/**
 * Use `getEq`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const getSetoid: <A>(E: Eq<A>) => Eq<Identity<A>>;
/**
 * @since 1.19.0
 */
export declare function getEq<A>(E: Eq<A>): Eq<Identity<A>>;
/**
 * @since 1.0.0
 */
export declare const identity: Monad1<URI> & Foldable2v1<URI> & Traversable2v1<URI> & Alt1<URI> & Comonad1<URI> & ChainRec1<URI>;
declare const alt: <A>(that: () => Identity<A>) => (fa: Identity<A>) => Identity<A>, ap: <A>(fa: Identity<A>) => <B>(fab: Identity<(a: A) => B>) => Identity<B>, apFirst: <B>(fb: Identity<B>) => <A>(fa: Identity<A>) => Identity<A>, apSecond: <B>(fb: Identity<B>) => <A>(fa: Identity<A>) => Identity<B>, chain: <A, B>(f: (a: A) => Identity<B>) => (ma: Identity<A>) => Identity<B>, chainFirst: <A, B>(f: (a: A) => Identity<B>) => (ma: Identity<A>) => Identity<A>, duplicate: <A>(ma: Identity<A>) => Identity<Identity<A>>, extend: <A, B>(f: (fa: Identity<A>) => B) => (ma: Identity<A>) => Identity<B>, flatten: <A>(mma: Identity<Identity<A>>) => Identity<A>, foldMap: <M>(M: import("./Monoid").Monoid<M>) => <A>(f: (a: A) => M) => (fa: Identity<A>) => M, map: <A, B>(f: (a: A) => B) => (fa: Identity<A>) => Identity<B>, reduce: <A, B>(b: B, f: (b: B, a: A) => B) => (fa: Identity<A>) => B, reduceRight: <A, B>(b: B, f: (a: A, b: B) => B) => (fa: Identity<A>) => B;
export { alt, ap, apFirst, apSecond, chain, chainFirst, duplicate, extend, flatten, foldMap, map, reduce, reduceRight };
