"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.findTarget = void 0;
const lodash_1 = __importStar(require("lodash"));
const debug_1 = require("../utils/debug");
const ensureAbsPath_1 = require("../utils/files/ensureAbsPath");
const modules_1 = require("../utils/modules");
function findTarget(config) {
    const target = config.target;
    if (!target) {
        throw new Error(`Please provide --target parameter!`);
    }
    const possiblePaths = [
        `@typechain/${target}`,
        `typechain-target-${target}`,
        (0, ensureAbsPath_1.ensureAbsPath)(target), // path
    ];
    const moduleInfo = (0, lodash_1.default)(possiblePaths).compact().map(modules_1.tryRequire).compact().first();
    if (!moduleInfo || !moduleInfo.module.default) {
        throw new Error(`Couldn't find ${config.target}. Tried loading: ${(0, lodash_1.compact)(possiblePaths).join(', ')}.\nPerhaps you forgot to install @typechain/${target}?`);
    }
    (0, debug_1.debug)('Plugin found at', moduleInfo.path);
    return new moduleInfo.module.default(config);
}
exports.findTarget = findTarget;
//# sourceMappingURL=findTarget.js.map