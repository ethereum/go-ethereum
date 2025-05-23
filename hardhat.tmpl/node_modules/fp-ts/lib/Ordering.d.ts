import { Semigroup } from './Semigroup';
import { Eq } from './Eq';
export declare type Ordering = -1 | 0 | 1;
/**
 * @since 1.0.0
 */
export declare const sign: (n: number) => Ordering;
/**
 * @since 1.19.0
 */
export declare const eqOrdering: Eq<Ordering>;
/**
 * Use `eqOrdering`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const setoidOrdering: Eq<Ordering>;
/**
 * @since 1.0.0
 */
export declare const semigroupOrdering: Semigroup<Ordering>;
/**
 * @since 1.0.0
 */
export declare const invert: (O: Ordering) => Ordering;
