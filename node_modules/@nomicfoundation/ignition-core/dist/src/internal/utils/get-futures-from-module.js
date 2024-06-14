"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getFuturesFromModule = void 0;
/**
 * Get the futures from a module, including its submodules.
 * No ordering is enforced.
 */
function getFuturesFromModule(module) {
    return [...module.futures].concat(Array.from(module.submodules).flatMap((sub) => getFuturesFromModule(sub)));
}
exports.getFuturesFromModule = getFuturesFromModule;
//# sourceMappingURL=get-futures-from-module.js.map