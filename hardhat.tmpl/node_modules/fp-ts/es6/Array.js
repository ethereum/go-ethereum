import { concat, identity, tuple } from './function';
import { none, some, isSome } from './Option';
import { getSemigroup, ordNumber, fromCompare } from './Ord';
import { getArraySetoid } from './Setoid';
import { pipeable } from './pipeable';
export var URI = 'Array';
/**
 * @since 1.17.0
 */
export var getShow = function (S) {
    return {
        show: function (arr) { return "[" + arr.map(S.show).join(', ') + "]"; }
    };
};
/**
 *
 * @example
 * import { getMonoid } from 'fp-ts/lib/Array'
 *
 * const M = getMonoid<number>()
 * assert.deepStrictEqual(M.concat([1, 2], [3, 4]), [1, 2, 3, 4])
 *
 * @since 1.0.0
 */
export var getMonoid = function () {
    return {
        concat: concat,
        empty: empty
    };
};
/**
 * Use `getEq`
 *
 * @since 1.0.0
 * @deprecated
 */
export var getSetoid = getEq;
/**
 * Derives a `Eq` over the `Array` of a given element type from the `Eq` of that type. The derived eq defines two
 * arrays as equal if all elements of both arrays are compared equal pairwise with the given eq `S`. In case of
 * arrays of different lengths, the result is non equality.
 *
 * @example
 * import { eqString } from 'fp-ts/lib/Eq'
 * import { getEq } from 'fp-ts/lib/Array'
 *
 * const E = getEq(eqString)
 * assert.strictEqual(E.equals(['a', 'b'], ['a', 'b']), true)
 * assert.strictEqual(E.equals(['a'], []), false)
 *
 * @since 1.19.0
 */
export function getEq(E) {
    // tslint:disable-next-line: deprecation
    return getArraySetoid(E);
}
/**
 * Derives an `Ord` over the Array of a given element type from the `Ord` of that type. The ordering between two such
 * arrays is equal to: the first non equal comparison of each arrays elements taken pairwise in increasing order, in
 * case of equality over all the pairwise elements; the longest array is considered the greatest, if both arrays have
 * the same length, the result is equality.
 *
 *
 * @example
 * import { getOrd } from 'fp-ts/lib/Array'
 * import { ordString } from 'fp-ts/lib/Ord'
 *
 * const O = getOrd(ordString)
 * assert.strictEqual(O.compare(['b'], ['a']), 1)
 * assert.strictEqual(O.compare(['a'], ['a']), 0)
 * assert.strictEqual(O.compare(['a'], ['b']), -1)
 *
 *
 * @since 1.2.0
 */
export var getOrd = function (O) {
    return fromCompare(function (a, b) {
        var aLen = a.length;
        var bLen = b.length;
        var len = Math.min(aLen, bLen);
        for (var i = 0; i < len; i++) {
            var order = O.compare(a[i], b[i]);
            if (order !== 0) {
                return order;
            }
        }
        return ordNumber.compare(aLen, bLen);
    });
};
export function traverse(F) {
    return array.traverse(F);
}
/**
 * An empty array
 *
 *
 * @since 1.9.0
 */
export var empty = [];
/**
 * Return a list of length `n` with element `i` initialized with `f(i)`
 *
 * @example
 * import { makeBy } from 'fp-ts/lib/Array'
 *
 * const double = (n: number): number => n * 2
 * assert.deepStrictEqual(makeBy(5, double), [0, 2, 4, 6, 8])
 *
 *
 * @since 1.10.0
 */
export var makeBy = function (n, f) {
    var r = [];
    for (var i = 0; i < n; i++) {
        r.push(f(i));
    }
    return r;
};
/**
 * Create an array containing a range of integers, including both endpoints
 *
 * @example
 * import { range } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(range(1, 5), [1, 2, 3, 4, 5])
 *
 *
 * @since 1.10.0
 */
export var range = function (start, end) {
    return makeBy(end - start + 1, function (i) { return start + i; });
};
/**
 * Create an array containing a value repeated the specified number of times
 *
 * @example
 * import { replicate } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(replicate(3, 'a'), ['a', 'a', 'a'])
 *
 *
 * @since 1.10.0
 */
export var replicate = function (n, a) {
    return makeBy(n, function () { return a; });
};
/**
 * Removes one level of nesting
 *
 * @example
 * import { flatten } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(flatten([[1], [2], [3]]), [1, 2, 3])
 *
 * @since 1.0.0
 */
export var flatten = function (ffa) {
    var rLen = 0;
    var len = ffa.length;
    for (var i = 0; i < len; i++) {
        rLen += ffa[i].length;
    }
    var r = Array(rLen);
    var start = 0;
    for (var i = 0; i < len; i++) {
        var arr = ffa[i];
        var l = arr.length;
        for (var j = 0; j < l; j++) {
            r[j + start] = arr[j];
        }
        start += l;
    }
    return r;
};
/**
 * Use `foldLeft`
 *
 * @since 1.0.0
 * @deprecated
 */
export var fold = function (as, onNil, onCons) {
    return isEmpty(as) ? onNil : onCons(as[0], as.slice(1));
};
/**
 * Use `foldLeft`
 *
 * @since 1.0.0
 * @deprecated
 */
export var foldL = function (as, onNil, onCons) {
    return isEmpty(as) ? onNil() : onCons(as[0], as.slice(1));
};
/**
 * Use `foldRight`
 *
 * @since 1.7.0
 * @deprecated
 */
export var foldr = function (as, onNil, onCons) {
    return isEmpty(as) ? onNil : onCons(as.slice(0, as.length - 1), as[as.length - 1]);
};
/**
 * Use `foldRight`
 *
 * @since 1.7.0
 * @deprecated
 */
export var foldrL = function (as, onNil, onCons) {
    return isEmpty(as) ? onNil() : onCons(as.slice(0, as.length - 1), as[as.length - 1]);
};
/**
 * Same as `reduce` but it carries over the intermediate steps
 *
 * ```ts
 * import { scanLeft } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(scanLeft([1, 2, 3], 10, (b, a) => b - a), [ 10, 9, 7, 4 ])
 * ```
 *
 *
 * @since 1.1.0
 */
export var scanLeft = function (as, b, f) {
    var l = as.length;
    var r = new Array(l + 1);
    r[0] = b;
    for (var i = 0; i < l; i++) {
        r[i + 1] = f(r[i], as[i]);
    }
    return r;
};
/**
 * Fold an array from the right, keeping all intermediate results instead of only the final result
 *
 * @example
 * import { scanRight } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(scanRight([1, 2, 3], 10, (a, b) => b - a), [ 4, 5, 7, 10 ])
 *
 *
 * @since 1.1.0
 */
export var scanRight = function (as, b, f) {
    var l = as.length;
    var r = new Array(l + 1);
    r[l] = b;
    for (var i = l - 1; i >= 0; i--) {
        r[i] = f(as[i], r[i + 1]);
    }
    return r;
};
/**
 * Test whether an array is empty
 *
 * @example
 * import { isEmpty } from 'fp-ts/lib/Array'
 *
 * assert.strictEqual(isEmpty([]), true)
 *
 * @since 1.0.0
 */
export var isEmpty = function (as) {
    return as.length === 0;
};
/**
 * Test whether an array is non empty narrowing down the type to `NonEmptyArray<A>`
 *
 * @since 1.19.0
 */
export function isNonEmpty(as) {
    return as.length > 0;
}
/**
 * Test whether an array contains a particular index
 *
 * @since 1.0.0
 */
export var isOutOfBound = function (i, as) {
    return i < 0 || i >= as.length;
};
/**
 * This function provides a safe way to read a value at a particular index from an array
 *
 * @example
 * import { lookup } from 'fp-ts/lib/Array'
 * import { some, none } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(lookup(1, [1, 2, 3]), some(2))
 * assert.deepStrictEqual(lookup(3, [1, 2, 3]), none)
 *
 * @since 1.14.0
 */
export var lookup = function (i, as) {
    return isOutOfBound(i, as) ? none : some(as[i]);
};
/**
 * Use `lookup` instead
 *
 * @since 1.0.0
 * @deprecated
 */
export var index = function (i, as) {
    return lookup(i, as);
};
/**
 * Attaches an element to the front of an array, creating a new non empty array
 *
 * @example
 * import { cons } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(cons(0, [1, 2, 3]), [0, 1, 2, 3])
 *
 * @since 1.0.0
 */
export var cons = function (a, as) {
    var len = as.length;
    var r = Array(len + 1);
    for (var i = 0; i < len; i++) {
        r[i + 1] = as[i];
    }
    r[0] = a;
    return r;
};
/**
 * Append an element to the end of an array, creating a new non empty array
 *
 * @example
 * import { snoc } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(snoc([1, 2, 3], 4), [1, 2, 3, 4])
 *
 * @since 1.0.0
 */
export var snoc = function (as, a) {
    var len = as.length;
    var r = Array(len + 1);
    for (var i = 0; i < len; i++) {
        r[i] = as[i];
    }
    r[len] = a;
    return r;
};
/**
 * Get the first element in an array, or `None` if the array is empty
 *
 * @example
 * import { head } from 'fp-ts/lib/Array'
 * import { some, none } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(head([1, 2, 3]), some(1))
 * assert.deepStrictEqual(head([]), none)
 *
 * @since 1.0.0
 */
export var head = function (as) {
    return isEmpty(as) ? none : some(as[0]);
};
/**
 * Get the last element in an array, or `None` if the array is empty
 *
 * @example
 * import { last } from 'fp-ts/lib/Array'
 * import { some, none } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(last([1, 2, 3]), some(3))
 * assert.deepStrictEqual(last([]), none)
 *
 * @since 1.0.0
 */
export var last = function (as) {
    return lookup(as.length - 1, as);
};
/**
 * Get all but the first element of an array, creating a new array, or `None` if the array is empty
 *
 * @example
 * import { tail } from 'fp-ts/lib/Array'
 * import { some, none } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(tail([1, 2, 3]), some([2, 3]))
 * assert.deepStrictEqual(tail([]), none)
 *
 * @since 1.0.0
 */
export var tail = function (as) {
    return isEmpty(as) ? none : some(as.slice(1));
};
/**
 * Get all but the last element of an array, creating a new array, or `None` if the array is empty
 *
 * @example
 * import { init } from 'fp-ts/lib/Array'
 * import { some, none } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(init([1, 2, 3]), some([1, 2]))
 * assert.deepStrictEqual(init([]), none)
 *
 * @since 1.0.0
 */
export var init = function (as) {
    var len = as.length;
    return len === 0 ? none : some(as.slice(0, len - 1));
};
/**
 * Use `takeLeft`
 *
 * @since 1.0.0
 * @deprecated
 */
export function take(n, as) {
    return as.slice(0, n);
}
/**
 * Use `takeRight`
 *
 * @since 1.10.0
 * @deprecated
 */
export var takeEnd = function (n, as) {
    return n === 0 ? empty : as.slice(-n);
};
export function takeWhile(as, predicate) {
    var i = spanIndexUncurry(as, predicate);
    var init = Array(i);
    for (var j = 0; j < i; j++) {
        init[j] = as[j];
    }
    return init;
}
var spanIndexUncurry = function (as, predicate) {
    var l = as.length;
    var i = 0;
    for (; i < l; i++) {
        if (!predicate(as[i])) {
            break;
        }
    }
    return i;
};
export function span(as, predicate) {
    var i = spanIndexUncurry(as, predicate);
    var init = Array(i);
    for (var j = 0; j < i; j++) {
        init[j] = as[j];
    }
    var l = as.length;
    var rest = Array(l - i);
    for (var j = i; j < l; j++) {
        rest[j - i] = as[j];
    }
    return { init: init, rest: rest };
}
/**
 * Use `dropLeft`
 *
 * @since 1.0.0
 * @deprecated
 */
export var drop = function (n, as) {
    return as.slice(n, as.length);
};
/**
 * Use `dropRight`
 *
 * @since 1.10.0
 * @deprecated
 */
export var dropEnd = function (n, as) {
    return as.slice(0, as.length - n);
};
/**
 * Use `dropLeftWhile`
 *
 * @since 1.0.0
 * @deprecated
 */
export var dropWhile = function (as, predicate) {
    var i = spanIndexUncurry(as, predicate);
    var l = as.length;
    var rest = Array(l - i);
    for (var j = i; j < l; j++) {
        rest[j - i] = as[j];
    }
    return rest;
};
function _findIndex(as, predicate) {
    var len = as.length;
    for (var i = 0; i < len; i++) {
        if (predicate(as[i])) {
            return some(i);
        }
    }
    return none;
}
export function findIndex() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1 ? function (as) { return _findIndex(as, args[0]); } : _findIndex(args[0], args[1]);
}
function _findFirst(as, predicate) {
    var len = as.length;
    for (var i = 0; i < len; i++) {
        if (predicate(as[i])) {
            return some(as[i]);
        }
    }
    return none;
}
export function findFirst() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1 ? function (as) { return _findFirst(as, args[0]); } : _findFirst(args[0], args[1]);
}
function _findFirstMap(arr, f) {
    var len = arr.length;
    for (var i = 0; i < len; i++) {
        var v = f(arr[i]);
        if (v.isSome()) {
            return v;
        }
    }
    return none;
}
export function findFirstMap() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1 ? function (as) { return _findFirstMap(as, args[0]); } : _findFirstMap(args[0], args[1]);
}
function _findLast(as, predicate) {
    var len = as.length;
    for (var i = len - 1; i >= 0; i--) {
        if (predicate(as[i])) {
            return some(as[i]);
        }
    }
    return none;
}
export function findLast() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1 ? function (as) { return _findLast(as, args[0]); } : _findLast(args[0], args[1]);
}
function _findLastMap(arr, f) {
    var len = arr.length;
    for (var i = len - 1; i >= 0; i--) {
        var v = f(arr[i]);
        if (v.isSome()) {
            return v;
        }
    }
    return none;
}
export function findLastMap() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1 ? function (as) { return _findLastMap(as, args[0]); } : _findLastMap(args[0], args[1]);
}
function _findLastIndex(as, predicate) {
    var len = as.length;
    for (var i = len - 1; i >= 0; i--) {
        if (predicate(as[i])) {
            return some(i);
        }
    }
    return none;
}
export function findLastIndex() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1 ? function (as) { return _findLastIndex(as, args[0]); } : _findLastIndex(args[0], args[1]);
}
/**
 * Use `array.filter` instead
 *
 * @since 1.0.0
 * @deprecated
 */
export var refine = function (as, refinement) {
    // tslint:disable-next-line: deprecation
    return filter(as, refinement);
};
/**
 * @since 1.0.0
 */
export var copy = function (as) {
    var l = as.length;
    var r = Array(l);
    for (var i = 0; i < l; i++) {
        r[i] = as[i];
    }
    return r;
};
/**
 * @since 1.0.0
 */
export var unsafeInsertAt = function (i, a, as) {
    var xs = copy(as);
    xs.splice(i, 0, a);
    return xs;
};
function _insertAt(i, a, as) {
    return i < 0 || i > as.length ? none : some(unsafeInsertAt(i, a, as));
}
export function insertAt() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 2 ? function (as) { return _insertAt(args[0], args[1], as); } : _insertAt(args[0], args[1], args[2]);
}
/**
 * @since 1.0.0
 */
export var unsafeUpdateAt = function (i, a, as) {
    if (as[i] === a) {
        return as;
    }
    else {
        var xs = copy(as);
        xs[i] = a;
        return xs;
    }
};
function _updateAt(i, a, as) {
    return isOutOfBound(i, as) ? none : some(unsafeUpdateAt(i, a, as));
}
export function updateAt() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 2 ? function (as) { return _updateAt(args[0], args[1], as); } : _updateAt(args[0], args[1], args[2]);
}
/**
 * @since 1.0.0
 */
export var unsafeDeleteAt = function (i, as) {
    var xs = copy(as);
    xs.splice(i, 1);
    return xs;
};
function _deleteAt(i, as) {
    return isOutOfBound(i, as) ? none : some(unsafeDeleteAt(i, as));
}
export function deleteAt() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1 ? function (as) { return _deleteAt(args[0], as); } : _deleteAt(args[0], args[1]);
}
function _modifyAt(as, i, f) {
    return isOutOfBound(i, as) ? none : some(unsafeUpdateAt(i, f(as[i]), as));
}
export function modifyAt() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 2 ? function (as) { return _modifyAt(as, args[0], args[1]); } : _modifyAt(args[0], args[1], args[2]);
}
/**
 * Reverse an array, creating a new array
 *
 * @example
 * import { reverse } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(reverse([1, 2, 3]), [3, 2, 1])
 *
 * @since 1.0.0
 */
export var reverse = function (as) {
    return copy(as).reverse();
};
/**
 * Extracts from an array of `Either` all the `Right` elements. All the `Right` elements are extracted in order
 *
 * @example
 * import { rights } from 'fp-ts/lib/Array'
 * import { right, left } from 'fp-ts/lib/Either'
 *
 * assert.deepStrictEqual(rights([right(1), left('foo'), right(2)]), [1, 2])
 *
 * @since 1.0.0
 */
export var rights = function (as) {
    var r = [];
    var len = as.length;
    for (var i = 0; i < len; i++) {
        var a = as[i];
        if (a.isRight()) {
            r.push(a.value);
        }
    }
    return r;
};
/**
 * Extracts from an array of `Either` all the `Left` elements. All the `Left` elements are extracted in order
 *
 * @example
 * import { lefts } from 'fp-ts/lib/Array'
 * import { left, right } from 'fp-ts/lib/Either'
 *
 * assert.deepStrictEqual(lefts([right(1), left('foo'), right(2)]), ['foo'])
 *
 * @since 1.0.0
 */
export var lefts = function (as) {
    var r = [];
    var len = as.length;
    for (var i = 0; i < len; i++) {
        var a = as[i];
        if (a.isLeft()) {
            r.push(a.value);
        }
    }
    return r;
};
/**
 * Sort the elements of an array in increasing order, creating a new array
 *
 * @example
 * import { sort } from 'fp-ts/lib/Array'
 * import { ordNumber } from 'fp-ts/lib/Ord'
 *
 * assert.deepStrictEqual(sort(ordNumber)([3, 2, 1]), [1, 2, 3])
 *
 * @since 1.0.0
 */
export var sort = function (O) { return function (as) {
    return copy(as).sort(O.compare);
}; };
/**
 * Apply a function to pairs of elements at the same index in two arrays, collecting the results in a new array. If one
 * input array is short, excess elements of the longer array are discarded.
 *
 * @example
 * import { zipWith } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(zipWith([1, 2, 3], ['a', 'b', 'c', 'd'], (n, s) => s + n), ['a1', 'b2', 'c3'])
 *
 * @since 1.0.0
 */
export var zipWith = function (fa, fb, f) {
    var fc = [];
    var len = Math.min(fa.length, fb.length);
    for (var i = 0; i < len; i++) {
        fc[i] = f(fa[i], fb[i]);
    }
    return fc;
};
/**
 * Takes two arrays and returns an array of corresponding pairs. If one input array is short, excess elements of the
 * longer array are discarded
 *
 * @example
 * import { zip } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(zip([1, 2, 3], ['a', 'b', 'c', 'd']), [[1, 'a'], [2, 'b'], [3, 'c']])
 *
 * @since 1.0.0
 */
export var zip = function (fa, fb) {
    return zipWith(fa, fb, tuple);
};
/**
 * The function is reverse of `zip`. Takes an array of pairs and return two corresponding arrays
 *
 * @example
 * import { unzip } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(unzip([[1, 'a'], [2, 'b'], [3, 'c']]), [[1, 2, 3], ['a', 'b', 'c']])
 *
 *
 * @since 1.13.0
 */
export var unzip = function (as) {
    var fa = [];
    var fb = [];
    for (var i = 0; i < as.length; i++) {
        fa[i] = as[i][0];
        fb[i] = as[i][1];
    }
    return [fa, fb];
};
function _rotate(n, xs) {
    var len = xs.length;
    if (n === 0 || len <= 1 || len === Math.abs(n)) {
        return xs;
    }
    else if (n < 0) {
        return _rotate(len + n, xs);
    }
    else {
        return xs.slice(-n).concat(xs.slice(0, len - n));
    }
}
export function rotate() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1 ? function (as) { return _rotate(args[0], as); } : _rotate(args[0], args[1]);
}
/**
 * Test if a value is a member of an array. Takes a `Eq<A>` as a single
 * argument which returns the function to use to search for a value of type `A` in
 * an array of type `Array<A>`.
 *
 * @example
 * import { elem } from 'fp-ts/lib/Array'
 * import { eqNumber } from 'fp-ts/lib/Eq'
 *
 * assert.strictEqual(elem(eqNumber)(1, [1, 2, 3]), true)
 * assert.strictEqual(elem(eqNumber)(4, [1, 2, 3]), false)
 *
 * @since 1.14.0
 */
export var elem = function (E) { return function (a, as) {
    var predicate = function (e) { return E.equals(e, a); };
    var i = 0;
    var len = as.length;
    for (; i < len; i++) {
        if (predicate(as[i])) {
            return true;
        }
    }
    return false;
}; };
/**
 * Use `elem` instead
 * @since 1.3.0
 * @deprecated
 */
export var member = function (E) {
    var has = elem(E);
    return function (as, a) { return has(a, as); };
};
/**
 * Remove duplicates from an array, keeping the first occurance of an element.
 *
 * @example
 * import { uniq } from 'fp-ts/lib/Array'
 * import { eqNumber } from 'fp-ts/lib/Eq'
 *
 * assert.deepStrictEqual(uniq(eqNumber)([1, 2, 1]), [1, 2])
 *
 *
 * @since 1.3.0
 */
export var uniq = function (E) {
    var elemE = elem(E);
    return function (as) {
        var r = [];
        var len = as.length;
        var i = 0;
        for (; i < len; i++) {
            var a = as[i];
            if (!elemE(a, r)) {
                r.push(a);
            }
        }
        return len === r.length ? as : r;
    };
};
/**
 * Sort the elements of an array in increasing order, where elements are compared using first `ords[0]`, then `ords[1]`,
 * etc...
 *
 * @example
 * import { sortBy } from 'fp-ts/lib/Array'
 * import { contramap, ordString, ordNumber } from 'fp-ts/lib/Ord'
 *
 * interface Person {
 *   name: string
 *   age: number
 * }
 * const byName = contramap((p: Person) => p.name, ordString)
 * const byAge = contramap((p: Person) => p.age, ordNumber)
 *
 * const sortByNameByAge = sortBy([byName, byAge])
 *
 * if (sortByNameByAge.isSome()) {
 *   const persons = [{ name: 'a', age: 1 }, { name: 'b', age: 3 }, { name: 'c', age: 2 }, { name: 'b', age: 2 }]
 *   assert.deepStrictEqual(sortByNameByAge.value(persons), [
 *     { name: 'a', age: 1 },
 *     { name: 'b', age: 2 },
 *     { name: 'b', age: 3 },
 *     { name: 'c', age: 2 }
 *   ])
 * }
 *
 *
 * @since 1.3.0
 */
export var sortBy = function (ords) {
    // tslint:disable-next-line: deprecation
    return fold(ords, none, function (head, tail) { return some(sortBy1(head, tail)); });
};
/**
 * Non failing version of `sortBy`
 * @example
 * import { sortBy1 } from 'fp-ts/lib/Array'
 * import { contramap, ordString, ordNumber } from 'fp-ts/lib/Ord'
 *
 * interface Person {
 *   name: string
 *   age: number
 * }
 * const byName = contramap((p: Person) => p.name, ordString)
 * const byAge = contramap((p: Person) => p.age, ordNumber)
 *
 * const sortByNameByAge = sortBy1(byName, [byAge])
 *
 * const persons = [{ name: 'a', age: 1 }, { name: 'b', age: 3 }, { name: 'c', age: 2 }, { name: 'b', age: 2 }]
 * assert.deepStrictEqual(sortByNameByAge(persons), [
 *   { name: 'a', age: 1 },
 *   { name: 'b', age: 2 },
 *   { name: 'b', age: 3 },
 *   { name: 'c', age: 2 }
 * ])
 *
 *
 * @since 1.3.0
 */
export var sortBy1 = function (head, tail) {
    return sort(tail.reduce(getSemigroup().concat, head));
};
/**
 * Use `filterMap`
 *
 * Apply a function to each element in an array, keeping only the results which contain a value, creating a new array.
 *
 * @example
 * import { mapOption } from 'fp-ts/lib/Array'
 * import { Option, some, none } from 'fp-ts/lib/Option'
 *
 * const f = (n: number): Option<number> => (n % 2 === 0 ? none : some(n))
 * assert.deepStrictEqual(mapOption([1, 2, 3], f), [1, 3])
 *
 * @since 1.0.0
 * @deprecated
 */
export var mapOption = function (as, f) {
    return array.filterMapWithIndex(as, function (_, a) { return f(a); });
};
/**
 * Use `compact`
 *
 * Filter an array of optional values, keeping only the elements which contain a value, creating a new array.
 *
 * @example
 * import { catOptions } from 'fp-ts/lib/Array'
 * import { some, none } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(catOptions([some(1), none, some(3)]), [1, 3])
 *
 * @since 1.0.0
 * @deprecated
 */
// tslint:disable-next-line: deprecation
export var catOptions = function (as) { return mapOption(as, identity); };
export function partitionMap() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1
        ? function (fa) { return array.partitionMapWithIndex(fa, function (_, a) { return args[0](a); }); }
        : array.partitionMapWithIndex(args[0], function (_, a) { return args[1](a); });
}
export function filter() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1 ? function (as) { return array.filter(as, args[0]); } : array.filter(args[0], args[1]);
}
export function partition() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1
        ? function (fa) { return array.partitionWithIndex(fa, function (_, a) { return args[0](a); }); }
        : array.partitionWithIndex(args[0], function (_, a) { return args[1](a); });
}
function _chop(as, f) {
    var result = [];
    var cs = as;
    while (isNonEmpty(cs)) {
        var _a = f(cs), b = _a[0], c = _a[1];
        result.push(b);
        cs = c;
    }
    return result;
}
export function chop() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1 ? function (as) { return _chop(as, args[0]); } : _chop(args[0], args[1]);
}
/**
 * Use `splitAt`
 *
 * @since 1.10.0
 * @deprecated
 */
export var split = function (n, as) {
    return [as.slice(0, n), as.slice(n)];
};
function _chunksOf(as, n) {
    // tslint:disable-next-line: deprecation
    return isOutOfBound(n - 1, as) ? [as] : chop(as, function (as) { return split(n, as); });
}
export function chunksOf() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1 ? function (as) { return _chunksOf(as, args[0]); } : _chunksOf(args[0], args[1]);
}
export function comprehension(input, f, g) {
    var go = function (scope, input) {
        if (input.length === 0) {
            return f.apply(void 0, scope) ? [g.apply(void 0, scope)] : empty;
        }
        else {
            return array.chain(input[0], function (x) { return go(snoc(scope, x), input.slice(1)); });
        }
    };
    return go(empty, input);
}
/**
 * Creates an array of unique values, in order, from all given arrays using a `Eq` for equality comparisons
 *
 * @example
 * import { union } from 'fp-ts/lib/Array'
 * import { eqNumber } from 'fp-ts/lib/Eq'
 *
 * assert.deepStrictEqual(union(eqNumber)([1, 2], [2, 3]), [1, 2, 3])
 *
 *
 * @since 1.12.0
 */
export var union = function (E) {
    var elemE = elem(E);
    // tslint:disable-next-line: deprecation
    return function (xs, ys) { return concat(xs, ys.filter(function (a) { return !elemE(a, xs); })); };
};
/**
 * Creates an array of unique values that are included in all given arrays using a `Eq` for equality
 * comparisons. The order and references of result values are determined by the first array.
 *
 * @example
 * import { intersection } from 'fp-ts/lib/Array'
 * import { eqNumber } from 'fp-ts/lib/Eq'
 *
 * assert.deepStrictEqual(intersection(eqNumber)([1, 2], [2, 3]), [2])
 *
 *
 * @since 1.12.0
 */
export var intersection = function (E) {
    var elemE = elem(E);
    return function (xs, ys) { return xs.filter(function (a) { return elemE(a, ys); }); };
};
/**
 * Creates an array of array values not included in the other given array using a `Eq` for equality
 * comparisons. The order and references of result values are determined by the first array.
 *
 * @example
 * import { difference } from 'fp-ts/lib/Array'
 * import { eqNumber } from 'fp-ts/lib/Eq'
 *
 * assert.deepStrictEqual(difference(eqNumber)([1, 2], [2, 3]), [1])
 *
 *
 * @since 1.12.0
 */
export var difference = function (E) {
    var elemE = elem(E);
    return function (xs, ys) { return xs.filter(function (a) { return !elemE(a, ys); }); };
};
/**
 * @since 1.0.0
 */
export var array = {
    URI: URI,
    map: function (fa, f) { return fa.map(function (a) { return f(a); }); },
    mapWithIndex: function (fa, f) { return fa.map(function (a, i) { return f(i, a); }); },
    compact: function (as) { return array.filterMap(as, identity); },
    separate: function (fa) {
        var left = [];
        var right = [];
        for (var _i = 0, fa_1 = fa; _i < fa_1.length; _i++) {
            var e = fa_1[_i];
            if (e._tag === 'Left') {
                left.push(e.value);
            }
            else {
                right.push(e.value);
            }
        }
        return {
            left: left,
            right: right
        };
    },
    filter: function (as, predicate) {
        return as.filter(predicate);
    },
    filterMap: function (as, f) { return array.filterMapWithIndex(as, function (_, a) { return f(a); }); },
    partition: function (fa, predicate) {
        return array.partitionWithIndex(fa, function (_, a) { return predicate(a); });
    },
    partitionMap: partitionMap,
    of: of,
    ap: function (fab, fa) { return flatten(array.map(fab, function (f) { return array.map(fa, f); })); },
    chain: function (fa, f) {
        var resLen = 0;
        var l = fa.length;
        var temp = new Array(l);
        for (var i = 0; i < l; i++) {
            var e = fa[i];
            var arr = f(e);
            resLen += arr.length;
            temp[i] = arr;
        }
        var r = Array(resLen);
        var start = 0;
        for (var i = 0; i < l; i++) {
            var arr = temp[i];
            var l_1 = arr.length;
            for (var j = 0; j < l_1; j++) {
                r[j + start] = arr[j];
            }
            start += l_1;
        }
        return r;
    },
    reduce: function (fa, b, f) { return array.reduceWithIndex(fa, b, function (_, b, a) { return f(b, a); }); },
    foldMap: function (M) {
        var foldMapWithIndexM = array.foldMapWithIndex(M);
        return function (fa, f) { return foldMapWithIndexM(fa, function (_, a) { return f(a); }); };
    },
    foldr: function (fa, b, f) { return array.foldrWithIndex(fa, b, function (_, a, b) { return f(a, b); }); },
    unfoldr: function (b, f) {
        var ret = [];
        var bb = b;
        while (true) {
            var mt = f(bb);
            if (isSome(mt)) {
                var _a = mt.value, a = _a[0], b_1 = _a[1];
                ret.push(a);
                bb = b_1;
            }
            else {
                break;
            }
        }
        return ret;
    },
    traverse: function (F) {
        var traverseWithIndexF = array.traverseWithIndex(F);
        return function (ta, f) { return traverseWithIndexF(ta, function (_, a) { return f(a); }); };
    },
    sequence: function (F) { return function (ta) {
        return array.reduce(ta, F.of(array.zero()), function (fas, fa) { return F.ap(F.map(fas, function (as) { return function (a) { return snoc(as, a); }; }), fa); });
    }; },
    zero: function () { return empty; },
    // tslint:disable-next-line: deprecation
    alt: function (fx, fy) { return concat(fx, fy); },
    extend: function (fa, f) { return fa.map(function (_, i, as) { return f(as.slice(i)); }); },
    wither: function (F) {
        var traverseF = array.traverse(F);
        return function (wa, f) { return F.map(traverseF(wa, f), array.compact); };
    },
    wilt: function (F) {
        var traverseF = array.traverse(F);
        return function (wa, f) { return F.map(traverseF(wa, f), array.separate); };
    },
    reduceWithIndex: function (fa, b, f) {
        var l = fa.length;
        var r = b;
        for (var i = 0; i < l; i++) {
            r = f(i, r, fa[i]);
        }
        return r;
    },
    foldMapWithIndex: function (M) { return function (fa, f) { return fa.reduce(function (b, a, i) { return M.concat(b, f(i, a)); }, M.empty); }; },
    foldrWithIndex: function (fa, b, f) { return fa.reduceRight(function (b, a, i) { return f(i, a, b); }, b); },
    traverseWithIndex: function (F) { return function (ta, f) {
        return array.reduceWithIndex(ta, F.of(array.zero()), function (i, fbs, a) {
            return F.ap(F.map(fbs, function (bs) { return function (b) { return snoc(bs, b); }; }), f(i, a));
        });
    }; },
    partitionMapWithIndex: function (fa, f) {
        var left = [];
        var right = [];
        for (var i = 0; i < fa.length; i++) {
            var e = f(i, fa[i]);
            if (e._tag === 'Left') {
                left.push(e.value);
            }
            else {
                right.push(e.value);
            }
        }
        return {
            left: left,
            right: right
        };
    },
    partitionWithIndex: function (fa, predicateWithIndex) {
        var left = [];
        var right = [];
        for (var i = 0; i < fa.length; i++) {
            var a = fa[i];
            if (predicateWithIndex(i, a)) {
                right.push(a);
            }
            else {
                left.push(a);
            }
        }
        return {
            left: left,
            right: right
        };
    },
    filterMapWithIndex: function (fa, f) {
        var result = [];
        for (var i = 0; i < fa.length; i++) {
            var optionB = f(i, fa[i]);
            if (isSome(optionB)) {
                result.push(optionB.value);
            }
        }
        return result;
    },
    filterWithIndex: function (fa, predicateWithIndex) {
        return fa.filter(function (a, i) { return predicateWithIndex(i, a); });
    }
};
//
// backporting
//
/**
 * @since 1.19.0
 */
export function of(a) {
    return [a];
}
/**
 * Break an array into its first element and remaining elements
 *
 * @example
 * import { foldLeft } from 'fp-ts/lib/Array'
 *
 * const len: <A>(as: Array<A>) => number = foldLeft(() => 0, (_, tail) => 1 + len(tail))
 * assert.strictEqual(len([1, 2, 3]), 3)
 *
 * @since 1.19.0
 */
export function foldLeft(onNil, onCons) {
    // tslint:disable-next-line: deprecation
    return function (as) { return foldL(as, onNil, onCons); };
}
/**
 * Break an array into its initial elements and the last element
 *
 * @since 1.19.0
 */
export function foldRight(onNil, onCons) {
    // tslint:disable-next-line: deprecation
    return function (as) { return foldrL(as, onNil, onCons); };
}
/**
 * Keep only a number of elements from the start of an array, creating a new array.
 * `n` must be a natural number
 *
 * @example
 * import { takeLeft } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(takeLeft(2)([1, 2, 3]), [1, 2])
 *
 * @since 1.19.0
 */
export function takeLeft(n) {
    // tslint:disable-next-line: deprecation
    return function (as) { return take(n, as); };
}
/**
 * Keep only a number of elements from the end of an array, creating a new array.
 * `n` must be a natural number
 *
 * @example
 * import { takeRight } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(takeRight(2)([1, 2, 3, 4, 5]), [4, 5])
 *
 * @since 1.19.0
 */
export function takeRight(n) {
    // tslint:disable-next-line: deprecation
    return function (as) { return takeEnd(n, as); };
}
export function takeLeftWhile(predicate) {
    // tslint:disable-next-line: deprecation
    return function (as) { return takeWhile(as, predicate); };
}
export function spanLeft(predicate) {
    return function (as) { return span(as, predicate); };
}
/**
 * Drop a number of elements from the start of an array, creating a new array
 *
 * @example
 * import { dropLeft } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(dropLeft(2)([1, 2, 3]), [3])
 *
 * @since 1.19.0
 */
export function dropLeft(n) {
    // tslint:disable-next-line: deprecation
    return function (as) { return drop(n, as); };
}
/**
 * Drop a number of elements from the end of an array, creating a new array
 *
 * @example
 * import { dropRight } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(dropRight(2)([1, 2, 3, 4, 5]), [1, 2, 3])
 *
 * @since 1.19.0
 */
export function dropRight(n) {
    // tslint:disable-next-line: deprecation
    return function (as) { return dropEnd(n, as); };
}
/**
 * Remove the longest initial subarray for which all element satisfy the specified predicate, creating a new array
 *
 * @example
 * import { dropLeftWhile } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(dropLeftWhile((n: number) => n % 2 === 1)([1, 3, 2, 4, 5]), [2, 4, 5])
 *
 * @since 1.19.0
 */
export function dropLeftWhile(predicate) {
    // tslint:disable-next-line: deprecation
    return function (as) { return dropWhile(as, predicate); };
}
/**
 * Splits an array into two pieces, the first piece has `n` elements.
 *
 * @example
 * import { splitAt } from 'fp-ts/lib/Array'
 *
 * assert.deepStrictEqual(splitAt(2)([1, 2, 3, 4, 5]), [[1, 2], [3, 4, 5]])
 *
 * @since 1.19.0
 */
export function splitAt(n) {
    // tslint:disable-next-line: deprecation
    return function (as) { return split(n, as); };
}
var _a = pipeable(array), alt = _a.alt, ap = _a.ap, apFirst = _a.apFirst, apSecond = _a.apSecond, chain = _a.chain, chainFirst = _a.chainFirst, duplicate = _a.duplicate, extend = _a.extend, 
// filter, // this top level function already exists
filterMap = _a.filterMap, filterMapWithIndex = _a.filterMapWithIndex, filterWithIndex = _a.filterWithIndex, foldMap = _a.foldMap, foldMapWithIndex = _a.foldMapWithIndex, map = _a.map, mapWithIndex = _a.mapWithIndex, 
// partition, // this top level function already exists
// partitionMap, // this top level function already exists
partitionMapWithIndex = _a.partitionMapWithIndex, partitionWithIndex = _a.partitionWithIndex, reduce = _a.reduce, reduceRight = _a.reduceRight, reduceRightWithIndex = _a.reduceRightWithIndex, reduceWithIndex = _a.reduceWithIndex, compact = _a.compact, separate = _a.separate;
export { alt, ap, apFirst, apSecond, chain, chainFirst, duplicate, extend, 
// filter, // this top level function already exists
filterMap, filterMapWithIndex, filterWithIndex, foldMap, foldMapWithIndex, map, mapWithIndex, 
// partition, // this top level function already exists
// partitionMap, // this top level function already exists
partitionMapWithIndex, partitionWithIndex, reduce, reduceRight, reduceRightWithIndex, reduceWithIndex, compact, separate };
