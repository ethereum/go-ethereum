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
import { toString } from './function';
import { ord, getSemigroup as getOrdSemigroup } from './Ord';
import { pipeable } from './pipeable';
import { fromEquals } from './Eq';
export var URI = 'Tuple';
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
        return "new Tuple(" + toString(this.fst) + ", " + toString(this.snd) + ")";
    };
    /** @obsolete */
    Tuple.prototype.toTuple = function () {
        return [this.fst, this.snd];
    };
    return Tuple;
}());
export { Tuple };
/**
 * @since 1.17.0
 */
export var getShow = function (SL, SA) {
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
export var getSetoid = getEq;
/**
 * @since 1.19.0
 */
export function getEq(EL, EA) {
    return fromEquals(function (x, y) { return EL.equals(x.fst, y.fst) && EA.equals(x.snd, y.snd); });
}
/**
 * To obtain the result, the `fst`s are `compare`d, and if they are `EQ`ual, the
 * `snd`s are `compare`d.
 *
 * @since 1.0.0
 */
export var getOrd = function (OL, OA) {
    return getOrdSemigroup().concat(ord.contramap(OL, fst), ord.contramap(OA, snd));
};
/**
 * @since 1.0.0
 */
export var getSemigroup = function (SL, SA) {
    return {
        concat: function (x, y) { return new Tuple(SL.concat(x.fst, y.fst), SA.concat(x.snd, y.snd)); }
    };
};
/**
 * @since 1.0.0
 */
export var getMonoid = function (ML, MA) {
    return {
        concat: getSemigroup(ML, MA).concat,
        empty: new Tuple(ML.empty, MA.empty)
    };
};
/**
 * @since 1.0.0
 */
export var getApply = function (S) {
    return {
        URI: URI,
        _L: undefined,
        map: tuple.map,
        ap: function (fab, fa) { return new Tuple(S.concat(fab.fst, fa.fst), fab.snd(fa.snd)); }
    };
};
/**
 * @since 1.0.0
 */
export var getApplicative = function (M) {
    return __assign({}, getApply(M), { of: function (a) { return new Tuple(M.empty, a); } });
};
/**
 * @since 1.0.0
 */
export var getChain = function (S) {
    return __assign({}, getApply(S), { chain: function (fa, f) {
            var _a = f(fa.snd), fst = _a.fst, snd = _a.snd;
            return new Tuple(S.concat(fa.fst, fst), snd);
        } });
};
/**
 * @since 1.0.0
 */
export var getMonad = function (M) {
    return __assign({}, getChain(M), { of: function (a) { return new Tuple(M.empty, a); } });
};
/**
 * @since 1.0.0
 */
export var getChainRec = function (M) {
    return __assign({}, getChain(M), { chainRec: function (a, f) {
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
export var tuple = {
    URI: URI,
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
export function swap(sa) {
    return sa.swap();
}
/**
 * @since 1.19.0
 */
export function fst(fa) {
    return fa.fst;
}
/**
 * @since 1.19.0
 */
export function snd(fa) {
    return fa.snd;
}
var _a = pipeable(tuple), bimap = _a.bimap, compose = _a.compose, duplicate = _a.duplicate, extend = _a.extend, foldMap = _a.foldMap, map = _a.map, mapLeft = _a.mapLeft, reduce = _a.reduce, reduceRight = _a.reduceRight;
export { bimap, compose, duplicate, extend, foldMap, map, mapLeft, reduce, reduceRight };
