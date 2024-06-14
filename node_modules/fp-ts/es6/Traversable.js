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
import { getFoldableComposition } from './Foldable';
import { getFunctorComposition } from './Functor';
export function traverse(F, 
// tslint:disable-next-line: deprecation
T) {
    return T.traverse(F);
}
// tslint:disable-next-line: deprecation
export function sequence(F, T) {
    return function (tfa) { return T.traverse(F)(tfa, function (fa) { return fa; }); };
}
// tslint:disable-next-line: deprecation
export function getTraversableComposition(F, G) {
    return __assign({}, getFunctorComposition(F, G), getFoldableComposition(F, G), { traverse: function (H) {
            var traverseF = F.traverse(H);
            var traverseG = G.traverse(H);
            return function (fga, f) { return traverseF(fga, function (ga) { return traverseG(ga, f); }); };
        } });
}
