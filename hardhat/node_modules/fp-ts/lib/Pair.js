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
var Ord_1 = require("./Ord");
var Ordering_1 = require("./Ordering");
var Eq_1 = require("./Eq");
exports.URI = 'Pair';
/**
 * @data
 * @constructor Pair
 * @since 1.0.0
 */
var Pair = /** @class */ (function () {
    function Pair(fst, snd) {
        this.fst = fst;
        this.snd = snd;
    }
    /** Map a function over the first field of a pair */
    Pair.prototype.first = function (f) {
        return new Pair(f(this.fst), this.snd);
    };
    /** Map a function over the second field of a pair */
    Pair.prototype.second = function (f) {
        return new Pair(this.fst, f(this.snd));
    };
    /** Swaps the elements in a pair */
    Pair.prototype.swap = function () {
        return new Pair(this.snd, this.fst);
    };
    Pair.prototype.map = function (f) {
        return new Pair(f(this.fst), f(this.snd));
    };
    Pair.prototype.ap = function (fab) {
        return new Pair(fab.fst(this.fst), fab.snd(this.snd));
    };
    /**
     * Flipped version of `ap`
     */
    Pair.prototype.ap_ = function (fb) {
        return fb.ap(this);
    };
    Pair.prototype.reduce = function (b, f) {
        return f(f(b, this.fst), this.snd);
    };
    Pair.prototype.extract = function () {
        return this.fst;
    };
    Pair.prototype.extend = function (f) {
        return new Pair(f(this), f(this.swap()));
    };
    return Pair;
}());
exports.Pair = Pair;
/**
 * @since 1.17.0
 */
exports.getShow = function (S) {
    return {
        show: function (p) { return "new Pair(" + S.show(p.fst) + ", " + S.show(p.snd) + ")"; }
    };
};
var map = function (fa, f) {
    return fa.map(f);
};
var of = function (a) {
    return new Pair(a, a);
};
var ap = function (fab, fa) {
    return fa.ap(fab);
};
var reduce = function (fa, b, f) {
    return fa.reduce(b, f);
};
var foldMap = function (M) { return function (fa, f) {
    return M.concat(f(fa.fst), f(fa.snd));
}; };
var foldr = function (fa, b, f) {
    return f(fa.fst, f(fa.snd, b));
};
var extract = function (fa) {
    return fa.extract();
};
var extend = function (fa, f) {
    return fa.extend(f);
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
    return Eq_1.fromEquals(function (x, y) { return S.equals(x.fst, y.fst) && S.equals(x.snd, y.snd); });
}
exports.getEq = getEq;
/**
 * @since 1.0.0
 */
exports.getOrd = function (O) {
    return Ord_1.fromCompare(function (x, y) { return Ordering_1.semigroupOrdering.concat(O.compare(x.fst, y.fst), O.compare(x.snd, y.snd)); });
};
/**
 * @since 1.0.0
 */
exports.getSemigroup = function (S) {
    return {
        concat: function (x, y) { return new Pair(S.concat(x.fst, y.fst), S.concat(x.snd, y.snd)); }
    };
};
/**
 * @since 1.0.0
 */
exports.getMonoid = function (M) {
    return __assign({}, exports.getSemigroup(M), { empty: new Pair(M.empty, M.empty) });
};
var traverse = function (F) { return function (ta, f) {
    return F.ap(F.map(f(ta.fst), function (b1) { return function (b2) { return new Pair(b1, b2); }; }), f(ta.snd));
}; };
var sequence = function (F) { return function (ta) {
    return F.ap(F.map(ta.fst, function (a1) { return function (a2) { return new Pair(a1, a2); }; }), ta.snd);
}; };
/**
 * @since 1.0.0
 */
exports.pair = {
    URI: exports.URI,
    map: map,
    of: of,
    ap: ap,
    reduce: reduce,
    foldMap: foldMap,
    foldr: foldr,
    traverse: traverse,
    sequence: sequence,
    extend: extend,
    extract: extract
};
