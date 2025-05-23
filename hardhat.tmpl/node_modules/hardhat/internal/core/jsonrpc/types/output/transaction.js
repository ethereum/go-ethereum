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
exports.rpcTransaction = void 0;
const t = __importStar(require("io-ts"));
const io_ts_1 = require("../../../../util/io-ts");
const access_list_1 = require("../access-list");
const base_types_1 = require("../base-types");
exports.rpcTransaction = t.type({
    blockHash: (0, io_ts_1.nullable)(base_types_1.rpcHash),
    blockNumber: (0, io_ts_1.nullable)(base_types_1.rpcQuantity),
    from: base_types_1.rpcAddress,
    gas: base_types_1.rpcQuantity,
    gasPrice: base_types_1.rpcQuantity,
    hash: base_types_1.rpcHash,
    input: base_types_1.rpcData,
    nonce: base_types_1.rpcQuantity,
    // This is also optional because Alchemy doesn't return to for deployment txs
    to: (0, io_ts_1.optional)((0, io_ts_1.nullable)(base_types_1.rpcAddress)),
    transactionIndex: (0, io_ts_1.nullable)(base_types_1.rpcQuantity),
    value: base_types_1.rpcQuantity,
    v: base_types_1.rpcQuantity,
    r: base_types_1.rpcQuantity,
    s: base_types_1.rpcQuantity,
    // EIP-2929/2930 properties
    type: (0, io_ts_1.optional)(base_types_1.rpcQuantity),
    chainId: (0, io_ts_1.optional)((0, io_ts_1.nullable)(base_types_1.rpcQuantity)),
    accessList: (0, io_ts_1.optional)(access_list_1.rpcAccessList),
    // EIP-1559 properties
    maxFeePerGas: (0, io_ts_1.optional)(base_types_1.rpcQuantity),
    maxPriorityFeePerGas: (0, io_ts_1.optional)(base_types_1.rpcQuantity),
}, "RpcTransaction");
//# sourceMappingURL=transaction.js.map