import { constant } from './function';
export function iapplyFirst(ixmonad) {
    return function (fa, fb) { return ixmonad.ichain(fa, function (a) { return ixmonad.ichain(fb, function () { return ixmonad.iof(a); }); }); };
}
export function iapplySecond(ixmonad) {
    return function (fa, fb) { return ixmonad.ichain(fa, constant(fb)); };
}
