import { max, min } from './Ord';
/**
 * @since 1.4.0
 */
export var getMinMaxDistributiveLattice = function (O) {
    return {
        meet: min(O),
        join: max(O)
    };
};
