"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var Array_1 = require("./Array");
var function_1 = require("./function");
var pipeable_1 = require("./pipeable");
var Eq_1 = require("./Eq");
exports.URI = 'Tree';
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
        return new Tree(value, function_1.concat(forest, this.forest.map(function (t) { return t.chain(f); })));
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
        return this.forest === Array_1.empty || this.forest.length === 0
            ? // tslint:disable-next-line: deprecation
                "make(" + function_1.toString(this.value) + ")"
            : // tslint:disable-next-line: deprecation
                "make(" + function_1.toString(this.value) + ", " + function_1.toString(this.forest) + ")";
    };
    return Tree;
}());
exports.Tree = Tree;
/**
 * @since 1.17.0
 */
function getShow(S) {
    var show = function (t) {
        return t.forest === Array_1.empty || t.forest.length === 0
            ? "make(" + S.show(t.value) + ")"
            : "make(" + S.show(t.value) + ", [" + t.forest.map(show).join(', ') + "])";
    };
    return {
        show: show
    };
}
exports.getShow = getShow;
var foldr = function (fa, b, f) {
    var r = b;
    var len = fa.forest.length;
    for (var i = len - 1; i >= 0; i--) {
        r = foldr(fa.forest[i], r, f);
    }
    return f(fa.value, r);
};
function traverse(F) {
    var traverseF = Array_1.array.traverse(F);
    var r = function (ta, f) {
        return F.ap(F.map(f(ta.value), function (value) { return function (forest) { return new Tree(value, forest); }; }), traverseF(ta.forest, function (t) { return r(t, f); }));
    };
    return r;
}
function sequence(F) {
    var traverseF = traverse(F);
    return function (ta) { return traverseF(ta, function_1.identity); };
}
/**
 * Use `getEq`
 *
 * @since 1.6.0
 * @deprecated
 */
exports.getSetoid = getEq;
/**
 * @since 1.19.0
 */
function getEq(E) {
    var SA;
    var R = Eq_1.fromEquals(function (x, y) { return E.equals(x.value, y.value) && SA.equals(x.forest, y.forest); });
    SA = Array_1.getEq(R);
    return R;
}
exports.getEq = getEq;
/**
 * @since 1.6.0
 */
exports.tree = {
    URI: exports.URI,
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
exports.drawForest = function (forest) {
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
exports.drawTree = function (tree) {
    return tree.value + exports.drawForest(tree.forest);
};
/**
 * Build a tree from a seed value
 *
 * @since 1.6.0
 */
exports.unfoldTree = function (b, f) {
    var _a = f(b), a = _a[0], bs = _a[1];
    return new Tree(a, exports.unfoldForest(bs, f));
};
/**
 * Build a tree from a seed value
 *
 * @since 1.6.0
 */
exports.unfoldForest = function (bs, f) {
    return bs.map(function (b) { return exports.unfoldTree(b, f); });
};
function unfoldTreeM(M) {
    var unfoldForestMM = unfoldForestM(M);
    return function (b, f) { return M.chain(f(b), function (_a) {
        var a = _a[0], bs = _a[1];
        return M.chain(unfoldForestMM(bs, f), function (ts) { return M.of(new Tree(a, ts)); });
    }); };
}
exports.unfoldTreeM = unfoldTreeM;
function unfoldForestM(M) {
    var traverseM = Array_1.array.traverse(M);
    var unfoldTree;
    return function (bs, f) {
        // tslint:disable-next-line
        if (unfoldTree === undefined) {
            unfoldTree = unfoldTreeM(M);
        }
        return traverseM(bs, function (b) { return unfoldTree(b, f); });
    };
}
exports.unfoldForestM = unfoldForestM;
/**
 * @since 1.14.0
 */
function elem(E) {
    var go = function (a, fa) {
        if (E.equals(a, fa.value)) {
            return true;
        }
        return fa.forest.some(function (tree) { return go(a, tree); });
    };
    return go;
}
exports.elem = elem;
//
// backporting
//
/**
 * @since 1.19.0
 */
function make(a, forest) {
    if (forest === void 0) { forest = Array_1.empty; }
    return new Tree(a, forest);
}
exports.make = make;
var _a = pipeable_1.pipeable(exports.tree), ap = _a.ap, apFirst = _a.apFirst, apSecond = _a.apSecond, chain = _a.chain, chainFirst = _a.chainFirst, duplicate = _a.duplicate, extend = _a.extend, flatten = _a.flatten, foldMap = _a.foldMap, map = _a.map, reduce = _a.reduce, reduceRight = _a.reduceRight;
exports.ap = ap;
exports.apFirst = apFirst;
exports.apSecond = apSecond;
exports.chain = chain;
exports.chainFirst = chainFirst;
exports.duplicate = duplicate;
exports.extend = extend;
exports.flatten = flatten;
exports.foldMap = foldMap;
exports.map = map;
exports.reduce = reduce;
exports.reduceRight = reduceRight;
