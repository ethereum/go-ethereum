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
var __exportStar = (this && this.__exportStar) || function(m, exports) {
    for (var p in m) if (p !== "default" && !Object.prototype.hasOwnProperty.call(exports, p)) __createBinding(exports, m, p);
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.normalizeName = void 0;
__exportStar(require("./codegen/createBarrelFiles"), exports);
__exportStar(require("./codegen/syntax"), exports);
__exportStar(require("./parser/abiParser"), exports);
var normalizeName_1 = require("./parser/normalizeName");
Object.defineProperty(exports, "normalizeName", { enumerable: true, get: function () { return normalizeName_1.normalizeName; } });
__exportStar(require("./parser/parseEvmType"), exports);
__exportStar(require("./typechain/runTypeChain"), exports);
__exportStar(require("./typechain/types"), exports);
__exportStar(require("./utils/files"), exports);
__exportStar(require("./utils/glob"), exports);
__exportStar(require("./utils/signatures"), exports);
//# sourceMappingURL=index.js.map