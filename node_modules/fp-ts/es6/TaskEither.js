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
import { getApplySemigroup as eitherGetApplySemigroup, getSemigroup as eitherGetSemigroup, left as eitherLeft, right as eitherRight } from './Either';
import * as eitherT from './EitherT';
import { constant, constIdentity, identity } from './function';
import { pipeable } from './pipeable';
import { fromIO as taskFromIO, getSemigroup as taskGetSemigroup, Task, task, tryCatch as taskTryCatch } from './Task';
export var URI = 'TaskEither';
var T = eitherT.getEitherT2v(task);
var foldT = eitherT.fold(task);
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
        return fb.ap(this.map(constant));
    };
    /**
     * Combine two (parallel) effectful actions, keeping only the result of the second
     * @since 1.5.0
     * @obsolete
     */
    TaskEither.prototype.applySecond = function (fb) {
        // tslint:disable-next-line: deprecation
        return fb.ap(this.map(constIdentity));
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
        return this.fold(f, identity);
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
        return new TaskEither(this.value.map(eitherRight));
    };
    TaskEither.prototype.filterOrElse = function (p, zero) {
        return new TaskEither(this.value.map(function (e) { return e.filterOrElse(p, zero); }));
    };
    TaskEither.prototype.filterOrElseL = function (p, zero) {
        return new TaskEither(this.value.map(function (e) { return e.filterOrElseL(p, zero); }));
    };
    return TaskEither;
}());
export { TaskEither };
/**
 * Use `rightTask`
 *
 * @since 1.0.0
 * @deprecated
 */
export var right = function (fa) {
    return new TaskEither(fa.map(eitherRight));
};
/**
 * Use `leftTask`
 *
 * @since 1.0.0
 * @deprecated
 */
export var left = function (fl) {
    return new TaskEither(fl.map(eitherLeft));
};
/**
 * @since 1.0.0
 */
export var fromEither = function (fa) {
    return new TaskEither(task.of(fa));
};
/**
 * Use `rightIO`
 *
 * @since 1.5.0
 * @deprecated
 */
export var fromIO = function (fa) {
    return rightIO(fa);
};
/**
 * Use `left2v`
 *
 * @since 1.3.0
 * @deprecated
 */
export var fromLeft = function (l) {
    return fromEither(eitherLeft(l));
};
/**
 * @since 1.6.0
 */
export var fromIOEither = function (fa) {
    return new TaskEither(taskFromIO(fa.value));
};
/**
 * @since 1.9.0
 */
export var getSemigroup = function (S) {
    var S2 = taskGetSemigroup(eitherGetSemigroup(S));
    return {
        concat: function (x, y) { return new TaskEither(S2.concat(x.value, y.value)); }
    };
};
/**
 * @since 1.9.0
 */
export var getApplySemigroup = function (S) {
    var S2 = taskGetSemigroup(eitherGetApplySemigroup(S));
    return {
        concat: function (x, y) { return new TaskEither(S2.concat(x.value, y.value)); }
    };
};
/**
 * @since 1.9.0
 */
export var getApplyMonoid = function (M) {
    return __assign({}, getApplySemigroup(M), { empty: right2v(M.empty) });
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
export var tryCatch = function (f, onrejected) {
    return new TaskEither(taskTryCatch(f, onrejected));
};
export function taskify(f) {
    return function () {
        var args = Array.prototype.slice.call(arguments);
        return new TaskEither(new Task(function () {
            return new Promise(function (resolve) {
                var cbResolver = function (e, r) {
                    return e != null ? resolve(eitherLeft(e)) : resolve(eitherRight(r));
                };
                f.apply(null, args.concat(cbResolver));
            });
        }));
    };
}
/**
 * Make sure that a resource is cleaned up in the event of an exception. The
 * release action is called regardless of whether the body action throws or
 * returns.
 *
 * @since 1.10.0
 */
export var bracket = function (acquire, use, release) {
    return acquire.chain(function (a) {
        return use(a)
            .attempt()
            .chain(function (e) { return release(a, e).chain(function () { return e.fold(left2v, taskEither.of); }); });
    });
};
/**
 * @since 1.0.0
 */
export var taskEither = {
    URI: URI,
    bimap: function (fla, f, g) { return fla.bimap(f, g); },
    map: function (fa, f) { return fa.map(f); },
    of: right2v,
    ap: function (fab, fa) { return fa.ap(fab); },
    chain: function (fa, f) { return fa.chain(f); },
    alt: function (fx, fy) { return fx.alt(fy); },
    fromIO: fromIO,
    fromTask: rightTask,
    throwError: left2v,
    fromEither: fromEither,
    fromOption: function (o, e) { return (o.isNone() ? left2v(e) : right2v(o.value)); }
};
/**
 * Like `TaskEither` but `ap` is sequential
 *
 * @since 1.10.0
 */
export var taskEitherSeq = __assign({}, taskEither, { ap: function (fab, fa) { return fab.chain(function (f) { return fa.map(f); }); } });
//
// backporting
//
/**
 * @since 1.19.0
 */
export function right2v(a) {
    return new TaskEither(T.of(a));
}
/**
 * @since 1.19.0
 */
export function left2v(e) {
    return fromEither(eitherLeft(e));
}
/**
 * @since 1.19.0
 */
export function rightIO(ma) {
    return rightTask(task.fromIO(ma));
}
/**
 * @since 1.19.0
 */
export function leftIO(me) {
    return leftTask(task.fromIO(me));
}
/**
 * @since 1.19.0
 */
export function rightTask(ma) {
    // tslint:disable-next-line: deprecation
    return right(ma);
}
/**
 * @since 1.19.0
 */
export function leftTask(me) {
    // tslint:disable-next-line: deprecation
    return left(me);
}
/**
 * @since 1.19.0
 */
export function fold(onLeft, onRight) {
    return function (ma) { return ma.foldTask(onLeft, onRight); };
}
/**
 * @since 1.19.0
 */
export function getOrElse(f) {
    return fold(f, task.of);
}
/**
 * @since 1.19.0
 */
export function orElse(f) {
    return function (ma) { return ma.orElse(f); };
}
var _a = pipeable(taskEither), alt = _a.alt, ap = _a.ap, apFirst = _a.apFirst, apSecond = _a.apSecond, bimap = _a.bimap, chain = _a.chain, chainFirst = _a.chainFirst, flatten = _a.flatten, map = _a.map, mapLeft = _a.mapLeft, fromOption = _a.fromOption, fromPredicate = _a.fromPredicate, filterOrElse = _a.filterOrElse;
export { alt, ap, apFirst, apSecond, bimap, chain, chainFirst, flatten, map, mapLeft, fromOption, fromPredicate, filterOrElse };
