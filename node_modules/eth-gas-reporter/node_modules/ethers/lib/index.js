"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    Object.defineProperty(o, k2, { enumerable: true, get: function() { return m[k]; } });
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
exports.Wordlist = exports.version = exports.wordlists = exports.utils = exports.logger = exports.errors = exports.constants = exports.FixedNumber = exports.BigNumber = exports.ContractFactory = exports.Contract = exports.BaseContract = exports.providers = exports.getDefaultProvider = exports.VoidSigner = exports.Wallet = exports.Signer = exports.ethers = void 0;
// To modify this file, you must update ./misc/admin/lib/cmds/update-exports.js
var ethers = __importStar(require("./ethers"));
exports.ethers = ethers;
try {
    var anyGlobal = window;
    if (anyGlobal._ethers == null) {
        anyGlobal._ethers = ethers;
    }
}
catch (error) { }
var ethers_1 = require("./ethers");
Object.defineProperty(exports, "Signer", { enumerable: true, get: function () { return ethers_1.Signer; } });
Object.defineProperty(exports, "Wallet", { enumerable: true, get: function () { return ethers_1.Wallet; } });
Object.defineProperty(exports, "VoidSigner", { enumerable: true, get: function () { return ethers_1.VoidSigner; } });
Object.defineProperty(exports, "getDefaultProvider", { enumerable: true, get: function () { return ethers_1.getDefaultProvider; } });
Object.defineProperty(exports, "providers", { enumerable: true, get: function () { return ethers_1.providers; } });
Object.defineProperty(exports, "BaseContract", { enumerable: true, get: function () { return ethers_1.BaseContract; } });
Object.defineProperty(exports, "Contract", { enumerable: true, get: function () { return ethers_1.Contract; } });
Object.defineProperty(exports, "ContractFactory", { enumerable: true, get: function () { return ethers_1.ContractFactory; } });
Object.defineProperty(exports, "BigNumber", { enumerable: true, get: function () { return ethers_1.BigNumber; } });
Object.defineProperty(exports, "FixedNumber", { enumerable: true, get: function () { return ethers_1.FixedNumber; } });
Object.defineProperty(exports, "constants", { enumerable: true, get: function () { return ethers_1.constants; } });
Object.defineProperty(exports, "errors", { enumerable: true, get: function () { return ethers_1.errors; } });
Object.defineProperty(exports, "logger", { enumerable: true, get: function () { return ethers_1.logger; } });
Object.defineProperty(exports, "utils", { enumerable: true, get: function () { return ethers_1.utils; } });
Object.defineProperty(exports, "wordlists", { enumerable: true, get: function () { return ethers_1.wordlists; } });
////////////////////////
// Compile-Time Constants
Object.defineProperty(exports, "version", { enumerable: true, get: function () { return ethers_1.version; } });
Object.defineProperty(exports, "Wordlist", { enumerable: true, get: function () { return ethers_1.Wordlist; } });
//# sourceMappingURL=index.js.map