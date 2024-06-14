"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
/**
 * @file Data structure which represents non-empty arrays
 */
var A = require("./Array");
var function_1 = require("./function");
var Option_1 = require("./Option");
var Semigroup_1 = require("./Semigroup");
var pipeable_1 = require("./pipeable");
exports.URI = 'NonEmptyArray2v';
/**
 * @since 1.17.0
 */
exports.getShow = function (S) {
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
function make(head, tail) {
    return A.cons(head, tail);
}
exports.make = make;
/**
 * @since 1.15.0
 */
function head(nea) {
    return nea[0];
}
exports.head = head;
/**
 * @since 1.15.0
 */
function tail(nea) {
    return nea.slice(1);
}
exports.tail = tail;
/**
 * @since 1.17.3
 */
exports.reverse = A.reverse;
/**
 * @since 1.15.0
 */
function min(ord) {
    var S = Semigroup_1.getMeetSemigroup(ord);
    return function (nea) { return nea.reduce(S.concat); };
}
exports.min = min;
/**
 * @since 1.15.0
 */
function max(ord) {
    var S = Semigroup_1.getJoinSemigroup(ord);
    return function (nea) { return nea.reduce(S.concat); };
}
exports.max = max;
/**
 * Builds a `NonEmptyArray` from an `Array` returning `none` if `as` is an empty array
 *
 * @since 1.15.0
 */
function fromArray(as) {
    return A.isNonEmpty(as) ? Option_1.some(as) : Option_1.none;
}
exports.fromArray = fromArray;
/**
 * Builds a `NonEmptyArray` from a provably (compile time) non empty `Array`.
 *
 * @since 1.15.0
 */
function fromNonEmptyArray(as) {
    return as;
}
exports.fromNonEmptyArray = fromNonEmptyArray;
/**
 * Builds a `Semigroup` instance for `NonEmptyArray`
 *
 * @since 1.15.0
 */
exports.getSemigroup = function () {
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
exports.getSetoid = getEq;
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
function getEq(E) {
    return A.getEq(E);
}
exports.getEq = getEq;
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
exports.group = function (E) { return function (as) {
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
exports.groupSort = function (O) {
    // tslint:disable-next-line: deprecation
    return function_1.compose(exports.group(O), A.sort(O));
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
            r[k] = exports.cons(a, []);
        }
    }
    return r;
}
function groupBy() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1 ? function (as) { return _groupBy(as, args[0]); } : _groupBy(args[0], args[1]);
}
exports.groupBy = groupBy;
/**
 * @since 1.15.0
 */
function last(nea) {
    return nea[nea.length - 1];
}
exports.last = last;
/**
 * @since 1.15.0
 */
function sort(O) {
    return A.sort(O);
}
exports.sort = sort;
function findFirst(nea, predicate) {
    // tslint:disable-next-line: deprecation
    return A.findFirst(nea, predicate);
}
exports.findFirst = findFirst;
function findLast(nea, predicate) {
    // tslint:disable-next-line: deprecation
    return A.findLast(nea, predicate);
}
exports.findLast = findLast;
/**
 * Use `Array`'s `findIndex`
 *
 * @since 1.15.0
 * @deprecated
 */
function findIndex(nea, predicate) {
    // tslint:disable-next-line: deprecation
    return A.findIndex(nea, predicate);
}
exports.findIndex = findIndex;
/**
 * Use `Array`'s `findLastIndex`
 *
 * @since 1.15.0
 * @deprecated
 */
function findLastIndex(nea, predicate) {
    // tslint:disable-next-line: deprecation
    return A.findLastIndex(nea, predicate);
}
exports.findLastIndex = findLastIndex;
function _insertAt(i, a, nea) {
    // tslint:disable-next-line: deprecation
    return A.insertAt(i, a, nea);
}
function insertAt() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 2
        ? function (nea) { return _insertAt(args[0], args[1], nea); }
        : _insertAt(args[0], args[1], args[2]);
}
exports.insertAt = insertAt;
function _updateAt(i, a, nea) {
    // tslint:disable-next-line: deprecation
    return A.updateAt(i, a, nea);
}
function updateAt() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 2
        ? function (nea) { return _updateAt(args[0], args[1], nea); }
        : _updateAt(args[0], args[1], args[2]);
}
exports.updateAt = updateAt;
function _modifyAt(nea, i, f) {
    // tslint:disable-next-line: deprecation
    return A.modifyAt(nea, i, f);
}
function modifyAt() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 2
        ? function (nea) { return _modifyAt(nea, args[0], args[1]); }
        : _modifyAt(args[0], args[1], args[2]);
}
exports.modifyAt = modifyAt;
/**
 * @since 1.17.0
 */
exports.copy = function (nea) {
    return A.copy(nea);
};
function _filter(nea, predicate) {
    // tslint:disable-next-line: deprecation
    return filterWithIndex(nea, function (_, a) { return predicate(a); });
}
function filter() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1 ? function (nea) { return _filter(nea, args[0]); } : _filter(args[0], args[1]);
}
exports.filter = filter;
function _filterWithIndex(nea, predicate) {
    return fromArray(nea.filter(function (a, i) { return predicate(i, a); }));
}
function filterWithIndex() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1
        ? function (nea) { return _filterWithIndex(nea, args[0]); }
        : _filterWithIndex(args[0], args[1]);
}
exports.filterWithIndex = filterWithIndex;
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
exports.snoc = A.snoc;
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
exports.cons = A.cons;
/**
 * @since 1.15.0
 */
exports.nonEmptyArray = {
    URI: exports.URI,
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
exports.of = A.array.of;
var _a = pipeable_1.pipeable(exports.nonEmptyArray), ap = _a.ap, apFirst = _a.apFirst, apSecond = _a.apSecond, chain = _a.chain, chainFirst = _a.chainFirst, duplicate = _a.duplicate, extend = _a.extend, flatten = _a.flatten, foldMap = _a.foldMap, foldMapWithIndex = _a.foldMapWithIndex, map = _a.map, mapWithIndex = _a.mapWithIndex, reduce = _a.reduce, reduceRight = _a.reduceRight, reduceRightWithIndex = _a.reduceRightWithIndex, reduceWithIndex = _a.reduceWithIndex;
exports.ap = ap;
exports.apFirst = apFirst;
exports.apSecond = apSecond;
exports.chain = chain;
exports.chainFirst = chainFirst;
exports.duplicate = duplicate;
exports.extend = extend;
exports.flatten = flatten;
exports.foldMap = foldMap;
exports.foldMapWithIndex = foldMapWithIndex;
exports.map = map;
exports.mapWithIndex = mapWithIndex;
exports.reduce = reduce;
exports.reduceRight = reduceRight;
exports.reduceRightWithIndex = reduceRightWithIndex;
exports.reduceWithIndex = reduceWithIndex;
