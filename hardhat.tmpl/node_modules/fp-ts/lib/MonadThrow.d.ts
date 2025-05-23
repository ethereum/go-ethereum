import { Either } from './Either';
import { HKT, Kind, Kind2, Kind3, Kind4, URIS, URIS2, URIS3, URIS4 } from './HKT';
import { Monad, Monad1, Monad2, Monad2C, Monad3, Monad3C, Monad4 } from './Monad';
import { Option } from './Option';
/**
 * The `MonadThrow` type class represents those monads which support errors via
 * `throwError`, where `throwError(e)` halts, yielding the error `e`.
 *
 * Laws:
 *
 * - Left zero: `M.chain(M.throwError(e), f) = M.throwError(e)`
 *
 * @since 1.16.0
 */
export interface MonadThrow<M> extends Monad<M> {
    readonly throwError: <E, A>(e: E) => HKT<M, A>;
    /** @deprecated */
    readonly fromEither: <E, A>(e: Either<E, A>) => HKT<M, A>;
    /** @deprecated */
    readonly fromOption: <E, A>(o: Option<A>, e: E) => HKT<M, A>;
}
export interface MonadThrow1<M extends URIS> extends Monad1<M> {
    readonly throwError: <E, A>(e: E) => Kind<M, A>;
    /** @deprecated */
    readonly fromEither: <E, A>(e: Either<E, A>) => Kind<M, A>;
    /** @deprecated */
    readonly fromOption: <E, A>(o: Option<A>, e: E) => Kind<M, A>;
}
export interface MonadThrow2<M extends URIS2> extends Monad2<M> {
    readonly throwError: <E, A>(e: E) => Kind2<M, E, A>;
    /** @deprecated */
    readonly fromEither: <E, A>(e: Either<E, A>) => Kind2<M, E, A>;
    /** @deprecated */
    readonly fromOption: <E, A>(o: Option<A>, e: E) => Kind2<M, E, A>;
}
export interface MonadThrow2C<M extends URIS2, E> extends Monad2C<M, E> {
    readonly throwError: <A>(e: E) => Kind2<M, E, A>;
    /** @deprecated */
    readonly fromEither: <A>(e: Either<E, A>) => Kind2<M, E, A>;
    /** @deprecated */
    readonly fromOption: <A>(o: Option<A>, e: E) => Kind2<M, E, A>;
}
export interface MonadThrow3<M extends URIS3> extends Monad3<M> {
    readonly throwError: <U, E, A>(e: E) => Kind3<M, U, E, A>;
    /** @deprecated */
    readonly fromEither: <U, E, A>(e: Either<E, A>) => Kind3<M, U, E, A>;
    /** @deprecated */
    readonly fromOption: <U, E, A>(o: Option<A>, e: E) => Kind3<M, U, E, A>;
}
export interface MonadThrow3C<M extends URIS3, U, E> extends Monad3C<M, U, E> {
    readonly throwError: <A>(e: E) => Kind3<M, U, E, A>;
    /** @deprecated */
    readonly fromEither: <A>(e: Either<E, A>) => Kind3<M, U, E, A>;
    /** @deprecated */
    readonly fromOption: <A>(o: Option<A>, e: E) => Kind3<M, U, E, A>;
}
export interface MonadThrow4<M extends URIS4> extends Monad4<M> {
    readonly throwError: <X, U, E, A>(e: E) => Kind4<M, X, U, E, A>;
    /** @deprecated */
    readonly fromEither: <X, U, E, A>(e: Either<E, A>) => Kind4<M, X, U, E, A>;
    /** @deprecated */
    readonly fromOption: <X, U, E, A>(o: Option<A>, e: E) => Kind4<M, X, U, E, A>;
}
