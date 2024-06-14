import { left as eitherLeft, right as eitherRight, toError, tryCatch2v as eitherTryCatch2v } from './Either';
import * as eitherT from './EitherT';
import { constant, constIdentity } from './function';
import { IO, io } from './IO';
import { pipeable } from './pipeable';
export var URI = 'IOEither';
var T = eitherT.getEitherT2v(io);
var foldT = eitherT.fold(io);
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
        return fb.ap(this.map(constant));
    };
    /**
     * Combine two effectful actions, keeping only the result of the second
     * @obsolete
     */
    IOEither.prototype.applySecond = function (fb) {
        // tslint:disable-next-line: deprecation
        return fb.ap(this.map(constIdentity));
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
export { IOEither };
/**
 * Use `rightIO`
 *
 * @since 1.6.0
 * @deprecated
 */
export var right = function (fa) {
    return new IOEither(fa.map(eitherRight));
};
/**
 * Use `leftIO`
 *
 * @since 1.6.0
 * @deprecated
 */
export var left = function (fa) {
    return new IOEither(fa.map(eitherLeft));
};
/**
 * @since 1.6.0
 */
export var fromEither = function (fa) {
    return new IOEither(io.of(fa));
};
/**
 * Use `left2v`
 *
 * @since 1.6.0
 * @deprecated
 */
export var fromLeft = function (l) {
    return fromEither(eitherLeft(l));
};
/**
 * Use `tryCatch2v` instead
 *
 * @since 1.6.0
 * @deprecated
 */
export var tryCatch = function (f, onerror) {
    if (onerror === void 0) { onerror = toError; }
    return tryCatch2v(f, onerror);
};
/**
 * @since 1.11.0
 */
export var tryCatch2v = function (f, onerror) {
    return new IOEither(new IO(function () { return eitherTryCatch2v(f, onerror); }));
};
/**
 * @since 1.6.0
 */
export var ioEither = {
    URI: URI,
    bimap: function (fla, f, g) { return fla.bimap(f, g); },
    map: function (fa, f) { return fa.map(f); },
    of: right2v,
    ap: function (fab, fa) { return fa.ap(fab); },
    chain: function (fa, f) { return fa.chain(f); },
    alt: function (fx, fy) { return fx.alt(fy); },
    // tslint:disable-next-line: deprecation
    throwError: fromLeft,
    fromEither: fromEither,
    // tslint:disable-next-line: deprecation
    fromOption: function (o, e) { return (o.isNone() ? fromLeft(e) : ioEither.of(o.value)); }
};
//
// backporting
//
/**
 * @since 1.19.0
 */
// tslint:disable-next-line: deprecation
export var left2v = fromLeft;
/**
 * @since 1.19.0
 */
export function right2v(a) {
    return new IOEither(T.of(a));
}
/**
 * @since 1.19.0
 */
// tslint:disable-next-line: deprecation
export var rightIO = right;
/**
 * @since 1.19.0
 */
// tslint:disable-next-line: deprecation
export var leftIO = left;
/**
 * @since 1.19.0
 */
export function fold(onLeft, onRight) {
    return function (ma) { return ma.foldIO(onLeft, onRight); };
}
/**
 * @since 1.19.0
 */
export function orElse(f) {
    return function (ma) { return ma.orElse(f); };
}
var _a = pipeable(ioEither), alt = _a.alt, ap = _a.ap, apFirst = _a.apFirst, apSecond = _a.apSecond, bimap = _a.bimap, chain = _a.chain, chainFirst = _a.chainFirst, flatten = _a.flatten, map = _a.map, mapLeft = _a.mapLeft, fromOption = _a.fromOption, fromPredicate = _a.fromPredicate, filterOrElse = _a.filterOrElse;
export { alt, ap, apFirst, apSecond, bimap, chain, chainFirst, flatten, map, mapLeft, fromOption, fromPredicate, filterOrElse };
