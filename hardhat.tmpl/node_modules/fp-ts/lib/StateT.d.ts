import { Applicative, Applicative1, Applicative2, Applicative3 } from './Applicative';
import { Chain, Chain1, Chain2, Chain3 } from './Chain';
import { Functor, Functor1, Functor2, Functor3 } from './Functor';
import { HKT, Kind, Kind2, Kind3, URIS, URIS2, URIS3 } from './HKT';
import { Monad, Monad1, Monad2, Monad3 } from './Monad';
import { State } from './State';
import { Endomorphism } from './function';
export interface StateT2v<M> {
    readonly map: <S, A, B>(fa: (s: S) => HKT<M, [A, S]>, f: (a: A) => B) => (s: S) => HKT<M, [B, S]>;
    readonly of: <S, A>(a: A) => (s: S) => HKT<M, [A, S]>;
    readonly ap: <S, A, B>(fab: (s: S) => HKT<M, [(a: A) => B, S]>, fa: (s: S) => HKT<M, [A, S]>) => (s: S) => HKT<M, [B, S]>;
    readonly chain: <S, A, B>(fa: (s: S) => HKT<M, [A, S]>, f: (a: A) => (s: S) => HKT<M, [B, S]>) => (s: S) => HKT<M, [B, S]>;
}
export interface StateT2v1<M extends URIS> {
    readonly map: <S, A, B>(fa: (s: S) => Kind<M, [A, S]>, f: (a: A) => B) => (s: S) => Kind<M, [B, S]>;
    readonly of: <S, A>(a: A) => (s: S) => Kind<M, [A, S]>;
    readonly ap: <S, A, B>(fab: (s: S) => Kind<M, [(a: A) => B, S]>, fa: (s: S) => Kind<M, [A, S]>) => (s: S) => Kind<M, [B, S]>;
    readonly chain: <S, A, B>(fa: (s: S) => Kind<M, [A, S]>, f: (a: A) => (s: S) => Kind<M, [B, S]>) => (s: S) => Kind<M, [B, S]>;
}
export interface StateT2v2<M extends URIS2> {
    readonly map: <L, S, A, B>(fa: (s: S) => Kind2<M, L, [A, S]>, f: (a: A) => B) => (s: S) => Kind2<M, L, [B, S]>;
    readonly of: <L, S, A>(a: A) => (s: S) => Kind2<M, L, [A, S]>;
    readonly ap: <L, S, A, B>(fab: (s: S) => Kind2<M, L, [(a: A) => B, S]>, fa: (s: S) => Kind2<M, L, [A, S]>) => (s: S) => Kind2<M, L, [B, S]>;
    readonly chain: <L, S, A, B>(fa: (s: S) => Kind2<M, L, [A, S]>, f: (a: A) => (s: S) => Kind2<M, L, [B, S]>) => (s: S) => Kind2<M, L, [B, S]>;
}
export interface StateT2v3<M extends URIS3> {
    readonly map: <U, L, S, A, B>(fa: (s: S) => Kind3<M, U, L, [A, S]>, f: (a: A) => B) => (s: S) => Kind3<M, U, L, [B, S]>;
    readonly of: <U, L, S, A>(a: A) => (s: S) => Kind3<M, U, L, [A, S]>;
    readonly ap: <U, L, S, A, B>(fab: (s: S) => Kind3<M, U, L, [(a: A) => B, S]>, fa: (s: S) => Kind3<M, U, L, [A, S]>) => (s: S) => Kind3<M, U, L, [B, S]>;
    readonly chain: <U, L, S, A, B>(fa: (s: S) => Kind3<M, U, L, [A, S]>, f: (a: A) => (s: S) => Kind3<M, U, L, [B, S]>) => (s: S) => Kind3<M, U, L, [B, S]>;
}
/**
 * @since 1.14.0
 */
export declare function get2v<F extends URIS3>(F: Applicative3<F>): <S, U, L>(s: S) => Kind3<F, U, L, [S, S]>;
export declare function get2v<F extends URIS2>(F: Applicative2<F>): <S, L>(s: S) => Kind2<F, L, [S, S]>;
export declare function get2v<F extends URIS>(F: Applicative1<F>): <S>(s: S) => Kind<F, [S, S]>;
export declare function get2v<F>(F: Applicative<F>): <S>(s: S) => HKT<F, [S, S]>;
/**
 * @since 1.0.0
 */
export declare function put<F extends URIS3>(F: Applicative3<F>): <S>(s: S) => <U, L>() => Kind3<F, U, L, [void, S]>;
export declare function put<F extends URIS2>(F: Applicative2<F>): <S>(s: S) => <L>() => Kind2<F, L, [void, S]>;
export declare function put<F extends URIS>(F: Applicative1<F>): <S>(s: S) => () => Kind<F, [void, S]>;
export declare function put<F>(F: Applicative<F>): <S>(s: S) => () => HKT<F, [void, S]>;
/**
 * @since 1.0.0
 */
export declare function modify<F extends URIS3>(F: Applicative3<F>): <S>(f: Endomorphism<S>) => <U, L>(s: S) => Kind3<F, U, L, [void, S]>;
export declare function modify<F extends URIS2>(F: Applicative2<F>): <S>(f: Endomorphism<S>) => <L>(s: S) => Kind2<F, L, [void, S]>;
export declare function modify<F extends URIS>(F: Applicative1<F>): <S>(f: Endomorphism<S>) => (s: S) => Kind<F, [void, S]>;
export declare function modify<F>(F: Applicative<F>): <S>(f: Endomorphism<S>) => (s: S) => HKT<F, [void, S]>;
/**
 * @since 1.0.0
 */
export declare function gets<F extends URIS3>(F: Applicative3<F>): <S, A>(f: (s: S) => A) => <U, L>(s: S) => Kind3<F, U, L, [A, S]>;
export declare function gets<F extends URIS2>(F: Applicative2<F>): <S, A>(f: (s: S) => A) => <L>(s: S) => Kind2<F, L, [A, S]>;
export declare function gets<F extends URIS>(F: Applicative1<F>): <S, A>(f: (s: S) => A) => (s: S) => Kind<F, [A, S]>;
export declare function gets<F>(F: Applicative<F>): <S, A>(f: (s: S) => A) => (s: S) => HKT<F, [A, S]>;
/**
 * @since 1.2.0
 */
export declare function fromState<F extends URIS3>(F: Applicative3<F>): <S, A, U, L>(fa: State<S, A>) => (s: S) => Kind3<F, U, L, [A, S]>;
export declare function fromState<F extends URIS2>(F: Applicative2<F>): <S, A, L>(fa: State<S, A>) => (s: S) => Kind2<F, L, [A, S]>;
export declare function fromState<F extends URIS>(F: Applicative1<F>): <S, A>(fa: State<S, A>) => (s: S) => Kind<F, [A, S]>;
export declare function fromState<F>(F: Applicative<F>): <S, A>(fa: State<S, A>) => (s: S) => HKT<F, [A, S]>;
/**
 * @since 1.2.0
 */
export declare function liftF<F extends URIS3>(F: Functor3<F>): <U, L, S, A>(fa: Kind3<F, U, L, A>) => (s: S) => Kind3<F, U, L, [A, S]>;
export declare function liftF<F extends URIS2>(F: Functor2<F>): <L, S, A>(fa: Kind2<F, L, A>) => (s: S) => Kind2<F, L, [A, S]>;
export declare function liftF<F extends URIS>(F: Functor1<F>): <S, A>(fa: Kind<F, A>) => (s: S) => Kind<F, [A, S]>;
export declare function liftF<F>(F: Functor<F>): <S, A>(fa: HKT<F, A>) => (s: S) => HKT<F, [A, S]>;
/**
 * @since 1.14.0
 */
export declare function getStateT2v<M extends URIS3>(M: Monad3<M>): StateT2v3<M>;
export declare function getStateT2v<M extends URIS2>(M: Monad2<M>): StateT2v2<M>;
export declare function getStateT2v<M extends URIS>(M: Monad1<M>): StateT2v1<M>;
export declare function getStateT2v<M>(M: Monad<M>): StateT2v<M>;
/** @deprecated */
export interface StateT<M> {
    readonly map: <S, A, B>(f: (a: A) => B, fa: (s: S) => HKT<M, [A, S]>) => (s: S) => HKT<M, [B, S]>;
    readonly of: <S, A>(a: A) => (s: S) => HKT<M, [A, S]>;
    readonly ap: <S, A, B>(fab: (s: S) => HKT<M, [(a: A) => B, S]>, fa: (s: S) => HKT<M, [A, S]>) => (s: S) => HKT<M, [B, S]>;
    readonly chain: <S, A, B>(f: (a: A) => (s: S) => HKT<M, [B, S]>, fa: (s: S) => HKT<M, [A, S]>) => (s: S) => HKT<M, [B, S]>;
}
/** @deprecated */
export interface StateT1<M extends URIS> {
    readonly map: <S, A, B>(f: (a: A) => B, fa: (s: S) => Kind<M, [A, S]>) => (s: S) => Kind<M, [B, S]>;
    readonly of: <S, A>(a: A) => (s: S) => Kind<M, [A, S]>;
    readonly ap: <S, A, B>(fab: (s: S) => Kind<M, [(a: A) => B, S]>, fa: (s: S) => Kind<M, [A, S]>) => (s: S) => Kind<M, [B, S]>;
    readonly chain: <S, A, B>(f: (a: A) => (s: S) => Kind<M, [B, S]>, fa: (s: S) => Kind<M, [A, S]>) => (s: S) => Kind<M, [B, S]>;
}
/** @deprecated */
export interface StateT2<M extends URIS2> {
    readonly map: <L, S, A, B>(f: (a: A) => B, fa: (s: S) => Kind2<M, L, [A, S]>) => (s: S) => Kind2<M, L, [B, S]>;
    readonly of: <L, S, A>(a: A) => (s: S) => Kind2<M, L, [A, S]>;
    readonly ap: <L, S, A, B>(fab: (s: S) => Kind2<M, L, [(a: A) => B, S]>, fa: (s: S) => Kind2<M, L, [A, S]>) => (s: S) => Kind2<M, L, [B, S]>;
    readonly chain: <L, S, A, B>(f: (a: A) => (s: S) => Kind2<M, L, [B, S]>, fa: (s: S) => Kind2<M, L, [A, S]>) => (s: S) => Kind2<M, L, [B, S]>;
}
/** @deprecated */
export interface StateT3<M extends URIS3> {
    readonly map: <U, L, S, A, B>(f: (a: A) => B, fa: (s: S) => Kind3<M, U, L, [A, S]>) => (s: S) => Kind3<M, U, L, [B, S]>;
    readonly of: <U, L, S, A>(a: A) => (s: S) => Kind3<M, U, L, [A, S]>;
    readonly ap: <U, L, S, A, B>(fab: (s: S) => Kind3<M, U, L, [(a: A) => B, S]>, fa: (s: S) => Kind3<M, U, L, [A, S]>) => (s: S) => Kind3<M, U, L, [B, S]>;
    readonly chain: <U, L, S, A, B>(f: (a: A) => (s: S) => Kind3<M, U, L, [B, S]>, fa: (s: S) => Kind3<M, U, L, [A, S]>) => (s: S) => Kind3<M, U, L, [B, S]>;
}
/**
 * @since 1.0.0
 * @deprecated
 */
export declare function map<F extends URIS3>(F: Functor3<F>): <U, L, S, A, B>(f: (a: A) => B, fa: (s: S) => Kind3<F, U, L, [A, S]>) => (s: S) => Kind3<F, U, L, [B, S]>;
/** @deprecated */
export declare function map<F extends URIS2>(F: Functor2<F>): <L, S, A, B>(f: (a: A) => B, fa: (s: S) => Kind2<F, L, [A, S]>) => (s: S) => Kind2<F, L, [B, S]>;
/** @deprecated */
export declare function map<F extends URIS>(F: Functor1<F>): <S, A, B>(f: (a: A) => B, fa: (s: S) => Kind<F, [A, S]>) => (s: S) => Kind<F, [B, S]>;
/** @deprecated */
export declare function map<F>(F: Functor<F>): <S, A, B>(f: (a: A) => B, fa: (s: S) => HKT<F, [A, S]>) => (s: S) => HKT<F, [B, S]>;
/**
 * @since 1.0.0
 * @deprecated
 */
export declare function ap<F extends URIS3>(F: Chain3<F>): <U, L, S, A, B>(fab: (s: S) => Kind3<F, U, L, [(a: A) => B, S]>, fa: (s: S) => Kind3<F, U, L, [A, S]>) => (s: S) => Kind3<F, U, L, [B, S]>;
/** @deprecated */
export declare function ap<F extends URIS2>(F: Chain2<F>): <L, S, A, B>(fab: (s: S) => Kind2<F, L, [(a: A) => B, S]>, fa: (s: S) => Kind2<F, L, [A, S]>) => (s: S) => Kind2<F, L, [B, S]>;
/** @deprecated */
export declare function ap<F extends URIS>(F: Chain1<F>): <S, A, B>(fab: (s: S) => Kind<F, [(a: A) => B, S]>, fa: (s: S) => Kind<F, [A, S]>) => (s: S) => Kind<F, [B, S]>;
/** @deprecated */
export declare function ap<F>(F: Chain<F>): <S, A, B>(fab: (s: S) => HKT<F, [(a: A) => B, S]>, fa: (s: S) => HKT<F, [A, S]>) => (s: S) => HKT<F, [B, S]>;
/**
 * @since 1.0.0
 * @deprecated
 */
export declare function chain<F extends URIS3>(F: Chain3<F>): <U, L, S, A, B>(f: (a: A) => (s: S) => Kind3<F, U, L, [B, S]>, fa: (s: S) => Kind3<F, U, L, [A, S]>) => (s: S) => Kind3<F, U, L, [B, S]>;
/** @deprecated */
export declare function chain<F extends URIS2>(F: Chain2<F>): <L, S, A, B>(f: (a: A) => (s: S) => Kind2<F, L, [B, S]>, fa: (s: S) => Kind2<F, L, [A, S]>) => (s: S) => Kind2<F, L, [B, S]>;
/** @deprecated */
export declare function chain<F extends URIS>(F: Chain1<F>): <S, A, B>(f: (a: A) => (s: S) => Kind<F, [B, S]>, fa: (s: S) => Kind<F, [A, S]>) => (s: S) => Kind<F, [B, S]>;
/** @deprecated */
export declare function chain<F>(F: Chain<F>): <S, A, B>(f: (a: A) => (s: S) => HKT<F, [B, S]>, fa: (s: S) => HKT<F, [A, S]>) => (s: S) => HKT<F, [B, S]>;
/**
 * Use `getStateT2v` instead
 * @since 1.0.0
 * @deprecated
 */
export declare function getStateT<M extends URIS3>(M: Monad3<M>): StateT3<M>;
/** @deprecated */
export declare function getStateT<M extends URIS2>(M: Monad2<M>): StateT2<M>;
/** @deprecated */
export declare function getStateT<M extends URIS>(M: Monad1<M>): StateT1<M>;
/** @deprecated */
export declare function getStateT<M>(M: Monad<M>): StateT<M>;
/**
 * @since 1.0.0
 * @deprecated
 */
export declare function of<F extends URIS3>(F: Applicative3<F>): <U, L, S, A>(a: A) => (s: S) => Kind3<F, U, L, [A, S]>;
/** @deprecated */
export declare function of<F extends URIS2>(F: Applicative2<F>): <L, S, A>(a: A) => (s: S) => Kind2<F, L, [A, S]>;
/** @deprecated */
export declare function of<F extends URIS>(F: Applicative1<F>): <S, A>(a: A) => (s: S) => Kind<F, [A, S]>;
/** @deprecated */
export declare function of<F>(F: Applicative<F>): <S, A>(a: A) => (s: S) => HKT<F, [A, S]>;
/**
 * @since 1.0.0
 * @deprecated
 */
export declare function get<F extends URIS3>(F: Applicative3<F>): <S>() => <U, L>(s: S) => Kind3<F, U, L, [S, S]>;
/** @deprecated */
export declare function get<F extends URIS2>(F: Applicative2<F>): <S>() => <L>(s: S) => Kind2<F, L, [S, S]>;
/** @deprecated */
export declare function get<F extends URIS>(F: Applicative1<F>): <S>() => (s: S) => Kind<F, [S, S]>;
/** @deprecated */
export declare function get<F>(F: Applicative<F>): <S>() => (s: S) => HKT<F, [S, S]>;
