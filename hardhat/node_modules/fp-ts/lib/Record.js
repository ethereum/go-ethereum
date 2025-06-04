"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var Monoid_1 = require("./Monoid");
var Option_1 = require("./Option");
var Eq_1 = require("./Eq");
var Show_1 = require("./Show");
var pipeable_1 = require("./pipeable");
/**
 * @since 1.17.0
 */
exports.getShow = function (S) {
    return {
        show: function (r) {
            // tslint:disable-next-line: deprecation
            var elements = collect(r, function (k, a) { return Show_1.showString.show(k) + ": " + S.show(a); }).join(', ');
            return elements === '' ? '{}' : "{ " + elements + " }";
        }
    };
};
/**
 * Calculate the number of key/value pairs in a record
 *
 * @since 1.10.0
 */
exports.size = function (d) {
    return Object.keys(d).length;
};
/**
 * Test whether a record is empty
 *
 * @since 1.10.0
 */
exports.isEmpty = function (d) {
    return Object.keys(d).length === 0;
};
function _collect(d, f) {
    var out = [];
    var keys = Object.keys(d).sort();
    for (var _i = 0, keys_1 = keys; _i < keys_1.length; _i++) {
        var key = keys_1[_i];
        out.push(f(key, d[key]));
    }
    return out;
}
function collect() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1 ? function (d) { return _collect(d, args[0]); } : _collect(args[0], args[1]);
}
exports.collect = collect;
function toArray(d) {
    // tslint:disable-next-line: deprecation
    return collect(d, function (k, a) { return [k, a]; });
}
exports.toArray = toArray;
function toUnfoldable(unfoldable) {
    return function (d) {
        var arr = toArray(d);
        var len = arr.length;
        return unfoldable.unfoldr(0, function (b) { return (b < len ? Option_1.some([arr[b], b + 1]) : Option_1.none); });
    };
}
exports.toUnfoldable = toUnfoldable;
function insert(k, a, d) {
    if (d[k] === a) {
        return d;
    }
    var r = Object.assign({}, d);
    r[k] = a;
    return r;
}
exports.insert = insert;
function remove(k, d) {
    if (!hasOwnProperty(k, d)) {
        return d;
    }
    var r = Object.assign({}, d);
    delete r[k];
    return r;
}
exports.remove = remove;
function _pop(k, d) {
    var a = exports.lookup(k, d);
    // tslint:disable-next-line: deprecation
    return a.isNone() ? Option_1.none : Option_1.some([a.value, remove(k, d)]);
}
function pop() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1 ? function (d) { return _pop(args[0], d); } : _pop(args[0], args[1]);
}
exports.pop = pop;
/**
 * Test whether one record contains all of the keys and values contained in another record
 *
 * @since 1.14.0
 */
exports.isSubrecord = function (E) { return function (d1, d2) {
    for (var k in d1) {
        if (!hasOwnProperty(k, d2) || !E.equals(d1[k], d2[k])) {
            return false;
        }
    }
    return true;
}; };
/**
 * Use `isSubrecord` instead
 * @since 1.10.0
 * @deprecated
 */
exports.isSubdictionary = exports.isSubrecord;
/**
 * Use `getEq`
 *
 * @since 1.10.0
 * @deprecated
 */
exports.getSetoid = getEq;
function getEq(E) {
    var isSubrecordE = exports.isSubrecord(E);
    return Eq_1.fromEquals(function (x, y) { return isSubrecordE(x, y) && isSubrecordE(y, x); });
}
exports.getEq = getEq;
function getMonoid(S) {
    // tslint:disable-next-line: deprecation
    return Monoid_1.getDictionaryMonoid(S);
}
exports.getMonoid = getMonoid;
/**
 * Lookup the value for a key in a record
 * @since 1.10.0
 */
exports.lookup = function (key, fa) {
    return hasOwnProperty(key, fa) ? Option_1.some(fa[key]) : Option_1.none;
};
function filter() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1 ? function (fa) { return exports.record.filter(fa, args[0]); } : exports.record.filter(args[0], args[1]);
}
exports.filter = filter;
/**
 * @since 1.10.0
 */
exports.empty = {};
function mapWithKey(fa, f) {
    return exports.record.mapWithIndex(fa, f);
}
exports.mapWithKey = mapWithKey;
function map() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1 ? function (fa) { return exports.record.map(fa, args[0]); } : exports.record.map(args[0], args[1]);
}
exports.map = map;
/**
 * Reduce object by iterating over it's values.
 *
 * @since 1.10.0
 *
 * @example
 * import { reduce } from 'fp-ts/lib/Record'
 *
 * const joinAllVals = (ob: {[k: string]: string}) => reduce(ob, '', (acc, val) => acc + val)
 *
 * assert.deepStrictEqual(joinAllVals({a: 'foo', b: 'bar'}), 'foobar')
 */
function reduce(fa, b, f) {
    return exports.record.reduce(fa, b, f);
}
exports.reduce = reduce;
function foldMap(M) {
    var foldMapM = exports.record.foldMap(M);
    return function () {
        var args = [];
        for (var _i = 0; _i < arguments.length; _i++) {
            args[_i] = arguments[_i];
        }
        return args.length === 1 ? function (fa) { return foldMapM(fa, args[0]); } : foldMapM(args[0], args[1]);
    };
}
exports.foldMap = foldMap;
/**
 * Use `reduceRight`
 *
 * @since 1.10.0
 * @deprecated
 */
function foldr(fa, b, f) {
    return exports.record.foldr(fa, b, f);
}
exports.foldr = foldr;
function reduceWithKey(fa, b, f) {
    return exports.record.reduceWithIndex(fa, b, f);
}
exports.reduceWithKey = reduceWithKey;
/**
 * Use `foldMapWithIndex`
 *
 * @since 1.12.0
 * @deprecated
 */
exports.foldMapWithKey = function (M) { return function (fa, f) {
    return exports.record.foldMapWithIndex(M)(fa, f);
}; };
function foldrWithKey(fa, b, f) {
    return exports.record.foldrWithIndex(fa, b, f);
}
exports.foldrWithKey = foldrWithKey;
/**
 * Create a record with one key/value pair
 *
 * @since 1.10.0
 */
exports.singleton = function (k, a) {
    var _a;
    return _a = {}, _a[k] = a, _a;
};
function traverseWithKey(F) {
    return exports.record.traverseWithIndex(F);
}
exports.traverseWithKey = traverseWithKey;
function traverse(F) {
    return exports.record.traverse(F);
}
exports.traverse = traverse;
function sequence(F) {
    return traverseWithIndex(F)(function (_, a) { return a; });
}
exports.sequence = sequence;
/**
 * @since 1.10.0
 */
exports.compact = function (fa) {
    var r = {};
    var keys = Object.keys(fa);
    for (var _i = 0, keys_2 = keys; _i < keys_2.length; _i++) {
        var key = keys_2[_i];
        var optionA = fa[key];
        if (optionA.isSome()) {
            r[key] = optionA.value;
        }
    }
    return r;
};
function partitionMap() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1
        ? function (fa) { return exports.record.partitionMap(fa, args[0]); }
        : exports.record.partitionMap(args[0], args[1]);
}
exports.partitionMap = partitionMap;
function partition() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1
        ? function (fa) { return exports.record.partition(fa, args[0]); }
        : exports.record.partition(args[0], args[1]);
}
exports.partition = partition;
/**
 * @since 1.10.0
 */
function separate(fa) {
    return exports.record.separate(fa);
}
exports.separate = separate;
function wither(F) {
    return exports.record.wither(F);
}
exports.wither = wither;
function wilt(F) {
    return exports.record.wilt(F);
}
exports.wilt = wilt;
function filterMap() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1
        ? function (fa) { return exports.record.filterMap(fa, args[0]); }
        : exports.record.filterMap(args[0], args[1]);
}
exports.filterMap = filterMap;
function partitionMapWithKey(fa, f) {
    var left = {};
    var right = {};
    var keys = Object.keys(fa);
    for (var _i = 0, keys_3 = keys; _i < keys_3.length; _i++) {
        var key = keys_3[_i];
        var e = f(key, fa[key]);
        if (e.isLeft()) {
            left[key] = e.value;
        }
        else {
            right[key] = e.value;
        }
    }
    return {
        left: left,
        right: right
    };
}
exports.partitionMapWithKey = partitionMapWithKey;
function partitionWithKey(fa, predicate) {
    return exports.record.partitionWithIndex(fa, predicate);
}
exports.partitionWithKey = partitionWithKey;
function filterMapWithKey(fa, f) {
    return exports.record.filterMapWithIndex(fa, f);
}
exports.filterMapWithKey = filterMapWithKey;
function filterWithKey(fa, predicate) {
    return exports.record.filterWithIndex(fa, predicate);
}
exports.filterWithKey = filterWithKey;
function fromFoldable(
// tslint:disable-next-line: deprecation
F) {
    return function (ta, f) {
        return F.reduce(ta, {}, function (b, _a) {
            var k = _a[0], a = _a[1];
            b[k] = hasOwnProperty(k, b) ? f(b[k], a) : a;
            return b;
        });
    };
}
exports.fromFoldable = fromFoldable;
function fromFoldableMap(M, 
// tslint:disable-next-line: deprecation
F) {
    return function (ta, f) {
        return F.reduce(ta, {}, function (r, a) {
            var _a = f(a), k = _a[0], b = _a[1];
            r[k] = hasOwnProperty(k, r) ? M.concat(r[k], b) : b;
            return r;
        });
    };
}
exports.fromFoldableMap = fromFoldableMap;
/**
 * @since 1.14.0
 */
function every(fa, predicate) {
    for (var k in fa) {
        if (!predicate(fa[k])) {
            return false;
        }
    }
    return true;
}
exports.every = every;
/**
 * @since 1.14.0
 */
function some(fa, predicate) {
    for (var k in fa) {
        if (predicate(fa[k])) {
            return true;
        }
    }
    return false;
}
exports.some = some;
/**
 * @since 1.14.0
 */
function elem(E) {
    return function (a, fa) { return some(fa, function (x) { return E.equals(x, a); }); };
}
exports.elem = elem;
function partitionMapWithIndex() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1
        ? function (fa) { return exports.record.partitionMapWithIndex(fa, args[0]); }
        : exports.record.partitionMapWithIndex(args[0], args[1]);
}
exports.partitionMapWithIndex = partitionMapWithIndex;
function partitionWithIndex() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1
        ? function (fa) { return exports.record.partitionWithIndex(fa, args[0]); }
        : exports.record.partitionWithIndex(args[0], args[1]);
}
exports.partitionWithIndex = partitionWithIndex;
function filterMapWithIndex() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1
        ? function (fa) { return exports.record.filterMapWithIndex(fa, args[0]); }
        : exports.record.filterMapWithIndex(args[0], args[1]);
}
exports.filterMapWithIndex = filterMapWithIndex;
function filterWithIndex() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1
        ? function (fa) { return exports.record.filterWithIndex(fa, args[0]); }
        : exports.record.filterWithIndex(args[0], args[1]);
}
exports.filterWithIndex = filterWithIndex;
function insertAt(k, a) {
    // tslint:disable-next-line: deprecation
    return function (r) { return insert(k, a, r); };
}
exports.insertAt = insertAt;
function deleteAt(k) {
    // tslint:disable-next-line: deprecation
    return function (r) { return remove(k, r); };
}
exports.deleteAt = deleteAt;
/**
 * @since 1.19.0
 */
exports.URI = 'Record';
function mapWithIndex(f) {
    return function (fa) { return exports.record.mapWithIndex(fa, f); };
}
exports.mapWithIndex = mapWithIndex;
function reduceWithIndex(b, f) {
    return function (fa) { return exports.record.reduceWithIndex(fa, b, f); };
}
exports.reduceWithIndex = reduceWithIndex;
function foldMapWithIndex(M) {
    var foldMapWithIndexM = exports.record.foldMapWithIndex(M);
    return function (f) { return function (fa) { return foldMapWithIndexM(fa, f); }; };
}
exports.foldMapWithIndex = foldMapWithIndex;
function reduceRightWithIndex(b, f) {
    return function (fa) { return exports.record.foldrWithIndex(fa, b, f); };
}
exports.reduceRightWithIndex = reduceRightWithIndex;
var _hasOwnProperty = Object.prototype.hasOwnProperty;
/**
 * @since 1.19.0
 */
function hasOwnProperty(k, d) {
    return _hasOwnProperty.call(d, k);
}
exports.hasOwnProperty = hasOwnProperty;
function traverseWithIndex(F) {
    var traverseWithIndexF = exports.record.traverseWithIndex(F);
    return function (f) { return function (ta) { return traverseWithIndexF(ta, f); }; };
}
exports.traverseWithIndex = traverseWithIndex;
function traverse2v(F) {
    var traverseWithIndexF = traverseWithIndex(F);
    return function (f) { return traverseWithIndexF(function (_, a) { return f(a); }); };
}
exports.traverse2v = traverse2v;
/**
 * @since 1.19.0
 */
exports.record = {
    URI: exports.URI,
    map: function (fa, f) { return exports.record.mapWithIndex(fa, function (_, a) { return f(a); }); },
    reduce: function (fa, b, f) { return exports.record.reduceWithIndex(fa, b, function (_, b, a) { return f(b, a); }); },
    foldMap: function (M) {
        var foldMapWithIndexM = exports.record.foldMapWithIndex(M);
        return function (fa, f) { return foldMapWithIndexM(fa, function (_, a) { return f(a); }); };
    },
    foldr: function (fa, b, f) { return exports.record.foldrWithIndex(fa, b, function (_, a, b) { return f(a, b); }); },
    traverse: function (F) {
        var traverseWithIndexF = exports.record.traverseWithIndex(F);
        return function (ta, f) { return traverseWithIndexF(ta, function (_, a) { return f(a); }); };
    },
    sequence: sequence,
    compact: function (fa) {
        var r = {};
        var keys = Object.keys(fa);
        for (var _i = 0, keys_4 = keys; _i < keys_4.length; _i++) {
            var key = keys_4[_i];
            var optionA = fa[key];
            if (Option_1.isSome(optionA)) {
                r[key] = optionA.value;
            }
        }
        return r;
    },
    separate: function (fa) {
        var left = {};
        var right = {};
        var keys = Object.keys(fa);
        for (var _i = 0, keys_5 = keys; _i < keys_5.length; _i++) {
            var key = keys_5[_i];
            var e = fa[key];
            switch (e._tag) {
                case 'Left':
                    left[key] = e.value;
                    break;
                case 'Right':
                    right[key] = e.value;
                    break;
            }
        }
        return {
            left: left,
            right: right
        };
    },
    filter: function (fa, predicate) {
        return exports.record.filterWithIndex(fa, function (_, a) { return predicate(a); });
    },
    filterMap: function (fa, f) { return exports.record.filterMapWithIndex(fa, function (_, a) { return f(a); }); },
    partition: function (fa, predicate) {
        return exports.record.partitionWithIndex(fa, function (_, a) { return predicate(a); });
    },
    partitionMap: function (fa, f) { return exports.record.partitionMapWithIndex(fa, function (_, a) { return f(a); }); },
    wither: function (F) {
        var traverseF = exports.record.traverse(F);
        return function (wa, f) { return F.map(traverseF(wa, f), exports.record.compact); };
    },
    wilt: function (F) {
        var traverseF = exports.record.traverse(F);
        return function (wa, f) { return F.map(traverseF(wa, f), exports.record.separate); };
    },
    mapWithIndex: function (fa, f) {
        var out = {};
        var keys = Object.keys(fa);
        for (var _i = 0, keys_6 = keys; _i < keys_6.length; _i++) {
            var key = keys_6[_i];
            out[key] = f(key, fa[key]);
        }
        return out;
    },
    reduceWithIndex: function (fa, b, f) {
        var out = b;
        var keys = Object.keys(fa).sort();
        var len = keys.length;
        for (var i = 0; i < len; i++) {
            var k = keys[i];
            out = f(k, out, fa[k]);
        }
        return out;
    },
    foldMapWithIndex: function (M) { return function (fa, f) {
        var out = M.empty;
        var keys = Object.keys(fa).sort();
        var len = keys.length;
        for (var i = 0; i < len; i++) {
            var k = keys[i];
            out = M.concat(out, f(k, fa[k]));
        }
        return out;
    }; },
    foldrWithIndex: function (fa, b, f) {
        var out = b;
        var keys = Object.keys(fa).sort();
        var len = keys.length;
        for (var i = len - 1; i >= 0; i--) {
            var k = keys[i];
            out = f(k, fa[k], out);
        }
        return out;
    },
    traverseWithIndex: function (F) { return function (ta, f) {
        var keys = Object.keys(ta);
        if (keys.length === 0) {
            return F.of(exports.empty);
        }
        var fr = F.of({});
        var _loop_1 = function (key) {
            fr = F.ap(F.map(fr, function (r) { return function (b) {
                r[key] = b;
                return r;
            }; }), f(key, ta[key]));
        };
        for (var _i = 0, keys_7 = keys; _i < keys_7.length; _i++) {
            var key = keys_7[_i];
            _loop_1(key);
        }
        return fr;
    }; },
    partitionMapWithIndex: function (fa, f) {
        var left = {};
        var right = {};
        var keys = Object.keys(fa);
        for (var _i = 0, keys_8 = keys; _i < keys_8.length; _i++) {
            var key = keys_8[_i];
            var e = f(key, fa[key]);
            switch (e._tag) {
                case 'Left':
                    left[key] = e.value;
                    break;
                case 'Right':
                    right[key] = e.value;
                    break;
            }
        }
        return {
            left: left,
            right: right
        };
    },
    partitionWithIndex: function (fa, predicateWithIndex) {
        var left = {};
        var right = {};
        var keys = Object.keys(fa);
        for (var _i = 0, keys_9 = keys; _i < keys_9.length; _i++) {
            var key = keys_9[_i];
            var a = fa[key];
            if (predicateWithIndex(key, a)) {
                right[key] = a;
            }
            else {
                left[key] = a;
            }
        }
        return {
            left: left,
            right: right
        };
    },
    filterMapWithIndex: function (fa, f) {
        var r = {};
        var keys = Object.keys(fa);
        for (var _i = 0, keys_10 = keys; _i < keys_10.length; _i++) {
            var key = keys_10[_i];
            var optionB = f(key, fa[key]);
            if (Option_1.isSome(optionB)) {
                r[key] = optionB.value;
            }
        }
        return r;
    },
    filterWithIndex: function (fa, predicateWithIndex) {
        var out = {};
        var changed = false;
        for (var key in fa) {
            if (_hasOwnProperty.call(fa, key)) {
                var a = fa[key];
                if (predicateWithIndex(key, a)) {
                    out[key] = a;
                }
                else {
                    changed = true;
                }
            }
        }
        return changed ? out : fa;
    }
};
var reduceRight = pipeable_1.pipeable(exports.record).reduceRight;
exports.reduceRight = reduceRight;
