"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var function_1 = require("./function");
function splitStrong(F) {
    return function (pab, pcd) {
        return F.compose(F.first(pab), F.second(pcd));
    };
}
exports.splitStrong = splitStrong;
function fanout(F) {
    var splitStrongF = splitStrong(F);
    return function (pab, pac) {
        var split = F.promap(F.id(), function_1.identity, function (a) { return [a, a]; });
        return F.compose(splitStrongF(pab, pac), split);
    };
}
exports.fanout = fanout;
