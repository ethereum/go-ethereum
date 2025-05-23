"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
/**
 * @since 1.0.0
 */
exports.fieldNumber = {
    add: function (x, y) { return x + y; },
    zero: 0,
    mul: function (x, y) { return x * y; },
    one: 1,
    sub: function (x, y) { return x - y; },
    degree: function (_) { return 1; },
    div: function (x, y) { return x / y; },
    mod: function (x, y) { return x % y; }
};
/**
 * The *greatest common divisor* of two values
 *
 * @since 1.0.0
 */
exports.gcd = function (E, field) {
    var zero = field.zero;
    var f = function (x, y) { return (E.equals(y, zero) ? x : f(y, field.mod(x, y))); };
    return f;
};
/**
 * The *least common multiple* of two values
 *
 * @since 1.0.0
 */
exports.lcm = function (E, F) {
    var zero = F.zero;
    var gcdSF = exports.gcd(E, F);
    return function (x, y) { return (E.equals(x, zero) || E.equals(y, zero) ? zero : F.div(F.mul(x, y), gcdSF(x, y))); };
};
