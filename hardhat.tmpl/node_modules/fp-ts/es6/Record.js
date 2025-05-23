import { getDictionaryMonoid } from './Monoid';
import { none, some as optionSome, isSome } from './Option';
import { fromEquals } from './Eq';
import { showString } from './Show';
import { pipeable } from './pipeable';
/**
 * @since 1.17.0
 */
export var getShow = function (S) {
    return {
        show: function (r) {
            // tslint:disable-next-line: deprecation
            var elements = collect(r, function (k, a) { return showString.show(k) + ": " + S.show(a); }).join(', ');
            return elements === '' ? '{}' : "{ " + elements + " }";
        }
    };
};
/**
 * Calculate the number of key/value pairs in a record
 *
 * @since 1.10.0
 */
export var size = function (d) {
    return Object.keys(d).length;
};
/**
 * Test whether a record is empty
 *
 * @since 1.10.0
 */
export var isEmpty = function (d) {
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
export function collect() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1 ? function (d) { return _collect(d, args[0]); } : _collect(args[0], args[1]);
}
export function toArray(d) {
    // tslint:disable-next-line: deprecation
    return collect(d, function (k, a) { return [k, a]; });
}
export function toUnfoldable(unfoldable) {
    return function (d) {
        var arr = toArray(d);
        var len = arr.length;
        return unfoldable.unfoldr(0, function (b) { return (b < len ? optionSome([arr[b], b + 1]) : none); });
    };
}
export function insert(k, a, d) {
    if (d[k] === a) {
        return d;
    }
    var r = Object.assign({}, d);
    r[k] = a;
    return r;
}
export function remove(k, d) {
    if (!hasOwnProperty(k, d)) {
        return d;
    }
    var r = Object.assign({}, d);
    delete r[k];
    return r;
}
function _pop(k, d) {
    var a = lookup(k, d);
    // tslint:disable-next-line: deprecation
    return a.isNone() ? none : optionSome([a.value, remove(k, d)]);
}
export function pop() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1 ? function (d) { return _pop(args[0], d); } : _pop(args[0], args[1]);
}
/**
 * Test whether one record contains all of the keys and values contained in another record
 *
 * @since 1.14.0
 */
export var isSubrecord = function (E) { return function (d1, d2) {
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
export var isSubdictionary = isSubrecord;
/**
 * Use `getEq`
 *
 * @since 1.10.0
 * @deprecated
 */
export var getSetoid = getEq;
export function getEq(E) {
    var isSubrecordE = isSubrecord(E);
    return fromEquals(function (x, y) { return isSubrecordE(x, y) && isSubrecordE(y, x); });
}
export function getMonoid(S) {
    // tslint:disable-next-line: deprecation
    return getDictionaryMonoid(S);
}
/**
 * Lookup the value for a key in a record
 * @since 1.10.0
 */
export var lookup = function (key, fa) {
    return hasOwnProperty(key, fa) ? optionSome(fa[key]) : none;
};
export function filter() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1 ? function (fa) { return record.filter(fa, args[0]); } : record.filter(args[0], args[1]);
}
/**
 * @since 1.10.0
 */
export var empty = {};
export function mapWithKey(fa, f) {
    return record.mapWithIndex(fa, f);
}
export function map() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1 ? function (fa) { return record.map(fa, args[0]); } : record.map(args[0], args[1]);
}
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
export function reduce(fa, b, f) {
    return record.reduce(fa, b, f);
}
export function foldMap(M) {
    var foldMapM = record.foldMap(M);
    return function () {
        var args = [];
        for (var _i = 0; _i < arguments.length; _i++) {
            args[_i] = arguments[_i];
        }
        return args.length === 1 ? function (fa) { return foldMapM(fa, args[0]); } : foldMapM(args[0], args[1]);
    };
}
/**
 * Use `reduceRight`
 *
 * @since 1.10.0
 * @deprecated
 */
export function foldr(fa, b, f) {
    return record.foldr(fa, b, f);
}
export function reduceWithKey(fa, b, f) {
    return record.reduceWithIndex(fa, b, f);
}
/**
 * Use `foldMapWithIndex`
 *
 * @since 1.12.0
 * @deprecated
 */
export var foldMapWithKey = function (M) { return function (fa, f) {
    return record.foldMapWithIndex(M)(fa, f);
}; };
export function foldrWithKey(fa, b, f) {
    return record.foldrWithIndex(fa, b, f);
}
/**
 * Create a record with one key/value pair
 *
 * @since 1.10.0
 */
export var singleton = function (k, a) {
    var _a;
    return _a = {}, _a[k] = a, _a;
};
export function traverseWithKey(F) {
    return record.traverseWithIndex(F);
}
export function traverse(F) {
    return record.traverse(F);
}
export function sequence(F) {
    return traverseWithIndex(F)(function (_, a) { return a; });
}
/**
 * @since 1.10.0
 */
export var compact = function (fa) {
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
export function partitionMap() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1
        ? function (fa) { return record.partitionMap(fa, args[0]); }
        : record.partitionMap(args[0], args[1]);
}
export function partition() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1
        ? function (fa) { return record.partition(fa, args[0]); }
        : record.partition(args[0], args[1]);
}
/**
 * @since 1.10.0
 */
export function separate(fa) {
    return record.separate(fa);
}
export function wither(F) {
    return record.wither(F);
}
export function wilt(F) {
    return record.wilt(F);
}
export function filterMap() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1
        ? function (fa) { return record.filterMap(fa, args[0]); }
        : record.filterMap(args[0], args[1]);
}
export function partitionMapWithKey(fa, f) {
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
export function partitionWithKey(fa, predicate) {
    return record.partitionWithIndex(fa, predicate);
}
export function filterMapWithKey(fa, f) {
    return record.filterMapWithIndex(fa, f);
}
export function filterWithKey(fa, predicate) {
    return record.filterWithIndex(fa, predicate);
}
export function fromFoldable(
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
export function fromFoldableMap(M, 
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
/**
 * @since 1.14.0
 */
export function every(fa, predicate) {
    for (var k in fa) {
        if (!predicate(fa[k])) {
            return false;
        }
    }
    return true;
}
/**
 * @since 1.14.0
 */
export function some(fa, predicate) {
    for (var k in fa) {
        if (predicate(fa[k])) {
            return true;
        }
    }
    return false;
}
/**
 * @since 1.14.0
 */
export function elem(E) {
    return function (a, fa) { return some(fa, function (x) { return E.equals(x, a); }); };
}
export function partitionMapWithIndex() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1
        ? function (fa) { return record.partitionMapWithIndex(fa, args[0]); }
        : record.partitionMapWithIndex(args[0], args[1]);
}
export function partitionWithIndex() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1
        ? function (fa) { return record.partitionWithIndex(fa, args[0]); }
        : record.partitionWithIndex(args[0], args[1]);
}
export function filterMapWithIndex() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1
        ? function (fa) { return record.filterMapWithIndex(fa, args[0]); }
        : record.filterMapWithIndex(args[0], args[1]);
}
export function filterWithIndex() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    return args.length === 1
        ? function (fa) { return record.filterWithIndex(fa, args[0]); }
        : record.filterWithIndex(args[0], args[1]);
}
export function insertAt(k, a) {
    // tslint:disable-next-line: deprecation
    return function (r) { return insert(k, a, r); };
}
export function deleteAt(k) {
    // tslint:disable-next-line: deprecation
    return function (r) { return remove(k, r); };
}
/**
 * @since 1.19.0
 */
export var URI = 'Record';
export function mapWithIndex(f) {
    return function (fa) { return record.mapWithIndex(fa, f); };
}
export function reduceWithIndex(b, f) {
    return function (fa) { return record.reduceWithIndex(fa, b, f); };
}
export function foldMapWithIndex(M) {
    var foldMapWithIndexM = record.foldMapWithIndex(M);
    return function (f) { return function (fa) { return foldMapWithIndexM(fa, f); }; };
}
export function reduceRightWithIndex(b, f) {
    return function (fa) { return record.foldrWithIndex(fa, b, f); };
}
var _hasOwnProperty = Object.prototype.hasOwnProperty;
/**
 * @since 1.19.0
 */
export function hasOwnProperty(k, d) {
    return _hasOwnProperty.call(d, k);
}
export function traverseWithIndex(F) {
    var traverseWithIndexF = record.traverseWithIndex(F);
    return function (f) { return function (ta) { return traverseWithIndexF(ta, f); }; };
}
export function traverse2v(F) {
    var traverseWithIndexF = traverseWithIndex(F);
    return function (f) { return traverseWithIndexF(function (_, a) { return f(a); }); };
}
/**
 * @since 1.19.0
 */
export var record = {
    URI: URI,
    map: function (fa, f) { return record.mapWithIndex(fa, function (_, a) { return f(a); }); },
    reduce: function (fa, b, f) { return record.reduceWithIndex(fa, b, function (_, b, a) { return f(b, a); }); },
    foldMap: function (M) {
        var foldMapWithIndexM = record.foldMapWithIndex(M);
        return function (fa, f) { return foldMapWithIndexM(fa, function (_, a) { return f(a); }); };
    },
    foldr: function (fa, b, f) { return record.foldrWithIndex(fa, b, function (_, a, b) { return f(a, b); }); },
    traverse: function (F) {
        var traverseWithIndexF = record.traverseWithIndex(F);
        return function (ta, f) { return traverseWithIndexF(ta, function (_, a) { return f(a); }); };
    },
    sequence: sequence,
    compact: function (fa) {
        var r = {};
        var keys = Object.keys(fa);
        for (var _i = 0, keys_4 = keys; _i < keys_4.length; _i++) {
            var key = keys_4[_i];
            var optionA = fa[key];
            if (isSome(optionA)) {
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
        return record.filterWithIndex(fa, function (_, a) { return predicate(a); });
    },
    filterMap: function (fa, f) { return record.filterMapWithIndex(fa, function (_, a) { return f(a); }); },
    partition: function (fa, predicate) {
        return record.partitionWithIndex(fa, function (_, a) { return predicate(a); });
    },
    partitionMap: function (fa, f) { return record.partitionMapWithIndex(fa, function (_, a) { return f(a); }); },
    wither: function (F) {
        var traverseF = record.traverse(F);
        return function (wa, f) { return F.map(traverseF(wa, f), record.compact); };
    },
    wilt: function (F) {
        var traverseF = record.traverse(F);
        return function (wa, f) { return F.map(traverseF(wa, f), record.separate); };
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
            return F.of(empty);
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
            if (isSome(optionB)) {
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
var reduceRight = pipeable(record).reduceRight;
export { reduceRight };
