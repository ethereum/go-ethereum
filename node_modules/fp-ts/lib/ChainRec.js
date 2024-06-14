"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
/**
 * @since 1.0.0
 */
exports.tailRec = function (f, a) {
    var v = f(a);
    while (v.isLeft()) {
        v = f(v.value);
    }
    return v.value;
};
