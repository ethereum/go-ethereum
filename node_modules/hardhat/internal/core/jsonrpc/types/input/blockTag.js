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
exports.optionalRpcOldBlockTag = exports.rpcOldBlockTag = exports.optionalRpcNewBlockTag = exports.rpcNewBlockTag = exports.rpcBlockTagName = exports.rpcNewBlockTagObjectWithHash = exports.rpcNewBlockTagObjectWithNumber = void 0;
const t = __importStar(require("io-ts"));
const io_ts_1 = require("../../../../util/io-ts");
const base_types_1 = require("../base-types");
exports.rpcNewBlockTagObjectWithNumber = t.type({
    blockNumber: base_types_1.rpcQuantity,
});
exports.rpcNewBlockTagObjectWithHash = t.type({
    blockHash: base_types_1.rpcData,
    requireCanonical: (0, io_ts_1.optionalOrNullable)(t.boolean),
});
exports.rpcBlockTagName = t.keyof({
    earliest: null,
    latest: null,
    pending: null,
    safe: null,
    finalized: null,
});
// This is the new kind of block tag as defined by https://github.com/ethereum/EIPs/blob/master/EIPS/eip-1898.md
exports.rpcNewBlockTag = t.union([
    base_types_1.rpcQuantity,
    exports.rpcNewBlockTagObjectWithNumber,
    exports.rpcNewBlockTagObjectWithHash,
    exports.rpcBlockTagName,
]);
exports.optionalRpcNewBlockTag = (0, io_ts_1.optionalOrNullable)(exports.rpcNewBlockTag);
// This is the old kind of block tag which is described in the ethereum wiki
exports.rpcOldBlockTag = t.union([base_types_1.rpcQuantity, exports.rpcBlockTagName]);
exports.optionalRpcOldBlockTag = (0, io_ts_1.optionalOrNullable)(exports.rpcOldBlockTag);
//# sourceMappingURL=blockTag.js.map