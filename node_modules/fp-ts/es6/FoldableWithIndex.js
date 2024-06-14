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
/**
 * @file A `Foldable` with an additional index.
 * A `FoldableWithIndex` instance must be compatible with its `Foldable` instance
 *
 * ```ts
 * reduce(fa, b, f) = reduceWithIndex(fa, b, (_, b, a) => f(b, a))
 * foldMap(M)(fa, f) = foldMapWithIndex(M)(fa, (_, a) => f(a))
 * foldr(fa, b, f) = foldrWithIndex(fa, b, (_, a, b) => f(a, b))
 * ```
 */
import { getFoldableComposition } from './Foldable2v';
export function getFoldableWithIndexComposition(F, G) {
    return __assign({}, getFoldableComposition(F, G), { reduceWithIndex: function (fga, b, f) {
            return F.reduceWithIndex(fga, b, function (fi, b, ga) { return G.reduceWithIndex(ga, b, function (gi, b, a) { return f([fi, gi], b, a); }); });
        }, foldMapWithIndex: function (M) {
            var foldMapWithIndexF = F.foldMapWithIndex(M);
            var foldMapWithIndexG = G.foldMapWithIndex(M);
            return function (fga, f) { return foldMapWithIndexF(fga, function (fi, ga) { return foldMapWithIndexG(ga, function (gi, a) { return f([fi, gi], a); }); }); };
        }, foldrWithIndex: function (fga, b, f) {
            return F.foldrWithIndex(fga, b, function (fi, ga, b) { return G.foldrWithIndex(ga, b, function (gi, a, b) { return f([fi, gi], a, b); }); });
        } });
}
