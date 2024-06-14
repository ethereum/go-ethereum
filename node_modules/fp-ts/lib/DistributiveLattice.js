"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var Ord_1 = require("./Ord");
/**
 * @since 1.4.0
 */
exports.getMinMaxDistributiveLattice = function (O) {
    return {
        meet: Ord_1.min(O),
        join: Ord_1.max(O)
    };
};
