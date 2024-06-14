"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports._base16To36 = exports._base36To16 = exports.BigNumber = exports.isBigNumberish = void 0;
/**
 *  BigNumber
 *
 *  A wrapper around the BN.js object. We use the BN.js library
 *  because it is used by elliptic, so it is required regardless.
 *
 */
var bn_js_1 = __importDefault(require("bn.js"));
var BN = bn_js_1.default.BN;
var bytes_1 = require("@ethersproject/bytes");
var logger_1 = require("@ethersproject/logger");
var _version_1 = require("./_version");
var logger = new logger_1.Logger(_version_1.version);
var _constructorGuard = {};
var MAX_SAFE = 0x1fffffffffffff;
function isBigNumberish(value) {
    return (value != null) && (BigNumber.isBigNumber(value) ||
        (typeof (value) === "number" && (value % 1) === 0) ||
        (typeof (value) === "string" && !!value.match(/^-?[0-9]+$/)) ||
        (0, bytes_1.isHexString)(value) ||
        (typeof (value) === "bigint") ||
        (0, bytes_1.isBytes)(value));
}
exports.isBigNumberish = isBigNumberish;
// Only warn about passing 10 into radix once
var _warnedToStringRadix = false;
var BigNumber = /** @class */ (function () {
    function BigNumber(constructorGuard, hex) {
        if (constructorGuard !== _constructorGuard) {
            logger.throwError("cannot call constructor directly; use BigNumber.from", logger_1.Logger.errors.UNSUPPORTED_OPERATION, {
                operation: "new (BigNumber)"
            });
        }
        this._hex = hex;
        this._isBigNumber = true;
        Object.freeze(this);
    }
    BigNumber.prototype.fromTwos = function (value) {
        return toBigNumber(toBN(this).fromTwos(value));
    };
    BigNumber.prototype.toTwos = function (value) {
        return toBigNumber(toBN(this).toTwos(value));
    };
    BigNumber.prototype.abs = function () {
        if (this._hex[0] === "-") {
            return BigNumber.from(this._hex.substring(1));
        }
        return this;
    };
    BigNumber.prototype.add = function (other) {
        return toBigNumber(toBN(this).add(toBN(other)));
    };
    BigNumber.prototype.sub = function (other) {
        return toBigNumber(toBN(this).sub(toBN(other)));
    };
    BigNumber.prototype.div = function (other) {
        var o = BigNumber.from(other);
        if (o.isZero()) {
            throwFault("division-by-zero", "div");
        }
        return toBigNumber(toBN(this).div(toBN(other)));
    };
    BigNumber.prototype.mul = function (other) {
        return toBigNumber(toBN(this).mul(toBN(other)));
    };
    BigNumber.prototype.mod = function (other) {
        var value = toBN(other);
        if (value.isNeg()) {
            throwFault("division-by-zero", "mod");
        }
        return toBigNumber(toBN(this).umod(value));
    };
    BigNumber.prototype.pow = function (other) {
        var value = toBN(other);
        if (value.isNeg()) {
            throwFault("negative-power", "pow");
        }
        return toBigNumber(toBN(this).pow(value));
    };
    BigNumber.prototype.and = function (other) {
        var value = toBN(other);
        if (this.isNegative() || value.isNeg()) {
            throwFault("unbound-bitwise-result", "and");
        }
        return toBigNumber(toBN(this).and(value));
    };
    BigNumber.prototype.or = function (other) {
        var value = toBN(other);
        if (this.isNegative() || value.isNeg()) {
            throwFault("unbound-bitwise-result", "or");
        }
        return toBigNumber(toBN(this).or(value));
    };
    BigNumber.prototype.xor = function (other) {
        var value = toBN(other);
        if (this.isNegative() || value.isNeg()) {
            throwFault("unbound-bitwise-result", "xor");
        }
        return toBigNumber(toBN(this).xor(value));
    };
    BigNumber.prototype.mask = function (value) {
        if (this.isNegative() || value < 0) {
            throwFault("negative-width", "mask");
        }
        return toBigNumber(toBN(this).maskn(value));
    };
    BigNumber.prototype.shl = function (value) {
        if (this.isNegative() || value < 0) {
            throwFault("negative-width", "shl");
        }
        return toBigNumber(toBN(this).shln(value));
    };
    BigNumber.prototype.shr = function (value) {
        if (this.isNegative() || value < 0) {
            throwFault("negative-width", "shr");
        }
        return toBigNumber(toBN(this).shrn(value));
    };
    BigNumber.prototype.eq = function (other) {
        return toBN(this).eq(toBN(other));
    };
    BigNumber.prototype.lt = function (other) {
        return toBN(this).lt(toBN(other));
    };
    BigNumber.prototype.lte = function (other) {
        return toBN(this).lte(toBN(other));
    };
    BigNumber.prototype.gt = function (other) {
        return toBN(this).gt(toBN(other));
    };
    BigNumber.prototype.gte = function (other) {
        return toBN(this).gte(toBN(other));
    };
    BigNumber.prototype.isNegative = function () {
        return (this._hex[0] === "-");
    };
    BigNumber.prototype.isZero = function () {
        return toBN(this).isZero();
    };
    BigNumber.prototype.toNumber = function () {
        try {
            return toBN(this).toNumber();
        }
        catch (error) {
            throwFault("overflow", "toNumber", this.toString());
        }
        return null;
    };
    BigNumber.prototype.toBigInt = function () {
        try {
            return BigInt(this.toString());
        }
        catch (e) { }
        return logger.throwError("this platform does not support BigInt", logger_1.Logger.errors.UNSUPPORTED_OPERATION, {
            value: this.toString()
        });
    };
    BigNumber.prototype.toString = function () {
        // Lots of people expect this, which we do not support, so check (See: #889)
        if (arguments.length > 0) {
            if (arguments[0] === 10) {
                if (!_warnedToStringRadix) {
                    _warnedToStringRadix = true;
                    logger.warn("BigNumber.toString does not accept any parameters; base-10 is assumed");
                }
            }
            else if (arguments[0] === 16) {
                logger.throwError("BigNumber.toString does not accept any parameters; use bigNumber.toHexString()", logger_1.Logger.errors.UNEXPECTED_ARGUMENT, {});
            }
            else {
                logger.throwError("BigNumber.toString does not accept parameters", logger_1.Logger.errors.UNEXPECTED_ARGUMENT, {});
            }
        }
        return toBN(this).toString(10);
    };
    BigNumber.prototype.toHexString = function () {
        return this._hex;
    };
    BigNumber.prototype.toJSON = function (key) {
        return { type: "BigNumber", hex: this.toHexString() };
    };
    BigNumber.from = function (value) {
        if (value instanceof BigNumber) {
            return value;
        }
        if (typeof (value) === "string") {
            if (value.match(/^-?0x[0-9a-f]+$/i)) {
                return new BigNumber(_constructorGuard, toHex(value));
            }
            if (value.match(/^-?[0-9]+$/)) {
                return new BigNumber(_constructorGuard, toHex(new BN(value)));
            }
            return logger.throwArgumentError("invalid BigNumber string", "value", value);
        }
        if (typeof (value) === "number") {
            if (value % 1) {
                throwFault("underflow", "BigNumber.from", value);
            }
            if (value >= MAX_SAFE || value <= -MAX_SAFE) {
                throwFault("overflow", "BigNumber.from", value);
            }
            return BigNumber.from(String(value));
        }
        var anyValue = value;
        if (typeof (anyValue) === "bigint") {
            return BigNumber.from(anyValue.toString());
        }
        if ((0, bytes_1.isBytes)(anyValue)) {
            return BigNumber.from((0, bytes_1.hexlify)(anyValue));
        }
        if (anyValue) {
            // Hexable interface (takes priority)
            if (anyValue.toHexString) {
                var hex = anyValue.toHexString();
                if (typeof (hex) === "string") {
                    return BigNumber.from(hex);
                }
            }
            else {
                // For now, handle legacy JSON-ified values (goes away in v6)
                var hex = anyValue._hex;
                // New-form JSON
                if (hex == null && anyValue.type === "BigNumber") {
                    hex = anyValue.hex;
                }
                if (typeof (hex) === "string") {
                    if ((0, bytes_1.isHexString)(hex) || (hex[0] === "-" && (0, bytes_1.isHexString)(hex.substring(1)))) {
                        return BigNumber.from(hex);
                    }
                }
            }
        }
        return logger.throwArgumentError("invalid BigNumber value", "value", value);
    };
    BigNumber.isBigNumber = function (value) {
        return !!(value && value._isBigNumber);
    };
    return BigNumber;
}());
exports.BigNumber = BigNumber;
// Normalize the hex string
function toHex(value) {
    // For BN, call on the hex string
    if (typeof (value) !== "string") {
        return toHex(value.toString(16));
    }
    // If negative, prepend the negative sign to the normalized positive value
    if (value[0] === "-") {
        // Strip off the negative sign
        value = value.substring(1);
        // Cannot have multiple negative signs (e.g. "--0x04")
        if (value[0] === "-") {
            logger.throwArgumentError("invalid hex", "value", value);
        }
        // Call toHex on the positive component
        value = toHex(value);
        // Do not allow "-0x00"
        if (value === "0x00") {
            return value;
        }
        // Negate the value
        return "-" + value;
    }
    // Add a "0x" prefix if missing
    if (value.substring(0, 2) !== "0x") {
        value = "0x" + value;
    }
    // Normalize zero
    if (value === "0x") {
        return "0x00";
    }
    // Make the string even length
    if (value.length % 2) {
        value = "0x0" + value.substring(2);
    }
    // Trim to smallest even-length string
    while (value.length > 4 && value.substring(0, 4) === "0x00") {
        value = "0x" + value.substring(4);
    }
    return value;
}
function toBigNumber(value) {
    return BigNumber.from(toHex(value));
}
function toBN(value) {
    var hex = BigNumber.from(value).toHexString();
    if (hex[0] === "-") {
        return (new BN("-" + hex.substring(3), 16));
    }
    return new BN(hex.substring(2), 16);
}
function throwFault(fault, operation, value) {
    var params = { fault: fault, operation: operation };
    if (value != null) {
        params.value = value;
    }
    return logger.throwError(fault, logger_1.Logger.errors.NUMERIC_FAULT, params);
}
// value should have no prefix
function _base36To16(value) {
    return (new BN(value, 36)).toString(16);
}
exports._base36To16 = _base36To16;
// value should have no prefix
function _base16To36(value) {
    return (new BN(value, 16)).toString(36);
}
exports._base16To36 = _base16To36;
//# sourceMappingURL=bignumber.js.map