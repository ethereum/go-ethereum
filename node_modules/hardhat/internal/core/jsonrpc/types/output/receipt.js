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
exports.rpcTransactionReceipt = void 0;
const t = __importStar(require("io-ts"));
const io_ts_1 = require("../../../../util/io-ts");
const base_types_1 = require("../base-types");
const log_1 = require("./log");
exports.rpcTransactionReceipt = t.type({
    transactionHash: base_types_1.rpcHash,
    transactionIndex: base_types_1.rpcQuantity,
    blockHash: base_types_1.rpcHash,
    blockNumber: base_types_1.rpcQuantity,
    from: base_types_1.rpcAddress,
    to: (0, io_ts_1.nullable)(base_types_1.rpcAddress),
    cumulativeGasUsed: base_types_1.rpcQuantity,
    gasUsed: base_types_1.rpcQuantity,
    contractAddress: (0, io_ts_1.nullable)(base_types_1.rpcAddress),
    logs: t.array(log_1.rpcLog, "RpcLog Array"),
    logsBloom: base_types_1.rpcData,
    // This should be just optional, but Alchemy returns null
    //
    // It shouldn't accept a number, but that's what Erigon returns.
    // See: https://github.com/ledgerwatch/erigon/issues/2288
    status: (0, io_ts_1.optional)((0, io_ts_1.nullable)(t.union([base_types_1.rpcQuantity, base_types_1.rpcQuantityAsNumber]))),
    root: (0, io_ts_1.optional)(base_types_1.rpcData),
    type: (0, io_ts_1.optional)(base_types_1.rpcQuantity),
    effectiveGasPrice: (0, io_ts_1.optional)(base_types_1.rpcQuantity),
}, "RpcTransactionReceipt");
//# sourceMappingURL=receipt.js.map