"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
/**
 * Log any value to the console for debugging purposes and then return a value. This will log the value's underlying
 * representation for low-level debugging
 *
 * @since 1.0.0
 */
exports.trace = function (message, out) {
    console.log(message); // tslint:disable-line:no-console
    return out();
};
/**
 * Log any value and return it
 *
 * @since 1.0.0
 */
exports.spy = function (a) {
    return exports.trace(a, function () { return a; });
};
function traceA(F) {
    return function (x) { return exports.trace(x, function () { return F.of(undefined); }); };
}
exports.traceA = traceA;
function traceM(F) {
    return function (a) { return exports.trace(a, function () { return F.of(a); }); };
}
exports.traceM = traceM;
