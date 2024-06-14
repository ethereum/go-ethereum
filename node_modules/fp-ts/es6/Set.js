import { fromEquals } from './Eq';
import { not, identity } from './function';
/**
 * @since 1.17.0
 */
export var getShow = function (SA) {
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
export var empty = new Set();
/**
 * @since 1.0.0
 */
export var toArray = function (O) { return function (x) {
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
export var getSetoid = getEq;
/**
 * @since 1.19.0
 */
export function getEq(E) {
    var subsetE = subset(E);
    return fromEquals(function (x, y) { return subsetE(x, y) && subsetE(y, x); });
}
/**
 * @since 1.0.0
 */
export var some = function (x, predicate) {
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
export var map = function (E) {
    var has = elem(E);
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
export var every = function (x, predicate) {
    return !some(x, not(predicate));
};
/**
 * @since 1.2.0
 */
export var chain = function (E) {
    var has = elem(E);
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
export var subset = function (E) {
    var has = elem(E);
    return function (x, y) { return every(x, function (a) { return has(a, y); }); };
};
export function filter(x, predicate) {
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
export function partition(x, predicate) {
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
/**
 * Use `elem` instead
 * @since 1.0.0
 * @deprecated
 */
export var member = function (E) {
    var has = elem(E);
    return function (set) { return function (a) { return has(a, set); }; };
};
/**
 * Test if a value is a member of a set
 *
 * @since 1.14.0
 */
export var elem = function (E) { return function (a, x) {
    return some(x, function (ax) { return E.equals(a, ax); });
}; };
/**
 * Form the union of two sets
 *
 * @since 1.0.0
 */
export var union = function (E) {
    var has = elem(E);
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
export var intersection = function (E) {
    var has = elem(E);
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
export var partitionMap = function (EL, ER) { return function (x, f) {
    var values = x.values();
    var e;
    var left = new Set();
    var right = new Set();
    var hasL = elem(EL);
    var hasR = elem(ER);
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
export var difference = function (E) {
    var d = difference2v(E);
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
export var difference2v = function (E) {
    var has = elem(E);
    return function (x, y) { return filter(x, function (a) { return !has(a, y); }); };
};
/**
 * @since 1.0.0
 */
export var getUnionMonoid = function (E) {
    return {
        concat: union(E),
        empty: empty
    };
};
/**
 * @since 1.0.0
 */
export var getIntersectionSemigroup = function (E) {
    return {
        concat: intersection(E)
    };
};
/**
 * @since 1.0.0
 */
export var reduce = function (O) {
    var toArrayO = toArray(O);
    return function (fa, b, f) { return toArrayO(fa).reduce(f, b); };
};
/**
 * @since 1.14.0
 */
export var foldMap = function (O, M) {
    var toArrayO = toArray(O);
    return function (fa, f) { return toArrayO(fa).reduce(function (b, a) { return M.concat(b, f(a)); }, M.empty); };
};
/**
 * Create a set with one element
 *
 * @since 1.0.0
 */
export var singleton = function (a) {
    return new Set([a]);
};
/**
 * Insert a value into a set
 *
 * @since 1.0.0
 */
export var insert = function (E) {
    var has = elem(E);
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
export var remove = function (E) { return function (a, x) {
    return filter(x, function (ax) { return !E.equals(a, ax); });
}; };
/**
 * Create a set from an array
 *
 * @since 1.2.0
 */
export var fromArray = function (E) { return function (as) {
    var len = as.length;
    var r = new Set();
    var has = elem(E);
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
export var compact = function (E) {
    var filterMapE = filterMap(E);
    return function (fa) { return filterMapE(fa, identity); };
};
/**
 * @since 1.12.0
 */
export var separate = function (EL, ER) { return function (fa) {
    var hasL = elem(EL);
    var hasR = elem(ER);
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
export var filterMap = function (E) {
    var has = elem(E);
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
