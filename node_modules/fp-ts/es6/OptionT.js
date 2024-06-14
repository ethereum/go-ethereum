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
import { getApplicativeComposition } from './Applicative';
import { none as optionNone, option, some as optionSome } from './Option';
export function fold(F) {
    return function (onNone, onSome, fa) { return F.map(fa, function (o) { return (o.isNone() ? onNone : onSome(o.value)); }); };
}
export function getOptionT2v(M) {
    var applicativeComposition = getApplicativeComposition(M, option);
    return __assign({}, applicativeComposition, { chain: function (fa, f) { return M.chain(fa, function (o) { return (o.isNone() ? M.of(optionNone) : f(o.value)); }); } });
}
/** @deprecated */
// tslint:disable-next-line: deprecation
export function chain(F) {
    return function (f, fa) { return F.chain(fa, function (o) { return (o.isNone() ? F.of(optionNone) : f(o.value)); }); };
}
// tslint:disable-next-line: deprecation
export function getOptionT(M) {
    var applicativeComposition = getApplicativeComposition(M, option);
    return __assign({}, applicativeComposition, { 
        // tslint:disable-next-line: deprecation
        chain: chain(M) });
}
/** @deprecated */
export function some(F) {
    return function (a) { return F.of(optionSome(a)); };
}
/** @deprecated */
export function none(F) {
    return function () { return F.of(optionNone); };
}
/** @deprecated */
export function fromOption(F) {
    return F.of;
}
/** @deprecated */
export function liftF(F) {
    return function (fa) { return F.map(fa, optionSome); };
}
/** @deprecated */
export function getOrElse(F) {
    return function (a) { return function (fa) { return F.map(fa, function (o) { return o.getOrElse(a); }); }; };
}
