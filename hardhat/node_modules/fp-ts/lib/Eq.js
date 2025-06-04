"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var pipeable_1 = require("./pipeable");
/**
 * @since 1.19.0
 */
exports.URI = 'Eq';
/**
 * @since 1.19.0
 */
function fromEquals(equals) {
    return {
        equals: function (x, y) { return x === y || equals(x, y); }
    };
}
exports.fromEquals = fromEquals;
/**
 * @since 1.19.0
 */
function strictEqual(a, b) {
    return a === b;
}
exports.strictEqual = strictEqual;
var eqStrict = { equals: strictEqual };
/**
 * @since 1.19.0
 */
exports.eqString = eqStrict;
/**
 * @since 1.19.0
 */
exports.eqNumber = eqStrict;
/**
 * @since 1.19.0
 */
exports.eqBoolean = eqStrict;
/**
 * @since 1.19.0
 */
function getStructEq(eqs) {
    return fromEquals(function (x, y) {
        for (var k in eqs) {
            if (!eqs[k].equals(x[k], y[k])) {
                return false;
            }
        }
        return true;
    });
}
exports.getStructEq = getStructEq;
/**
 * Given a tuple of `Eq`s returns a `Eq` for the tuple
 *
 * @example
 * import { getTupleEq, eqString, eqNumber, eqBoolean } from 'fp-ts/lib/Eq'
 *
 * const E = getTupleEq(eqString, eqNumber, eqBoolean)
 * assert.strictEqual(E.equals(['a', 1, true], ['a', 1, true]), true)
 * assert.strictEqual(E.equals(['a', 1, true], ['b', 1, true]), false)
 * assert.strictEqual(E.equals(['a', 1, true], ['a', 2, true]), false)
 * assert.strictEqual(E.equals(['a', 1, true], ['a', 1, false]), false)
 *
 * @since 1.19.0
 */
function getTupleEq() {
    var eqs = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        eqs[_i] = arguments[_i];
    }
    return fromEquals(function (x, y) { return eqs.every(function (E, i) { return E.equals(x[i], y[i]); }); });
}
exports.getTupleEq = getTupleEq;
/**
 * @since 1.19.0
 */
exports.eq = {
    URI: exports.URI,
    contramap: function (fa, f) { return fromEquals(function (x, y) { return fa.equals(f(x), f(y)); }); }
};
var contramap = pipeable_1.pipeable(exports.eq).contramap;
exports.contramap = contramap;
/**
 * @since 1.19.0
 */
exports.eqDate = exports.eq.contramap(exports.eqNumber, function (date) { return date.valueOf(); });
