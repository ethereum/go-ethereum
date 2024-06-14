import { applyFirst } from './Apply';
import { unsafeMonoidArray } from './Monoid';
import { none, some } from './Option';
import { max, min } from './Ord';
// tslint:disable-next-line: deprecation
export function getFoldableComposition(F, G) {
    return {
        reduce: function (fga, b, f) { return F.reduce(fga, b, function (b, ga) { return G.reduce(ga, b, f); }); }
    };
}
// tslint:disable-next-line: deprecation
export function foldMap(F, M) {
    return function (fa, f) { return F.reduce(fa, M.empty, function (acc, x) { return M.concat(acc, f(x)); }); };
}
// tslint:disable-next-line: deprecation
export function foldr(F) {
    var toArrayF = toArray(F);
    return function (fa, b, f) { return toArrayF(fa).reduceRight(function (acc, a) { return f(a, acc); }, b); };
}
// tslint:disable-next-line: deprecation
export function fold(F, M) {
    return function (fa) { return F.reduce(fa, M.empty, M.concat); };
}
export function foldM(
// tslint:disable-next-line: deprecation
F, M) {
    return function (f, b, fa) { return F.reduce(fa, M.of(b), function (mb, a) { return M.chain(mb, function (b) { return f(b, a); }); }); };
}
export function traverse_(M, 
// tslint:disable-next-line: deprecation
F) {
    var toArrayF = toArray(F);
    // tslint:disable-next-line: deprecation
    var applyFirstM = applyFirst(M);
    var initialValue = M.of(undefined);
    return function (f, fa) { return toArrayF(fa).reduce(function (mu, a) { return applyFirstM(mu, f(a)); }, initialValue); };
}
// tslint:disable-next-line: deprecation
export function sequence_(M, F) {
    // tslint:disable-next-line: deprecation
    var traverse_MF = traverse_(M, F);
    return function (fa) { return traverse_MF(function (ma) { return ma; }, fa); };
}
// tslint:disable-next-line: deprecation
export function oneOf(F, P) {
    return function (fga) { return F.reduce(fga, P.zero(), function (acc, a) { return P.alt(acc, a); }); };
}
// tslint:disable-next-line: deprecation
export function intercalate(F, M) {
    return function (sep) {
        function go(_a, x) {
            var init = _a.init, acc = _a.acc;
            return init ? { init: false, acc: x } : { init: false, acc: M.concat(M.concat(acc, sep), x) };
        }
        return function (fm) { return F.reduce(fm, { init: true, acc: M.empty }, go).acc; };
    };
}
// tslint:disable-next-line: deprecation
export function sum(F, S) {
    return function (fa) { return F.reduce(fa, S.zero, function (b, a) { return S.add(b, a); }); };
}
// tslint:disable-next-line: deprecation
export function product(F, S) {
    return function (fa) { return F.reduce(fa, S.one, function (b, a) { return S.mul(b, a); }); };
}
// tslint:disable-next-line: deprecation
export function elem(F, E) {
    return function (a, fa) { return F.reduce(fa, false, function (b, x) { return b || E.equals(x, a); }); };
}
// tslint:disable-next-line: deprecation
export function find(F) {
    return function (fa, p) {
        return F.reduce(fa, none, function (b, a) {
            if (b.isNone() && p(a)) {
                return some(a);
            }
            else {
                return b;
            }
        });
    };
}
// tslint:disable-next-line: deprecation
export function minimum(F, O) {
    var minO = min(O);
    return function (fa) { return F.reduce(fa, none, function (b, a) { return (b.isNone() ? some(a) : some(minO(b.value, a))); }); };
}
// tslint:disable-next-line: deprecation
export function maximum(F, O) {
    var maxO = max(O);
    return function (fa) { return F.reduce(fa, none, function (b, a) { return (b.isNone() ? some(a) : some(maxO(b.value, a))); }); };
}
// tslint:disable-next-line: deprecation
export function toArray(F) {
    // tslint:disable-next-line: deprecation
    var foldMapF = foldMap(F, unsafeMonoidArray);
    return function (fa) { return foldMapF(fa, function (a) { return [a]; }); };
}
export function traverse(M, 
// tslint:disable-next-line: deprecation
F) {
    // tslint:disable-next-line: deprecation
    var traverseMF = traverse_(M, F);
    return function (fa, f) { return traverseMF(f, fa); };
}
