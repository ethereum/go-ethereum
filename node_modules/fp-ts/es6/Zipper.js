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
import { array, cons, drop, empty, isEmpty, isOutOfBound, snoc, take, getShow as getArrayShow } from './Array';
import { decrement, increment, toString } from './function';
import { none, some } from './Option';
export var URI = 'Zipper';
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
        return snoc(this.lefts, this.focus).concat(this.rights);
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
            return none;
        }
        else {
            return fromArray(this.toArray(), newIndex);
        }
    };
    /**
     * @since 1.9.0
     */
    Zipper.prototype.up = function () {
        return this.move(decrement);
    };
    /**
     * @since 1.9.0
     */
    Zipper.prototype.down = function () {
        return this.move(increment);
    };
    /**
     * Moves focus to the start of the zipper.
     * @since 1.9.0
     */
    Zipper.prototype.start = function () {
        if (isEmpty(this.lefts)) {
            return this;
        }
        else {
            // tslint:disable-next-line: deprecation
            return new Zipper(empty, this.lefts[0], snoc(drop(1, this.lefts), this.focus).concat(this.rights));
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
            return new Zipper(snoc(this.lefts, this.focus).concat(take(len - 1, this.rights)), this.rights[len - 1], empty);
        }
    };
    /**
     * Inserts an element to the left of focus and focuses on the new element.
     * @since 1.9.0
     */
    Zipper.prototype.insertLeft = function (a) {
        return new Zipper(this.lefts, a, cons(this.focus, this.rights));
    };
    /**
     * Inserts an element to the right of focus and focuses on the new element.
     * @since 1.9.0
     */
    Zipper.prototype.insertRight = function (a) {
        return new Zipper(snoc(this.lefts, this.focus), a, this.rights);
    };
    /**
     * Deletes the element at focus and moves the focus to the left. If there is no element on the left,
     * focus is moved to the right.
     * @since 1.9.0
     */
    Zipper.prototype.deleteLeft = function () {
        var len = this.lefts.length;
        return fromArray(this.lefts.concat(this.rights), len > 0 ? len - 1 : 0);
    };
    /**
     * Deletes the element at focus and moves the focus to the right. If there is no element on the right,
     * focus is moved to the left.
     * @since 1.9.0
     */
    Zipper.prototype.deleteRight = function () {
        var lenl = this.lefts.length;
        var lenr = this.rights.length;
        return fromArray(this.lefts.concat(this.rights), lenr > 0 ? lenl : lenl - 1);
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
        return new Zipper(array.ap(fab.lefts, this.lefts), fab.focus(this.focus), array.ap(fab.rights, this.rights));
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
        return "new Zipper(" + toString(this.lefts) + ", " + toString(this.focus) + ", " + toString(this.rights) + ")";
    };
    return Zipper;
}());
export { Zipper };
/**
 * @since 1.17.0
 */
export var getShow = function (S) {
    var SA = getArrayShow(S);
    return {
        show: function (z) { return "new Zipper(" + SA.show(z.lefts) + ", " + S.show(z.focus) + ", " + SA.show(z.rights) + ")"; }
    };
};
/**
 * @since 1.9.0
 */
export var fromArray = function (as, focusIndex) {
    if (focusIndex === void 0) { focusIndex = 0; }
    if (isEmpty(as) || isOutOfBound(focusIndex, as)) {
        return none;
    }
    else {
        // tslint:disable-next-line: deprecation
        return some(new Zipper(take(focusIndex, as), as[focusIndex], drop(focusIndex + 1, as)));
    }
};
/**
 * @since 1.9.0
 */
export var fromNonEmptyArray = function (nea) {
    return new Zipper(empty, nea.head, nea.tail);
};
/**
 * @since 1.17.0
 */
export var fromNonEmptyArray2v = function (nea) {
    return new Zipper(empty, nea[0], nea.slice(1));
};
var map = function (fa, f) {
    return fa.map(f);
};
var of = function (a) {
    return new Zipper(empty, a, empty);
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
    var traverseF = array.traverse(F);
    return function (ta, f) {
        return F.ap(F.ap(F.map(traverseF(ta.lefts, f), function (lefts) { return function (focus) { return function (rights) { return new Zipper(lefts, focus, rights); }; }; }), f(ta.focus)), traverseF(ta.rights, f));
    };
}
function sequence(F) {
    var sequenceF = array.sequence(F);
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
        return f(new Zipper(take(i, fa.lefts), a, snoc(drop(i + 1, fa.lefts), fa.focus).concat(fa.rights)));
    });
    var rights = fa.rights.map(function (a, i) {
        // tslint:disable-next-line: deprecation
        return f(new Zipper(snoc(fa.lefts, fa.focus).concat(take(i, fa.rights)), a, drop(i + 1, fa.rights)));
    });
    return new Zipper(lefts, f(fa), rights);
};
/**
 * @since 1.9.0
 */
export var getSemigroup = function (S) {
    return {
        concat: function (x, y) { return new Zipper(x.lefts.concat(y.lefts), S.concat(x.focus, y.focus), x.rights.concat(y.rights)); }
    };
};
/**
 * @since 1.9.0
 */
export var getMonoid = function (M) {
    return __assign({}, getSemigroup(M), { empty: new Zipper(empty, M.empty, empty) });
};
/**
 * @since 1.9.0
 */
export var zipper = {
    URI: URI,
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
