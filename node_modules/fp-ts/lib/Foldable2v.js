"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var Monoid_1 = require("./Monoid");
var Option_1 = require("./Option");
var Ord_1 = require("./Ord");
var function_1 = require("./function");
var Foldable_1 = require("./Foldable");
var Apply_1 = require("./Apply");
function getFoldableComposition(F, G) {
    return {
        reduce: function (fga, b, f) { return F.reduce(fga, b, function (b, ga) { return G.reduce(ga, b, f); }); },
        foldMap: function (M) {
            var foldMapF = F.foldMap(M);
            var foldMapG = G.foldMap(M);
            return function (fa, f) { return foldMapF(fa, function (ga) { return foldMapG(ga, f); }); };
        },
        foldr: function (fa, b, f) { return F.foldr(fa, b, function (ga, b) { return G.foldr(ga, b, f); }); }
    };
}
exports.getFoldableComposition = getFoldableComposition;
function fold(M, F) {
    return function (fa) { return F.reduce(fa, M.empty, M.concat); };
}
exports.fold = fold;
function foldM(M, F) {
    return function (fa, b, f) { return F.reduce(fa, M.of(b), function (mb, a) { return M.chain(mb, function (b) { return f(b, a); }); }); };
}
exports.foldM = foldM;
function sequence_(M, F) {
    var traverseMF = traverse_(M, F);
    return function (fa) { return traverseMF(fa, function_1.identity); };
}
exports.sequence_ = sequence_;
function oneOf(P, F) {
    return function (fga) { return F.reduce(fga, P.zero(), P.alt); };
}
exports.oneOf = oneOf;
function intercalate(M, F) {
    return function (sep, fm) {
        var go = function (_a, x) {
            var init = _a.init, acc = _a.acc;
            return init ? { init: false, acc: x } : { init: false, acc: M.concat(M.concat(acc, sep), x) };
        };
        return F.reduce(fm, { init: true, acc: M.empty }, go).acc;
    };
}
exports.intercalate = intercalate;
function sum(S, F) {
    return function (fa) { return F.reduce(fa, S.zero, S.add); };
}
exports.sum = sum;
function product(S, F) {
    return function (fa) { return F.reduce(fa, S.one, S.mul); };
}
exports.product = product;
function elem(E, F) {
    return function (a, fa) { return F.reduce(fa, false, function (b, x) { return b || E.equals(x, a); }); };
}
exports.elem = elem;
function findFirst(F) {
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
exports.findFirst = findFirst;
function min(O, F) {
    var minO = Ord_1.min(O);
    return function (fa) { return F.reduce(fa, Option_1.none, function (b, a) { return (b.isNone() ? Option_1.some(a) : Option_1.some(minO(b.value, a))); }); };
}
exports.min = min;
function max(O, F) {
    var maxO = Ord_1.max(O);
    return function (fa) { return F.reduce(fa, Option_1.none, function (b, a) { return (b.isNone() ? Option_1.some(a) : Option_1.some(maxO(b.value, a))); }); };
}
exports.max = max;
function toArray(F) {
    // tslint:disable-next-line: deprecation
    var foldMapF = Foldable_1.foldMap(F, Monoid_1.unsafeMonoidArray);
    return function (fa) { return foldMapF(fa, function (a) { return [a]; }); };
}
exports.toArray = toArray;
function traverse_(M, F) {
    // tslint:disable-next-line: deprecation
    var toArrayF = toArray(F);
    // tslint:disable-next-line: deprecation
    var applyFirstM = Apply_1.applyFirst(M);
    var initialValue = M.of(undefined);
    return function (fa, f) { return toArrayF(fa).reduce(function (mu, a) { return applyFirstM(mu, f(a)); }, initialValue); };
}
exports.traverse_ = traverse_;
function member(E, F) {
    // tslint:disable-next-line: deprecation
    return elem(E, F);
}
exports.member = member;
