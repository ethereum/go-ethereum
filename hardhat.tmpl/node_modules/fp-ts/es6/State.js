import { constant, constIdentity } from './function';
import { pipeable } from './pipeable';
export var URI = 'State';
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
        return fb.ap(this.map(constant));
    };
    /**
     * Combine two effectful actions, keeping only the result of the second
     * @since 1.7.0
     * @obsolete
     */
    State.prototype.applySecond = function (fb) {
        // tslint:disable-next-line: deprecation
        return fb.ap(this.map(constIdentity));
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
export { State };
/**
 * Get the current state
 *
 * @since 1.0.0
 */
export var get = function () {
    return new State(function (s) { return [s, s]; });
};
/**
 * Set the state
 *
 * @since 1.0.0
 */
export var put = function (s) {
    return new State(function () { return [undefined, s]; });
};
/**
 * Modify the state by applying a function to the current state
 *
 * @since 1.0.0
 */
export var modify = function (f) {
    return new State(function (s) { return [undefined, f(s)]; });
};
/**
 * Get a value which depends on the current state
 *
 * @since 1.0.0
 */
export var gets = function (f) {
    return new State(function (s) { return [f(s), s]; });
};
/**
 * @since 1.0.0
 */
export var state = {
    URI: URI,
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
export function of(a) {
    return new State(function (s) { return [a, s]; });
}
/**
 * Run a computation in the `State` monad, discarding the final state
 *
 * @since 1.19.0
 */
export function evalState(ma, s) {
    return ma.eval(s);
}
/**
 * Run a computation in the `State` monad discarding the result
 *
 * @since 1.19.0
 */
export function execState(ma, s) {
    return ma.exec(s);
}
var _a = pipeable(state), ap = _a.ap, apFirst = _a.apFirst, apSecond = _a.apSecond, chain = _a.chain, chainFirst = _a.chainFirst, flatten = _a.flatten, map = _a.map;
export { ap, apFirst, apSecond, chain, chainFirst, flatten, map };
