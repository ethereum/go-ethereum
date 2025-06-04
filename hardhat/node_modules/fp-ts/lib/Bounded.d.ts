/**
 * @file The `Bounded` type class represents totally ordered types that have an upper and lower boundary.
 *
 * Instances should satisfy the following law in addition to the `Ord` laws:
 *
 * - Bounded: `bottom <= a <= top`
 */
import { Ord } from './Ord';
/**
 * @since 1.0.0
 */
export interface Bounded<A> extends Ord<A> {
    readonly top: A;
    readonly bottom: A;
}
/**
 * @since 1.0.0
 */
export declare const boundedNumber: Bounded<number>;
