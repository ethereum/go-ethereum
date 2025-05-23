"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var function_1 = require("./function");
function duplicate(E) {
    return function (ma) { return E.extend(ma, function_1.identity); };
}
exports.duplicate = duplicate;
