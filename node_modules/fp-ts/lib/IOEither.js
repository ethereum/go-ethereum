"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var Either_1 = require("./Either");
var eitherT = require("./EitherT");
var function_1 = require("./function");
var IO_1 = require("./IO");
var pipeable_1 = require("./pipeable");
exports.URI = 'IOEither';
var T = eitherT.getEitherT2v(IO_1.io);
var foldT = eitherT.fold(IO_1.io);
/**
 * @since 1.6.0
 */
var IOEither = /** @class */ (function () {
    function IOEither(value) {
        this.value = value;
    }
    /**
     * Runs the inner io
     */
    IOEither.prototype.run = function () {
        return this.value.run();
    };
    /** @obsolete */
    IOEither.prototype.map = function (f) {
        return new IOEither(T.map(this.value, f));
    };
    /** @obsolete */
    IOEither.prototype.ap = function (fab) {
        return new IOEither(T.ap(fab.value, this.value));
    };
    /**
     * Flipped version of `ap`
     * @obsolete
     */
    IOEither.prototype.ap_ = function (fb) {
        return fb.ap(this);
    };
    /**
     * Combine two effectful actions, keeping only the result of the first
     * @obsolete
     */
    IOEither.prototype.applyFirst = function (fb) {
        return fb.ap(this.map(function_1.constant));
    };
    /**
     * Combine two effectful actions, keeping only the result of the second
     * @obsolete
     */
    IOEither.prototype.applySecond = function (fb) {
        // tslint:disable-next-line: deprecation
        return fb.ap(this.map(function_1.constIdentity));
    };
    /** @obsolete */
    IOEither.prototype.chain = function (f) {
        return new IOEither(T.chain(this.value, function (a) { return f(a).value; }));
    };
    /** @obsolete */
    IOEither.prototype.fold = function (left, right) {
        return foldT(left, right, this.value);
    };
    /**
     * Similar to `fold`, but the result is flattened.
     *
     * @since 1.19.0
     * @obsolete
     */
    IOEither.prototype.foldIO = function (left, right) {
        return this.value.chain(function (fa) { return fa.fold(left, right); });
    };
    /**
     * Similar to `fold`, but the result is flattened.
     *
     * @since 1.19.0
     * @obsolete
     */
    IOEither.prototype.foldIOEither = function (onLeft, onRight) {
        return new IOEither(this.value.chain(function (e) { return e.fold(onLeft, onRight).value; }));
    };
    /** @obsolete */
    IOEither.prototype.mapLeft = function (f) {
        return new IOEither(this.value.map(function (e) { return e.mapLeft(f); }));
    };
    /** @obsolete */
    IOEither.prototype.orElse = function (f) {
        return new IOEither(this.value.chain(function (e) { return e.fold(function (l) { return f(l).value; }, function (a) { return T.of(a); }); }));
    };
    /** @obsolete */
    IOEither.prototype.alt = function (fy) {
        return this.orElse(function () { return fy; });
    };
    /** @obsolete */
    IOEither.prototype.bimap = function (f, g) {
        return new IOEither(this.value.map(function (e) { return e.bimap(f, g); }));
    };
    return IOEither;
}());
exports.IOEither = IOEither;
/**
 * Use `rightIO`
 *
 * @since 1.6.0
 * @deprecated
 */
exports.right = function (fa) {
    return new IOEither(fa.map(Either_1.right));
};
/**
 * Use `leftIO`
 *
 * @since 1.6.0
 * @deprecated
 */
exports.left = function (fa) {
    return new IOEither(fa.map(Either_1.left));
};
/**
 * @since 1.6.0
 */
exports.fromEither = function (fa) {
    return new IOEither(IO_1.io.of(fa));
};
/**
 * Use `left2v`
 *
 * @since 1.6.0
 * @deprecated
 */
exports.fromLeft = function (l) {
    return exports.fromEither(Either_1.left(l));
};
/**
 * Use `tryCatch2v` instead
 *
 * @since 1.6.0
 * @deprecated
 */
exports.tryCatch = function (f, onerror) {
    if (onerror === void 0) { onerror = Either_1.toError; }
    return exports.tryCatch2v(f, onerror);
};
/**
 * @since 1.11.0
 */
exports.tryCatch2v = function (f, onerror) {
    return new IOEither(new IO_1.IO(function () { return Either_1.tryCatch2v(f, onerror); }));
};
/**
 * @since 1.6.0
 */
exports.ioEither = {
    URI: exports.URI,
    bimap: function (fla, f, g) { return fla.bimap(f, g); },
    map: function (fa, f) { return fa.map(f); },
    of: right2v,
    ap: function (fab, fa) { return fa.ap(fab); },
    chain: function (fa, f) { return fa.chain(f); },
    alt: function (fx, fy) { return fx.alt(fy); },
    // tslint:disable-next-line: deprecation
    throwError: exports.fromLeft,
    fromEither: exports.fromEither,
    // tslint:disable-next-line: deprecation
    fromOption: function (o, e) { return (o.isNone() ? exports.fromLeft(e) : exports.ioEither.of(o.value)); }
};
//
// backporting
//
/**
 * @since 1.19.0
 */
// tslint:disable-next-line: deprecation
exports.left2v = exports.fromLeft;
/**
 * @since 1.19.0
 */
function right2v(a) {
    return new IOEither(T.of(a));
}
exports.right2v = right2v;
/**
 * @since 1.19.0
 */
// tslint:disable-next-line: deprecation
exports.rightIO = exports.right;
/**
 * @since 1.19.0
 */
// tslint:disable-next-line: deprecation
exports.leftIO = exports.left;
/**
 * @since 1.19.0
 */
function fold(onLeft, onRight) {
    return function (ma) { return ma.foldIO(onLeft, onRight); };
}
exports.fold = fold;
/**
 * @since 1.19.0
 */
function orElse(f) {
    return function (ma) { return ma.orElse(f); };
}
exports.orElse = orElse;
var _a = pipeable_1.pipeable(exports.ioEither), alt = _a.alt, ap = _a.ap, apFirst = _a.apFirst, apSecond = _a.apSecond, bimap = _a.bimap, chain = _a.chain, chainFirst = _a.chainFirst, flatten = _a.flatten, map = _a.map, mapLeft = _a.mapLeft, fromOption = _a.fromOption, fromPredicate = _a.fromPredicate, filterOrElse = _a.filterOrElse;
exports.alt = alt;
exports.ap = ap;
exports.apFirst = apFirst;
exports.apSecond = apSecond;
exports.bimap = bimap;
exports.chain = chain;
exports.chainFirst = chainFirst;
exports.flatten = flatten;
exports.map = map;
exports.mapLeft = mapLeft;
exports.fromOption = fromOption;
exports.fromPredicate = fromPredicate;
exports.filterOrElse = filterOrElse;
