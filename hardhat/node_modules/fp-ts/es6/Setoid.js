/**
 * @file This type class is deprecated, please use `Eq` instead.
 */
/**
 * Use `Eq.fromEquals` instead
 * @since 1.14.0
 * @deprecated
 */
// tslint:disable-next-line: deprecation
export var fromEquals = function (equals) {
    return {
        equals: function (x, y) { return x === y || equals(x, y); }
    };
};
/**
 * Use `Eq.strictEqual` instead
 * @since 1.0.0
 * @deprecated
 */
export var strictEqual = function (a, b) {
    return a === b;
};
// tslint:disable-next-line: deprecation
var setoidStrict = { equals: strictEqual };
/**
 * Use `Eq.eqString` instead
 * @since 1.0.0
 * @deprecated
 */
// tslint:disable-next-line: deprecation
export var setoidString = setoidStrict;
/**
 * Use `Eq.eqNumber` instead
 * @since 1.0.0
 * @deprecated
 */
// tslint:disable-next-line: deprecation
export var setoidNumber = setoidStrict;
/**
 * Use `Eq.eqBoolean` instead
 * @since 1.0.0
 * @deprecated
 */
// tslint:disable-next-line: deprecation
export var setoidBoolean = setoidStrict;
/**
 * Use `Array.getMonoid` instead
 * @since 1.0.0
 * @deprecated
 */
// tslint:disable-next-line: deprecation
export var getArraySetoid = function (S) {
    // tslint:disable-next-line: deprecation
    return fromEquals(function (xs, ys) { return xs.length === ys.length && xs.every(function (x, i) { return S.equals(x, ys[i]); }); });
};
/**
 * Use `Eq.getStructEq` instead
 * @since 1.14.2
 * @deprecated
 */
export var getStructSetoid = function (
// tslint:disable-next-line: deprecation
setoids
// tslint:disable-next-line: deprecation
) {
    // tslint:disable-next-line: deprecation
    return fromEquals(function (x, y) {
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
export var getRecordSetoid = function (
// tslint:disable-next-line: deprecation
setoids
// tslint:disable-next-line: deprecation
) {
    // tslint:disable-next-line: deprecation
    return getStructSetoid(setoids);
};
/**
 * Use `Eq.getTupleEq` instead
 * @since 1.14.2
 * @deprecated
 */
// tslint:disable-next-line: deprecation
export var getTupleSetoid = function () {
    var setoids = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        setoids[_i] = arguments[_i];
    }
    // tslint:disable-next-line: deprecation
    return fromEquals(function (x, y) { return setoids.every(function (S, i) { return S.equals(x[i], y[i]); }); });
};
/**
 * Use `Eq.getTupleEq` instead
 * @since 1.0.0
 * @deprecated
 */
// tslint:disable-next-line: deprecation
export var getProductSetoid = function (SA, SB) {
    // tslint:disable-next-line: deprecation
    return getTupleSetoid(SA, SB);
};
/**
 * Use `Eq.contramap` instead
 * @since 1.2.0
 * @deprecated
 */
// tslint:disable-next-line: deprecation
export var contramap = function (f, fa) {
    // tslint:disable-next-line: deprecation
    return fromEquals(function (x, y) { return fa.equals(f(x), f(y)); });
};
/**
 * Use `Eq.eqDate` instead
 * @since 1.4.0
 * @deprecated
 */
// tslint:disable-next-line: deprecation
export var setoidDate = contramap(function (date) { return date.valueOf(); }, setoidNumber);
