/**
 * @file A `DistributiveLattice` must satisfy the following laws in addition to `Lattice` laws:
 *
 * - Distributivity for meet: `a ∨ (b ∧ c) = (a ∨ b) ∧ (a ∨ c)`
 * - Distributivity for join: `a ∧ (b ∨ c) = (a ∧ b) ∨ (a ∧ c)`
 */
import { Lattice } from './Lattice';
import { Ord } from './Ord';
/**
 * @since 1.4.0
 */
export interface DistributiveLattice<A> extends Lattice<A> {
}
/**
 * @since 1.4.0
 */
export declare const getMinMaxDistributiveLattice: <A>(O: Ord<A>) => DistributiveLattice<A>;
