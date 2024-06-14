export var URI = 'Traced';
/**
 * @since 1.16.0
 */
var Traced = /** @class */ (function () {
    function Traced(run) {
        this.run = run;
    }
    /** @obsolete */
    Traced.prototype.map = function (f) {
        var _this = this;
        return new Traced(function (p) { return f(_this.run(p)); });
    };
    return Traced;
}());
export { Traced };
/**
 * Extracts a value at a relative position which depends on the current value.
 * @since 1.16.0
 */
export var tracks = function (M, f) { return function (wa) {
    return wa.run(f(wa.run(M.empty)));
}; };
/**
 * Get the current position
 * @since 1.16.0
 */
export var listen = function (wa) {
    return new Traced(function (e) { return [wa.run(e), e]; });
};
/**
 * Get a value which depends on the current position
 * @since 1.16.0
 */
export var listens = function (wa, f) {
    return new Traced(function (e) { return [wa.run(e), f(e)]; });
};
/**
 * Apply a function to the current position
 * @since 1.16.0
 */
export var censor = function (wa, f) {
    return new Traced(function (e) { return wa.run(f(e)); });
};
/**
 * @since 1.16.0
 */
export function getComonad(monoid) {
    function extend(wa, f) {
        return new Traced(function (p1) { return f(new Traced(function (p2) { return wa.run(monoid.concat(p1, p2)); })); });
    }
    function extract(wa) {
        return wa.run(monoid.empty);
    }
    return {
        URI: URI,
        _L: undefined,
        map: map,
        extend: extend,
        extract: extract
    };
}
function map(wa, f) {
    return wa.map(f);
}
/**
 * @since 1.16.0
 */
export var traced = {
    URI: URI,
    map: map
};
