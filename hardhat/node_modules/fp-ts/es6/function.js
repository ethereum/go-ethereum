/**
 * @since 1.0.0
 */
export var identity = function (a) {
    return a;
};
/**
 * @since 1.0.0
 */
export var unsafeCoerce = identity;
/**
 * @since 1.0.0
 */
export var not = function (predicate) {
    return function (a) { return !predicate(a); };
};
export function or(p1, p2) {
    return function (a) { return p1(a) || p2(a); };
}
/**
 * @since 1.0.0
 * @deprecated
 */
export var and = function (p1, p2) {
    return function (a) { return p1(a) && p2(a); };
};
/**
 * @since 1.0.0
 */
export var constant = function (a) {
    return function () { return a; };
};
/**
 * A thunk that returns always `true`
 *
 * @since 1.0.0
 */
export var constTrue = function () {
    return true;
};
/**
 * A thunk that returns always `false`
 *
 * @since 1.0.0
 */
export var constFalse = function () {
    return false;
};
/**
 * A thunk that returns always `null`
 *
 * @since 1.0.0
 */
export var constNull = function () {
    return null;
};
/**
 * A thunk that returns always `undefined`
 *
 * @since 1.0.0
 */
export var constUndefined = function () {
    return;
};
/**
 * A thunk that returns always `void`
 *
 * @since 1.14.0
 */
export var constVoid = function () {
    return;
};
/**
 * Flips the order of the arguments to a function of two arguments.
 *
 * @since 1.0.0
 */
// tslint:disable-next-line: deprecation
export var flip = function (f) {
    return function (b) { return function (a) { return f(a)(b); }; };
};
/**
 * The `on` function is used to change the domain of a binary operator.
 *
 * @since 1.0.0
 * @deprecated
 */
// tslint:disable-next-line: deprecation
export var on = function (op) { return function (f) {
    return function (x, y) { return op(f(x), f(y)); };
}; };
export function compose() {
    var fns = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        fns[_i] = arguments[_i];
    }
    var len = fns.length - 1;
    return function (x) {
        var y = x;
        for (var i = len; i > -1; i--) {
            y = fns[i].call(this, y);
        }
        return y;
    };
}
export function pipe() {
    var fns = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        fns[_i] = arguments[_i];
    }
    var len = fns.length - 1;
    return function (x) {
        var y = x;
        for (var i = 0; i <= len; i++) {
            y = fns[i].call(this, y);
        }
        return y;
    };
}
/**
 * @since 1.0.0
 * @deprecated
 */
export var concat = function (x, y) {
    var lenx = x.length;
    if (lenx === 0) {
        return y;
    }
    var leny = y.length;
    if (leny === 0) {
        return x;
    }
    var r = Array(lenx + leny);
    for (var i = 0; i < lenx; i++) {
        r[i] = x[i];
    }
    for (var i = 0; i < leny; i++) {
        r[i + lenx] = y[i];
    }
    return r;
};
/**
 * @since 1.0.0
 * @deprecated
 */
export function curried(f, n, acc) {
    return function (x) {
        // tslint:disable-next-line: deprecation
        var combined = concat(acc, [x]);
        // tslint:disable-next-line: deprecation
        return n === 0 ? f.apply(this, combined) : curried(f, n - 1, combined);
    };
}
export function curry(f) {
    // tslint:disable-next-line: deprecation
    return curried(f, f.length - 1, []);
}
/* tslint:disable-next-line */
var getFunctionName = function (f) { return f.displayName || f.name || "<function" + f.length + ">"; };
/**
 * @since 1.0.0
 * @deprecated
 */
export var toString = function (x) {
    if (typeof x === 'string') {
        return JSON.stringify(x);
    }
    if (x instanceof Date) {
        return "new Date('" + x.toISOString() + "')";
    }
    if (Array.isArray(x)) {
        // tslint:disable-next-line: deprecation
        return "[" + x.map(toString).join(', ') + "]";
    }
    if (typeof x === 'function') {
        return getFunctionName(x);
    }
    if (x == null) {
        return String(x);
    }
    if (typeof x.toString === 'function' && x.toString !== Object.prototype.toString) {
        return x.toString();
    }
    try {
        return JSON.stringify(x, null, 2);
    }
    catch (e) {
        return String(x);
    }
};
/**
 * @since 1.0.0
 */
export var tuple = function () {
    var t = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        t[_i] = arguments[_i];
    }
    return t;
};
/**
 * @since 1.0.0
 * @deprecated
 */
export var tupleCurried = function (a) { return function (b) {
    return [a, b];
}; };
/**
 * Applies a function to an argument ($)
 *
 * @since 1.0.0
 * @deprecated
 */
export var apply = function (f) { return function (a) {
    return f(a);
}; };
/**
 * Applies an argument to a function (#)
 *
 * @since 1.0.0
 * @deprecated
 */
export var applyFlipped = function (a) { return function (f) {
    return f(a);
}; };
/**
 * For use with phantom fields
 *
 * @since 1.0.0
 * @deprecated
 */
export var phantom = undefined;
/**
 * A thunk that returns always the `identity` function.
 * For use with `applySecond` methods.
 *
 * @since 1.5.0
 * @deprecated
 */
export var constIdentity = function () {
    return identity;
};
/**
 * @since 1.9.0
 */
export var increment = function (n) {
    return n + 1;
};
/**
 * @since 1.9.0
 */
export var decrement = function (n) {
    return n - 1;
};
/**
 * @since 1.18.0
 */
export function absurd(_) {
    throw new Error('Called `absurd` function which should be uncallable');
}
export function flow(ab, bc, cd, de, ef, fg, gh, hi, ij) {
    switch (arguments.length) {
        case 1:
            return ab;
        case 2:
            return function () {
                return bc(ab.apply(this, arguments));
            };
        case 3:
            return function () {
                return cd(bc(ab.apply(this, arguments)));
            };
        case 4:
            return function () {
                return de(cd(bc(ab.apply(this, arguments))));
            };
        case 5:
            return function () {
                return ef(de(cd(bc(ab.apply(this, arguments)))));
            };
        case 6:
            return function () {
                return fg(ef(de(cd(bc(ab.apply(this, arguments))))));
            };
        case 7:
            return function () {
                return gh(fg(ef(de(cd(bc(ab.apply(this, arguments)))))));
            };
        case 8:
            return function () {
                return hi(gh(fg(ef(de(cd(bc(ab.apply(this, arguments))))))));
            };
        case 9:
            return function () {
                return ij(hi(gh(fg(ef(de(cd(bc(ab.apply(this, arguments)))))))));
            };
    }
}
