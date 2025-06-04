/**
 * @file Lift a computation from the `Task` monad
 */
import { HKT, Kind, Kind2, Kind3, URIS, URIS2, URIS3 } from './HKT';
import { Task } from './Task';
import { Monad, Monad1, Monad2, Monad3, Monad2C, Monad3C } from './Monad';
/**
 * @since 1.10.0
 */
export interface MonadTask<M> extends Monad<M> {
    readonly fromTask: <A>(fa: Task<A>) => HKT<M, A>;
}
export interface MonadTask1<M extends URIS> extends Monad1<M> {
    readonly fromTask: <A>(fa: Task<A>) => Kind<M, A>;
}
export interface MonadTask2<M extends URIS2> extends Monad2<M> {
    readonly fromTask: <L, A>(fa: Task<A>) => Kind2<M, L, A>;
}
export interface MonadTask3<M extends URIS3> extends Monad3<M> {
    readonly fromTask: <U, L, A>(fa: Task<A>) => Kind3<M, U, L, A>;
}
export interface MonadTask2C<M extends URIS2, L> extends Monad2C<M, L> {
    readonly fromTask: <A>(fa: Task<A>) => Kind2<M, L, A>;
}
export interface MonadTask3C<M extends URIS3, U, L> extends Monad3C<M, U, L> {
    readonly fromTask: <A>(fa: Task<A>) => Kind3<M, U, L, A>;
}
