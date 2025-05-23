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
/**
 * @file `Task<A>` represents an asynchronous computation that yields a value of type `A` and **never fails**.
 * If you want to represent an asynchronous computation that may fail, please see `TaskEither`.
 */
var Either_1 = require("./Either");
var function_1 = require("./function");
var pipeable_1 = require("./pipeable");
exports.URI = 'Task';
/**
 * @since 1.0.0
 */
var Task = /** @class */ (function () {
    function Task(run) {
        this.run = run;
    }
    /** @obsolete */
    Task.prototype.map = function (f) {
        var _this = this;
        return new Task(function () { return _this.run().then(f); });
    };
    /** @obsolete */
    Task.prototype.ap = function (fab) {
        var _this = this;
        return new Task(function () { return Promise.all([fab.run(), _this.run()]).then(function (_a) {
            var f = _a[0], a = _a[1];
            return f(a);
        }); });
    };
    /**
     * Flipped version of `ap`
     * @obsolete
     */
    Task.prototype.ap_ = function (fb) {
        return fb.ap(this);
    };
    /**
     * Combine two effectful actions, keeping only the result of the first
     * @since 1.6.0
     * @obsolete
     */
    Task.prototype.applyFirst = function (fb) {
        return fb.ap(this.map(function_1.constant));
    };
    /**
     * Combine two effectful actions, keeping only the result of the second
     * @since 1.5.0
     * @obsolete
     */
    Task.prototype.applySecond = function (fb) {
        // tslint:disable-next-line: deprecation
        return fb.ap(this.map(function_1.constIdentity));
    };
    /** @obsolete */
    Task.prototype.chain = function (f) {
        var _this = this;
        return new Task(function () { return _this.run().then(function (a) { return f(a).run(); }); });
    };
    Task.prototype.inspect = function () {
        return this.toString();
    };
    Task.prototype.toString = function () {
        // tslint:disable-next-line: deprecation
        return "new Task(" + function_1.toString(this.run) + ")";
    };
    return Task;
}());
exports.Task = Task;
/**
 * @since 1.0.0
 */
exports.getRaceMonoid = function () {
    return {
        concat: function (x, y) {
            return new Task(function () {
                return new Promise(function (resolve, reject) {
                    var running = true;
                    var resolveFirst = function (a) {
                        if (running) {
                            running = false;
                            resolve(a);
                        }
                    };
                    var rejectFirst = function (e) {
                        if (running) {
                            running = false;
                            reject(e);
                        }
                    };
                    x.run().then(resolveFirst, rejectFirst);
                    y.run().then(resolveFirst, rejectFirst);
                });
            });
        },
        empty: exports.never
    };
};
/**
 * @since 1.0.0
 */
exports.getSemigroup = function (S) {
    return {
        concat: function (x, y) { return new Task(function () { return x.run().then(function (rx) { return y.run().then(function (ry) { return S.concat(rx, ry); }); }); }); }
    };
};
/**
 * @since 1.0.0
 */
exports.getMonoid = function (M) {
    return __assign({}, exports.getSemigroup(M), { empty: of(M.empty) });
};
/**
 * @since 1.0.0
 */
exports.tryCatch = function (f, onrejected) {
    return new Task(function () { return f().then(Either_1.right, function (reason) { return Either_1.left(onrejected(reason)); }); });
};
/**
 * Lifts an IO action into a Task
 *
 * @since 1.0.0
 */
exports.fromIO = function (io) {
    return new Task(function () { return Promise.resolve(io.run()); });
};
/**
 * Use `delay2v`
 *
 * @since 1.7.0
 * @deprecated
 */
exports.delay = function (millis, a) {
    return delay2v(millis)(of(a));
};
/**
 * @since 1.0.0
 */
exports.task = {
    URI: exports.URI,
    map: function (fa, f) { return fa.map(f); },
    of: of,
    ap: function (fab, fa) { return fa.ap(fab); },
    chain: function (fa, f) { return fa.chain(f); },
    fromIO: exports.fromIO,
    fromTask: function_1.identity
};
/**
 * Like `Task` but `ap` is sequential
 *
 * @since 1.10.0
 */
exports.taskSeq = __assign({}, exports.task, { ap: function (fab, fa) { return fab.chain(function (f) { return fa.map(f); }); } });
//
// backporting
//
/**
 * @since 1.19.0
 */
function of(a) {
    return new Task(function () { return Promise.resolve(a); });
}
exports.of = of;
/**
 * @since 1.19.0
 */
exports.never = new Task(function () { return new Promise(function (_) { return undefined; }); });
/**
 * @since 1.19.0
 */
function delay2v(millis) {
    return function (ma) {
        return new Task(function () {
            return new Promise(function (resolve) {
                setTimeout(function () {
                    // tslint:disable-next-line: no-floating-promises
                    ma.run().then(resolve);
                }, millis);
            });
        });
    };
}
exports.delay2v = delay2v;
var _a = pipeable_1.pipeable(exports.task), ap = _a.ap, apFirst = _a.apFirst, apSecond = _a.apSecond, chain = _a.chain, chainFirst = _a.chainFirst, flatten = _a.flatten, map = _a.map;
exports.ap = ap;
exports.apFirst = apFirst;
exports.apSecond = apSecond;
exports.chain = chain;
exports.chainFirst = chainFirst;
exports.flatten = flatten;
exports.map = map;
