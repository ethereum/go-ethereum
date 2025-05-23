"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var function_1 = require("./function");
var pipeable_1 = require("./pipeable");
exports.URI = 'Store';
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
        return "new Store(" + function_1.toString(this.peek) + ", " + function_1.toString(this.pos) + ")";
    };
    return Store;
}());
exports.Store = Store;
/**
 * Extract a value from a position which depends on the current position
 *
 * @since 1.0.0
 */
function peeks(f) {
    return function (wa) { return wa.peek(f(wa.pos)); };
}
exports.peeks = peeks;
/**
 * Reposition the focus at the specified position, which depends on the current position
 *
 * @since 1.0.0
 */
exports.seeks = function (f) { return function (sa) {
    return new Store(sa.peek, f(sa.pos));
}; };
function experiment(F) {
    return function (f) { return function (wa) { return F.map(f(wa.pos), function (s) { return wa.peek(s); }); }; };
}
exports.experiment = experiment;
/**
 * @since 1.0.0
 */
exports.store = {
    URI: exports.URI,
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
function seek(s) {
    return function (wa) { return new Store(wa.peek, s); };
}
exports.seek = seek;
var _a = pipeable_1.pipeable(exports.store), duplicate = _a.duplicate, extend = _a.extend, map = _a.map;
exports.duplicate = duplicate;
exports.extend = extend;
exports.map = map;
