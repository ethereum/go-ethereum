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
var Reader_1 = require("./Reader");
var readerT = require("./ReaderT");
var taskEither = require("./TaskEither");
var readerTTaskEither = readerT.getReaderT2v(taskEither.taskEither);
exports.URI = 'ReaderTaskEither';
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
        return fb.ap(this.map(function_1.constant));
    };
    /**
     * Combine two effectful actions, keeping only the result of the second
     * @obsolete
     */
    ReaderTaskEither.prototype.applySecond = function (fb) {
        // tslint:disable-next-line: deprecation
        return fb.ap(this.map(function_1.constIdentity));
    };
    /** @obsolete */
    ReaderTaskEither.prototype.chain = function (f) {
        return new ReaderTaskEither(readerTTaskEither.chain(this.value, function (a) { return f(a).value; }));
    };
    /** @obsolete */
    ReaderTaskEither.prototype.fold = function (left, right) {
        var _this = this;
        return new Reader_1.Reader(function (e) { return _this.value(e).fold(left, right); });
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
exports.ReaderTaskEither = ReaderTaskEither;
/**
 * @since 1.6.0
 */
exports.ask = function () {
    return new ReaderTaskEither(function (e) { return taskEither.taskEither.of(e); });
};
/**
 * @since 1.6.0
 */
exports.asks = function (f) {
    return new ReaderTaskEither(function (e) { return taskEither.taskEither.of(f(e)); });
};
/**
 * @since 1.6.0
 */
exports.local = function (f) { return function (fa) {
    return fa.local(f);
}; };
/**
 * Use `rightTask`
 *
 * @since 1.6.0
 * @deprecated
 */
exports.right = function (fa) {
    return new ReaderTaskEither(function () { return taskEither.rightTask(fa); });
};
/**
 * Use `leftTask`
 *
 * @since 1.6.0
 * @deprecated
 */
exports.left = function (fa) {
    return new ReaderTaskEither(function () { return taskEither.leftTask(fa); });
};
/**
 * @since 1.6.0
 */
exports.fromTaskEither = function (fa) {
    return new ReaderTaskEither(function () { return fa; });
};
var readerTfromReader = readerT.fromReader(taskEither.taskEither);
/**
 * Use `rightReader`
 *
 * @since 1.6.0
 * @deprecated
 */
exports.fromReader = function (fa) {
    return new ReaderTaskEither(readerTfromReader(fa));
};
/**
 * @since 1.6.0
 */
exports.fromEither = function (fa) {
    return exports.fromTaskEither(taskEither.fromEither(fa));
};
/**
 * Use `rightIO`
 *
 * @since 1.6.0
 * @deprecated
 */
exports.fromIO = function (fa) {
    return exports.fromTaskEither(taskEither.rightIO(fa));
};
/**
 * Use `left2v`
 *
 * @since 1.6.0
 * @deprecated
 */
exports.fromLeft = function (l) {
    return exports.fromTaskEither(taskEither.left2v(l));
};
/**
 * @since 1.6.0
 */
exports.fromIOEither = function (fa) {
    return exports.fromTaskEither(taskEither.fromIOEither(fa));
};
/**
 * @since 1.6.0
 */
exports.tryCatch = function (f, onrejected) {
    return new ReaderTaskEither(function (e) { return taskEither.tryCatch(function () { return f(e); }, function (reason) { return onrejected(reason, e); }); });
};
/**
 * @since 1.6.0
 */
exports.readerTaskEither = {
    URI: exports.URI,
    map: function (fa, f) { return fa.map(f); },
    of: function (a) { return new ReaderTaskEither(readerTTaskEither.of(a)); },
    ap: function (fab, fa) { return fa.ap(fab); },
    chain: function (fa, f) { return fa.chain(f); },
    alt: function (fx, fy) { return fx.alt(fy); },
    bimap: function (fla, f, g) { return fla.bimap(f, g); },
    fromIO: exports.fromIO,
    // tslint:disable-next-line: deprecation
    fromTask: exports.right,
    // tslint:disable-next-line: deprecation
    throwError: exports.fromLeft,
    fromEither: exports.fromEither,
    // tslint:disable-next-line: deprecation
    fromOption: function (o, e) { return (o.isNone() ? exports.fromLeft(e) : exports.readerTaskEither.of(o.value)); }
};
/**
 * Like `readerTaskEither` but `ap` is sequential
 * @since 1.10.0
 */
exports.readerTaskEitherSeq = __assign({}, exports.readerTaskEither, { ap: function (fab, fa) { return fab.chain(function (f) { return fa.map(f); }); } });
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
exports.right2v = exports.readerTaskEither.of;
/**
 * @since 1.19.0
 */
// tslint:disable-next-line: deprecation
exports.rightReader = exports.fromReader;
/**
 * @since 1.19.0
 */
// tslint:disable-next-line: deprecation
exports.rightIO = exports.fromIO;
/**
 * @since 1.19.0
 */
// tslint:disable-next-line: deprecation
exports.rightTask = exports.right;
/**
 * @since 1.19.0
 */
// tslint:disable-next-line: deprecation
exports.leftTask = exports.left;
var _a = pipeable_1.pipeable(exports.readerTaskEither), alt = _a.alt, ap = _a.ap, apFirst = _a.apFirst, apSecond = _a.apSecond, bimap = _a.bimap, chain = _a.chain, chainFirst = _a.chainFirst, flatten = _a.flatten, map = _a.map, mapLeft = _a.mapLeft, fromOption = _a.fromOption, fromPredicate = _a.fromPredicate, filterOrElse = _a.filterOrElse;
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
