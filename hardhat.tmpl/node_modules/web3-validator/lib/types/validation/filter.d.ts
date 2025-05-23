import { Filter } from 'web3-types';
/**
 * First we check if all properties in the provided value are expected,
 * then because all Filter properties are optional, we check if the expected properties
 * are defined. If defined and they're not the expected type, we immediately return false,
 * otherwise we return true after all checks pass.
 */
export declare const isFilterObject: (value: Filter) => boolean;
//# sourceMappingURL=filter.d.ts.map