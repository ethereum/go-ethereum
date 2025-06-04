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
import { either, left as eitherLeft, right as eitherRight } from './Either';
export function fold(F) {
    return function (left, right, fa) { return F.map(fa, function (e) { return (e.isLeft() ? left(e.value) : right(e.value)); }); };
}
export function getEitherT2v(M) {
    var applicativeComposition = getApplicativeComposition(M, either);
    return __assign({}, applicativeComposition, { chain: function (fa, f) { return M.chain(fa, function (e) { return (e.isLeft() ? M.of(eitherLeft(e.value)) : f(e.value)); }); } });
}
/** @deprecated */
// tslint:disable-next-line: deprecation
export function chain(F) {
    return function (f, fa) { return F.chain(fa, function (e) { return (e.isLeft() ? F.of(eitherLeft(e.value)) : f(e.value)); }); };
}
/** @deprecated */
// tslint:disable-next-line: deprecation
export function getEitherT(M) {
    var applicativeComposition = getApplicativeComposition(M, either);
    return __assign({}, applicativeComposition, { 
        // tslint:disable-next-line: deprecation
        chain: chain(M) });
}
/** @deprecated */
export function right(F) {
    return function (fa) { return F.map(fa, eitherRight); };
}
/** @deprecated */
export function left(F) {
    return function (fl) { return F.map(fl, eitherLeft); };
}
/** @deprecated */
export function fromEither(F) {
    return F.of;
}
/** @deprecated */
export function mapLeft(F) {
    return function (f) { return function (fa) { return F.map(fa, function (e) { return e.mapLeft(f); }); }; };
}
/** @deprecated */
export function bimap(F) {
    return function (fa, f, g) { return F.map(fa, function (e) { return e.bimap(f, g); }); };
}
