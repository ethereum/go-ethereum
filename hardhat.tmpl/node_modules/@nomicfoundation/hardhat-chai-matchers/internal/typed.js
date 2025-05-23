"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.tryDereference = void 0;
function tryDereference(value, type) {
    const { Typed } = require("ethers");
    try {
        return Typed.dereference(value, type);
    }
    catch {
        return undefined;
    }
}
exports.tryDereference = tryDereference;
//# sourceMappingURL=typed.js.map