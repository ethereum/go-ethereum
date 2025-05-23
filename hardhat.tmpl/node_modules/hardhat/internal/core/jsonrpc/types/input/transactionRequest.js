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
exports.rpcTransactionRequest = void 0;
const t = __importStar(require("io-ts"));
const io_ts_1 = require("../../../../util/io-ts");
const access_list_1 = require("../access-list");
const base_types_1 = require("../base-types");
const authorization_list_1 = require("../authorization-list");
// Type used by eth_sendTransaction
exports.rpcTransactionRequest = t.type({
    from: base_types_1.rpcAddress,
    to: (0, io_ts_1.optionalOrNullable)(base_types_1.rpcAddress),
    gas: (0, io_ts_1.optionalOrNullable)(base_types_1.rpcQuantity),
    gasPrice: (0, io_ts_1.optionalOrNullable)(base_types_1.rpcQuantity),
    value: (0, io_ts_1.optionalOrNullable)(base_types_1.rpcQuantity),
    nonce: (0, io_ts_1.optionalOrNullable)(base_types_1.rpcQuantity),
    data: (0, io_ts_1.optionalOrNullable)(base_types_1.rpcData),
    accessList: (0, io_ts_1.optionalOrNullable)(access_list_1.rpcAccessList),
    chainId: (0, io_ts_1.optionalOrNullable)(base_types_1.rpcQuantity),
    maxFeePerGas: (0, io_ts_1.optionalOrNullable)(base_types_1.rpcQuantity),
    maxPriorityFeePerGas: (0, io_ts_1.optionalOrNullable)(base_types_1.rpcQuantity),
    blobs: (0, io_ts_1.optionalOrNullable)(t.array(base_types_1.rpcData)),
    blobVersionedHashes: (0, io_ts_1.optionalOrNullable)(t.array(base_types_1.rpcHash)),
    authorizationList: (0, io_ts_1.optionalOrNullable)(authorization_list_1.rpcAuthorizationList),
}, "RpcTransactionRequest");
//# sourceMappingURL=transactionRequest.js.map