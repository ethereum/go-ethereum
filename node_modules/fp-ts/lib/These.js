"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var function_1 = require("./function");
var Option_1 = require("./Option");
var Eq_1 = require("./Eq");
var pipeable_1 = require("./pipeable");
exports.URI = 'These';
var This = /** @class */ (function () {
    function This(value) {
        this.value = value;
        this._tag = 'This';
    }
    /** @obsolete */
    This.prototype.map = function (f) {
        return this;
    };
    /** @obsolete */
    This.prototype.bimap = function (f, g) {
        return new This(f(this.value));
    };
    /** @obsolete */
    This.prototype.reduce = function (b, f) {
        return b;
    };
    /**
     * Applies a function to each case in the data structure
     * @obsolete
     */
    This.prototype.fold = function (onLeft, onRight, onBoth) {
        return onLeft(this.value);
    };
    This.prototype.inspect = function () {
        return this.toString();
    };
    This.prototype.toString = function () {
        // tslint:disable-next-line: deprecation
        return "left(" + function_1.toString(this.value) + ")";
    };
    /**
     * Returns `true` if the these is `This`, `false` otherwise
     * @obsolete
     */
    This.prototype.isThis = function () {
        return true;
    };
    /**
     * Returns `true` if the these is `That`, `false` otherwise
     * @obsolete
     */
    This.prototype.isThat = function () {
        return false;
    };
    /**
     * Returns `true` if the these is `Both`, `false` otherwise
     * @obsolete
     */
    This.prototype.isBoth = function () {
        return false;
    };
    return This;
}());
exports.This = This;
var That = /** @class */ (function () {
    function That(value) {
        this.value = value;
        this._tag = 'That';
    }
    That.prototype.map = function (f) {
        return new That(f(this.value));
    };
    That.prototype.bimap = function (f, g) {
        return new That(g(this.value));
    };
    That.prototype.reduce = function (b, f) {
        return f(b, this.value);
    };
    That.prototype.fold = function (onLeft, onRight, onBoth) {
        return onRight(this.value);
    };
    That.prototype.inspect = function () {
        return this.toString();
    };
    That.prototype.toString = function () {
        // tslint:disable-next-line: deprecation
        return "right(" + function_1.toString(this.value) + ")";
    };
    That.prototype.isThis = function () {
        return false;
    };
    That.prototype.isThat = function () {
        return true;
    };
    That.prototype.isBoth = function () {
        return false;
    };
    return That;
}());
exports.That = That;
var Both = /** @class */ (function () {
    function Both(l, a) {
        this.l = l;
        this.a = a;
        this._tag = 'Both';
    }
    Both.prototype.map = function (f) {
        return new Both(this.l, f(this.a));
    };
    Both.prototype.bimap = function (f, g) {
        return new Both(f(this.l), g(this.a));
    };
    Both.prototype.reduce = function (b, f) {
        return f(b, this.a);
    };
    Both.prototype.fold = function (onLeft, onRight, onBoth) {
        return onBoth(this.l, this.a);
    };
    Both.prototype.inspect = function () {
        return this.toString();
    };
    Both.prototype.toString = function () {
        // tslint:disable-next-line: deprecation
        return "both(" + function_1.toString(this.l) + ", " + function_1.toString(this.a) + ")";
    };
    Both.prototype.isThis = function () {
        return false;
    };
    Both.prototype.isThat = function () {
        return false;
    };
    Both.prototype.isBoth = function () {
        return true;
    };
    return Both;
}());
exports.Both = Both;
/**
 * @since 1.17.0
 */
exports.getShow = function (SL, SA) {
    return {
        show: function (t) {
            return t.fold(function (l) { return "left(" + SL.show(l) + ")"; }, function (a) { return "right(" + SA.show(a) + ")"; }, function (l, a) { return "both(" + SL.show(l) + ", " + SA.show(a) + ")"; });
        }
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
    return Eq_1.fromEquals(function (x, y) {
        return x.isThis()
            ? y.isThis() && EL.equals(x.value, y.value)
            : x.isThat()
                ? y.isThat() && EA.equals(x.value, y.value)
                : y.isBoth() && EL.equals(x.l, y.l) && EA.equals(x.a, y.a);
    });
}
exports.getEq = getEq;
/**
 * @since 1.0.0
 */
exports.getSemigroup = function (SL, SA) {
    return {
        concat: function (x, y) {
            return x.isThis()
                ? y.isThis()
                    ? exports.left(SL.concat(x.value, y.value))
                    : y.isThat()
                        ? exports.both(x.value, y.value)
                        : exports.both(SL.concat(x.value, y.l), y.a)
                : x.isThat()
                    ? y.isThis()
                        ? exports.both(y.value, x.value)
                        : y.isThat()
                            ? exports.right(SA.concat(x.value, y.value))
                            : exports.both(y.l, SA.concat(x.value, y.a))
                    : y.isThis()
                        ? exports.both(SL.concat(x.l, y.value), x.a)
                        : y.isThat()
                            ? exports.both(x.l, SA.concat(x.a, y.value))
                            : exports.both(SL.concat(x.l, y.l), SA.concat(x.a, y.a));
        }
    };
};
/**
 * Use `right`
 *
 * @since 1.0.0
 * @deprecated
 */
exports.that = function (a) {
    return new That(a);
};
/**
 * @since 1.0.0
 */
exports.getMonad = function (S) {
    var chain = function (fa, f) {
        if (fa.isThis()) {
            return exports.left(fa.value);
        }
        else if (fa.isThat()) {
            return f(fa.value);
        }
        else {
            var fb = f(fa.a);
            return fb.isThis()
                ? exports.left(S.concat(fa.l, fb.value))
                : fb.isThat()
                    ? exports.both(fa.l, fb.value)
                    : exports.both(S.concat(fa.l, fb.l), fb.a);
        }
    };
    return {
        URI: exports.URI,
        _L: undefined,
        map: exports.these.map,
        of: exports.right,
        ap: function (fab, fa) { return chain(fab, function (f) { return exports.these.map(fa, f); }); },
        chain: chain
    };
};
/**
 * Use `left`
 *
 * @since 1.0.0
 * @deprecated
 */
exports.this_ = function (l) {
    return new This(l);
};
/**
 * @since 1.0.0
 */
exports.both = function (l, a) {
    return new Both(l, a);
};
/**
 * Use `toTuple`
 *
 * @since 1.0.0
 * @deprecated
 */
exports.fromThese = function (defaultThis, defaultThat) { return function (fa) {
    return fa.isThis() ? [fa.value, defaultThat] : fa.isThat() ? [defaultThis, fa.value] : [fa.l, fa.a];
}; };
/**
 * Use `getLeft`
 *
 * @since 1.0.0
 * @deprecated
 */
exports.theseLeft = function (fa) {
    return fa.isThis() ? Option_1.some(fa.value) : fa.isThat() ? Option_1.none : Option_1.some(fa.l);
};
/**
 * Use `getRight`
 *
 * @since 1.0.0
 * @deprecated
 */
exports.theseRight = function (fa) {
    return fa.isThis() ? Option_1.none : fa.isThat() ? Option_1.some(fa.value) : Option_1.some(fa.a);
};
/**
 * Use `isLeft`
 *
 * @since 1.0.0
 * @deprecated
 */
exports.isThis = function (fa) {
    return fa.isThis();
};
/**
 * Use `isRight`
 *
 * @since 1.0.0
 * @deprecated
 */
exports.isThat = function (fa) {
    return fa.isThat();
};
/**
 * Returns `true` if the these is an instance of `Both`, `false` otherwise
 *
 * @since 1.0.0
 */
exports.isBoth = function (fa) {
    return fa.isBoth();
};
/**
 * Use `leftOrBoth`
 *
 * @since 1.13.0
 * @deprecated
 */
exports.thisOrBoth = function (defaultThis, ma) {
    return ma.isNone() ? exports.left(defaultThis) : exports.both(defaultThis, ma.value);
};
/**
 * Use `rightOrBoth`
 *
 * @since 1.13.0
 * @deprecated
 */
exports.thatOrBoth = function (defaultThat, ml) {
    return ml.isNone() ? exports.right(defaultThat) : exports.both(ml.value, defaultThat);
};
/**
 * Use `getLeftOnly`
 *
 * @since 1.13.0
 * @deprecated
 */
exports.theseThis = function (fa) {
    return fa.isThis() ? Option_1.some(fa.value) : Option_1.none;
};
/**
 * Use `getRightOnly`
 *
 * @since 1.13.0
 * @deprecated
 */
exports.theseThat = function (fa) {
    return fa.isThat() ? Option_1.some(fa.value) : Option_1.none;
};
/**
 * Takes a pair of `Option`s and attempts to create a `These` from them
 *
 * @example
 * import { fromOptions, left, right, both } from 'fp-ts/lib/These'
 * import { none, some } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(fromOptions(none, none), none)
 * assert.deepStrictEqual(fromOptions(some('a'), none), some(left('a')))
 * assert.deepStrictEqual(fromOptions(none, some(1)), some(right(1)))
 * assert.deepStrictEqual(fromOptions(some('a'), some(1)), some(both('a', 1)))
 *
 * @since 1.13.0
 */
exports.fromOptions = function (fl, fa) {
    return fl.foldL(function () { return fa.fold(Option_1.none, function (a) { return Option_1.some(exports.right(a)); }); }, function (l) { return fa.foldL(function () { return Option_1.some(exports.left(l)); }, function (a) { return Option_1.some(exports.both(l, a)); }); });
};
/**
 * @example
 * import { fromEither, left, right } from 'fp-ts/lib/These'
 * import * as E from 'fp-ts/lib/Either'
 *
 * assert.deepStrictEqual(fromEither(E.left('a')), left('a'))
 * assert.deepStrictEqual(fromEither(E.right(1)), right(1))
 *
 * @since 1.13.0
 */
exports.fromEither = function (fa) {
    return fa.isLeft() ? exports.left(fa.value) : exports.right(fa.value);
};
/**
 * @since 1.0.0
 */
exports.these = {
    URI: exports.URI,
    map: function (fa, f) { return fa.map(f); },
    bimap: function (fla, f, g) { return fla.bimap(f, g); },
    reduce: function (fa, b, f) { return fa.reduce(b, f); },
    foldMap: function (M) { return function (fa, f) { return (fa.isThis() ? M.empty : fa.isThat() ? f(fa.value) : f(fa.a)); }; },
    foldr: function (fa, b, f) { return (fa.isThis() ? b : fa.isThat() ? f(fa.value, b) : f(fa.a, b)); },
    traverse: function (F) { return function (ta, f) {
        return ta.isThis()
            ? F.of(exports.left(ta.value))
            : ta.isThat()
                ? F.map(f(ta.value), exports.right)
                : F.map(f(ta.a), function (b) { return exports.both(ta.l, b); });
    }; },
    sequence: function (F) { return function (ta) {
        return ta.isThis()
            ? F.of(exports.left(ta.value))
            : ta.isThat()
                ? F.map(ta.value, exports.right)
                : F.map(ta.a, function (b) { return exports.both(ta.l, b); });
    }; }
};
//
// backporting
//
/**
 * @since 1.19.0
 */
// tslint:disable-next-line: deprecation
exports.left = exports.this_;
/**
 * @since 1.19.0
 */
// tslint:disable-next-line: deprecation
exports.right = exports.that;
/**
 * Returns `true` if the these is an instance of `Left`, `false` otherwise
 *
 * @since 1.19.0
 */
// tslint:disable-next-line: deprecation
exports.isLeft = exports.isThis;
/**
 * Returns `true` if the these is an instance of `Right`, `false` otherwise
 *
 * @since 1.19.0
 */
// tslint:disable-next-line: deprecation
exports.isRight = exports.isThat;
/**
 * @example
 * import { toTuple, left, right, both } from 'fp-ts/lib/These'
 *
 * assert.deepStrictEqual(toTuple('a', 1)(left('b')), ['b', 1])
 * assert.deepStrictEqual(toTuple('a', 1)(right(2)), ['a', 2])
 * assert.deepStrictEqual(toTuple('a', 1)(both('b', 2)), ['b', 2])
 *
 * @since 1.19.0
 */
// tslint:disable-next-line: deprecation
exports.toTuple = exports.fromThese;
/**
 * Returns an `L` value if possible
 *
 * @example
 * import { getLeft, left, right, both } from 'fp-ts/lib/These'
 * import { none, some } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(getLeft(left('a')), some('a'))
 * assert.deepStrictEqual(getLeft(right(1)), none)
 * assert.deepStrictEqual(getLeft(both('a', 1)), some('a'))
 *
 * @since 1.19.0
 */
// tslint:disable-next-line: deprecation
exports.getLeft = exports.theseLeft;
/**
 * Returns an `A` value if possible
 *
 * @example
 * import { getRight, left, right, both } from 'fp-ts/lib/These'
 * import { none, some } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(getRight(left('a')), none)
 * assert.deepStrictEqual(getRight(right(1)), some(1))
 * assert.deepStrictEqual(getRight(both('a', 1)), some(1))
 *
 * @since 1.19.0
 */
// tslint:disable-next-line: deprecation
exports.getRight = exports.theseRight;
/**
 * @example
 * import { leftOrBoth, left, both } from 'fp-ts/lib/These'
 * import { none, some } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(leftOrBoth('a')(none), left('a'))
 * assert.deepStrictEqual(leftOrBoth('a')(some(1)), both('a', 1))
 *
 * @since 1.19.0
 */
function leftOrBoth(defaultLeft) {
    // tslint:disable-next-line: deprecation
    return function (ma) { return exports.thisOrBoth(defaultLeft, ma); };
}
exports.leftOrBoth = leftOrBoth;
/**
 * @example
 * import { rightOrBoth, right, both } from 'fp-ts/lib/These'
 * import { none, some } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(rightOrBoth(1)(none), right(1))
 * assert.deepStrictEqual(rightOrBoth(1)(some('a')), both('a', 1))
 *
 * @since 1.19.0
 */
function rightOrBoth(defaultRight) {
    // tslint:disable-next-line: deprecation
    return function (me) { return exports.thatOrBoth(defaultRight, me); };
}
exports.rightOrBoth = rightOrBoth;
/**
 * Returns the `L` value if and only if the value is constructed with `Left`
 *
 * @example
 * import { getLeftOnly, left, right, both } from 'fp-ts/lib/These'
 * import { none, some } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(getLeftOnly(left('a')), some('a'))
 * assert.deepStrictEqual(getLeftOnly(right(1)), none)
 * assert.deepStrictEqual(getLeftOnly(both('a', 1)), none)
 *
 * @since 1.19.0
 */
// tslint:disable-next-line: deprecation
exports.getLeftOnly = exports.theseThis;
/**
 * Returns the `A` value if and only if the value is constructed with `Right`
 *
 * @example
 * import { getRightOnly, left, right, both } from 'fp-ts/lib/These'
 * import { none, some } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(getRightOnly(left('a')), none)
 * assert.deepStrictEqual(getRightOnly(right(1)), some(1))
 * assert.deepStrictEqual(getRightOnly(both('a', 1)), none)
 *
 *
 * @since 1.19.0
 */
// tslint:disable-next-line: deprecation
exports.getRightOnly = exports.theseThat;
/**
 * @since 1.19.0
 */
function fold(onLeft, onRight, onBoth) {
    return function (fa) { return fa.fold(onLeft, onRight, onBoth); };
}
exports.fold = fold;
var _a = pipeable_1.pipeable(exports.these), bimap = _a.bimap, foldMap = _a.foldMap, map = _a.map, mapLeft = _a.mapLeft, reduce = _a.reduce, reduceRight = _a.reduceRight;
exports.bimap = bimap;
exports.foldMap = foldMap;
exports.map = map;
exports.mapLeft = mapLeft;
exports.reduce = reduce;
exports.reduceRight = reduceRight;
