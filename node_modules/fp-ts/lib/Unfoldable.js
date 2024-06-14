"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var Option_1 = require("./Option");
var Traversable_1 = require("./Traversable");
function replicate(U) {
    return function (a, n) {
        function step(n) {
            return n <= 0 ? Option_1.none : Option_1.option.of([a, n - 1]);
        }
        return U.unfoldr(n, step);
    };
}
exports.replicate = replicate;
function empty(U) {
    return U.unfoldr(undefined, function () { return Option_1.none; });
}
exports.empty = empty;
function singleton(U) {
    var replicateU = replicate(U);
    return function (a) { return replicateU(a, 1); };
}
exports.singleton = singleton;
function replicateA(F, 
// tslint:disable-next-line: deprecation
UT) {
    var sequenceFUT = Traversable_1.sequence(F, UT);
    var replicateUT = replicate(UT);
    return function (n, ma) { return sequenceFUT(replicateUT(ma, n)); };
}
exports.replicateA = replicateA;
