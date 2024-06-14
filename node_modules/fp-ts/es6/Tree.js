import { empty, getEq as getArrayEq, array } from './Array';
import { concat, identity, toString } from './function';
import { pipeable } from './pipeable';
import { fromEquals } from './Eq';
export var URI = 'Tree';
/**
 * @since 1.6.0
 */
var Tree = /** @class */ (function () {
    function Tree(value, forest) {
        this.value = value;
        this.forest = forest;
    }
    /** @obsolete */
    Tree.prototype.map = function (f) {
        return new Tree(f(this.value), this.forest.map(function (tree) { return tree.map(f); }));
    };
    /** @obsolete */
    Tree.prototype.ap = function (fab) {
        var _this = this;
        return fab.chain(function (f) { return _this.map(f); }); // <- derived
    };
    /**
     * Flipped version of `ap`
     * @since 1.6.0
     * @obsolete
     */
    Tree.prototype.ap_ = function (fb) {
        return fb.ap(this);
    };
    /** @obsolete */
    Tree.prototype.chain = function (f) {
        var _a = f(this.value), value = _a.value, forest = _a.forest;
        // tslint:disable-next-line: deprecation
        return new Tree(value, concat(forest, this.forest.map(function (t) { return t.chain(f); })));
    };
    /** @obsolete */
    Tree.prototype.extract = function () {
        return this.value;
    };
    /** @obsolete */
    Tree.prototype.extend = function (f) {
        return new Tree(f(this), this.forest.map(function (t) { return t.extend(f); }));
    };
    /** @obsolete */
    Tree.prototype.reduce = function (b, f) {
        var r = f(b, this.value);
        var len = this.forest.length;
        for (var i = 0; i < len; i++) {
            r = this.forest[i].reduce(r, f);
        }
        return r;
    };
    Tree.prototype.inspect = function () {
        return this.toString();
    };
    Tree.prototype.toString = function () {
        return this.forest === empty || this.forest.length === 0
            ? // tslint:disable-next-line: deprecation
                "make(" + toString(this.value) + ")"
            : // tslint:disable-next-line: deprecation
                "make(" + toString(this.value) + ", " + toString(this.forest) + ")";
    };
    return Tree;
}());
export { Tree };
/**
 * @since 1.17.0
 */
export function getShow(S) {
    var show = function (t) {
        return t.forest === empty || t.forest.length === 0
            ? "make(" + S.show(t.value) + ")"
            : "make(" + S.show(t.value) + ", [" + t.forest.map(show).join(', ') + "])";
    };
    return {
        show: show
    };
}
var foldr = function (fa, b, f) {
    var r = b;
    var len = fa.forest.length;
    for (var i = len - 1; i >= 0; i--) {
        r = foldr(fa.forest[i], r, f);
    }
    return f(fa.value, r);
};
function traverse(F) {
    var traverseF = array.traverse(F);
    var r = function (ta, f) {
        return F.ap(F.map(f(ta.value), function (value) { return function (forest) { return new Tree(value, forest); }; }), traverseF(ta.forest, function (t) { return r(t, f); }));
    };
    return r;
}
function sequence(F) {
    var traverseF = traverse(F);
    return function (ta) { return traverseF(ta, identity); };
}
/**
 * Use `getEq`
 *
 * @since 1.6.0
 * @deprecated
 */
export var getSetoid = getEq;
/**
 * @since 1.19.0
 */
export function getEq(E) {
    var SA;
    var R = fromEquals(function (x, y) { return E.equals(x.value, y.value) && SA.equals(x.forest, y.forest); });
    SA = getArrayEq(R);
    return R;
}
/**
 * @since 1.6.0
 */
export var tree = {
    URI: URI,
    map: function (fa, f) { return fa.map(f); },
    of: function (a) { return make(a); },
    ap: function (fab, fa) { return fa.ap(fab); },
    chain: function (fa, f) { return fa.chain(f); },
    reduce: function (fa, b, f) { return fa.reduce(b, f); },
    foldMap: function (M) { return function (fa, f) { return fa.reduce(M.empty, function (acc, a) { return M.concat(acc, f(a)); }); }; },
    foldr: foldr,
    traverse: traverse,
    sequence: sequence,
    extract: function (wa) { return wa.extract(); },
    extend: function (wa, f) { return wa.extend(f); }
};
var draw = function (indentation, forest) {
    var r = '';
    var len = forest.length;
    var tree;
    for (var i = 0; i < len; i++) {
        tree = forest[i];
        var isLast = i === len - 1;
        r += indentation + (isLast ? '└' : '├') + '─ ' + tree.value;
        r += draw(indentation + (len > 1 && !isLast ? '│  ' : '   '), tree.forest);
    }
    return r;
};
/**
 * Neat 2-dimensional drawing of a forest
 *
 * @since 1.6.0
 */
export var drawForest = function (forest) {
    return draw('\n', forest);
};
/**
 * Neat 2-dimensional drawing of a tree
 *
 * @example
 * import { Tree, drawTree, tree } from 'fp-ts/lib/Tree'
 *
 * const fa = new Tree('a', [
 *   tree.of('b'),
 *   tree.of('c'),
 *   new Tree('d', [tree.of('e'), tree.of('f')])
 * ])
 *
 * assert.strictEqual(drawTree(fa), `a
 * ├─ b
 * ├─ c
 * └─ d
 *    ├─ e
 *    └─ f`)
 *
 *
 * @since 1.6.0
 */
export var drawTree = function (tree) {
    return tree.value + drawForest(tree.forest);
};
/**
 * Build a tree from a seed value
 *
 * @since 1.6.0
 */
export var unfoldTree = function (b, f) {
    var _a = f(b), a = _a[0], bs = _a[1];
    return new Tree(a, unfoldForest(bs, f));
};
/**
 * Build a tree from a seed value
 *
 * @since 1.6.0
 */
export var unfoldForest = function (bs, f) {
    return bs.map(function (b) { return unfoldTree(b, f); });
};
export function unfoldTreeM(M) {
    var unfoldForestMM = unfoldForestM(M);
    return function (b, f) { return M.chain(f(b), function (_a) {
        var a = _a[0], bs = _a[1];
        return M.chain(unfoldForestMM(bs, f), function (ts) { return M.of(new Tree(a, ts)); });
    }); };
}
export function unfoldForestM(M) {
    var traverseM = array.traverse(M);
    var unfoldTree;
    return function (bs, f) {
        // tslint:disable-next-line
        if (unfoldTree === undefined) {
            unfoldTree = unfoldTreeM(M);
        }
        return traverseM(bs, function (b) { return unfoldTree(b, f); });
    };
}
/**
 * @since 1.14.0
 */
export function elem(E) {
    var go = function (a, fa) {
        if (E.equals(a, fa.value)) {
            return true;
        }
        return fa.forest.some(function (tree) { return go(a, tree); });
    };
    return go;
}
//
// backporting
//
/**
 * @since 1.19.0
 */
export function make(a, forest) {
    if (forest === void 0) { forest = empty; }
    return new Tree(a, forest);
}
var _a = pipeable(tree), ap = _a.ap, apFirst = _a.apFirst, apSecond = _a.apSecond, chain = _a.chain, chainFirst = _a.chainFirst, duplicate = _a.duplicate, extend = _a.extend, flatten = _a.flatten, foldMap = _a.foldMap, map = _a.map, reduce = _a.reduce, reduceRight = _a.reduceRight;
export { ap, apFirst, apSecond, chain, chainFirst, duplicate, extend, flatten, foldMap, map, reduce, reduceRight };
