/**
 * @since 1.0.0
 */
export var sign = function (n) {
    return n <= -1 ? -1 : n >= 1 ? 1 : 0;
};
/**
 * @since 1.19.0
 */
export var eqOrdering = {
    equals: function (x, y) { return x === y; }
};
/**
 * Use `eqOrdering`
 *
 * @since 1.0.0
 * @deprecated
 */
export var setoidOrdering = eqOrdering;
/**
 * @since 1.0.0
 */
export var semigroupOrdering = {
    concat: function (x, y) { return (x !== 0 ? x : y); }
};
/**
 * @since 1.0.0
 */
export var invert = function (O) {
    switch (O) {
        case -1:
            return 1;
        case 1:
            return -1;
        default:
            return 0;
    }
};
