/**
 * Log any value to the console for debugging purposes and then return a value. This will log the value's underlying
 * representation for low-level debugging
 *
 * @since 1.0.0
 */
export var trace = function (message, out) {
    console.log(message); // tslint:disable-line:no-console
    return out();
};
/**
 * Log any value and return it
 *
 * @since 1.0.0
 */
export var spy = function (a) {
    return trace(a, function () { return a; });
};
export function traceA(F) {
    return function (x) { return trace(x, function () { return F.of(undefined); }); };
}
export function traceM(F) {
    return function (a) { return trace(a, function () { return F.of(a); }); };
}
