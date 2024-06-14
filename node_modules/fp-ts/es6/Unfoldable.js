import { none, option } from './Option';
import { sequence } from './Traversable';
export function replicate(U) {
    return function (a, n) {
        function step(n) {
            return n <= 0 ? none : option.of([a, n - 1]);
        }
        return U.unfoldr(n, step);
    };
}
export function empty(U) {
    return U.unfoldr(undefined, function () { return none; });
}
export function singleton(U) {
    var replicateU = replicate(U);
    return function (a) { return replicateU(a, 1); };
}
export function replicateA(F, 
// tslint:disable-next-line: deprecation
UT) {
    var sequenceFUT = sequence(F, UT);
    var replicateUT = replicate(UT);
    return function (n, ma) { return sequenceFUT(replicateUT(ma, n)); };
}
