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
import { constant, constIdentity } from './function';
import { pipeable } from './pipeable';
import { Reader } from './Reader';
import * as readerT from './ReaderT';
import * as taskEither from './TaskEither';
var readerTTaskEither = readerT.getReaderT2v(taskEither.taskEither);
export var URI = 'ReaderTaskEither';
/**
 * @since 1.6.0
 */
var ReaderTaskEither = /** @class */ (function () {
    function ReaderTaskEither(value) {
        this.value = value;
    }
    /** Runs the inner `TaskEither` */
    ReaderTaskEither.prototype.run = function (e) {
        return this.value(e).run();
    };
    /** @obsolete */
    ReaderTaskEither.prototype.map = function (f) {
        return new ReaderTaskEither(readerTTaskEither.map(this.value, f));
    };
    /** @obsolete */
    ReaderTaskEither.prototype.ap = function (fab) {
        return new ReaderTaskEither(readerTTaskEither.ap(fab.value, this.value));
    };
    /**
     * Flipped version of `ap`
     * @obsolete
     */
    ReaderTaskEither.prototype.ap_ = function (fb) {
        return fb.ap(this);
    };
    /**
     * Combine two effectful actions, keeping only the result of the first
     * @obsolete
     */
    ReaderTaskEither.prototype.applyFirst = function (fb) {
        return fb.ap(this.map(constant));
    };
    /**
     * Combine two effectful actions, keeping only the result of the second
     * @obsolete
     */
    ReaderTaskEither.prototype.applySecond = function (fb) {
        // tslint:disable-next-line: deprecation
        return fb.ap(this.map(constIdentity));
    };
    /** @obsolete */
    ReaderTaskEither.prototype.chain = function (f) {
        return new ReaderTaskEither(readerTTaskEither.chain(this.value, function (a) { return f(a).value; }));
    };
    /** @obsolete */
    ReaderTaskEither.prototype.fold = function (left, right) {
        var _this = this;
        return new Reader(function (e) { return _this.value(e).fold(left, right); });
    };
    /** @obsolete */
    ReaderTaskEither.prototype.mapLeft = function (f) {
        var _this = this;
        return new ReaderTaskEither(function (e) { return _this.value(e).mapLeft(f); });
    };
    /**
     * Transforms the failure value of the `ReaderTaskEither` into a new `ReaderTaskEither`
     * @obsolete
     */
    ReaderTaskEither.prototype.orElse = function (f) {
        var _this = this;
        return new ReaderTaskEither(function (e) { return _this.value(e).orElse(function (l) { return f(l).value(e); }); });
    };
    /** @obsolete */
    ReaderTaskEither.prototype.alt = function (fy) {
        return this.orElse(function () { return fy; });
    };
    /** @obsolete */
    ReaderTaskEither.prototype.bimap = function (f, g) {
        var _this = this;
        return new ReaderTaskEither(function (e) { return _this.value(e).bimap(f, g); });
    };
    /**
     * @since 1.6.1
     * @obsolete
     */
    ReaderTaskEither.prototype.local = function (f) {
        var _this = this;
        return new ReaderTaskEither(function (e) { return _this.value(f(e)); });
    };
    return ReaderTaskEither;
}());
export { ReaderTaskEither };
/**
 * @since 1.6.0
 */
export var ask = function () {
    return new ReaderTaskEither(function (e) { return taskEither.taskEither.of(e); });
};
/**
 * @since 1.6.0
 */
export var asks = function (f) {
    return new ReaderTaskEither(function (e) { return taskEither.taskEither.of(f(e)); });
};
/**
 * @since 1.6.0
 */
export var local = function (f) { return function (fa) {
    return fa.local(f);
}; };
/**
 * Use `rightTask`
 *
 * @since 1.6.0
 * @deprecated
 */
export var right = function (fa) {
    return new ReaderTaskEither(function () { return taskEither.rightTask(fa); });
};
/**
 * Use `leftTask`
 *
 * @since 1.6.0
 * @deprecated
 */
export var left = function (fa) {
    return new ReaderTaskEither(function () { return taskEither.leftTask(fa); });
};
/**
 * @since 1.6.0
 */
export var fromTaskEither = function (fa) {
    return new ReaderTaskEither(function () { return fa; });
};
var readerTfromReader = readerT.fromReader(taskEither.taskEither);
/**
 * Use `rightReader`
 *
 * @since 1.6.0
 * @deprecated
 */
export var fromReader = function (fa) {
    return new ReaderTaskEither(readerTfromReader(fa));
};
/**
 * @since 1.6.0
 */
export var fromEither = function (fa) {
    return fromTaskEither(taskEither.fromEither(fa));
};
/**
 * Use `rightIO`
 *
 * @since 1.6.0
 * @deprecated
 */
export var fromIO = function (fa) {
    return fromTaskEither(taskEither.rightIO(fa));
};
/**
 * Use `left2v`
 *
 * @since 1.6.0
 * @deprecated
 */
export var fromLeft = function (l) {
    return fromTaskEither(taskEither.left2v(l));
};
/**
 * @since 1.6.0
 */
export var fromIOEither = function (fa) {
    return fromTaskEither(taskEither.fromIOEither(fa));
};
/**
 * @since 1.6.0
 */
export var tryCatch = function (f, onrejected) {
    return new ReaderTaskEither(function (e) { return taskEither.tryCatch(function () { return f(e); }, function (reason) { return onrejected(reason, e); }); });
};
/**
 * @since 1.6.0
 */
export var readerTaskEither = {
    URI: URI,
    map: function (fa, f) { return fa.map(f); },
    of: function (a) { return new ReaderTaskEither(readerTTaskEither.of(a)); },
    ap: function (fab, fa) { return fa.ap(fab); },
    chain: function (fa, f) { return fa.chain(f); },
    alt: function (fx, fy) { return fx.alt(fy); },
    bimap: function (fla, f, g) { return fla.bimap(f, g); },
    fromIO: fromIO,
    // tslint:disable-next-line: deprecation
    fromTask: right,
    // tslint:disable-next-line: deprecation
    throwError: fromLeft,
    fromEither: fromEither,
    // tslint:disable-next-line: deprecation
    fromOption: function (o, e) { return (o.isNone() ? fromLeft(e) : readerTaskEither.of(o.value)); }
};
/**
 * Like `readerTaskEither` but `ap` is sequential
 * @since 1.10.0
 */
export var readerTaskEitherSeq = __assign({}, readerTaskEither, { ap: function (fab, fa) { return fab.chain(function (f) { return fa.map(f); }); } });
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
export var right2v = readerTaskEither.of;
/**
 * @since 1.19.0
 */
// tslint:disable-next-line: deprecation
export var rightReader = fromReader;
/**
 * @since 1.19.0
 */
// tslint:disable-next-line: deprecation
export var rightIO = fromIO;
/**
 * @since 1.19.0
 */
// tslint:disable-next-line: deprecation
export var rightTask = right;
/**
 * @since 1.19.0
 */
// tslint:disable-next-line: deprecation
export var leftTask = left;
var _a = pipeable(readerTaskEither), alt = _a.alt, ap = _a.ap, apFirst = _a.apFirst, apSecond = _a.apSecond, bimap = _a.bimap, chain = _a.chain, chainFirst = _a.chainFirst, flatten = _a.flatten, map = _a.map, mapLeft = _a.mapLeft, fromOption = _a.fromOption, fromPredicate = _a.fromPredicate, filterOrElse = _a.filterOrElse;
export { alt, ap, apFirst, apSecond, bimap, chain, chainFirst, flatten, map, mapLeft, fromOption, fromPredicate, filterOrElse };
