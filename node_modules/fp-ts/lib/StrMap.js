"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var R = require("./Record");
var Semigroup_1 = require("./Semigroup");
var Eq_1 = require("./Eq");
exports.URI = 'StrMap';
var liftSeparated = function (_a) {
    var left = _a.left, right = _a.right;
    return {
        left: new StrMap(left),
        right: new StrMap(right)
    };
};
/**
 * @data
 * @constructor StrMap
 * @since 1.0.0
 */
var StrMap = /** @class */ (function () {
    function StrMap(value) {
        this.value = value;
    }
    StrMap.prototype.mapWithKey = function (f) {
        // tslint:disable-next-line: deprecation
        return new StrMap(R.mapWithKey(this.value, f));
    };
    StrMap.prototype.map = function (f) {
        return this.mapWithKey(function (_, a) { return f(a); });
    };
    StrMap.prototype.reduce = function (b, f) {
        return R.reduce(this.value, b, f);
    };
    /**
     * @since 1.12.0
     */
    StrMap.prototype.foldr = function (b, f) {
        // tslint:disable-next-line: deprecation
        return R.foldr(this.value, b, f);
    };
    /**
     * @since 1.12.0
     */
    StrMap.prototype.reduceWithKey = function (b, f) {
        // tslint:disable-next-line: deprecation
        return R.reduceWithKey(this.value, b, f);
    };
    /**
     * @since 1.12.0
     */
    StrMap.prototype.foldrWithKey = function (b, f) {
        // tslint:disable-next-line: deprecation
        return R.foldrWithKey(this.value, b, f);
    };
    StrMap.prototype.filter = function (p) {
        return this.filterWithKey(function (_, a) { return p(a); });
    };
    /**
     * @since 1.12.0
     */
    StrMap.prototype.filterMap = function (f) {
        return this.filterMapWithKey(function (_, a) { return f(a); });
    };
    /**
     * @since 1.12.0
     */
    StrMap.prototype.partition = function (p) {
        return this.partitionWithKey(function (_, a) { return p(a); });
    };
    /**
     * @since 1.12.0
     */
    StrMap.prototype.partitionMap = function (f) {
        return this.partitionMapWithKey(function (_, a) { return f(a); });
    };
    /**
     * @since 1.12.0
     */
    StrMap.prototype.separate = function () {
        return liftSeparated(R.separate(this.value));
    };
    /**
     * Use `partitionMapWithKey` instead
     * @since 1.12.0
     * @deprecated
     */
    StrMap.prototype.partitionMapWithIndex = function (f) {
        return this.partitionMapWithKey(f);
    };
    /**
     * @since 1.14.0
     */
    StrMap.prototype.partitionMapWithKey = function (f) {
        // tslint:disable-next-line: deprecation
        return liftSeparated(R.partitionMapWithKey(this.value, f));
    };
    /**
     * Use `partitionWithKey` instead
     * @since 1.12.0
     * @deprecated
     */
    StrMap.prototype.partitionWithIndex = function (p) {
        return this.partitionWithKey(p);
    };
    /**
     * @since 1.14.0
     */
    StrMap.prototype.partitionWithKey = function (p) {
        // tslint:disable-next-line: deprecation
        return liftSeparated(R.partitionWithKey(this.value, p));
    };
    /**
     * Use `filterMapWithKey` instead
     * @since 1.12.0
     * @deprecated
     */
    StrMap.prototype.filterMapWithIndex = function (f) {
        return this.filterMapWithKey(f);
    };
    /**
     * @since 1.14.0
     */
    StrMap.prototype.filterMapWithKey = function (f) {
        // tslint:disable-next-line: deprecation
        return new StrMap(R.filterMapWithKey(this.value, f));
    };
    /**
     * Use `filterWithKey` instead
     * @since 1.12.0
     * @deprecated
     */
    StrMap.prototype.filterWithIndex = function (p) {
        return this.filterWithKey(p);
    };
    /**
     * @since 1.14.0
     */
    StrMap.prototype.filterWithKey = function (p) {
        // tslint:disable-next-line: deprecation
        return new StrMap(R.filterWithKey(this.value, p));
    };
    /**
     * @since 1.14.0
     */
    StrMap.prototype.every = function (predicate) {
        return R.every(this.value, predicate);
    };
    /**
     * @since 1.14.0
     */
    StrMap.prototype.some = function (predicate) {
        return R.some(this.value, predicate);
    };
    return StrMap;
}());
exports.StrMap = StrMap;
/**
 * @since 1.17.0
 */
exports.getShow = function (S) {
    var SR = R.getShow(S);
    return {
        show: function (s) { return "new StrMap(" + SR.show(s.value) + ")"; }
    };
};
/**
 *
 * @since 1.10.0
 */
var empty = new StrMap(R.empty);
var concat = function (S) {
    var concat = R.getMonoid(S).concat;
    return function (x, y) { return new StrMap(concat(x.value, y.value)); };
};
/**
 *
 * @since 1.0.0
 */
exports.getMonoid = function (S) {
    if (S === void 0) { S = Semigroup_1.getLastSemigroup(); }
    return {
        concat: concat(S),
        empty: empty
    };
};
var map = function (fa, f) {
    return fa.map(f);
};
var reduce = function (fa, b, f) {
    return fa.reduce(b, f);
};
var foldMap = function (M) {
    var foldMapM = R.foldMap(M);
    // tslint:disable-next-line: deprecation
    return function (fa, f) { return foldMapM(fa.value, f); };
};
var foldr = function (fa, b, f) {
    return fa.foldr(b, f);
};
var reduceWithIndex = function (fa, b, f) {
    return fa.reduceWithKey(b, f);
};
var foldMapWithIndex = function (M) {
    // tslint:disable-next-line: deprecation
    var foldMapWithKey = R.foldMapWithKey(M);
    return function (fa, f) { return foldMapWithKey(fa.value, f); };
};
var foldrWithIndex = function (fa, b, f) {
    return fa.foldrWithKey(b, f);
};
function traverseWithKey(F) {
    // tslint:disable-next-line: deprecation
    var traverseWithKeyF = R.traverseWithKey(F);
    return function (ta, f) { return F.map(traverseWithKeyF(ta.value, f), function (d) { return new StrMap(d); }); };
}
exports.traverseWithKey = traverseWithKey;
function traverse(F) {
    var traverseWithKeyF = traverseWithKey(F);
    return function (ta, f) { return traverseWithKeyF(ta, function (_, a) { return f(a); }); };
}
function sequence(F) {
    var traverseWithKeyF = traverseWithKey(F);
    return function (ta) { return traverseWithKeyF(ta, function (_, a) { return a; }); };
}
/**
 * Test whether one dictionary contains all of the keys and values contained in another dictionary
 *
 * @since 1.0.0
 */
exports.isSubdictionary = function (E) {
    var isSubrecordS = R.isSubrecord(E);
    return function (d1, d2) { return isSubrecordS(d1.value, d2.value); };
};
/**
 * Calculate the number of key/value pairs in a dictionary
 *
 * @since 1.0.0
 */
exports.size = function (d) {
    return R.size(d.value);
};
/**
 * Test whether a dictionary is empty
 *
 * @since 1.0.0
 */
exports.isEmpty = function (d) {
    return R.isEmpty(d.value);
};
/**
 * Use `getEq`
 *
 * @since 1.0.0
 * @deprecated
 */
exports.getSetoid = getEq;
/**
 * @since 1.19.0
 */
function getEq(E) {
    var isSubrecordS = R.isSubrecord(E);
    return Eq_1.fromEquals(function (x, y) { return isSubrecordS(x.value, y.value) && isSubrecordS(y.value, x.value); });
}
exports.getEq = getEq;
/**
 * Create a dictionary with one key/value pair
 *
 * @since 1.0.0
 */
exports.singleton = function (k, a) {
    return new StrMap(R.singleton(k, a));
};
/**
 * Lookup the value for a key in a dictionary
 *
 * @since 1.0.0
 */
exports.lookup = function (k, d) {
    return R.lookup(k, d.value);
};
function fromFoldable(
// tslint:disable-next-line: deprecation
F) {
    var fromFoldableF = R.fromFoldable(F);
    return function (ta, onConflict) { return new StrMap(fromFoldableF(ta, onConflict)); };
}
exports.fromFoldable = fromFoldable;
/**
 *
 * @since 1.0.0
 */
exports.collect = function (d, f) {
    // tslint:disable-next-line: deprecation
    return R.collect(d.value, f);
};
/**
 *
 * @since 1.0.0
 */
exports.toArray = function (d) {
    return R.toArray(d.value);
};
function toUnfoldable(U) {
    var toUnfoldableU = R.toUnfoldable(U);
    return function (d) { return toUnfoldableU(d.value); };
}
exports.toUnfoldable = toUnfoldable;
/**
 * Insert or replace a key/value pair in a map
 *
 * @since 1.0.0
 */
exports.insert = function (k, a, d) {
    // tslint:disable-next-line: deprecation
    var value = R.insert(k, a, d.value);
    return value === d.value ? d : new StrMap(value);
};
/**
 * Delete a key and value from a map
 *
 * @since 1.0.0
 */
exports.remove = function (k, d) {
    // tslint:disable-next-line: deprecation
    var value = R.remove(k, d.value);
    return value === d.value ? d : new StrMap(value);
};
/**
 * Delete a key and value from a map, returning the value as well as the subsequent map
 *
 * @since 1.0.0
 */
exports.pop = function (k, d) {
    // tslint:disable-next-line: deprecation
    return R.pop(k, d.value).map(function (_a) {
        var a = _a[0], d = _a[1];
        return [a, new StrMap(d)];
    });
};
/**
 * @since 1.14.0
 */
function elem(E) {
    return function (a, fa) { return fa.some(function (x) { return E.equals(x, a); }); };
}
exports.elem = elem;
var filterMap = function (fa, f) {
    return fa.filterMap(f);
};
var filter = function (fa, p) {
    return fa.filter(p);
};
var compact = function (fa) {
    return new StrMap(R.compact(fa.value));
};
var separate = function (fa) {
    return fa.separate();
};
var partitionMap = function (fa, f) {
    return fa.partitionMap(f);
};
var partition = function (fa, p) {
    return fa.partition(p);
};
var wither = function (F) {
    var traverseF = traverse(F);
    return function (wa, f) { return F.map(traverseF(wa, f), compact); };
};
var wilt = function (F) {
    var traverseF = traverse(F);
    return function (wa, f) { return F.map(traverseF(wa, f), separate); };
};
var mapWithIndex = function (fa, f) {
    return fa.mapWithKey(f);
};
// tslint:disable-next-line: deprecation
var traverseWithIndex = traverseWithKey;
var partitionMapWithIndex = function (fa, f) {
    return fa.partitionMapWithKey(f);
};
var partitionWithIndex = function (fa, p) {
    return fa.partitionWithKey(p);
};
var filterMapWithIndex = function (fa, f) {
    return fa.filterMapWithKey(f);
};
var filterWithIndex = function (fa, p) {
    return fa.filterWithKey(p);
};
/**
 * @since 1.0.0
 */
exports.strmap = {
    URI: exports.URI,
    map: map,
    reduce: reduce,
    foldMap: foldMap,
    foldr: foldr,
    traverse: traverse,
    sequence: sequence,
    compact: compact,
    separate: separate,
    filter: filter,
    filterMap: filterMap,
    partition: partition,
    partitionMap: partitionMap,
    wither: wither,
    wilt: wilt,
    mapWithIndex: mapWithIndex,
    reduceWithIndex: reduceWithIndex,
    foldMapWithIndex: foldMapWithIndex,
    foldrWithIndex: foldrWithIndex,
    traverseWithIndex: traverseWithIndex,
    partitionMapWithIndex: partitionMapWithIndex,
    partitionWithIndex: partitionWithIndex,
    filterMapWithIndex: filterMapWithIndex,
    filterWithIndex: filterWithIndex
};
