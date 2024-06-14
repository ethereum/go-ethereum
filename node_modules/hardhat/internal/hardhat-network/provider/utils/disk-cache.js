"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.getForkCacheDirPath = void 0;
const path_1 = __importDefault(require("path"));
function getForkCacheDirPath(paths) {
    return path_1.default.join(paths.cache, "hardhat-network-fork");
}
exports.getForkCacheDirPath = getForkCacheDirPath;
//# sourceMappingURL=disk-cache.js.map