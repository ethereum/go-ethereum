/**
 * @file Data structure which represents non-empty arrays
 */
import * as A from './Array';
import { compose } from './function';
import { none, some } from './Option';
import { getJoinSemigroup, getMeetSemigroup } from './Semigroup';
import { pipeable } from './pipeable';
export var URI = 'NonEmptyArray2v';
/**
 * @since 1.17.0
 */
export var getShow = function (S) {
    var SA = A.getShow(S);
    return {
        show: function (arr) { return "make(" + S.show(arr[0]) + ", " + SA.show(arr.slice(1)) + ")"; }
    };
};
/**
 * Use `cons` instead
 *
 * @since 1.15.0
 * @deprecated
 */
export function make(head, tail) {
    return A.cons(head, tail);
}
/**
 * @since 1.15.0
 */
export function head(nea) {
    return nea[0];
}
/**
 * @since 1.15.0
 */
export function tail(nea) {
    return nea.slice(1);
}
/**
 * @since 1.17.3
 */
export var reverse = A.reverse;
/**
 * @since 1.15.0
 */
export function min(ord) {
    var S = getMeetSemigroup(ord);
    return function (nea) { return nea.reduce(S.concat); };
}
/**
 * @since 1.15.0
 */
export function max(ord) {
    var S = getJoinSemigroup(ord);
    return function (nea) { return nea.reduce(S.concat); };
}
/**
 * Builds a `NonEmptyArray` from an `Array` returning `none` if `as` is an empty array
 *
 * @since 1.15.0
 */
export function fromArray(as) {
    return A.isNonEmpty(as) ? some(as) : none;
}
/**
 * Builds a `NonEmptyArray` from a provably (compile time) non empty `Array`.
 *
 * @since 1.15.0
 */
export function fromNonEmptyArray(as) {
    return as;
}
/**
 * Builds a `Semigroup` instance for `NonEmptyArray`
 *
 * @since 1.15.0
 */
export var getSemigroup = function () {
    return {
        concat: function (x, y) { return x.concat(y); }
    };
};
/**
 * Use `getEq`
 *
 * @since 1.15.0
 * @deprecated
 */
export var getSetoid = getEq;
/**
 * @example
 * import { fromNonEmptyArray, getEq, make } from 'fp-ts/lib/NonEmptyArray2v'
 * import { eqNumber } from 'fp-ts/lib/Eq'
 *
 * const S = getEq(eqNumber)
 * assert.strictEqual(S.equals(make(1, [2]), fromNonEmptyArray([1, 2])), true)
 * assert.strictEqual(S.equals(make(1, [2]), fromNonEmptyArray([1, 3])), false)
 *
 * @since 1.19.0
 */
export function getEq(E) {
    return A.getEq(E);
}
/**
 * Group equal, consecutive elements of an array into non empty arrays.
 *
 * @example
 * import { make, group } from 'fp-ts/lib/NonEmptyArray2v'
 * import { ordNumber } from 'fp-ts/lib/Ord'
 *
 * assert.deepStrictEqual(group(ordNumber)([1, 2, 1, 1]), [
 *   make(1, []),
 *   make(2, []),
 *   make(1, [1])
 * ])
 *
 * @since 1.15.0
 */
export var group = function (E) { return function (as) {
    var len = as.length;
    if (len === 0) {
        return A.empty;
    }
    var r = [];
    var head = as[0];
    var nea = fromNonEmptyArray([head]);
    for (var i = 1; i < len; i++) {
        var x = as[i];
        if (E.equals(x, head)) {
            nea.push(x);
        }
        else {
            r.push(nea);
            head = x;
            nea = fromNonEmptyArray([head]);
        }
    }
    r.push(nea);
    return r;
}; };
/**
 * Sort and then group the elements of an array into non empty arrays.
 *
 * @example
 * import { make, groupSort } from 'fp-ts/lib/NonEmptyArray2v'
 * import { ordNumber } from 'fp-ts/lib/Ord'
 *
 * assert.deepStrictEqual(groupSort(ordNumber)([1, 2, 1, 1]), [make(1, [1, 1]), make(2, [])])
 *
 * @since 1.15.0
 */
export var groupSort = function (O) {
    // tslint:disable-next-line: deprecation
    return compose(group(O), A.sort(O));
};
function _groupBy(as, f) {
    var r = {};
    for (var _i = 0, as_1 = as; _i < as_1.length; _i++) {
        var a = as_1[_i];
        var k = f(a);
        if (r.hasOwnProperty(k)) {
            r[k].push(a);
        }
        else {
            r[k] = cons(a, []);
        }
    }
    return r;
}
export function groupBy() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1 ? function (as) { return _groupBy(as, args[0]); } : _groupBy(args[0], args[1]);
}
/**
 * @since 1.15.0
 */
export function last(nea) {
    return nea[nea.length - 1];
}
/**
 * @since 1.15.0
 */
export function sort(O) {
    return A.sort(O);
}
export function findFirst(nea, predicate) {
    // tslint:disable-next-line: deprecation
    return A.findFirst(nea, predicate);
}
export function findLast(nea, predicate) {
    // tslint:disable-next-line: deprecation
    return A.findLast(nea, predicate);
}
/**
 * Use `Array`'s `findIndex`
 *
 * @since 1.15.0
 * @deprecated
 */
export function findIndex(nea, predicate) {
    // tslint:disable-next-line: deprecation
    return A.findIndex(nea, predicate);
}
/**
 * Use `Array`'s `findLastIndex`
 *
 * @since 1.15.0
 * @deprecated
 */
export function findLastIndex(nea, predicate) {
    // tslint:disable-next-line: deprecation
    return A.findLastIndex(nea, predicate);
}
function _insertAt(i, a, nea) {
    // tslint:disable-next-line: deprecation
    return A.insertAt(i, a, nea);
}
export function insertAt() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 2
        ? function (nea) { return _insertAt(args[0], args[1], nea); }
        : _insertAt(args[0], args[1], args[2]);
}
function _updateAt(i, a, nea) {
    // tslint:disable-next-line: deprecation
    return A.updateAt(i, a, nea);
}
export function updateAt() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 2
        ? function (nea) { return _updateAt(args[0], args[1], nea); }
        : _updateAt(args[0], args[1], args[2]);
}
function _modifyAt(nea, i, f) {
    // tslint:disable-next-line: deprecation
    return A.modifyAt(nea, i, f);
}
export function modifyAt() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 2
        ? function (nea) { return _modifyAt(nea, args[0], args[1]); }
        : _modifyAt(args[0], args[1], args[2]);
}
/**
 * @since 1.17.0
 */
export var copy = function (nea) {
    return A.copy(nea);
};
function _filter(nea, predicate) {
    // tslint:disable-next-line: deprecation
    return filterWithIndex(nea, function (_, a) { return predicate(a); });
}
export function filter() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1 ? function (nea) { return _filter(nea, args[0]); } : _filter(args[0], args[1]);
}
function _filterWithIndex(nea, predicate) {
    return fromArray(nea.filter(function (a, i) { return predicate(i, a); }));
}
export function filterWithIndex() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1
        ? function (nea) { return _filterWithIndex(nea, args[0]); }
        : _filterWithIndex(args[0], args[1]);
}
/**
 * Append an element to the end of an array, creating a new non empty array
 *
 * @example
 * import { snoc } from 'fp-ts/lib/NonEmptyArray2v'
 *
 * assert.deepStrictEqual(snoc([1, 2, 3], 4), [1, 2, 3, 4])
 *
 * @since 1.16.0
 */
export var snoc = A.snoc;
/**
 * Append an element to the front of an array, creating a new non empty array
 *
 * @example
 * import { cons } from 'fp-ts/lib/NonEmptyArray2v'
 *
 * assert.deepStrictEqual(cons(1, [2, 3, 4]), [1, 2, 3, 4])
 *
 * @since 1.16.0
 */
export var cons = A.cons;
/**
 * @since 1.15.0
 */
export var nonEmptyArray = {
    URI: URI,
    map: A.array.map,
    mapWithIndex: A.array.mapWithIndex,
    of: A.array.of,
    ap: A.array.ap,
    chain: A.array.chain,
    extend: A.array.extend,
    extract: head,
    reduce: A.array.reduce,
    foldMap: A.array.foldMap,
    foldr: A.array.foldr,
    traverse: A.array.traverse,
    sequence: A.array.sequence,
    reduceWithIndex: A.array.reduceWithIndex,
    foldMapWithIndex: A.array.foldMapWithIndex,
    foldrWithIndex: A.array.foldrWithIndex,
    traverseWithIndex: A.array.traverseWithIndex
};
//
// backporting
//
/**
 * @since 1.19.0
 */
export var of = A.array.of;
var _a = pipeable(nonEmptyArray), ap = _a.ap, apFirst = _a.apFirst, apSecond = _a.apSecond, chain = _a.chain, chainFirst = _a.chainFirst, duplicate = _a.duplicate, extend = _a.extend, flatten = _a.flatten, foldMap = _a.foldMap, foldMapWithIndex = _a.foldMapWithIndex, map = _a.map, mapWithIndex = _a.mapWithIndex, reduce = _a.reduce, reduceRight = _a.reduceRight, reduceRightWithIndex = _a.reduceRightWithIndex, reduceWithIndex = _a.reduceWithIndex;
export { ap, apFirst, apSecond, chain, chainFirst, duplicate, extend, flatten, foldMap, foldMapWithIndex, map, mapWithIndex, reduce, reduceRight, reduceRightWithIndex, reduceWithIndex };
