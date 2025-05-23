var __assign = (this && this.__assign) || function () {
    __assign = Object.assign || function(t) {
        for (var s, i = 1, n = arguments.length; i < n; i++) {
            s = arguments[i];
            for (var p in s) if (Object.prototype.hasOwnProperty.call(s, p))
                t[p] = s[p];
        }
        return t;
    };
    return __assign.apply(this, arguments);
};
import { getMinMaxDistributiveLattice } from './DistributiveLattice';
/**
 * @since 1.4.0
 */
export var getMinMaxBoundedDistributiveLattice = function (O) { return function (min, max) {
    return __assign({}, getMinMaxDistributiveLattice(O), { zero: min, one: max });
}; };
