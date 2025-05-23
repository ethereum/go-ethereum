"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var Eq_1 = require("./Eq");
var function_1 = require("./function");
/**
 * @since 1.17.0
 */
exports.getShow = function (SA) {
    return {
        show: function (s) {
            var elements = '';
            s.forEach(function (a) {
                elements += SA.show(a) + ', ';
            });
            if (elements !== '') {
                elements = elements.substring(0, elements.length - 2);
            }
            return "new Set([" + elements + "])";
        }
    };
};
/**
 * @since 1.14.0
 */
exports.empty = new Set();
/**
 * @since 1.0.0
 */
exports.toArray = function (O) { return function (x) {
    var r = [];
    x.forEach(function (e) { return r.push(e); });
    return r.sort(O.compare);
}; };
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
    var subsetE = exports.subset(E);
    return Eq_1.fromEquals(function (x, y) { return subsetE(x, y) && subsetE(y, x); });
}
exports.getEq = getEq;
/**
 * @since 1.0.0
 */
exports.some = function (x, predicate) {
    var values = x.values();
    var e;
    var found = false;
    while (!found && !(e = values.next()).done) {
        found = predicate(e.value);
    }
    return found;
};
/**
 * Projects a Set through a function
 *
 * @since 1.2.0
 */
exports.map = function (E) {
    var has = exports.elem(E);
    return function (set, f) {
        var r = new Set();
        set.forEach(function (e) {
            var v = f(e);
            if (!has(v, r)) {
                r.add(v);
            }
        });
        return r;
    };
};
/**
 * @since 1.0.0
 */
exports.every = function (x, predicate) {
    return !exports.some(x, function_1.not(predicate));
};
/**
 * @since 1.2.0
 */
exports.chain = function (E) {
    var has = exports.elem(E);
    return function (set, f) {
        var r = new Set();
        set.forEach(function (e) {
            f(e).forEach(function (e) {
                if (!has(e, r)) {
                    r.add(e);
                }
            });
        });
        return r;
    };
};
/**
 * `true` if and only if every element in the first set is an element of the second set
 *
 * @since 1.0.0
 */
exports.subset = function (E) {
    var has = exports.elem(E);
    return function (x, y) { return exports.every(x, function (a) { return has(a, y); }); };
};
function filter(x, predicate) {
    var values = x.values();
    var e;
    var r = new Set();
    while (!(e = values.next()).done) {
        var value = e.value;
        if (predicate(value)) {
            r.add(value);
        }
    }
    return r;
}
exports.filter = filter;
function partition(x, predicate) {
    var values = x.values();
    var e;
    var right = new Set();
    var left = new Set();
    while (!(e = values.next()).done) {
        var value = e.value;
        if (predicate(value)) {
            right.add(value);
        }
        else {
            left.add(value);
        }
    }
    return { left: left, right: right };
}
exports.partition = partition;
/**
 * Use `elem` instead
 * @since 1.0.0
 * @deprecated
 */
exports.member = function (E) {
    var has = exports.elem(E);
    return function (set) { return function (a) { return has(a, set); }; };
};
/**
 * Test if a value is a member of a set
 *
 * @since 1.14.0
 */
exports.elem = function (E) { return function (a, x) {
    return exports.some(x, function (ax) { return E.equals(a, ax); });
}; };
/**
 * Form the union of two sets
 *
 * @since 1.0.0
 */
exports.union = function (E) {
    var has = exports.elem(E);
    return function (x, y) {
        var r = new Set(x);
        y.forEach(function (e) {
            if (!has(e, r)) {
                r.add(e);
            }
        });
        return r;
    };
};
/**
 * The set of elements which are in both the first and second set
 *
 * @since 1.0.0
 */
exports.intersection = function (E) {
    var has = exports.elem(E);
    return function (x, y) {
        var r = new Set();
        x.forEach(function (e) {
            if (has(e, y)) {
                r.add(e);
            }
        });
        return r;
    };
};
/**
 * @since 1.2.0
 */
exports.partitionMap = function (EL, ER) { return function (x, f) {
    var values = x.values();
    var e;
    var left = new Set();
    var right = new Set();
    var hasL = exports.elem(EL);
    var hasR = exports.elem(ER);
    while (!(e = values.next()).done) {
        var v = f(e.value);
        if (v.isLeft()) {
            if (!hasL(v.value, left)) {
                left.add(v.value);
            }
        }
        else {
            if (!hasR(v.value, right)) {
                right.add(v.value);
            }
        }
    }
    return { left: left, right: right };
}; };
/**
 * Use `difference2v` instead
 *
 * @since 1.0.0
 * @deprecated
 */
exports.difference = function (E) {
    var d = exports.difference2v(E);
    return function (x, y) { return d(y, x); };
};
/**
 * Form the set difference (`x` - `y`)
 *
 * @example
 * import { difference2v } from 'fp-ts/lib/Set'
 * import { eqNumber } from 'fp-ts/lib/Eq'
 *
 * assert.deepStrictEqual(difference2v(eqNumber)(new Set([1, 2]), new Set([1, 3])), new Set([2]))
 *
 *
 * @since 1.12.0
 */
exports.difference2v = function (E) {
    var has = exports.elem(E);
    return function (x, y) { return filter(x, function (a) { return !has(a, y); }); };
};
/**
 * @since 1.0.0
 */
exports.getUnionMonoid = function (E) {
    return {
        concat: exports.union(E),
        empty: exports.empty
    };
};
/**
 * @since 1.0.0
 */
exports.getIntersectionSemigroup = function (E) {
    return {
        concat: exports.intersection(E)
    };
};
/**
 * @since 1.0.0
 */
exports.reduce = function (O) {
    var toArrayO = exports.toArray(O);
    return function (fa, b, f) { return toArrayO(fa).reduce(f, b); };
};
/**
 * @since 1.14.0
 */
exports.foldMap = function (O, M) {
    var toArrayO = exports.toArray(O);
    return function (fa, f) { return toArrayO(fa).reduce(function (b, a) { return M.concat(b, f(a)); }, M.empty); };
};
/**
 * Create a set with one element
 *
 * @since 1.0.0
 */
exports.singleton = function (a) {
    return new Set([a]);
};
/**
 * Insert a value into a set
 *
 * @since 1.0.0
 */
exports.insert = function (E) {
    var has = exports.elem(E);
    return function (a, x) {
        if (!has(a, x)) {
            var r = new Set(x);
            r.add(a);
            return r;
        }
        else {
            return x;
        }
    };
};
/**
 * Delete a value from a set
 *
 * @since 1.0.0
 */
exports.remove = function (E) { return function (a, x) {
    return filter(x, function (ax) { return !E.equals(a, ax); });
}; };
/**
 * Create a set from an array
 *
 * @since 1.2.0
 */
exports.fromArray = function (E) { return function (as) {
    var len = as.length;
    var r = new Set();
    var has = exports.elem(E);
    for (var i = 0; i < len; i++) {
        var a = as[i];
        if (!has(a, r)) {
            r.add(a);
        }
    }
    return r;
}; };
/**
 * @since 1.12.0
 */
exports.compact = function (E) {
    var filterMapE = exports.filterMap(E);
    return function (fa) { return filterMapE(fa, function_1.identity); };
};
/**
 * @since 1.12.0
 */
exports.separate = function (EL, ER) { return function (fa) {
    var hasL = exports.elem(EL);
    var hasR = exports.elem(ER);
    var left = new Set();
    var right = new Set();
    fa.forEach(function (e) {
        if (e.isLeft()) {
            if (!hasL(e.value, left)) {
                left.add(e.value);
            }
        }
        else {
            if (!hasR(e.value, right)) {
                right.add(e.value);
            }
        }
    });
    return { left: left, right: right };
}; };
/**
 * @since 1.12.0
 */
exports.filterMap = function (E) {
    var has = exports.elem(E);
    return function (fa, f) {
        var r = new Set();
        fa.forEach(function (a) {
            var ob = f(a);
            if (ob.isSome() && !has(ob.value, r)) {
                r.add(ob.value);
            }
        });
        return r;
    };
};
