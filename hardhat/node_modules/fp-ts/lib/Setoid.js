"use strict";
/**
 * @file This type class is deprecated, please use `Eq` instead.
 */
Object.defineProperty(exports, "__esModule", { value: true });
/**
 * Use `Eq.fromEquals` instead
 * @since 1.14.0
 * @deprecated
 */
// tslint:disable-next-line: deprecation
exports.fromEquals = function (equals) {
    return {
        equals: function (x, y) { return x === y || equals(x, y); }
    };
};
/**
 * Use `Eq.strictEqual` instead
 * @since 1.0.0
 * @deprecated
 */
exports.strictEqual = function (a, b) {
    return a === b;
};
// tslint:disable-next-line: deprecation
var setoidStrict = { equals: exports.strictEqual };
/**
 * Use `Eq.eqString` instead
 * @since 1.0.0
 * @deprecated
 */
// tslint:disable-next-line: deprecation
exports.setoidString = setoidStrict;
/**
 * Use `Eq.eqNumber` instead
 * @since 1.0.0
 * @deprecated
 */
// tslint:disable-next-line: deprecation
exports.setoidNumber = setoidStrict;
/**
 * Use `Eq.eqBoolean` instead
 * @since 1.0.0
 * @deprecated
 */
// tslint:disable-next-line: deprecation
exports.setoidBoolean = setoidStrict;
/**
 * Use `Array.getMonoid` instead
 * @since 1.0.0
 * @deprecated
 */
// tslint:disable-next-line: deprecation
exports.getArraySetoid = function (S) {
    // tslint:disable-next-line: deprecation
    return exports.fromEquals(function (xs, ys) { return xs.length === ys.length && xs.every(function (x, i) { return S.equals(x, ys[i]); }); });
};
/**
 * Use `Eq.getStructEq` instead
 * @since 1.14.2
 * @deprecated
 */
exports.getStructSetoid = function (
// tslint:disable-next-line: deprecation
setoids
// tslint:disable-next-line: deprecation
) {
    // tslint:disable-next-line: deprecation
    return exports.fromEquals(function (x, y) {
        for (var k in setoids) {
            if (!setoids[k].equals(x[k], y[k])) {
                return false;
            }
        }
        return true;
    });
};
/**
 * Use `Eq.getStructEq` instead
 * @since 1.0.0
 * @deprecated
 */
exports.getRecordSetoid = function (
// tslint:disable-next-line: deprecation
setoids
// tslint:disable-next-line: deprecation
) {
    // tslint:disable-next-line: deprecation
    return exports.getStructSetoid(setoids);
};
/**
 * Use `Eq.getTupleEq` instead
 * @since 1.14.2
 * @deprecated
 */
// tslint:disable-next-line: deprecation
exports.getTupleSetoid = function () {
    var setoids = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        setoids[_i] = arguments[_i];
    }
    // tslint:disable-next-line: deprecation
    return exports.fromEquals(function (x, y) { return setoids.every(function (S, i) { return S.equals(x[i], y[i]); }); });
};
/**
 * Use `Eq.getTupleEq` instead
 * @since 1.0.0
 * @deprecated
 */
// tslint:disable-next-line: deprecation
exports.getProductSetoid = function (SA, SB) {
    // tslint:disable-next-line: deprecation
    return exports.getTupleSetoid(SA, SB);
};
/**
 * Use `Eq.contramap` instead
 * @since 1.2.0
 * @deprecated
 */
// tslint:disable-next-line: deprecation
exports.contramap = function (f, fa) {
    // tslint:disable-next-line: deprecation
    return exports.fromEquals(function (x, y) { return fa.equals(f(x), f(y)); });
};
/**
 * Use `Eq.eqDate` instead
 * @since 1.4.0
 * @deprecated
 */
// tslint:disable-next-line: deprecation
exports.setoidDate = exports.contramap(function (date) { return date.valueOf(); }, exports.setoidNumber);
