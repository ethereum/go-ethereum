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
import { getFunctorComposition } from './Functor';
import { fromEither, none, some } from './Option';
export function getCompactableComposition(F, G) {
    var FC = getFunctorComposition(F, G);
    var CC = __assign({}, FC, { compact: function (fga) { return F.map(fga, G.compact); }, separate: function (fge) {
            var left = CC.compact(FC.map(fge, function (e) { return e.fold(some, function () { return none; }); }));
            var right = CC.compact(FC.map(fge, fromEither));
            return { left: left, right: right };
        } });
    return CC;
}
