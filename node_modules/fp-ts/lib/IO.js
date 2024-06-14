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
var function_1 = require("./function");
var pipeable_1 = require("./pipeable");
exports.URI = 'IO';
/**
 * @since 1.0.0
 */
var IO = /** @class */ (function () {
    function IO(run) {
        this.run = run;
    }
    /** @obsolete */
    IO.prototype.map = function (f) {
        var _this = this;
        return new IO(function () { return f(_this.run()); });
    };
    /** @obsolete */
    IO.prototype.ap = function (fab) {
        var _this = this;
        return new IO(function () { return fab.run()(_this.run()); });
    };
    /**
     * Flipped version of `ap`
     * @obsolete
     */
    IO.prototype.ap_ = function (fb) {
        return fb.ap(this);
    };
    /**
     * Combine two effectful actions, keeping only the result of the first
     * @since 1.6.0
     * @obsolete
     */
    IO.prototype.applyFirst = function (fb) {
        return fb.ap(this.map(function_1.constant));
    };
    /**
     * Combine two effectful actions, keeping only the result of the second
     * @since 1.5.0
     * @obsolete
     */
    IO.prototype.applySecond = function (fb) {
        // tslint:disable-next-line: deprecation
        return fb.ap(this.map(function_1.constIdentity));
    };
    /** @obsolete */
    IO.prototype.chain = function (f) {
        var _this = this;
        return new IO(function () { return f(_this.run()).run(); });
    };
    IO.prototype.inspect = function () {
        return this.toString();
    };
    IO.prototype.toString = function () {
        // tslint:disable-next-line: deprecation
        return "new IO(" + function_1.toString(this.run) + ")";
    };
    return IO;
}());
exports.IO = IO;
/**
 * @since 1.0.0
 */
exports.getSemigroup = function (S) {
    return {
        concat: function (x, y) {
            return new IO(function () {
                var xr = x.run();
                var yr = y.run();
                return S.concat(xr, yr);
            });
        }
    };
};
/**
 * @since 1.0.0
 */
exports.getMonoid = function (M) {
    return __assign({}, exports.getSemigroup(M), { empty: exports.io.of(M.empty) });
};
/**
 * @since 1.0.0
 */
exports.io = {
    URI: exports.URI,
    map: function (fa, f) { return fa.map(f); },
    of: function (a) { return new IO(function () { return a; }); },
    ap: function (fab, fa) { return fa.ap(fab); },
    chain: function (fa, f) { return fa.chain(f); },
    fromIO: function_1.identity
};
//
// backporting
//
var _a = pipeable_1.pipeable(exports.io), ap = _a.ap, apFirst = _a.apFirst, apSecond = _a.apSecond, chain = _a.chain, chainFirst = _a.chainFirst, flatten = _a.flatten, map = _a.map;
exports.ap = ap;
exports.apFirst = apFirst;
exports.apSecond = apSecond;
exports.chain = chain;
exports.chainFirst = chainFirst;
exports.flatten = flatten;
exports.map = map;
