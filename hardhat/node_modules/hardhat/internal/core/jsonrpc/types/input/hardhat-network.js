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
exports.rpcIntervalMining = exports.optionalRpcHardhatNetworkConfig = exports.rpcHardhatNetworkConfig = exports.rpcForkConfig = void 0;
const t = __importStar(require("io-ts"));
const io_ts_1 = require("../../../../util/io-ts");
const base_types_1 = require("../base-types");
exports.rpcForkConfig = (0, io_ts_1.optional)(t.type({
    jsonRpcUrl: t.string,
    blockNumber: (0, io_ts_1.optional)(t.number),
    httpHeaders: (0, io_ts_1.optional)(t.record(t.string, t.string, "httpHeaders")),
}, "RpcForkConfig"));
exports.rpcHardhatNetworkConfig = t.type({
    forking: (0, io_ts_1.optional)(exports.rpcForkConfig),
}, "HardhatNetworkConfig");
exports.optionalRpcHardhatNetworkConfig = (0, io_ts_1.optional)(exports.rpcHardhatNetworkConfig);
const isNumberPair = (x) => Array.isArray(x) &&
    x.length === 2 &&
    Number.isInteger(x[0]) &&
    Number.isInteger(x[1]);
// TODO: This can be simplified
const rpcIntervalMiningRange = new t.Type("Interval mining range", isNumberPair, (u, c) => isNumberPair(u) && u[0] >= 0 && u[1] >= u[0]
    ? t.success(u)
    : t.failure(u, c), t.identity);
exports.rpcIntervalMining = t.union([
    base_types_1.rpcUnsignedInteger,
    rpcIntervalMiningRange,
]);
//# sourceMappingURL=hardhat-network.js.map