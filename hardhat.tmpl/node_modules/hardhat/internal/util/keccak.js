"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.keccak256 = void 0;
const keccak_1 = __importDefault(require("keccak"));
function keccak256(data) {
    return (0, keccak_1.default)("keccak256").update(Buffer.from(data)).digest();
}
exports.keccak256 = keccak256;
//# sourceMappingURL=keccak.js.map