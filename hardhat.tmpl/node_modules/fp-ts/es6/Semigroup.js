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
/**
 * @file See [Getting started with fp-ts: Semigroup](https://dev.to/gcanti/getting-started-with-fp-ts-semigroup-2mf7)
 */
import { max, min } from './Ord';
import { concat, identity } from './function';
/**
 * @since 1.0.0
 */
export var fold = function (S) { return function (a) { return function (as) {
    return as.reduce(S.concat, a);
}; }; };
/**
 * @since 1.0.0
 */
export var getFirstSemigroup = function () {
    return { concat: identity };
};
/**
 * @since 1.0.0
 */
export var getLastSemigroup = function () {
    return { concat: function (_, y) { return y; } };
};
/**
 * Given a tuple of semigroups returns a semigroup for the tuple
 *
 * @example
 * import { getTupleSemigroup, semigroupString, semigroupSum, semigroupAll } from 'fp-ts/lib/Semigroup'
 *
 * const S1 = getTupleSemigroup(semigroupString, semigroupSum)
 * assert.deepStrictEqual(S1.concat(['a', 1], ['b', 2]), ['ab', 3])
 *
 * const S2 = getTupleSemigroup(semigroupString, semigroupSum, semigroupAll)
 * assert.deepStrictEqual(S2.concat(['a', 1, true], ['b', 2, false]), ['ab', 3, false])
 *
 * @since 1.14.0
 */
export var getTupleSemigroup = function () {
    var semigroups = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        semigroups[_i] = arguments[_i];
    }
    return {
        concat: function (x, y) { return semigroups.map(function (s, i) { return s.concat(x[i], y[i]); }); }
    };
};
/**
 * Use `getTupleSemigroup` instead
 * @since 1.0.0
 * @deprecated
 */
export var getProductSemigroup = function (SA, SB) {
    return getTupleSemigroup(SA, SB);
};
/**
 * @since 1.0.0
 */
export var getDualSemigroup = function (S) {
    return {
        concat: function (x, y) { return S.concat(y, x); }
    };
};
/**
 * @since 1.0.0
 */
export var getFunctionSemigroup = function (S) { return function () {
    return {
        concat: function (f, g) { return function (a) { return S.concat(f(a), g(a)); }; }
    };
}; };
/**
 * @since 1.14.0
 */
export var getStructSemigroup = function (semigroups) {
    return {
        concat: function (x, y) {
            var r = {};
            for (var _i = 0, _a = Object.keys(semigroups); _i < _a.length; _i++) {
                var key = _a[_i];
                r[key] = semigroups[key].concat(x[key], y[key]);
            }
            return r;
        }
    };
};
/**
 * Use `getStructSemigroup` instead
 * @since 1.0.0
 * @deprecated
 */
export var getRecordSemigroup = function (semigroups) {
    return getStructSemigroup(semigroups);
};
/**
 * @since 1.0.0
 */
export var getMeetSemigroup = function (O) {
    return {
        concat: min(O)
    };
};
/**
 * @since 1.0.0
 */
export var getJoinSemigroup = function (O) {
    return {
        concat: max(O)
    };
};
/**
 * Boolean semigroup under conjunction
 * @since 1.0.0
 */
export var semigroupAll = {
    concat: function (x, y) { return x && y; }
};
/**
 * Boolean semigroup under disjunction
 * @since 1.0.0
 */
export var semigroupAny = {
    concat: function (x, y) { return x || y; }
};
/**
 * Use `Array`'s `getMonoid`
 *
 * @since 1.0.0
 * @deprecated
 */
export var getArraySemigroup = function () {
    return { concat: concat };
};
export function getDictionarySemigroup(S) {
    return {
        concat: function (x, y) {
            var r = __assign({}, x);
            var keys = Object.keys(y);
            var len = keys.length;
            for (var i = 0; i < len; i++) {
                var k = keys[i];
                r[k] = x.hasOwnProperty(k) ? S.concat(x[k], y[k]) : y[k];
            }
            return r;
        }
    };
}
// tslint:disable-next-line: deprecation
var semigroupAnyDictionary = getDictionarySemigroup(getLastSemigroup());
/**
 * Returns a `Semigroup` instance for objects preserving their type
 *
 * @example
 * import { getObjectSemigroup } from 'fp-ts/lib/Semigroup'
 *
 * interface Person {
 *   name: string
 *   age: number
 * }
 *
 * const S = getObjectSemigroup<Person>()
 * assert.deepStrictEqual(S.concat({ name: 'name', age: 23 }, { name: 'name', age: 24 }), { name: 'name', age: 24 })
 *
 * @since 1.4.0
 */
export var getObjectSemigroup = function () {
    return semigroupAnyDictionary;
};
/**
 * Number `Semigroup` under addition
 * @since 1.0.0
 */
export var semigroupSum = {
    concat: function (x, y) { return x + y; }
};
/**
 * Number `Semigroup` under multiplication
 * @since 1.0.0
 */
export var semigroupProduct = {
    concat: function (x, y) { return x * y; }
};
/**
 * @since 1.0.0
 */
export var semigroupString = {
    concat: function (x, y) { return x + y; }
};
/**
 * @since 1.0.0
 */
export var semigroupVoid = {
    concat: function () { return undefined; }
};
