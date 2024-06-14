"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.resolveArgsToFutures = void 0;
const type_guards_1 = require("../type-guards");
function resolveArgsToFutures(args) {
    return args.flatMap(_resolveArgToFutures);
}
exports.resolveArgsToFutures = resolveArgsToFutures;
function _resolveArgToFutures(argument) {
    if ((0, type_guards_1.isFuture)(argument)) {
        return [argument];
    }
    if (Array.isArray(argument)) {
        return resolveArgsToFutures(argument);
    }
    if (typeof argument === "object" && argument !== null) {
        return resolveArgsToFutures(Object.values(argument));
    }
    return [];
}
//# sourceMappingURL=utils.js.map