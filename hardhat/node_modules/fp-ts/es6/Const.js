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
import { fromEquals } from './Eq';
import { toString } from './function';
import { pipeable } from './pipeable';
export var URI = 'Const';
/**
 * @since 1.0.0
 */
var Const = /** @class */ (function () {
    /**
     * Use `make`
     *
     * @deprecated
     */
    function Const(value) {
        this.value = value;
    }
    /** @obsolete */
    Const.prototype.map = function (f) {
        return this;
    };
    /** @obsolete */
    Const.prototype.contramap = function (f) {
        return this;
    };
    /** @obsolete */
    Const.prototype.fold = function (f) {
        return f(this.value);
    };
    Const.prototype.inspect = function () {
        return this.toString();
    };
    Const.prototype.toString = function () {
        // tslint:disable-next-line: deprecation
        return "make(" + toString(this.value) + ")";
    };
    return Const;
}());
export { Const };
/**
 * @since 1.17.0
 */
export var getShow = function (S) {
    return {
        show: function (c) { return "make(" + S.show(c.value) + ")"; }
    };
};
/**
 * Use `getEq`
 *
 * @since 1.0.0
 * @deprecated
 */
export var getSetoid = getEq;
/**
 * @since 1.19.0
 */
export function getEq(S) {
    return fromEquals(function (x, y) { return S.equals(x.value, y.value); });
}
/**
 * @since 1.0.0
 */
export var getApply = function (S) {
    return {
        URI: URI,
        _L: undefined,
        map: const_.map,
        ap: function (fab, fa) { return make(S.concat(fab.value, fa.value)); }
    };
};
/**
 * @since 1.0.0
 */
export var getApplicative = function (M) {
    return __assign({}, getApply(M), { of: function () { return make(M.empty); } });
};
/**
 * @since 1.0.0
 */
export var const_ = {
    URI: URI,
    map: function (fa, f) { return fa.map(f); },
    contramap: function (fa, f) { return fa.contramap(f); }
};
//
// backporting
//
/**
 * @since 1.19.0
 */
export function make(l) {
    // tslint:disable-next-line: deprecation
    return new Const(l);
}
var _a = pipeable(const_), contramap = _a.contramap, map = _a.map;
export { contramap, map };
