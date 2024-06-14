"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var function_1 = require("./function");
function splitChoice(F) {
    return function (pab, pcd) {
        return F.compose(F.left(pab), F.right(pcd));
    };
}
exports.splitChoice = splitChoice;
function fanin(F) {
    var splitChoiceF = splitChoice(F);
    return function (pac, pbc) {
        var join = F.promap(F.id(), function (e) { return e.fold(function_1.identity, function_1.identity); }, function_1.identity);
        return F.compose(join, splitChoiceF(pac, pbc));
    };
}
exports.fanin = fanin;
