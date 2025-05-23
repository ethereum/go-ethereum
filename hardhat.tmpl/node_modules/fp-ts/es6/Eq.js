import { pipeable } from './pipeable';
/**
 * @since 1.19.0
 */
export var URI = 'Eq';
/**
 * @since 1.19.0
 */
export function fromEquals(equals) {
    return {
        equals: function (x, y) { return x === y || equals(x, y); }
    };
}
/**
 * @since 1.19.0
 */
export function strictEqual(a, b) {
    return a === b;
}
var eqStrict = { equals: strictEqual };
/**
 * @since 1.19.0
 */
export var eqString = eqStrict;
/**
 * @since 1.19.0
 */
export var eqNumber = eqStrict;
/**
 * @since 1.19.0
 */
export var eqBoolean = eqStrict;
/**
 * @since 1.19.0
 */
export function getStructEq(eqs) {
    return fromEquals(function (x, y) {
        for (var k in eqs) {
            if (!eqs[k].equals(x[k], y[k])) {
                return false;
            }
        }
        return true;
    });
}
/**
 * Given a tuple of `Eq`s returns a `Eq` for the tuple
 *
 * @example
 * import { getTupleEq, eqString, eqNumber, eqBoolean } from 'fp-ts/lib/Eq'
 *
 * const E = getTupleEq(eqString, eqNumber, eqBoolean)
 * assert.strictEqual(E.equals(['a', 1, true], ['a', 1, true]), true)
 * assert.strictEqual(E.equals(['a', 1, true], ['b', 1, true]), false)
 * assert.strictEqual(E.equals(['a', 1, true], ['a', 2, true]), false)
 * assert.strictEqual(E.equals(['a', 1, true], ['a', 1, false]), false)
 *
 * @since 1.19.0
 */
export function getTupleEq() {
    var eqs = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        eqs[_i] = arguments[_i];
    }
    return fromEquals(function (x, y) { return eqs.every(function (E, i) { return E.equals(x[i], y[i]); }); });
}
/**
 * @since 1.19.0
 */
export var eq = {
    URI: URI,
    contramap: function (fa, f) { return fromEquals(function (x, y) { return fa.equals(f(x), f(y)); }); }
};
var contramap = pipeable(eq).contramap;
export { contramap };
/**
 * @since 1.19.0
 */
export var eqDate = eq.contramap(eqNumber, function (date) { return date.valueOf(); });
