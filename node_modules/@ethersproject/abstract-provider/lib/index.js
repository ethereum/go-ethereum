"use strict";
var __extends = (this && this.__extends) || (function () {
    var extendStatics = function (d, b) {
        extendStatics = Object.setPrototypeOf ||
            ({ __proto__: [] } instanceof Array && function (d, b) { d.__proto__ = b; }) ||
            function (d, b) { for (var p in b) if (Object.prototype.hasOwnProperty.call(b, p)) d[p] = b[p]; };
        return extendStatics(d, b);
    };
    return function (d, b) {
        if (typeof b !== "function" && b !== null)
            throw new TypeError("Class extends value " + String(b) + " is not a constructor or null");
        extendStatics(d, b);
        function __() { this.constructor = d; }
        d.prototype = b === null ? Object.create(b) : (__.prototype = b.prototype, new __());
    };
})();
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
var __generator = (this && this.__generator) || function (thisArg, body) {
    var _ = { label: 0, sent: function() { if (t[0] & 1) throw t[1]; return t[1]; }, trys: [], ops: [] }, f, y, t, g;
    return g = { next: verb(0), "throw": verb(1), "return": verb(2) }, typeof Symbol === "function" && (g[Symbol.iterator] = function() { return this; }), g;
    function verb(n) { return function (v) { return step([n, v]); }; }
    function step(op) {
        if (f) throw new TypeError("Generator is already executing.");
        while (_) try {
            if (f = 1, y && (t = op[0] & 2 ? y["return"] : op[0] ? y["throw"] || ((t = y["return"]) && t.call(y), 0) : y.next) && !(t = t.call(y, op[1])).done) return t;
            if (y = 0, t) op = [op[0] & 2, t.value];
            switch (op[0]) {
                case 0: case 1: t = op; break;
                case 4: _.label++; return { value: op[1], done: false };
                case 5: _.label++; y = op[1]; op = [0]; continue;
                case 7: op = _.ops.pop(); _.trys.pop(); continue;
                default:
                    if (!(t = _.trys, t = t.length > 0 && t[t.length - 1]) && (op[0] === 6 || op[0] === 2)) { _ = 0; continue; }
                    if (op[0] === 3 && (!t || (op[1] > t[0] && op[1] < t[3]))) { _.label = op[1]; break; }
                    if (op[0] === 6 && _.label < t[1]) { _.label = t[1]; t = op; break; }
                    if (t && _.label < t[2]) { _.label = t[2]; _.ops.push(op); break; }
                    if (t[2]) _.ops.pop();
                    _.trys.pop(); continue;
            }
            op = body.call(thisArg, _);
        } catch (e) { op = [6, e]; y = 0; } finally { f = t = 0; }
        if (op[0] & 5) throw op[1]; return { value: op[0] ? op[1] : void 0, done: true };
    }
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Provider = exports.TransactionOrderForkEvent = exports.TransactionForkEvent = exports.BlockForkEvent = exports.ForkEvent = void 0;
var bignumber_1 = require("@ethersproject/bignumber");
var bytes_1 = require("@ethersproject/bytes");
var properties_1 = require("@ethersproject/properties");
var logger_1 = require("@ethersproject/logger");
var _version_1 = require("./_version");
var logger = new logger_1.Logger(_version_1.version);
;
;
//export type CallTransactionable = {
//    call(transaction: TransactionRequest): Promise<TransactionResponse>;
//};
var ForkEvent = /** @class */ (function (_super) {
    __extends(ForkEvent, _super);
    function ForkEvent() {
        return _super !== null && _super.apply(this, arguments) || this;
    }
    ForkEvent.isForkEvent = function (value) {
        return !!(value && value._isForkEvent);
    };
    return ForkEvent;
}(properties_1.Description));
exports.ForkEvent = ForkEvent;
var BlockForkEvent = /** @class */ (function (_super) {
    __extends(BlockForkEvent, _super);
    function BlockForkEvent(blockHash, expiry) {
        var _this = this;
        if (!(0, bytes_1.isHexString)(blockHash, 32)) {
            logger.throwArgumentError("invalid blockHash", "blockHash", blockHash);
        }
        _this = _super.call(this, {
            _isForkEvent: true,
            _isBlockForkEvent: true,
            expiry: (expiry || 0),
            blockHash: blockHash
        }) || this;
        return _this;
    }
    return BlockForkEvent;
}(ForkEvent));
exports.BlockForkEvent = BlockForkEvent;
var TransactionForkEvent = /** @class */ (function (_super) {
    __extends(TransactionForkEvent, _super);
    function TransactionForkEvent(hash, expiry) {
        var _this = this;
        if (!(0, bytes_1.isHexString)(hash, 32)) {
            logger.throwArgumentError("invalid transaction hash", "hash", hash);
        }
        _this = _super.call(this, {
            _isForkEvent: true,
            _isTransactionForkEvent: true,
            expiry: (expiry || 0),
            hash: hash
        }) || this;
        return _this;
    }
    return TransactionForkEvent;
}(ForkEvent));
exports.TransactionForkEvent = TransactionForkEvent;
var TransactionOrderForkEvent = /** @class */ (function (_super) {
    __extends(TransactionOrderForkEvent, _super);
    function TransactionOrderForkEvent(beforeHash, afterHash, expiry) {
        var _this = this;
        if (!(0, bytes_1.isHexString)(beforeHash, 32)) {
            logger.throwArgumentError("invalid transaction hash", "beforeHash", beforeHash);
        }
        if (!(0, bytes_1.isHexString)(afterHash, 32)) {
            logger.throwArgumentError("invalid transaction hash", "afterHash", afterHash);
        }
        _this = _super.call(this, {
            _isForkEvent: true,
            _isTransactionOrderForkEvent: true,
            expiry: (expiry || 0),
            beforeHash: beforeHash,
            afterHash: afterHash
        }) || this;
        return _this;
    }
    return TransactionOrderForkEvent;
}(ForkEvent));
exports.TransactionOrderForkEvent = TransactionOrderForkEvent;
///////////////////////////////
// Exported Abstracts
var Provider = /** @class */ (function () {
    function Provider() {
        var _newTarget = this.constructor;
        logger.checkAbstract(_newTarget, Provider);
        (0, properties_1.defineReadOnly)(this, "_isProvider", true);
    }
    Provider.prototype.getFeeData = function () {
        return __awaiter(this, void 0, void 0, function () {
            var _a, block, gasPrice, lastBaseFeePerGas, maxFeePerGas, maxPriorityFeePerGas;
            return __generator(this, function (_b) {
                switch (_b.label) {
                    case 0: return [4 /*yield*/, (0, properties_1.resolveProperties)({
                            block: this.getBlock("latest"),
                            gasPrice: this.getGasPrice().catch(function (error) {
                                // @TODO: Why is this now failing on Calaveras?
                                //console.log(error);
                                return null;
                            })
                        })];
                    case 1:
                        _a = _b.sent(), block = _a.block, gasPrice = _a.gasPrice;
                        lastBaseFeePerGas = null, maxFeePerGas = null, maxPriorityFeePerGas = null;
                        if (block && block.baseFeePerGas) {
                            // We may want to compute this more accurately in the future,
                            // using the formula "check if the base fee is correct".
                            // See: https://eips.ethereum.org/EIPS/eip-1559
                            lastBaseFeePerGas = block.baseFeePerGas;
                            maxPriorityFeePerGas = bignumber_1.BigNumber.from("1500000000");
                            maxFeePerGas = block.baseFeePerGas.mul(2).add(maxPriorityFeePerGas);
                        }
                        return [2 /*return*/, { lastBaseFeePerGas: lastBaseFeePerGas, maxFeePerGas: maxFeePerGas, maxPriorityFeePerGas: maxPriorityFeePerGas, gasPrice: gasPrice }];
                }
            });
        });
    };
    // Alias for "on"
    Provider.prototype.addListener = function (eventName, listener) {
        return this.on(eventName, listener);
    };
    // Alias for "off"
    Provider.prototype.removeListener = function (eventName, listener) {
        return this.off(eventName, listener);
    };
    Provider.isProvider = function (value) {
        return !!(value && value._isProvider);
    };
    return Provider;
}());
exports.Provider = Provider;
//# sourceMappingURL=index.js.map