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
import { identity, concat } from './function';
import { fold as foldSemigroup, getDictionarySemigroup, getDualSemigroup, getFunctionSemigroup, getJoinSemigroup, getMeetSemigroup, semigroupAll, semigroupAny, semigroupProduct, semigroupString, semigroupSum, semigroupVoid, getStructSemigroup, getTupleSemigroup } from './Semigroup';
/**
 * @since 1.0.0
 */
export var fold = function (M) {
    return foldSemigroup(M)(M.empty);
};
/**
 * Given a tuple of monoids returns a monoid for the tuple
 *
 * @example
 * import { getTupleMonoid, monoidString, monoidSum, monoidAll } from 'fp-ts/lib/Monoid'
 *
 * const M1 = getTupleMonoid(monoidString, monoidSum)
 * assert.deepStrictEqual(M1.concat(['a', 1], ['b', 2]), ['ab', 3])
 *
 * const M2 = getTupleMonoid(monoidString, monoidSum, monoidAll)
 * assert.deepStrictEqual(M2.concat(['a', 1, true], ['b', 2, false]), ['ab', 3, false])
 *
 * @since 1.0.0
 */
export var getTupleMonoid = function () {
    var monoids = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        monoids[_i] = arguments[_i];
    }
    return __assign({}, getTupleSemigroup.apply(void 0, monoids), { empty: monoids.map(function (m) { return m.empty; }) });
};
/**
 * Use `getTupleMonoid` instead
 * @since 1.0.0
 * @deprecated
 */
export var getProductMonoid = function (MA, MB) {
    return getTupleMonoid(MA, MB);
};
/**
 * @since 1.0.0
 */
export var getDualMonoid = function (M) {
    return __assign({}, getDualSemigroup(M), { empty: M.empty });
};
/**
 * Boolean monoid under conjunction
 * @since 1.0.0
 */
export var monoidAll = __assign({}, semigroupAll, { empty: true });
/**
 * Boolean monoid under disjunction
 * @since 1.0.0
 */
export var monoidAny = __assign({}, semigroupAny, { empty: false });
var emptyArray = [];
/**
 * @since 1.0.0
 */
export var unsafeMonoidArray = {
    concat: concat,
    empty: emptyArray
};
/**
 * Use `Array`'s `getMonoid`
 *
 * @since 1.0.0
 * @deprecated
 */
export var getArrayMonoid = function () {
    return unsafeMonoidArray;
};
var emptyObject = {};
export function getDictionaryMonoid(S) {
    return __assign({}, getDictionarySemigroup(S), { empty: emptyObject });
}
/**
 * Number monoid under addition
 * @since 1.0.0
 */
export var monoidSum = __assign({}, semigroupSum, { empty: 0 });
/**
 * Number monoid under multiplication
 * @since 1.0.0
 */
export var monoidProduct = __assign({}, semigroupProduct, { empty: 1 });
/**
 * @since 1.0.0
 */
export var monoidString = __assign({}, semigroupString, { empty: '' });
/**
 * @since 1.0.0
 */
export var monoidVoid = __assign({}, semigroupVoid, { empty: undefined });
/**
 * @since 1.0.0
 */
export var getFunctionMonoid = function (M) { return function () {
    return __assign({}, getFunctionSemigroup(M)(), { empty: function () { return M.empty; } });
}; };
/**
 * @since 1.0.0
 */
export var getEndomorphismMonoid = function () {
    return {
        concat: function (x, y) { return function (a) { return x(y(a)); }; },
        empty: identity
    };
};
/**
 * @since 1.14.0
 */
export var getStructMonoid = function (monoids) {
    var empty = {};
    for (var _i = 0, _a = Object.keys(monoids); _i < _a.length; _i++) {
        var key = _a[_i];
        empty[key] = monoids[key].empty;
    }
    return __assign({}, getStructSemigroup(monoids), { empty: empty });
};
/**
 * Use `getStructMonoid` instead
 * @since 1.0.0
 * @deprecated
 */
export var getRecordMonoid = function (monoids) {
    return getStructMonoid(monoids);
};
/**
 * @since 1.9.0
 */
export var getMeetMonoid = function (B) {
    return __assign({}, getMeetSemigroup(B), { empty: B.top });
};
/**
 * @since 1.9.0
 */
export var getJoinMonoid = function (B) {
    return __assign({}, getJoinSemigroup(B), { empty: B.bottom });
};
