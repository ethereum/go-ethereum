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
exports.rpcAuthorizationList = void 0;
const t = __importStar(require("io-ts"));
const base_types_1 = require("./base-types");
const rpcAuthorizationListTuple = t.type({
    chainId: base_types_1.rpcQuantity,
    address: base_types_1.rpcAddress,
    nonce: base_types_1.rpcQuantity,
    yParity: base_types_1.rpcParity,
    r: base_types_1.rpcHash,
    s: base_types_1.rpcHash,
});
exports.rpcAuthorizationList = t.array(rpcAuthorizationListTuple);
//# sourceMappingURL=authorization-list.js.map