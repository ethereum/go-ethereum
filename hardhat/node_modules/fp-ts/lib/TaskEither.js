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
var Either_1 = require("./Either");
var eitherT = require("./EitherT");
var function_1 = require("./function");
var pipeable_1 = require("./pipeable");
var Task_1 = require("./Task");
exports.URI = 'TaskEither';
var T = eitherT.getEitherT2v(Task_1.task);
var foldT = eitherT.fold(Task_1.task);
/**
 * @since 1.0.0
 */
var TaskEither = /** @class */ (function () {
    function TaskEither(value) {
        this.value = value;
    }
    /** Runs the inner `Task` */
    TaskEither.prototype.run = function () {
        return this.value.run();
    };
    /** @obsolete */
    TaskEither.prototype.map = function (f) {
        return new TaskEither(T.map(this.value, f));
    };
    /** @obsolete */
    TaskEither.prototype.ap = function (fab) {
        return new TaskEither(T.ap(fab.value, this.value));
    };
    /**
     * Flipped version of `ap`
     * @obsolete
     */
    TaskEither.prototype.ap_ = function (fb) {
        return fb.ap(this);
    };
    /**
     * Combine two (parallel) effectful actions, keeping only the result of the first
     * @since 1.6.0
     * @obsolete
     */
    TaskEither.prototype.applyFirst = function (fb) {
        return fb.ap(this.map(function_1.constant));
    };
    /**
     * Combine two (parallel) effectful actions, keeping only the result of the second
     * @since 1.5.0
     * @obsolete
     */
    TaskEither.prototype.applySecond = function (fb) {
        // tslint:disable-next-line: deprecation
        return fb.ap(this.map(function_1.constIdentity));
    };
    /**
     * Combine two (sequential) effectful actions, keeping only the result of the first
     * @since 1.12.0
     * @obsolete
     */
    TaskEither.prototype.chainFirst = function (fb) {
        return this.chain(function (a) { return fb.map(function () { return a; }); });
    };
    /**
     * Combine two (sequential) effectful actions, keeping only the result of the second
     * @since 1.12.0
     * @obsolete
     */
    TaskEither.prototype.chainSecond = function (fb) {
        return this.chain(function () { return fb; });
    };
    /** @obsolete */
    TaskEither.prototype.chain = function (f) {
        return new TaskEither(T.chain(this.value, function (a) { return f(a).value; }));
    };
    /** @obsolete */
    TaskEither.prototype.fold = function (onLeft, onRight) {
        return foldT(onLeft, onRight, this.value);
    };
    /**
     * Similar to `fold`, but the result is flattened.
     * @since 1.10.0
     * @obsolete
     */
    TaskEither.prototype.foldTask = function (onLeft, onRight) {
        return this.value.chain(function (e) { return e.fold(onLeft, onRight); });
    };
    /**
     * Similar to `fold`, but the result is flattened.
     * @since 1.10.0
     * @obsolete
     */
    TaskEither.prototype.foldTaskEither = function (onLeft, onRight) {
        return new TaskEither(this.value.chain(function (e) { return e.fold(onLeft, onRight).value; }));
    };
    /**
     * Similar to `fold`, return the value from Right or the given argument if Left.
     * @since 1.17.0
     * @obsolete
     */
    TaskEither.prototype.getOrElse = function (a) {
        return this.getOrElseL(function () { return a; });
    };
    /**
     * @since 1.17.0
     * @obsolete
     */
    TaskEither.prototype.getOrElseL = function (f) {
        return this.fold(f, function_1.identity);
    };
    /** @obsolete */
    TaskEither.prototype.mapLeft = function (f) {
        return new TaskEither(this.value.map(function (e) { return e.mapLeft(f); }));
    };
    /**
     * Transforms the failure value of the `TaskEither` into a new `TaskEither`
     * @obsolete
     */
    TaskEither.prototype.orElse = function (f) {
        return new TaskEither(this.value.chain(function (e) { return e.fold(function (l) { return f(l).value; }, T.of); }));
    };
    /**
     * @since 1.6.0
     * @obsolete
     */
    TaskEither.prototype.alt = function (fy) {
        return this.orElse(function () { return fy; });
    };
    /**
     * @since 1.2.0
     * @obsolete
     */
    TaskEither.prototype.bimap = function (f, g) {
        return new TaskEither(this.value.map(function (e) { return e.bimap(f, g); }));
    };
    /**
     * Return `Right` if the given action succeeds, `Left` if it throws
     * @since 1.10.0
     * @obsolete
     */
    TaskEither.prototype.attempt = function () {
        return new TaskEither(this.value.map(Either_1.right));
    };
    TaskEither.prototype.filterOrElse = function (p, zero) {
        return new TaskEither(this.value.map(function (e) { return e.filterOrElse(p, zero); }));
    };
    TaskEither.prototype.filterOrElseL = function (p, zero) {
        return new TaskEither(this.value.map(function (e) { return e.filterOrElseL(p, zero); }));
    };
    return TaskEither;
}());
exports.TaskEither = TaskEither;
/**
 * Use `rightTask`
 *
 * @since 1.0.0
 * @deprecated
 */
exports.right = function (fa) {
    return new TaskEither(fa.map(Either_1.right));
};
/**
 * Use `leftTask`
 *
 * @since 1.0.0
 * @deprecated
 */
exports.left = function (fl) {
    return new TaskEither(fl.map(Either_1.left));
};
/**
 * @since 1.0.0
 */
exports.fromEither = function (fa) {
    return new TaskEither(Task_1.task.of(fa));
};
/**
 * Use `rightIO`
 *
 * @since 1.5.0
 * @deprecated
 */
exports.fromIO = function (fa) {
    return rightIO(fa);
};
/**
 * Use `left2v`
 *
 * @since 1.3.0
 * @deprecated
 */
exports.fromLeft = function (l) {
    return exports.fromEither(Either_1.left(l));
};
/**
 * @since 1.6.0
 */
exports.fromIOEither = function (fa) {
    return new TaskEither(Task_1.fromIO(fa.value));
};
/**
 * @since 1.9.0
 */
exports.getSemigroup = function (S) {
    var S2 = Task_1.getSemigroup(Either_1.getSemigroup(S));
    return {
        concat: function (x, y) { return new TaskEither(S2.concat(x.value, y.value)); }
    };
};
/**
 * @since 1.9.0
 */
exports.getApplySemigroup = function (S) {
    var S2 = Task_1.getSemigroup(Either_1.getApplySemigroup(S));
    return {
        concat: function (x, y) { return new TaskEither(S2.concat(x.value, y.value)); }
    };
};
/**
 * @since 1.9.0
 */
exports.getApplyMonoid = function (M) {
    return __assign({}, exports.getApplySemigroup(M), { empty: right2v(M.empty) });
};
/**
 * Transforms a `Promise` into a `TaskEither`, catching the possible error.
 *
 * @example
 * import { createHash } from 'crypto'
 * import { TaskEither, tryCatch } from 'fp-ts/lib/TaskEither'
 * import { createReadStream } from 'fs'
 * import { left } from 'fp-ts/lib/Either'
 *
 * const md5 = (path: string): TaskEither<string, string> => {
 *   const mkHash = (p: string) =>
 *     new Promise<string>((resolve, reject) => {
 *       const hash = createHash('md5')
 *       const rs = createReadStream(p)
 *       rs.on('error', (error: Error) => reject(error.message))
 *       rs.on('data', (chunk: string) => hash.update(chunk))
 *       rs.on('end', () => {
 *         return resolve(hash.digest('hex'))
 *       })
 *     })
 *   return tryCatch(() => mkHash(path), message => `cannot create md5 hash: ${String(message)}`)
 * }
 *
 * md5('foo')
 *   .run()
 *   .then(x => {
 *     assert.deepStrictEqual(x, left(`cannot create md5 hash: ENOENT: no such file or directory, open 'foo'`))
 *   })
 *
 *
 * @since 1.0.0
 */
exports.tryCatch = function (f, onrejected) {
    return new TaskEither(Task_1.tryCatch(f, onrejected));
};
function taskify(f) {
    return function () {
        var args = Array.prototype.slice.call(arguments);
        return new TaskEither(new Task_1.Task(function () {
            return new Promise(function (resolve) {
                var cbResolver = function (e, r) {
                    return e != null ? resolve(Either_1.left(e)) : resolve(Either_1.right(r));
                };
                f.apply(null, args.concat(cbResolver));
            });
        }));
    };
}
exports.taskify = taskify;
/**
 * Make sure that a resource is cleaned up in the event of an exception. The
 * release action is called regardless of whether the body action throws or
 * returns.
 *
 * @since 1.10.0
 */
exports.bracket = function (acquire, use, release) {
    return acquire.chain(function (a) {
        return use(a)
            .attempt()
            .chain(function (e) { return release(a, e).chain(function () { return e.fold(left2v, exports.taskEither.of); }); });
    });
};
/**
 * @since 1.0.0
 */
exports.taskEither = {
    URI: exports.URI,
    bimap: function (fla, f, g) { return fla.bimap(f, g); },
    map: function (fa, f) { return fa.map(f); },
    of: right2v,
    ap: function (fab, fa) { return fa.ap(fab); },
    chain: function (fa, f) { return fa.chain(f); },
    alt: function (fx, fy) { return fx.alt(fy); },
    fromIO: exports.fromIO,
    fromTask: rightTask,
    throwError: left2v,
    fromEither: exports.fromEither,
    fromOption: function (o, e) { return (o.isNone() ? left2v(e) : right2v(o.value)); }
};
/**
 * Like `TaskEither` but `ap` is sequential
 *
 * @since 1.10.0
 */
exports.taskEitherSeq = __assign({}, exports.taskEither, { ap: function (fab, fa) { return fab.chain(function (f) { return fa.map(f); }); } });
//
// backporting
//
/**
 * @since 1.19.0
 */
function right2v(a) {
    return new TaskEither(T.of(a));
}
exports.right2v = right2v;
/**
 * @since 1.19.0
 */
function left2v(e) {
    return exports.fromEither(Either_1.left(e));
}
exports.left2v = left2v;
/**
 * @since 1.19.0
 */
function rightIO(ma) {
    return rightTask(Task_1.task.fromIO(ma));
}
exports.rightIO = rightIO;
/**
 * @since 1.19.0
 */
function leftIO(me) {
    return leftTask(Task_1.task.fromIO(me));
}
exports.leftIO = leftIO;
/**
 * @since 1.19.0
 */
function rightTask(ma) {
    // tslint:disable-next-line: deprecation
    return exports.right(ma);
}
exports.rightTask = rightTask;
/**
 * @since 1.19.0
 */
function leftTask(me) {
    // tslint:disable-next-line: deprecation
    return exports.left(me);
}
exports.leftTask = leftTask;
/**
 * @since 1.19.0
 */
function fold(onLeft, onRight) {
    return function (ma) { return ma.foldTask(onLeft, onRight); };
}
exports.fold = fold;
/**
 * @since 1.19.0
 */
function getOrElse(f) {
    return fold(f, Task_1.task.of);
}
exports.getOrElse = getOrElse;
/**
 * @since 1.19.0
 */
function orElse(f) {
    return function (ma) { return ma.orElse(f); };
}
exports.orElse = orElse;
var _a = pipeable_1.pipeable(exports.taskEither), alt = _a.alt, ap = _a.ap, apFirst = _a.apFirst, apSecond = _a.apSecond, bimap = _a.bimap, chain = _a.chain, chainFirst = _a.chainFirst, flatten = _a.flatten, map = _a.map, mapLeft = _a.mapLeft, fromOption = _a.fromOption, fromPredicate = _a.fromPredicate, filterOrElse = _a.filterOrElse;
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
