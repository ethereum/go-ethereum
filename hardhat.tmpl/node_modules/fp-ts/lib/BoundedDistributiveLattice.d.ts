/**
 * @file A `BoundedDistributiveLattice` is a lattice that is both bounded and distributive
 */
import { BoundedLattice } from './BoundedLattice';
import { DistributiveLattice } from './DistributiveLattice';
import { Ord } from './Ord';
/**
 * @since 1.4.0
 */
export interface BoundedDistributiveLattice<A> extends BoundedLattice<A>, DistributiveLattice<A> {
}
/**
 * @since 1.4.0
 */
export declare const getMinMaxBoundedDistributiveLattice: <A>(O: Ord<A>) => (min: A, max: A) => BoundedDistributiveLattice<A>;
