import { Applicative2C } from './Applicative';
import { Apply2C } from './Apply';
import { Contravariant2 } from './Contravariant';
import { Functor2 } from './Functor';
import { Monoid } from './Monoid';
import { Semigroup } from './Semigroup';
import { Eq } from './Eq';
import { Show } from './Show';
declare module './HKT' {
    interface URItoKind2<L, A> {
        Const: Const<L, A>;
    }
}
export declare const URI = "Const";
export declare type URI = typeof URI;
/**
 * @since 1.0.0
 */
export declare class Const<L, A> {
    readonly value: L;
    readonly _A: A;
    readonly _L: L;
    readonly _URI: URI;
    /**
     * Use `make`
     *
     * @deprecated
     */
    constructor(value: L);
    /** @obsolete */
    map<B>(f: (a: A) => B): Const<L, B>;
    /** @obsolete */
    contramap<B>(f: (b: B) => A): Const<L, B>;
    /** @obsolete */
    fold<B>(f: (l: L) => B): B;
    inspect(): string;
    toString(): string;
}
/**
 * @since 1.17.0
 */
export declare const getShow: <L, A>(S: Show<L>) => Show<Const<L, A>>;
/**
 * Use `getEq`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const getSetoid: <L, A>(S: Eq<L>) => Eq<Const<L, A>>;
/**
 * @since 1.19.0
 */
export declare function getEq<L, A>(S: Eq<L>): Eq<Const<L, A>>;
/**
 * @since 1.0.0
 */
export declare const getApply: <L>(S: Semigroup<L>) => Apply2C<"Const", L>;
/**
 * @since 1.0.0
 */
export declare const getApplicative: <L>(M: Monoid<L>) => Applicative2C<"Const", L>;
/**
 * @since 1.0.0
 */
export declare const const_: Functor2<URI> & Contravariant2<URI>;
/**
 * @since 1.19.0
 */
export declare function make<L, A = never>(l: L): Const<L, A>;
declare const contramap: <A, B>(f: (b: B) => A) => <L>(fa: Const<L, A>) => Const<L, B>, map: <A, B>(f: (a: A) => B) => <L>(fa: Const<L, A>) => Const<L, B>;
export { contramap, map };
