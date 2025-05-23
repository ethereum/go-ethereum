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
var Eq_1 = require("./Eq");
var function_1 = require("./function");
var Ordering_1 = require("./Ordering");
/**
 * @since 1.19.0
 */
exports.URI = 'Ord';
/**
 * @since 1.0.0
 * @deprecated
 */
exports.unsafeCompare = function (x, y) {
    return x < y ? -1 : x > y ? 1 : 0;
};
/**
 * @since 1.0.0
 */
exports.ordString = __assign({}, Eq_1.eqString, { 
    // tslint:disable-next-line: deprecation
    compare: exports.unsafeCompare });
/**
 * @since 1.0.0
 */
exports.ordNumber = __assign({}, Eq_1.eqNumber, { 
    // tslint:disable-next-line: deprecation
    compare: exports.unsafeCompare });
/**
 * @since 1.0.0
 */
exports.ordBoolean = __assign({}, Eq_1.eqBoolean, { 
    // tslint:disable-next-line: deprecation
    compare: exports.unsafeCompare });
/**
 * Test whether one value is _strictly less than_ another
 *
 * @since 1.19.0
 */
exports.lt = function (O) { return function (x, y) {
    return O.compare(x, y) === -1;
}; };
/**
 * Use `lt`
 *
 * @since 1.0.0
 * @deprecated
 */
exports.lessThan = exports.lt;
/**
 * Test whether one value is _strictly greater than_ another
 *
 * @since 1.19.0
 */
exports.gt = function (O) { return function (x, y) {
    return O.compare(x, y) === 1;
}; };
/**
 * Use `gt`
 *
 * @since 1.0.0
 * @deprecated
 */
exports.greaterThan = exports.gt;
/**
 * Test whether one value is _non-strictly less than_ another
 *
 * @since 1.19.0
 */
exports.leq = function (O) { return function (x, y) {
    return O.compare(x, y) !== 1;
}; };
/**
 * Use `leq`
 *
 * @since 1.0.0
 * @deprecated
 */
exports.lessThanOrEq = exports.leq;
/**
 * Test whether one value is _non-strictly greater than_ another
 *
 * @since 1.19.0
 */
exports.geq = function (O) { return function (x, y) {
    return O.compare(x, y) !== -1;
}; };
/**
 * Use `geq`
 *
 * @since 1.0.0
 * @deprecated
 */
exports.greaterThanOrEq = exports.geq;
/**
 * Take the minimum of two values. If they are considered equal, the first argument is chosen
 *
 * @since 1.0.0
 */
exports.min = function (O) { return function (x, y) {
    return O.compare(x, y) === 1 ? y : x;
}; };
/**
 * Take the maximum of two values. If they are considered equal, the first argument is chosen
 *
 * @since 1.0.0
 */
exports.max = function (O) { return function (x, y) {
    return O.compare(x, y) === -1 ? y : x;
}; };
/**
 * Clamp a value between a minimum and a maximum
 *
 * @since 1.0.0
 */
exports.clamp = function (O) {
    var minO = exports.min(O);
    var maxO = exports.max(O);
    return function (low, hi) { return function (x) { return maxO(minO(x, hi), low); }; };
};
/**
 * Test whether a value is between a minimum and a maximum (inclusive)
 *
 * @since 1.0.0
 */
exports.between = function (O) {
    var lessThanO = exports.lt(O);
    var greaterThanO = exports.gt(O);
    return function (low, hi) { return function (x) { return (lessThanO(x, low) || greaterThanO(x, hi) ? false : true); }; };
};
/**
 * @since 1.0.0
 */
exports.fromCompare = function (compare) {
    var optimizedCompare = function (x, y) { return (x === y ? 0 : compare(x, y)); };
    return {
        equals: function (x, y) { return optimizedCompare(x, y) === 0; },
        compare: optimizedCompare
    };
};
function _contramap(f, O) {
    // tslint:disable-next-line: deprecation
    return exports.fromCompare(function_1.on(O.compare)(f));
}
function contramap() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1 ? function (O) { return _contramap(args[0], O); } : _contramap(args[0], args[1]);
}
exports.contramap = contramap;
/**
 * @since 1.0.0
 */
exports.getSemigroup = function () {
    return {
        concat: function (x, y) { return exports.fromCompare(function (a, b) { return Ordering_1.semigroupOrdering.concat(x.compare(a, b), y.compare(a, b)); }); }
    };
};
/**
 * Given a tuple of `Ord`s returns an `Ord` for the tuple
 *
 * @example
 * import { getTupleOrd, ordString, ordNumber, ordBoolean } from 'fp-ts/lib/Ord'
 *
 * const O = getTupleOrd(ordString, ordNumber, ordBoolean)
 * assert.strictEqual(O.compare(['a', 1, true], ['b', 2, true]), -1)
 * assert.strictEqual(O.compare(['a', 1, true], ['a', 2, true]), -1)
 * assert.strictEqual(O.compare(['a', 1, true], ['a', 1, false]), 1)
 *
 * @since 1.14.3
 */
exports.getTupleOrd = function () {
    var ords = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        ords[_i] = arguments[_i];
    }
    var len = ords.length;
    return exports.fromCompare(function (x, y) {
        var i = 0;
        for (; i < len - 1; i++) {
            var r = ords[i].compare(x[i], y[i]);
            if (r !== 0) {
                return r;
            }
        }
        return ords[i].compare(x[i], y[i]);
    });
};
/**
 * Use `getTupleOrd` instead
 * @since 1.0.0
 * @deprecated
 */
exports.getProductOrd = function (OA, OB) {
    return exports.getTupleOrd(OA, OB);
};
/**
 * @since 1.3.0
 */
exports.getDualOrd = function (O) {
    return exports.fromCompare(function (x, y) { return O.compare(y, x); });
};
/**
 * @since 1.19.0
 */
exports.ord = {
    URI: exports.URI,
    // tslint:disable-next-line: deprecation
    contramap: function (fa, f) { return contramap(f, fa); }
};
/**
 * @since 1.4.0
 */
exports.ordDate = exports.ord.contramap(exports.ordNumber, function (date) { return date.valueOf(); });
