import { toString } from './function';
import { pipeable } from './pipeable';
export var URI = 'Store';
/**
 * @since 1.0.0
 */
var Store = /** @class */ (function () {
    function Store(peek, pos) {
        this.peek = peek;
        this.pos = pos;
    }
    /**
     * Reposition the focus at the specified position
     * @obsolete
     */
    Store.prototype.seek = function (s) {
        return new Store(this.peek, s);
    };
    /** @obsolete */
    Store.prototype.map = function (f) {
        var _this = this;
        return new Store(function (s) { return f(_this.peek(s)); }, this.pos);
    };
    /** @obsolete */
    Store.prototype.extract = function () {
        return this.peek(this.pos);
    };
    /** @obsolete */
    Store.prototype.extend = function (f) {
        var _this = this;
        return new Store(function (s) { return f(_this.seek(s)); }, this.pos);
    };
    /* istanbul ignore next */
    Store.prototype.inspect = function () {
        return this.toString();
    };
    /* istanbul ignore next */
    Store.prototype.toString = function () {
        // tslint:disable-next-line: deprecation
        return "new Store(" + toString(this.peek) + ", " + toString(this.pos) + ")";
    };
    return Store;
}());
export { Store };
/**
 * Extract a value from a position which depends on the current position
 *
 * @since 1.0.0
 */
export function peeks(f) {
    return function (wa) { return wa.peek(f(wa.pos)); };
}
/**
 * Reposition the focus at the specified position, which depends on the current position
 *
 * @since 1.0.0
 */
export var seeks = function (f) { return function (sa) {
    return new Store(sa.peek, f(sa.pos));
}; };
export function experiment(F) {
    return function (f) { return function (wa) { return F.map(f(wa.pos), function (s) { return wa.peek(s); }); }; };
}
/**
 * @since 1.0.0
 */
export var store = {
    URI: URI,
    map: function (fa, f) { return fa.map(f); },
    extract: function (wa) { return wa.extract(); },
    extend: function (wa, f) { return wa.extend(f); }
};
//
// backporting
//
/**
 * Reposition the focus at the specified position
 *
 * @since 1.19.0
 */
export function seek(s) {
    return function (wa) { return new Store(wa.peek, s); };
}
var _a = pipeable(store), duplicate = _a.duplicate, extend = _a.extend, map = _a.map;
export { duplicate, extend, map };
