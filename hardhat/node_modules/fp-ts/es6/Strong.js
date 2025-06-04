import { identity } from './function';
export function splitStrong(F) {
    return function (pab, pcd) {
        return F.compose(F.first(pab), F.second(pcd));
    };
}
export function fanout(F) {
    var splitStrongF = splitStrong(F);
    return function (pab, pac) {
        var split = F.promap(F.id(), identity, function (a) { return [a, a]; });
        return F.compose(splitStrongF(pab, pac), split);
    };
}
