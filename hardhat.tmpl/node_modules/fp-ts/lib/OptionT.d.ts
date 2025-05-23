import { Applicative, Applicative1, Applicative2, Applicative2C, Applicative3C, ApplicativeComposition, ApplicativeComposition11, ApplicativeComposition21, ApplicativeComposition2C1, ApplicativeComposition3C1 } from './Applicative';
import { Functor, Functor1, Functor2, Functor2C, Functor3C } from './Functor';
import { HKT, Kind, Kind2, Kind3, URIS, URIS2, URIS3 } from './HKT';
import { Monad, Monad1, Monad2, Monad2C, Monad3C } from './Monad';
import { Option, URI } from './Option';
export interface OptionT2v<M> extends ApplicativeComposition<M, URI> {
    readonly chain: <A, B>(fa: HKT<M, Option<A>>, f: (a: A) => HKT<M, Option<B>>) => HKT<M, Option<B>>;
}
export interface OptionT2v1<M extends URIS> extends ApplicativeComposition11<M, URI> {
    readonly chain: <A, B>(fa: Kind<M, Option<A>>, f: (a: A) => Kind<M, Option<B>>) => Kind<M, Option<B>>;
}
export interface OptionT2v2<M extends URIS2> extends ApplicativeComposition21<M, URI> {
    readonly chain: <L, A, B>(fa: Kind2<M, L, Option<A>>, f: (a: A) => Kind2<M, L, Option<B>>) => Kind2<M, L, Option<B>>;
}
export interface OptionT2v2C<M extends URIS2, L> extends ApplicativeComposition2C1<M, URI, L> {
    readonly chain: <A, B>(fa: Kind2<M, L, Option<A>>, f: (a: A) => Kind2<M, L, Option<B>>) => Kind2<M, L, Option<B>>;
}
export interface OptionT2v3C<M extends URIS3, U, L> extends ApplicativeComposition3C1<M, URI, U, L> {
    readonly chain: <A, B>(fa: Kind3<M, U, L, Option<A>>, f: (a: A) => Kind3<M, U, L, Option<B>>) => Kind3<M, U, L, Option<B>>;
}
/**
 * @since 1.0.0
 */
export declare function fold<F extends URIS3, U, L>(F: Functor3C<F, U, L>): <R, A>(onNone: R, onSome: (a: A) => R, fa: Kind3<F, U, L, Option<A>>) => Kind3<F, U, L, R>;
export declare function fold<F extends URIS2>(F: Functor2<F>): <L, R, A>(onNone: R, onSome: (a: A) => R, fa: Kind2<F, L, Option<A>>) => Kind2<F, L, R>;
export declare function fold<F extends URIS2, L>(F: Functor2C<F, L>): <R, A>(onNone: R, onSome: (a: A) => R, fa: Kind2<F, L, Option<A>>) => Kind2<F, L, R>;
export declare function fold<F extends URIS>(F: Functor1<F>): <R, A>(onNone: R, onSome: (a: A) => R, fa: Kind<F, Option<A>>) => Kind<F, R>;
export declare function fold<F>(F: Functor<F>): <R, A>(onNone: R, onSome: (a: A) => R, fa: HKT<F, Option<A>>) => HKT<F, R>;
/**
 * @since 1.14.0
 */
export declare function getOptionT2v<M extends URIS3, U, L>(M: Monad3C<M, U, L>): OptionT2v3C<M, U, L>;
export declare function getOptionT2v<M extends URIS2>(M: Monad2<M>): OptionT2v2<M>;
export declare function getOptionT2v<M extends URIS2, L>(M: Monad2C<M, L>): OptionT2v2C<M, L>;
export declare function getOptionT2v<M extends URIS>(M: Monad1<M>): OptionT2v1<M>;
export declare function getOptionT2v<M>(M: Monad<M>): OptionT2v<M>;
/** @deprecated */
export interface OptionT<M> extends ApplicativeComposition<M, URI> {
    readonly chain: <A, B>(f: (a: A) => HKT<M, Option<B>>, fa: HKT<M, Option<A>>) => HKT<M, Option<B>>;
}
/** @deprecated */
export interface OptionT1<M extends URIS> extends ApplicativeComposition11<M, URI> {
    readonly chain: <A, B>(f: (a: A) => Kind<M, Option<B>>, fa: Kind<M, Option<A>>) => Kind<M, Option<B>>;
}
/** @deprecated */
export interface OptionT2<M extends URIS2> extends ApplicativeComposition21<M, URI> {
    readonly chain: <L, A, B>(f: (a: A) => Kind2<M, L, Option<B>>, fa: Kind2<M, L, Option<A>>) => Kind2<M, L, Option<B>>;
}
/** @deprecated */
export interface OptionT2C<M extends URIS2, L> extends ApplicativeComposition2C1<M, URI, L> {
    readonly chain: <A, B>(f: (a: A) => Kind2<M, L, Option<B>>, fa: Kind2<M, L, Option<A>>) => Kind2<M, L, Option<B>>;
}
/** @deprecated */
export interface OptionT3C<M extends URIS3, U, L> extends ApplicativeComposition3C1<M, URI, U, L> {
    readonly chain: <A, B>(f: (a: A) => Kind3<M, U, L, Option<B>>, fa: Kind3<M, U, L, Option<A>>) => Kind3<M, U, L, Option<B>>;
}
/**
 * Use `getOptionT2v` instead
 * @since 1.0.0
 * @deprecated
 */
export declare function chain<F extends URIS3, U, L>(F: Monad3C<F, U, L>): OptionT3C<F, U, L>['chain'];
/** @deprecated */
export declare function chain<F extends URIS2>(F: Monad2<F>): OptionT2<F>['chain'];
/** @deprecated */
export declare function chain<F extends URIS2, L>(F: Monad2C<F, L>): OptionT2C<F, L>['chain'];
/** @deprecated */
export declare function chain<F extends URIS>(F: Monad1<F>): OptionT1<F>['chain'];
/** @deprecated */
export declare function chain<F>(F: Monad<F>): OptionT<F>['chain'];
/**
 * Use `getOptionT2v` instead
 * @since 1.0.0
 * @deprecated
 */
export declare function getOptionT<M extends URIS3, U, L>(M: Monad3C<M, U, L>): OptionT3C<M, U, L>;
/** @deprecated */
export declare function getOptionT<M extends URIS2>(M: Monad2<M>): OptionT2<M>;
/** @deprecated */
export declare function getOptionT<M extends URIS2, L>(M: Monad2C<M, L>): OptionT2C<M, L>;
/** @deprecated */
export declare function getOptionT<M extends URIS>(M: Monad1<M>): OptionT1<M>;
/** @deprecated */
export declare function getOptionT<M>(M: Monad<M>): OptionT<M>;
/**
 * @since 1.0.0
 * @deprecated
 */
export declare function some<F extends URIS3, U, L>(F: Applicative3C<F, U, L>): <A>(a: A) => Kind3<F, U, L, Option<A>>;
/** @deprecated */
export declare function some<F extends URIS2>(F: Applicative2<F>): <L, A>(a: A) => Kind2<F, L, Option<A>>;
/** @deprecated */
export declare function some<F extends URIS2, L>(F: Applicative2C<F, L>): <A>(a: A) => Kind2<F, L, Option<A>>;
/** @deprecated */
export declare function some<F extends URIS>(F: Applicative1<F>): <A>(a: A) => Kind<F, Option<A>>;
/** @deprecated */
export declare function some<F>(F: Applicative<F>): <A>(a: A) => HKT<F, Option<A>>;
/**
 * @since 1.0.0
 * @deprecated
 */
export declare function none<F extends URIS3, U, L>(F: Applicative3C<F, U, L>): () => Kind3<F, U, L, Option<never>>;
/** @deprecated */
export declare function none<F extends URIS2>(F: Applicative2<F>): <L>() => Kind2<F, L, Option<never>>;
/** @deprecated */
export declare function none<F extends URIS2, L>(F: Applicative2C<F, L>): () => Kind2<F, L, Option<never>>;
/** @deprecated */
export declare function none<F extends URIS>(F: Applicative1<F>): () => Kind<F, Option<never>>;
/** @deprecated */
export declare function none<F>(F: Applicative<F>): () => HKT<F, Option<never>>;
/**
 * @since 1.0.0
 * @deprecated
 */
export declare function fromOption<F extends URIS3, U, L>(F: Applicative3C<F, U, L>): <A>(fa: Option<A>) => Kind3<F, U, L, Option<A>>;
export declare function fromOption<F extends URIS2>(F: Applicative2<F>): <L, A>(fa: Option<A>) => Kind2<F, L, Option<A>>;
/** @deprecated */
export declare function fromOption<F extends URIS2, L>(F: Applicative2C<F, L>): <A>(fa: Option<A>) => Kind2<F, L, Option<A>>;
/** @deprecated */
export declare function fromOption<F extends URIS>(F: Applicative1<F>): <A>(fa: Option<A>) => Kind<F, Option<A>>;
/** @deprecated */
export declare function fromOption<F>(F: Applicative<F>): <A>(fa: Option<A>) => HKT<F, Option<A>>;
/**
 * @since 1.0.0
 * @deprecated
 */
export declare function liftF<F extends URIS3, U, L>(F: Functor3C<F, U, L>): <A>(fa: Kind3<F, U, L, A>) => Kind3<F, U, L, Option<A>>;
/** @deprecated */
export declare function liftF<F extends URIS2>(F: Functor2<F>): <L, A>(fa: Kind2<F, L, A>) => Kind2<F, L, Option<A>>;
/** @deprecated */
export declare function liftF<F extends URIS2, L>(F: Functor2C<F, L>): <A>(fa: Kind2<F, L, A>) => Kind2<F, L, Option<A>>;
/** @deprecated */
export declare function liftF<F extends URIS>(F: Functor1<F>): <A>(fa: Kind<F, A>) => Kind<F, Option<A>>;
/** @deprecated */
export declare function liftF<F>(F: Functor<F>): <A>(fa: HKT<F, A>) => HKT<F, Option<A>>;
/**
 * @since 1.0.0
 * @deprecated
 */
export declare function getOrElse<F extends URIS3, U, L>(F: Functor3C<F, U, L>): <A>(a: A) => (fa: Kind3<F, U, L, Option<A>>) => Kind3<F, U, L, A>;
/** @deprecated */
export declare function getOrElse<F extends URIS2>(F: Functor2<F>): <A>(a: A) => <L>(fa: Kind2<F, L, Option<A>>) => Kind2<F, L, A>;
/** @deprecated */
export declare function getOrElse<F extends URIS2, L>(F: Functor2C<F, L>): <A>(a: A) => (fa: Kind2<F, L, Option<A>>) => Kind2<F, L, A>;
/** @deprecated */
export declare function getOrElse<F extends URIS>(F: Functor1<F>): <A>(a: A) => (fa: Kind<F, Option<A>>) => Kind<F, A>;
/** @deprecated */
export declare function getOrElse<F>(F: Functor<F>): <A>(a: A) => (fa: HKT<F, Option<A>>) => HKT<F, A>;
