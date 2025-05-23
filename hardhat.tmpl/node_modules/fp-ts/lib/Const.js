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
var pipeable_1 = require("./pipeable");
exports.URI = 'Const';
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
        return "make(" + function_1.toString(this.value) + ")";
    };
    return Const;
}());
exports.Const = Const;
/**
 * @since 1.17.0
 */
exports.getShow = function (S) {
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
exports.getSetoid = getEq;
/**
 * @since 1.19.0
 */
function getEq(S) {
    return Eq_1.fromEquals(function (x, y) { return S.equals(x.value, y.value); });
}
exports.getEq = getEq;
/**
 * @since 1.0.0
 */
exports.getApply = function (S) {
    return {
        URI: exports.URI,
        _L: undefined,
        map: exports.const_.map,
        ap: function (fab, fa) { return make(S.concat(fab.value, fa.value)); }
    };
};
/**
 * @since 1.0.0
 */
exports.getApplicative = function (M) {
    return __assign({}, exports.getApply(M), { of: function () { return make(M.empty); } });
};
/**
 * @since 1.0.0
 */
exports.const_ = {
    URI: exports.URI,
    map: function (fa, f) { return fa.map(f); },
    contramap: function (fa, f) { return fa.contramap(f); }
};
//
// backporting
//
/**
 * @since 1.19.0
 */
function make(l) {
    // tslint:disable-next-line: deprecation
    return new Const(l);
}
exports.make = make;
var _a = pipeable_1.pipeable(exports.const_), contramap = _a.contramap, map = _a.map;
exports.contramap = contramap;
exports.map = map;
