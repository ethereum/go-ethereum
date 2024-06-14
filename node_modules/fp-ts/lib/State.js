"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var function_1 = require("./function");
var pipeable_1 = require("./pipeable");
exports.URI = 'State';
/**
 * @since 1.0.0
 */
var State = /** @class */ (function () {
    function State(run) {
        this.run = run;
    }
    /** @obsolete */
    State.prototype.eval = function (s) {
        return this.run(s)[0];
    };
    /** @obsolete */
    State.prototype.exec = function (s) {
        return this.run(s)[1];
    };
    /** @obsolete */
    State.prototype.map = function (f) {
        var _this = this;
        return new State(function (s) {
            var _a = _this.run(s), a = _a[0], s1 = _a[1];
            return [f(a), s1];
        });
    };
    /** @obsolete */
    State.prototype.ap = function (fab) {
        var _this = this;
        return fab.chain(function (f) { return _this.map(f); }); // <= derived
    };
    /**
     * Flipped version of `ap`
     * @obsolete
     */
    State.prototype.ap_ = function (fb) {
        return fb.ap(this);
    };
    /**
     * Combine two effectful actions, keeping only the result of the first
     * @since 1.7.0
     * @obsolete
     */
    State.prototype.applyFirst = function (fb) {
        return fb.ap(this.map(function_1.constant));
    };
    /**
     * Combine two effectful actions, keeping only the result of the second
     * @since 1.7.0
     * @obsolete
     */
    State.prototype.applySecond = function (fb) {
        // tslint:disable-next-line: deprecation
        return fb.ap(this.map(function_1.constIdentity));
    };
    /** @obsolete */
    State.prototype.chain = function (f) {
        var _this = this;
        return new State(function (s) {
            var _a = _this.run(s), a = _a[0], s1 = _a[1];
            return f(a).run(s1);
        });
    };
    return State;
}());
exports.State = State;
/**
 * Get the current state
 *
 * @since 1.0.0
 */
exports.get = function () {
    return new State(function (s) { return [s, s]; });
};
/**
 * Set the state
 *
 * @since 1.0.0
 */
exports.put = function (s) {
    return new State(function () { return [undefined, s]; });
};
/**
 * Modify the state by applying a function to the current state
 *
 * @since 1.0.0
 */
exports.modify = function (f) {
    return new State(function (s) { return [undefined, f(s)]; });
};
/**
 * Get a value which depends on the current state
 *
 * @since 1.0.0
 */
exports.gets = function (f) {
    return new State(function (s) { return [f(s), s]; });
};
/**
 * @since 1.0.0
 */
exports.state = {
    URI: exports.URI,
    map: function (fa, f) { return fa.map(f); },
    of: of,
    ap: function (fab, fa) { return fa.ap(fab); },
    chain: function (fa, f) { return fa.chain(f); }
};
//
// backporting
//
/**
 * @since 1.19.0
 */
function of(a) {
    return new State(function (s) { return [a, s]; });
}
exports.of = of;
/**
 * Run a computation in the `State` monad, discarding the final state
 *
 * @since 1.19.0
 */
function evalState(ma, s) {
    return ma.eval(s);
}
exports.evalState = evalState;
/**
 * Run a computation in the `State` monad discarding the result
 *
 * @since 1.19.0
 */
function execState(ma, s) {
    return ma.exec(s);
}
exports.execState = execState;
var _a = pipeable_1.pipeable(exports.state), ap = _a.ap, apFirst = _a.apFirst, apSecond = _a.apSecond, chain = _a.chain, chainFirst = _a.chainFirst, flatten = _a.flatten, map = _a.map;
exports.ap = ap;
exports.apFirst = apFirst;
exports.apSecond = apSecond;
exports.chain = chain;
exports.chainFirst = chainFirst;
exports.flatten = flatten;
exports.map = map;
