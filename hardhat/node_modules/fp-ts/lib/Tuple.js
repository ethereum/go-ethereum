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
var Ord_1 = require("./Ord");
var pipeable_1 = require("./pipeable");
var Eq_1 = require("./Eq");
exports.URI = 'Tuple';
/**
 * @since 1.0.0
 */
var Tuple = /** @class */ (function () {
    function Tuple(fst, snd) {
        this.fst = fst;
        this.snd = snd;
    }
    /** @obsolete */
    Tuple.prototype.compose = function (ab) {
        return new Tuple(this.fst, ab.snd);
    };
    /** @obsolete */
    Tuple.prototype.map = function (f) {
        return new Tuple(this.fst, f(this.snd));
    };
    /** @obsolete */
    Tuple.prototype.bimap = function (f, g) {
        return new Tuple(f(this.fst), g(this.snd));
    };
    /** @obsolete */
    Tuple.prototype.extract = function () {
        return this.snd;
    };
    /** @obsolete */
    Tuple.prototype.extend = function (f) {
        return new Tuple(this.fst, f(this));
    };
    /** @obsolete */
    Tuple.prototype.reduce = function (b, f) {
        return f(b, this.snd);
    };
    /**
     * Exchange the first and second components of a tuple
     * @obsolete
     */
    Tuple.prototype.swap = function () {
        return new Tuple(this.snd, this.fst);
    };
    Tuple.prototype.inspect = function () {
        return this.toString();
    };
    Tuple.prototype.toString = function () {
        // tslint:disable-next-line: deprecation
        return "new Tuple(" + function_1.toString(this.fst) + ", " + function_1.toString(this.snd) + ")";
    };
    /** @obsolete */
    Tuple.prototype.toTuple = function () {
        return [this.fst, this.snd];
    };
    return Tuple;
}());
exports.Tuple = Tuple;
/**
 * @since 1.17.0
 */
exports.getShow = function (SL, SA) {
    return {
        show: function (t) { return "new Tuple(" + SL.show(t.fst) + ", " + SA.show(t.snd) + ")"; }
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
function getEq(EL, EA) {
    return Eq_1.fromEquals(function (x, y) { return EL.equals(x.fst, y.fst) && EA.equals(x.snd, y.snd); });
}
exports.getEq = getEq;
/**
 * To obtain the result, the `fst`s are `compare`d, and if they are `EQ`ual, the
 * `snd`s are `compare`d.
 *
 * @since 1.0.0
 */
exports.getOrd = function (OL, OA) {
    return Ord_1.getSemigroup().concat(Ord_1.ord.contramap(OL, fst), Ord_1.ord.contramap(OA, snd));
};
/**
 * @since 1.0.0
 */
exports.getSemigroup = function (SL, SA) {
    return {
        concat: function (x, y) { return new Tuple(SL.concat(x.fst, y.fst), SA.concat(x.snd, y.snd)); }
    };
};
/**
 * @since 1.0.0
 */
exports.getMonoid = function (ML, MA) {
    return {
        concat: exports.getSemigroup(ML, MA).concat,
        empty: new Tuple(ML.empty, MA.empty)
    };
};
/**
 * @since 1.0.0
 */
exports.getApply = function (S) {
    return {
        URI: exports.URI,
        _L: undefined,
        map: exports.tuple.map,
        ap: function (fab, fa) { return new Tuple(S.concat(fab.fst, fa.fst), fab.snd(fa.snd)); }
    };
};
/**
 * @since 1.0.0
 */
exports.getApplicative = function (M) {
    return __assign({}, exports.getApply(M), { of: function (a) { return new Tuple(M.empty, a); } });
};
/**
 * @since 1.0.0
 */
exports.getChain = function (S) {
    return __assign({}, exports.getApply(S), { chain: function (fa, f) {
            var _a = f(fa.snd), fst = _a.fst, snd = _a.snd;
            return new Tuple(S.concat(fa.fst, fst), snd);
        } });
};
/**
 * @since 1.0.0
 */
exports.getMonad = function (M) {
    return __assign({}, exports.getChain(M), { of: function (a) { return new Tuple(M.empty, a); } });
};
/**
 * @since 1.0.0
 */
exports.getChainRec = function (M) {
    return __assign({}, exports.getChain(M), { chainRec: function (a, f) {
            var result = f(a);
            var acc = M.empty;
            while (result.snd.isLeft()) {
                acc = M.concat(acc, result.fst);
                result = f(result.snd.value);
            }
            return new Tuple(M.concat(acc, result.fst), result.snd.value);
        } });
};
/**
 * @since 1.0.0
 */
exports.tuple = {
    URI: exports.URI,
    compose: function (bc, fa) { return fa.compose(bc); },
    map: function (fa, f) { return fa.map(f); },
    bimap: function (fla, f, g) { return fla.bimap(f, g); },
    extract: function (wa) { return wa.extract(); },
    extend: function (wa, f) { return wa.extend(f); },
    reduce: function (fa, b, f) { return fa.reduce(b, f); },
    foldMap: function (_) { return function (fa, f) { return f(fa.snd); }; },
    foldr: function (fa, b, f) { return f(fa.snd, b); },
    traverse: function (F) { return function (ta, f) {
        return F.map(f(ta.snd), function (b) { return new Tuple(ta.fst, b); });
    }; },
    sequence: function (F) { return function (ta) {
        return F.map(ta.snd, function (b) { return new Tuple(ta.fst, b); });
    }; }
};
//
// backporting
//
/**
 * @since 1.19.0
 */
function swap(sa) {
    return sa.swap();
}
exports.swap = swap;
/**
 * @since 1.19.0
 */
function fst(fa) {
    return fa.fst;
}
exports.fst = fst;
/**
 * @since 1.19.0
 */
function snd(fa) {
    return fa.snd;
}
exports.snd = snd;
var _a = pipeable_1.pipeable(exports.tuple), bimap = _a.bimap, compose = _a.compose, duplicate = _a.duplicate, extend = _a.extend, foldMap = _a.foldMap, map = _a.map, mapLeft = _a.mapLeft, reduce = _a.reduce, reduceRight = _a.reduceRight;
exports.bimap = bimap;
exports.compose = compose;
exports.duplicate = duplicate;
exports.extend = extend;
exports.foldMap = foldMap;
exports.map = map;
exports.mapLeft = mapLeft;
exports.reduce = reduce;
exports.reduceRight = reduceRight;
