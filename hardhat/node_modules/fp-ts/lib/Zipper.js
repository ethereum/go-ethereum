"use strict";
var __assign = (this && this.__assign) || function () {
    __assign = Object.assign || function(t) {
        for (var s, i = 1, n = arguments.length; i < n; i++) {
            s = arguments[i];
            for (var p in s) if (Object.prototype.hasOwnProperty.call(s, p))
                t[p] = s[p];
        }
        return t;
    };
    return __assign.apply(this, arguments);
};
Object.defineProperty(exports, "__esModule", { value: true });
var Array_1 = require("./Array");
var function_1 = require("./function");
var Option_1 = require("./Option");
exports.URI = 'Zipper';
/**
 * @since 1.9.0
 */
var Zipper = /** @class */ (function () {
    function Zipper(lefts, focus, rights) {
        this.lefts = lefts;
        this.focus = focus;
        this.rights = rights;
        this.length = lefts.length + 1 + rights.length;
    }
    /**
     * Update the focus in this zipper.
     * @since 1.9.0
     */
    Zipper.prototype.update = function (a) {
        return new Zipper(this.lefts, a, this.rights);
    };
    /**
     * Apply `f` to the focus and update with the result.
     * @since 1.9.0
     */
    Zipper.prototype.modify = function (f) {
        return this.update(f(this.focus));
    };
    /**
     * @since 1.9.0
     */
    Zipper.prototype.toArray = function () {
        return Array_1.snoc(this.lefts, this.focus).concat(this.rights);
    };
    /**
     * @since 1.9.0
     */
    Zipper.prototype.isOutOfBound = function (index) {
        return index < 0 || index >= this.length;
    };
    /**
     * Moves focus in the zipper, or `None` if there is no such element.
     * @since 1.9.0
     */
    Zipper.prototype.move = function (f) {
        var newIndex = f(this.lefts.length);
        if (this.isOutOfBound(newIndex)) {
            return Option_1.none;
        }
        else {
            return exports.fromArray(this.toArray(), newIndex);
        }
    };
    /**
     * @since 1.9.0
     */
    Zipper.prototype.up = function () {
        return this.move(function_1.decrement);
    };
    /**
     * @since 1.9.0
     */
    Zipper.prototype.down = function () {
        return this.move(function_1.increment);
    };
    /**
     * Moves focus to the start of the zipper.
     * @since 1.9.0
     */
    Zipper.prototype.start = function () {
        if (Array_1.isEmpty(this.lefts)) {
            return this;
        }
        else {
            // tslint:disable-next-line: deprecation
            return new Zipper(Array_1.empty, this.lefts[0], Array_1.snoc(Array_1.drop(1, this.lefts), this.focus).concat(this.rights));
        }
    };
    /**
     * Moves focus to the end of the zipper.
     * @since 1.9.0
     */
    Zipper.prototype.end = function () {
        var len = this.rights.length;
        if (len === 0) {
            return this;
        }
        else {
            // tslint:disable-next-line: deprecation
            return new Zipper(Array_1.snoc(this.lefts, this.focus).concat(Array_1.take(len - 1, this.rights)), this.rights[len - 1], Array_1.empty);
        }
    };
    /**
     * Inserts an element to the left of focus and focuses on the new element.
     * @since 1.9.0
     */
    Zipper.prototype.insertLeft = function (a) {
        return new Zipper(this.lefts, a, Array_1.cons(this.focus, this.rights));
    };
    /**
     * Inserts an element to the right of focus and focuses on the new element.
     * @since 1.9.0
     */
    Zipper.prototype.insertRight = function (a) {
        return new Zipper(Array_1.snoc(this.lefts, this.focus), a, this.rights);
    };
    /**
     * Deletes the element at focus and moves the focus to the left. If there is no element on the left,
     * focus is moved to the right.
     * @since 1.9.0
     */
    Zipper.prototype.deleteLeft = function () {
        var len = this.lefts.length;
        return exports.fromArray(this.lefts.concat(this.rights), len > 0 ? len - 1 : 0);
    };
    /**
     * Deletes the element at focus and moves the focus to the right. If there is no element on the right,
     * focus is moved to the left.
     * @since 1.9.0
     */
    Zipper.prototype.deleteRight = function () {
        var lenl = this.lefts.length;
        var lenr = this.rights.length;
        return exports.fromArray(this.lefts.concat(this.rights), lenr > 0 ? lenl : lenl - 1);
    };
    /**
     * @since 1.9.0
     */
    Zipper.prototype.map = function (f) {
        return new Zipper(this.lefts.map(f), f(this.focus), this.rights.map(f));
    };
    /**
     * @since 1.9.0
     */
    Zipper.prototype.ap = function (fab) {
        return new Zipper(Array_1.array.ap(fab.lefts, this.lefts), fab.focus(this.focus), Array_1.array.ap(fab.rights, this.rights));
    };
    /**
     * @since 1.9.0
     */
    Zipper.prototype.reduce = function (b, f) {
        return this.rights.reduce(f, f(this.lefts.reduce(f, b), this.focus));
    };
    Zipper.prototype.inspect = function () {
        return this.toString();
    };
    Zipper.prototype.toString = function () {
        // tslint:disable-next-line: deprecation
        return "new Zipper(" + function_1.toString(this.lefts) + ", " + function_1.toString(this.focus) + ", " + function_1.toString(this.rights) + ")";
    };
    return Zipper;
}());
exports.Zipper = Zipper;
/**
 * @since 1.17.0
 */
exports.getShow = function (S) {
    var SA = Array_1.getShow(S);
    return {
        show: function (z) { return "new Zipper(" + SA.show(z.lefts) + ", " + S.show(z.focus) + ", " + SA.show(z.rights) + ")"; }
    };
};
/**
 * @since 1.9.0
 */
exports.fromArray = function (as, focusIndex) {
    if (focusIndex === void 0) { focusIndex = 0; }
    if (Array_1.isEmpty(as) || Array_1.isOutOfBound(focusIndex, as)) {
        return Option_1.none;
    }
    else {
        // tslint:disable-next-line: deprecation
        return Option_1.some(new Zipper(Array_1.take(focusIndex, as), as[focusIndex], Array_1.drop(focusIndex + 1, as)));
    }
};
/**
 * @since 1.9.0
 */
exports.fromNonEmptyArray = function (nea) {
    return new Zipper(Array_1.empty, nea.head, nea.tail);
};
/**
 * @since 1.17.0
 */
exports.fromNonEmptyArray2v = function (nea) {
    return new Zipper(Array_1.empty, nea[0], nea.slice(1));
};
var map = function (fa, f) {
    return fa.map(f);
};
var of = function (a) {
    return new Zipper(Array_1.empty, a, Array_1.empty);
};
var ap = function (fab, fa) {
    return fa.ap(fab);
};
var reduce = function (fa, b, f) {
    return fa.reduce(b, f);
};
var foldMap = function (M) { return function (fa, f) {
    var lefts = fa.lefts.reduce(function (acc, a) { return M.concat(acc, f(a)); }, M.empty);
    var rights = fa.rights.reduce(function (acc, a) { return M.concat(acc, f(a)); }, M.empty);
    return M.concat(M.concat(lefts, f(fa.focus)), rights);
}; };
var foldr = function (fa, b, f) {
    var rights = fa.rights.reduceRight(function (acc, a) { return f(a, acc); }, b);
    var focus = f(fa.focus, rights);
    return fa.lefts.reduceRight(function (acc, a) { return f(a, acc); }, focus);
};
function traverse(F) {
    var traverseF = Array_1.array.traverse(F);
    return function (ta, f) {
        return F.ap(F.ap(F.map(traverseF(ta.lefts, f), function (lefts) { return function (focus) { return function (rights) { return new Zipper(lefts, focus, rights); }; }; }), f(ta.focus)), traverseF(ta.rights, f));
    };
}
function sequence(F) {
    var sequenceF = Array_1.array.sequence(F);
    return function (ta) {
        return F.ap(F.ap(F.map(sequenceF(ta.lefts), function (lefts) { return function (focus) { return function (rights) { return new Zipper(lefts, focus, rights); }; }; }), ta.focus), sequenceF(ta.rights));
    };
}
var extract = function (fa) {
    return fa.focus;
};
var extend = function (fa, f) {
    var lefts = fa.lefts.map(function (a, i) {
        // tslint:disable-next-line: deprecation
        return f(new Zipper(Array_1.take(i, fa.lefts), a, Array_1.snoc(Array_1.drop(i + 1, fa.lefts), fa.focus).concat(fa.rights)));
    });
    var rights = fa.rights.map(function (a, i) {
        // tslint:disable-next-line: deprecation
        return f(new Zipper(Array_1.snoc(fa.lefts, fa.focus).concat(Array_1.take(i, fa.rights)), a, Array_1.drop(i + 1, fa.rights)));
    });
    return new Zipper(lefts, f(fa), rights);
};
/**
 * @since 1.9.0
 */
exports.getSemigroup = function (S) {
    return {
        concat: function (x, y) { return new Zipper(x.lefts.concat(y.lefts), S.concat(x.focus, y.focus), x.rights.concat(y.rights)); }
    };
};
/**
 * @since 1.9.0
 */
exports.getMonoid = function (M) {
    return __assign({}, exports.getSemigroup(M), { empty: new Zipper(Array_1.empty, M.empty, Array_1.empty) });
};
/**
 * @since 1.9.0
 */
exports.zipper = {
    URI: exports.URI,
    map: map,
    of: of,
    ap: ap,
    extend: extend,
    extract: extract,
    reduce: reduce,
    foldMap: foldMap,
    foldr: foldr,
    traverse: traverse,
    sequence: sequence
};
