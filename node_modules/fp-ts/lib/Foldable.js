"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var Apply_1 = require("./Apply");
var Monoid_1 = require("./Monoid");
var Option_1 = require("./Option");
var Ord_1 = require("./Ord");
// tslint:disable-next-line: deprecation
function getFoldableComposition(F, G) {
    return {
        reduce: function (fga, b, f) { return F.reduce(fga, b, function (b, ga) { return G.reduce(ga, b, f); }); }
    };
}
exports.getFoldableComposition = getFoldableComposition;
// tslint:disable-next-line: deprecation
function foldMap(F, M) {
    return function (fa, f) { return F.reduce(fa, M.empty, function (acc, x) { return M.concat(acc, f(x)); }); };
}
exports.foldMap = foldMap;
// tslint:disable-next-line: deprecation
function foldr(F) {
    var toArrayF = toArray(F);
    return function (fa, b, f) { return toArrayF(fa).reduceRight(function (acc, a) { return f(a, acc); }, b); };
}
exports.foldr = foldr;
// tslint:disable-next-line: deprecation
function fold(F, M) {
    return function (fa) { return F.reduce(fa, M.empty, M.concat); };
}
exports.fold = fold;
function foldM(
// tslint:disable-next-line: deprecation
F, M) {
    return function (f, b, fa) { return F.reduce(fa, M.of(b), function (mb, a) { return M.chain(mb, function (b) { return f(b, a); }); }); };
}
exports.foldM = foldM;
function traverse_(M, 
// tslint:disable-next-line: deprecation
F) {
    var toArrayF = toArray(F);
    // tslint:disable-next-line: deprecation
    var applyFirstM = Apply_1.applyFirst(M);
    var initialValue = M.of(undefined);
    return function (f, fa) { return toArrayF(fa).reduce(function (mu, a) { return applyFirstM(mu, f(a)); }, initialValue); };
}
exports.traverse_ = traverse_;
// tslint:disable-next-line: deprecation
function sequence_(M, F) {
    // tslint:disable-next-line: deprecation
    var traverse_MF = traverse_(M, F);
    return function (fa) { return traverse_MF(function (ma) { return ma; }, fa); };
}
exports.sequence_ = sequence_;
// tslint:disable-next-line: deprecation
function oneOf(F, P) {
    return function (fga) { return F.reduce(fga, P.zero(), function (acc, a) { return P.alt(acc, a); }); };
}
exports.oneOf = oneOf;
// tslint:disable-next-line: deprecation
function intercalate(F, M) {
    return function (sep) {
        function go(_a, x) {
            var init = _a.init, acc = _a.acc;
            return init ? { init: false, acc: x } : { init: false, acc: M.concat(M.concat(acc, sep), x) };
        }
        return function (fm) { return F.reduce(fm, { init: true, acc: M.empty }, go).acc; };
    };
}
exports.intercalate = intercalate;
// tslint:disable-next-line: deprecation
function sum(F, S) {
    return function (fa) { return F.reduce(fa, S.zero, function (b, a) { return S.add(b, a); }); };
}
exports.sum = sum;
// tslint:disable-next-line: deprecation
function product(F, S) {
    return function (fa) { return F.reduce(fa, S.one, function (b, a) { return S.mul(b, a); }); };
}
exports.product = product;
// tslint:disable-next-line: deprecation
function elem(F, E) {
    return function (a, fa) { return F.reduce(fa, false, function (b, x) { return b || E.equals(x, a); }); };
}
exports.elem = elem;
// tslint:disable-next-line: deprecation
function find(F) {
    return function (fa, p) {
        return F.reduce(fa, Option_1.none, function (b, a) {
            if (b.isNone() && p(a)) {
                return Option_1.some(a);
            }
            else {
                return b;
            }
        });
    };
}
exports.find = find;
// tslint:disable-next-line: deprecation
function minimum(F, O) {
    var minO = Ord_1.min(O);
    return function (fa) { return F.reduce(fa, Option_1.none, function (b, a) { return (b.isNone() ? Option_1.some(a) : Option_1.some(minO(b.value, a))); }); };
}
exports.minimum = minimum;
// tslint:disable-next-line: deprecation
function maximum(F, O) {
    var maxO = Ord_1.max(O);
    return function (fa) { return F.reduce(fa, Option_1.none, function (b, a) { return (b.isNone() ? Option_1.some(a) : Option_1.some(maxO(b.value, a))); }); };
}
exports.maximum = maximum;
// tslint:disable-next-line: deprecation
function toArray(F) {
    // tslint:disable-next-line: deprecation
    var foldMapF = foldMap(F, Monoid_1.unsafeMonoidArray);
    return function (fa) { return foldMapF(fa, function (a) { return [a]; }); };
}
exports.toArray = toArray;
function traverse(M, 
// tslint:disable-next-line: deprecation
F) {
    // tslint:disable-next-line: deprecation
    var traverseMF = traverse_(M, F);
    return function (fa, f) { return traverseMF(f, fa); };
}
exports.traverse = traverse;
