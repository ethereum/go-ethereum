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
import { constIdentity, toString, constant, identity } from './function';
import { pipeable } from './pipeable';
export var URI = 'IO';
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
        return fb.ap(this.map(constant));
    };
    /**
     * Combine two effectful actions, keeping only the result of the second
     * @since 1.5.0
     * @obsolete
     */
    IO.prototype.applySecond = function (fb) {
        // tslint:disable-next-line: deprecation
        return fb.ap(this.map(constIdentity));
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
        return "new IO(" + toString(this.run) + ")";
    };
    return IO;
}());
export { IO };
/**
 * @since 1.0.0
 */
export var getSemigroup = function (S) {
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
export var getMonoid = function (M) {
    return __assign({}, getSemigroup(M), { empty: io.of(M.empty) });
};
/**
 * @since 1.0.0
 */
export var io = {
    URI: URI,
    map: function (fa, f) { return fa.map(f); },
    of: function (a) { return new IO(function () { return a; }); },
    ap: function (fab, fa) { return fa.ap(fab); },
    chain: function (fa, f) { return fa.chain(f); },
    fromIO: identity
};
//
// backporting
//
var _a = pipeable(io), ap = _a.ap, apFirst = _a.apFirst, apSecond = _a.apSecond, chain = _a.chain, chainFirst = _a.chainFirst, flatten = _a.flatten, map = _a.map;
export { ap, apFirst, apSecond, chain, chainFirst, flatten, map };
