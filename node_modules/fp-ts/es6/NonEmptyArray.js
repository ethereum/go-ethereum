import { array, findFirst as arrayFindFirst, findIndex as arrayFindIndex, findLast as arrayFindLast, findLastIndex as arrayFindLastIndex, insertAt as arrayInsertAt, last, lookup, sort, updateAt as arrayUpdateAt, getEq as getArrayEq } from './Array';
import { compose, toString } from './function';
import { none, some } from './Option';
import { fold, getJoinSemigroup, getMeetSemigroup } from './Semigroup';
import { fromEquals } from './Eq';
export var URI = 'NonEmptyArray';
/**
 * @since 1.0.0
 */
var NonEmptyArray = /** @class */ (function () {
    function NonEmptyArray(head, tail) {
        this.head = head;
        this.tail = tail;
    }
    /**
     * Converts this `NonEmptyArray` to a plain `Array`
     *
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     *
     * assert.deepStrictEqual(new NonEmptyArray(1, [2, 3]).toArray(), [1, 2, 3])
     */
    NonEmptyArray.prototype.toArray = function () {
        return [this.head].concat(this.tail);
    };
    /**
     * Converts this `NonEmptyArray` to a plain `Array` using the given map function
     *
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     *
     * assert.deepStrictEqual(new NonEmptyArray('a', ['bb', 'ccc']).toArrayMap(s => s.length), [1, 2, 3])
     *
     * @since 1.14.0
     */
    NonEmptyArray.prototype.toArrayMap = function (f) {
        return [f(this.head)].concat(this.tail.map(function (a) { return f(a); }));
    };
    /**
     * Concatenates this `NonEmptyArray` and passed `Array`
     *
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     *
     * assert.deepStrictEqual(new NonEmptyArray<number>(1, []).concatArray([2]), new NonEmptyArray(1, [2]))
     */
    NonEmptyArray.prototype.concatArray = function (as) {
        return new NonEmptyArray(this.head, this.tail.concat(as));
    };
    /**
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     *
     * const double = (n: number): number => n * 2
     * assert.deepStrictEqual(new NonEmptyArray(1, [2]).map(double), new NonEmptyArray(2, [4]))
     */
    NonEmptyArray.prototype.map = function (f) {
        return new NonEmptyArray(f(this.head), this.tail.map(f));
    };
    NonEmptyArray.prototype.mapWithIndex = function (f) {
        return new NonEmptyArray(f(0, this.head), array.mapWithIndex(this.tail, function (i, a) { return f(i + 1, a); }));
    };
    /**
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     *
     * const x = new NonEmptyArray(1, [2])
     * const double = (n: number): number => n * 2
     * assert.deepStrictEqual(x.ap(new NonEmptyArray(double, [double])).toArray(), [2, 4, 2, 4])
     */
    NonEmptyArray.prototype.ap = function (fab) {
        var _this = this;
        return fab.chain(function (f) { return _this.map(f); }); // <= derived
    };
    /**
     * Flipped version of `ap`
     *
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     *
     * const x = new NonEmptyArray(1, [2])
     * const double = (n: number) => n * 2
     * assert.deepStrictEqual(new NonEmptyArray(double, [double]).ap_(x).toArray(), [2, 4, 2, 4])
     */
    NonEmptyArray.prototype.ap_ = function (fb) {
        return fb.ap(this);
    };
    /**
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     *
     * const x = new NonEmptyArray(1, [2])
     * const f = (a: number) => new NonEmptyArray(a, [4])
     * assert.deepStrictEqual(x.chain(f).toArray(), [1, 4, 2, 4])
     */
    NonEmptyArray.prototype.chain = function (f) {
        return f(this.head).concatArray(array.chain(this.tail, function (a) { return f(a).toArray(); }));
    };
    /**
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     *
     * const x = new NonEmptyArray(1, [2])
     * const y = new NonEmptyArray(3, [4])
     * assert.deepStrictEqual(x.concat(y).toArray(), [1, 2, 3, 4])
     */
    NonEmptyArray.prototype.concat = function (y) {
        return this.concatArray(y.toArray());
    };
    /**
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     *
     * const x = new NonEmptyArray('a', ['b'])
     * assert.strictEqual(x.reduce('', (b, a) => b + a), 'ab')
     */
    NonEmptyArray.prototype.reduce = function (b, f) {
        return array.reduce(this.toArray(), b, f);
    };
    /**
     * @since 1.12.0
     */
    NonEmptyArray.prototype.reduceWithIndex = function (b, f) {
        return array.reduceWithIndex(this.toArray(), b, f);
    };
    /**
     * @since 1.12.0
     */
    NonEmptyArray.prototype.foldr = function (b, f) {
        return this.foldrWithIndex(b, function (_, a, b) { return f(a, b); });
    };
    /**
     * @since 1.12.0
     */
    NonEmptyArray.prototype.foldrWithIndex = function (b, f) {
        return f(0, this.head, this.tail.reduceRight(function (acc, a, i) { return f(i + 1, a, acc); }, b));
    };
    /**
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     * import { fold, monoidSum } from 'fp-ts/lib/Monoid'
     *
     * const sum = (as: NonEmptyArray<number>) => fold(monoidSum)(as.toArray())
     * assert.deepStrictEqual(new NonEmptyArray(1, [2, 3, 4]).extend(sum), new NonEmptyArray(10, [9, 7, 4]))
     */
    NonEmptyArray.prototype.extend = function (f) {
        return unsafeFromArray(array.extend(this.toArray(), function (as) { return f(unsafeFromArray(as)); }));
    };
    /**
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     *
     * assert.strictEqual(new NonEmptyArray(1, [2, 3]).extract(), 1)
     */
    NonEmptyArray.prototype.extract = function () {
        return this.head;
    };
    /**
     * Same as `toString`
     */
    NonEmptyArray.prototype.inspect = function () {
        return this.toString();
    };
    /**
     * Return stringified representation of this `NonEmptyArray`
     */
    NonEmptyArray.prototype.toString = function () {
        // tslint:disable-next-line: deprecation
        return "new NonEmptyArray(" + toString(this.head) + ", " + toString(this.tail) + ")";
    };
    /**
     * Gets minimum of this `NonEmptyArray` using specified `Ord` instance
     *
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     * import { ordNumber } from 'fp-ts/lib/Ord'
     *
     * assert.strictEqual(new NonEmptyArray(1, [2, 3]).min(ordNumber), 1)
     *
     * @since 1.3.0
     */
    NonEmptyArray.prototype.min = function (ord) {
        return fold(getMeetSemigroup(ord))(this.head)(this.tail);
    };
    /**
     * Gets maximum of this `NonEmptyArray` using specified `Ord` instance
     *
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     * import { ordNumber } from 'fp-ts/lib/Ord'
     *
     * assert.strictEqual(new NonEmptyArray(1, [2, 3]).max(ordNumber), 3)
     *
     * @since 1.3.0
     */
    NonEmptyArray.prototype.max = function (ord) {
        return fold(getJoinSemigroup(ord))(this.head)(this.tail);
    };
    /**
     * Gets last element of this `NonEmptyArray`
     *
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     *
     * assert.strictEqual(new NonEmptyArray(1, [2, 3]).last(), 3)
     * assert.strictEqual(new NonEmptyArray(1, []).last(), 1)
     *
     * @since 1.6.0
     */
    NonEmptyArray.prototype.last = function () {
        return last(this.tail).getOrElse(this.head);
    };
    /**
     * Sorts this `NonEmptyArray` using specified `Ord` instance
     *
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     * import { ordNumber } from 'fp-ts/lib/Ord'
     *
     * assert.deepStrictEqual(new NonEmptyArray(3, [2, 1]).sort(ordNumber), new NonEmptyArray(1, [2, 3]))
     *
     * @since 1.6.0
     */
    NonEmptyArray.prototype.sort = function (ord) {
        return unsafeFromArray(sort(ord)(this.toArray()));
    };
    /**
     * Reverts this `NonEmptyArray`
     *
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     *
     * assert.deepStrictEqual(new NonEmptyArray(1, [2, 3]).reverse(), new NonEmptyArray(3, [2, 1]))
     *
     * @since 1.6.0
     */
    NonEmptyArray.prototype.reverse = function () {
        return unsafeFromArray(this.toArray().reverse());
    };
    /**
     * @since 1.10.0
     */
    NonEmptyArray.prototype.length = function () {
        return 1 + this.tail.length;
    };
    /**
     * This function provides a safe way to read a value at a particular index from an NonEmptyArray
     *
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     * import { some, none } from 'fp-ts/lib/Option'
     *
     * assert.deepStrictEqual(new NonEmptyArray(1, [2, 3]).lookup(1), some(2))
     * assert.deepStrictEqual(new NonEmptyArray(1, [2, 3]).lookup(3), none)
     *
     * @since 1.14.0
     */
    NonEmptyArray.prototype.lookup = function (i) {
        return i === 0 ? some(this.head) : lookup(i - 1, this.tail);
    };
    /**
     * Use `lookup` instead
     * @since 1.11.0
     * @deprecated
     */
    NonEmptyArray.prototype.index = function (i) {
        return this.lookup(i);
    };
    NonEmptyArray.prototype.findFirst = function (predicate) {
        // tslint:disable-next-line: deprecation
        return predicate(this.head) ? some(this.head) : arrayFindFirst(this.tail, predicate);
    };
    NonEmptyArray.prototype.findLast = function (predicate) {
        // tslint:disable-next-line: deprecation
        var a = arrayFindLast(this.tail, predicate);
        return a.isSome() ? a : predicate(this.head) ? some(this.head) : none;
    };
    /**
     * Find the first index for which a predicate holds
     *
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     * import { some, none } from 'fp-ts/lib/Option'
     *
     * assert.deepStrictEqual(new NonEmptyArray(1, [2, 3]).findIndex(x => x === 2), some(1))
     * assert.deepStrictEqual(new NonEmptyArray<number>(1, []).findIndex(x => x === 2), none)
     *
     * @since 1.11.0
     */
    NonEmptyArray.prototype.findIndex = function (predicate) {
        if (predicate(this.head)) {
            return some(0);
        }
        else {
            // tslint:disable-next-line: deprecation
            var i = arrayFindIndex(this.tail, predicate);
            return i.isSome() ? some(i.value + 1) : none;
        }
    };
    /**
     * Returns the index of the last element of the list which matches the predicate
     *
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     * import { some, none } from 'fp-ts/lib/Option'
     *
     * interface X {
     *   a: number
     *   b: number
     * }
     * const xs: NonEmptyArray<X> = new NonEmptyArray({ a: 1, b: 0 }, [{ a: 1, b: 1 }])
     * assert.deepStrictEqual(xs.findLastIndex(x => x.a === 1), some(1))
     * assert.deepStrictEqual(xs.findLastIndex(x => x.a === 4), none)
     *
     * @since 1.11.0
     */
    NonEmptyArray.prototype.findLastIndex = function (predicate) {
        // tslint:disable-next-line: deprecation
        var i = arrayFindLastIndex(this.tail, predicate);
        return i.isSome() ? some(i.value + 1) : predicate(this.head) ? some(0) : none;
    };
    /**
     * Insert an element at the specified index, creating a new NonEmptyArray, or returning `None` if the index is out of bounds
     *
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     * import { some } from 'fp-ts/lib/Option'
     *
     * assert.deepStrictEqual(new NonEmptyArray(1, [2, 3, 4]).insertAt(2, 5), some(new NonEmptyArray(1, [2, 5, 3, 4])))
     *
     * @since 1.11.0
     */
    NonEmptyArray.prototype.insertAt = function (i, a) {
        if (i === 0) {
            return some(new NonEmptyArray(a, this.toArray()));
        }
        else {
            // tslint:disable-next-line: deprecation
            var t = arrayInsertAt(i - 1, a, this.tail);
            return t.isSome() ? some(new NonEmptyArray(this.head, t.value)) : none;
        }
    };
    /**
     * Change the element at the specified index, creating a new NonEmptyArray, or returning `None` if the index is out of bounds
     *
     * @example
     * import { NonEmptyArray } from 'fp-ts/lib/NonEmptyArray'
     * import { some, none } from 'fp-ts/lib/Option'
     *
     * assert.deepStrictEqual(new NonEmptyArray(1, [2, 3]).updateAt(1, 1), some(new NonEmptyArray(1, [1, 3])))
     * assert.deepStrictEqual(new NonEmptyArray(1, []).updateAt(1, 1), none)
     *
     * @since 1.11.0
     */
    NonEmptyArray.prototype.updateAt = function (i, a) {
        if (i === 0) {
            return this.head === a ? some(this) : some(new NonEmptyArray(a, this.tail));
        }
        else {
            // tslint:disable-next-line: deprecation
            var t = arrayUpdateAt(i - 1, a, this.tail);
            return t.isSome() ? (t.value === this.tail ? some(this) : some(new NonEmptyArray(this.head, t.value))) : none;
        }
    };
    NonEmptyArray.prototype.filter = function (predicate) {
        return this.filterWithIndex(function (_, a) { return predicate(a); });
    };
    /**
     * @since 1.12.0
     */
    NonEmptyArray.prototype.filterWithIndex = function (predicate) {
        var t = array.filterWithIndex(this.tail, function (i, a) { return predicate(i + 1, a); });
        return predicate(0, this.head) ? some(new NonEmptyArray(this.head, t)) : fromArray(t);
    };
    /**
     * @since 1.14.0
     */
    NonEmptyArray.prototype.some = function (predicate) {
        return predicate(this.head) || this.tail.some(function (a) { return predicate(a); });
    };
    /**
     * @since 1.14.0
     */
    NonEmptyArray.prototype.every = function (predicate) {
        return predicate(this.head) && this.tail.every(function (a) { return predicate(a); });
    };
    return NonEmptyArray;
}());
export { NonEmptyArray };
var unsafeFromArray = function (as) {
    return new NonEmptyArray(as[0], as.slice(1));
};
/**
 * Builds a `NonEmptyArray` from an `Array` returning `none` if `as` is an empty array
 *
 * @since 1.0.0
 */
export var fromArray = function (as) {
    return as.length > 0 ? some(unsafeFromArray(as)) : none;
};
var map = function (fa, f) {
    return fa.map(f);
};
var mapWithIndex = function (fa, f) {
    return fa.mapWithIndex(f);
};
var of = function (a) {
    return new NonEmptyArray(a, []);
};
var ap = function (fab, fa) {
    return fa.ap(fab);
};
var chain = function (fa, f) {
    return fa.chain(f);
};
var concat = function (fx, fy) {
    return fx.concat(fy);
};
/**
 * Builds a `Semigroup` instance for `NonEmptyArray`
 *
 * @since 1.0.0
 */
export var getSemigroup = function () {
    return { concat: concat };
};
/**
 * Use `getEq`
 *
 * @since 1.14.0
 * @deprecated
 */
export var getSetoid = getEq;
/**
 * @example
 * import { NonEmptyArray, getEq } from 'fp-ts/lib/NonEmptyArray'
 * import { eqNumber } from 'fp-ts/lib/Eq'
 *
 * const E = getEq(eqNumber)
 * assert.strictEqual(E.equals(new NonEmptyArray(1, []), new NonEmptyArray(1, [])), true)
 * assert.strictEqual(E.equals(new NonEmptyArray(1, []), new NonEmptyArray(1, [2])), false)
 *
 * @since 1.19.0
 */
export function getEq(S) {
    var eqTail = getArrayEq(S);
    return fromEquals(function (x, y) { return S.equals(x.head, y.head) && eqTail.equals(x.tail, y.tail); });
}
/**
 * Group equal, consecutive elements of an array into non empty arrays.
 *
 * @example
 * import { NonEmptyArray, group } from 'fp-ts/lib/NonEmptyArray'
 * import { ordNumber } from 'fp-ts/lib/Ord'
 *
 * assert.deepStrictEqual(group(ordNumber)([1, 2, 1, 1]), [
 *   new NonEmptyArray(1, []),
 *   new NonEmptyArray(2, []),
 *   new NonEmptyArray(1, [1])
 * ])
 *
 * @since 1.7.0
 */
export var group = function (S) { return function (as) {
    var r = [];
    var len = as.length;
    if (len === 0) {
        return r;
    }
    var head = as[0];
    var tail = [];
    for (var i = 1; i < len; i++) {
        var x = as[i];
        if (S.equals(x, head)) {
            tail.push(x);
        }
        else {
            r.push(new NonEmptyArray(head, tail));
            head = x;
            tail = [];
        }
    }
    r.push(new NonEmptyArray(head, tail));
    return r;
}; };
/**
 * Sort and then group the elements of an array into non empty arrays.
 *
 * @example
 * import { NonEmptyArray, groupSort } from 'fp-ts/lib/NonEmptyArray'
 * import { ordNumber } from 'fp-ts/lib/Ord'
 *
 * assert.deepStrictEqual(groupSort(ordNumber)([1, 2, 1, 1]), [new NonEmptyArray(1, [1, 1]), new NonEmptyArray(2, [])])
 *
 * @since 1.7.0
 */
export var groupSort = function (O) {
    // tslint:disable-next-line: deprecation
    return compose(group(O), sort(O));
};
var reduce = function (fa, b, f) {
    return fa.reduce(b, f);
};
var foldMap = function (M) { return function (fa, f) {
    return fa.tail.reduce(function (acc, a) { return M.concat(acc, f(a)); }, f(fa.head));
}; };
var foldr = function (fa, b, f) {
    return fa.foldr(b, f);
};
var reduceWithIndex = function (fa, b, f) {
    return fa.reduceWithIndex(b, f);
};
var foldMapWithIndex = function (M) { return function (fa, f) {
    return fa.tail.reduce(function (acc, a, i) { return M.concat(acc, f(i + 1, a)); }, f(0, fa.head));
}; };
var foldrWithIndex = function (fa, b, f) {
    return fa.foldrWithIndex(b, f);
};
var extend = function (fa, f) {
    return fa.extend(f);
};
var extract = function (fa) {
    return fa.extract();
};
function traverse(F) {
    var traverseWithIndexF = traverseWithIndex(F);
    return function (ta, f) { return traverseWithIndexF(ta, function (_, a) { return f(a); }); };
}
function sequence(F) {
    var sequenceF = array.sequence(F);
    return function (ta) {
        return F.ap(F.map(ta.head, function (a) { return function (as) { return new NonEmptyArray(a, as); }; }), sequenceF(ta.tail));
    };
}
/**
 * Splits an array into sub-non-empty-arrays stored in an object, based on the result of calling a `string`-returning
 * function on each element, and grouping the results according to values returned
 *
 * @example
 * import { NonEmptyArray, groupBy } from 'fp-ts/lib/NonEmptyArray'
 *
 * assert.deepStrictEqual(groupBy(['foo', 'bar', 'foobar'], a => String(a.length)), {
 *   '3': new NonEmptyArray('foo', ['bar']),
 *   '6': new NonEmptyArray('foobar', [])
 * })
 *
 * @since 1.10.0
 */
export var groupBy = function (as, f) {
    var r = {};
    for (var _i = 0, as_1 = as; _i < as_1.length; _i++) {
        var a = as_1[_i];
        var k = f(a);
        if (r.hasOwnProperty(k)) {
            r[k].tail.push(a);
        }
        else {
            r[k] = new NonEmptyArray(a, []);
        }
    }
    return r;
};
var traverseWithIndex = function (F) {
    var traverseWithIndexF = array.traverseWithIndex(F);
    return function (ta, f) {
        var fb = f(0, ta.head);
        var fbs = traverseWithIndexF(ta.tail, function (i, a) { return f(i + 1, a); });
        return F.ap(F.map(fb, function (b) { return function (bs) { return new NonEmptyArray(b, bs); }; }), fbs);
    };
};
/**
 * @since 1.0.0
 */
export var nonEmptyArray = {
    URI: URI,
    extend: extend,
    extract: extract,
    map: map,
    mapWithIndex: mapWithIndex,
    of: of,
    ap: ap,
    chain: chain,
    reduce: reduce,
    foldMap: foldMap,
    foldr: foldr,
    traverse: traverse,
    sequence: sequence,
    reduceWithIndex: reduceWithIndex,
    foldMapWithIndex: foldMapWithIndex,
    foldrWithIndex: foldrWithIndex,
    traverseWithIndex: traverseWithIndex
};
