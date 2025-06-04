"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
var crypto_1 = __importDefault(require("crypto"));
var hash_utils_1 = require("./hash-utils");
exports.sha256 = hash_utils_1.createHashFunction(function () { return crypto_1.default.createHash("sha256"); });
//# sourceMappingURL=sha256.js.map