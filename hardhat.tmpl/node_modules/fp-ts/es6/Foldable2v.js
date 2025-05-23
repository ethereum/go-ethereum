import { unsafeMonoidArray } from './Monoid';
import { none, some } from './Option';
import { max as maxOrd, min as minOrd } from './Ord';
import { identity } from './function';
import { foldMap } from './Foldable';
import { applyFirst } from './Apply';
export function getFoldableComposition(F, G) {
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
export function fold(M, F) {
    return function (fa) { return F.reduce(fa, M.empty, M.concat); };
}
export function foldM(M, F) {
    return function (fa, b, f) { return F.reduce(fa, M.of(b), function (mb, a) { return M.chain(mb, function (b) { return f(b, a); }); }); };
}
export function sequence_(M, F) {
    var traverseMF = traverse_(M, F);
    return function (fa) { return traverseMF(fa, identity); };
}
export function oneOf(P, F) {
    return function (fga) { return F.reduce(fga, P.zero(), P.alt); };
}
export function intercalate(M, F) {
    return function (sep, fm) {
        var go = function (_a, x) {
            var init = _a.init, acc = _a.acc;
            return init ? { init: false, acc: x } : { init: false, acc: M.concat(M.concat(acc, sep), x) };
        };
        return F.reduce(fm, { init: true, acc: M.empty }, go).acc;
    };
}
export function sum(S, F) {
    return function (fa) { return F.reduce(fa, S.zero, S.add); };
}
export function product(S, F) {
    return function (fa) { return F.reduce(fa, S.one, S.mul); };
}
export function elem(E, F) {
    return function (a, fa) { return F.reduce(fa, false, function (b, x) { return b || E.equals(x, a); }); };
}
export function findFirst(F) {
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
export function min(O, F) {
    var minO = minOrd(O);
    return function (fa) { return F.reduce(fa, none, function (b, a) { return (b.isNone() ? some(a) : some(minO(b.value, a))); }); };
}
export function max(O, F) {
    var maxO = maxOrd(O);
    return function (fa) { return F.reduce(fa, none, function (b, a) { return (b.isNone() ? some(a) : some(maxO(b.value, a))); }); };
}
export function toArray(F) {
    // tslint:disable-next-line: deprecation
    var foldMapF = foldMap(F, unsafeMonoidArray);
    return function (fa) { return foldMapF(fa, function (a) { return [a]; }); };
}
export function traverse_(M, F) {
    // tslint:disable-next-line: deprecation
    var toArrayF = toArray(F);
    // tslint:disable-next-line: deprecation
    var applyFirstM = applyFirst(M);
    var initialValue = M.of(undefined);
    return function (fa, f) { return toArrayF(fa).reduce(function (mu, a) { return applyFirstM(mu, f(a)); }, initialValue); };
}
export function member(E, F) {
    // tslint:disable-next-line: deprecation
    return elem(E, F);
}
