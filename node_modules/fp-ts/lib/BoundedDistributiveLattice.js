"use strict";
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
Object.defineProperty(exports, "__esModule", { value: true });
var DistributiveLattice_1 = require("./DistributiveLattice");
/**
 * @since 1.4.0
 */
exports.getMinMaxBoundedDistributiveLattice = function (O) { return function (min, max) {
    return __assign({}, DistributiveLattice_1.getMinMaxDistributiveLattice(O), { zero: min, one: max });
}; };
