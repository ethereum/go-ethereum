"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
function flatten(chain) {
    return function (mma) { return chain.chain(mma, function (ma) { return ma; }); };
}
exports.flatten = flatten;
