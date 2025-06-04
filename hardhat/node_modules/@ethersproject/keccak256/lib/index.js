"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.keccak256 = void 0;
var js_sha3_1 = __importDefault(require("js-sha3"));
var bytes_1 = require("@ethersproject/bytes");
function keccak256(data) {
    return '0x' + js_sha3_1.default.keccak_256((0, bytes_1.arrayify)(data));
}
exports.keccak256 = keccak256;
//# sourceMappingURL=index.js.map