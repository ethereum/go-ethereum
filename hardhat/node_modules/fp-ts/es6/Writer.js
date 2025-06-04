import { pipeable } from './pipeable';
export var URI = 'Writer';
/**
 * @since 1.0.0
 */
var Writer = /** @class */ (function () {
    function Writer(run) {
        this.run = run;
    }
    /** @obsolete */
    Writer.prototype.eval = function () {
        return this.run()[0];
    };
    /** @obsolete */
    Writer.prototype.exec = function () {
        return this.run()[1];
    };
    /** @obsolete */
    Writer.prototype.map = function (f) {
        var _this = this;
        return new Writer(function () {
            var _a = _this.run(), a = _a[0], w = _a[1];
            return [f(a), w];
        });
    };
    return Writer;
}());
export { Writer };
/**
 * Appends a value to the accumulator
 *
 * @since 1.0.0
 */
export var tell = function (w) {
    return new Writer(function () { return [undefined, w]; });
};
/**
 * Modifies the result to include the changes to the accumulator
 *
 * @since 1.3.0
 */
export var listen = function (fa) {
    return new Writer(function () {
        var _a = fa.run(), a = _a[0], w = _a[1];
        return [[a, w], w];
    });
};
/**
 * Applies the returned function to the accumulator
 *
 * @since 1.3.0
 */
export var pass = function (fa) {
    return new Writer(function () {
        var _a = fa.run(), _b = _a[0], a = _b[0], f = _b[1], w = _a[1];
        return [a, f(w)];
    });
};
/**
 * Use `listens2v`
 *
 * @since 1.3.0
 * @deprecated
 */
export var listens = function (fa, f) {
    return new Writer(function () {
        var _a = fa.run(), a = _a[0], w = _a[1];
        return [[a, f(w)], w];
    });
};
/**
 * Use `censor2v`
 *
 * @since 1.3.0
 * @deprecated
 */
export var censor = function (fa, f) {
    return new Writer(function () {
        var _a = fa.run(), a = _a[0], w = _a[1];
        return [a, f(w)];
    });
};
/**
 *
 * @since 1.0.0
 */
export var getMonad = function (M) {
    return {
        URI: URI,
        _L: undefined,
        map: writer.map,
        of: function (a) { return new Writer(function () { return [a, M.empty]; }); },
        ap: function (fab, fa) {
            return new Writer(function () {
                var _a = fab.run(), f = _a[0], w1 = _a[1];
                var _b = fa.run(), a = _b[0], w2 = _b[1];
                return [f(a), M.concat(w1, w2)];
            });
        },
        chain: function (fa, f) {
            return new Writer(function () {
                var _a = fa.run(), a = _a[0], w1 = _a[1];
                var _b = f(a).run(), b = _b[0], w2 = _b[1];
                return [b, M.concat(w1, w2)];
            });
        }
    };
};
/**
 * @since 1.0.0
 */
export var writer = {
    URI: URI,
    map: function (fa, f) { return fa.map(f); }
};
//
// backporting
//
/**
 * @since 1.19.0
 */
export function evalWriter(fa) {
    return fa.eval();
}
/**
 * @since 1.19.0
 */
export function execWriter(fa) {
    return fa.exec();
}
/**
 * Projects a value from modifications made to the accumulator during an action
 *
 * @since 1.19.0
 */
export function listens2v(f) {
    // tslint:disable-next-line: deprecation
    return function (fa) { return listens(fa, f); };
}
/**
 * Modify the final accumulator value by applying a function
 *
 * @since 1.19.0
 */
export function censor2v(f) {
    // tslint:disable-next-line: deprecation
    return function (fa) { return censor(fa, f); };
}
var map = pipeable(writer).map;
export { map };
