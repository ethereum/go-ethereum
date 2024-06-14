/**
 * @file Lift a computation from the `IO` monad
 */
import { HKT, Kind, Kind2, Kind3, URIS, URIS2, URIS3 } from './HKT';
import { IO } from './IO';
import { Monad, Monad1, Monad2, Monad3, Monad2C, Monad3C } from './Monad';
/**
 * @since 1.10.0
 */
export interface MonadIO<M> extends Monad<M> {
    readonly fromIO: <A>(fa: IO<A>) => HKT<M, A>;
}
export interface MonadIO1<M extends URIS> extends Monad1<M> {
    readonly fromIO: <A>(fa: IO<A>) => Kind<M, A>;
}
export interface MonadIO2<M extends URIS2> extends Monad2<M> {
    readonly fromIO: <L, A>(fa: IO<A>) => Kind2<M, L, A>;
}
export interface MonadIO3<M extends URIS3> extends Monad3<M> {
    readonly fromIO: <U, L, A>(fa: IO<A>) => Kind3<M, U, L, A>;
}
export interface MonadIO2C<M extends URIS2, L> extends Monad2C<M, L> {
    readonly fromIO: <A>(fa: IO<A>) => Kind2<M, L, A>;
}
export interface MonadIO3C<M extends URIS3, U, L> extends Monad3C<M, U, L> {
    readonly fromIO: <A>(fa: IO<A>) => Kind3<M, U, L, A>;
}
