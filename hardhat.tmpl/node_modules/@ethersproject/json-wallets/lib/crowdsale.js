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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.decrypt = exports.CrowdsaleAccount = void 0;
var aes_js_1 = __importDefault(require("aes-js"));
var address_1 = require("@ethersproject/address");
var bytes_1 = require("@ethersproject/bytes");
var keccak256_1 = require("@ethersproject/keccak256");
var pbkdf2_1 = require("@ethersproject/pbkdf2");
var strings_1 = require("@ethersproject/strings");
var properties_1 = require("@ethersproject/properties");
var logger_1 = require("@ethersproject/logger");
var _version_1 = require("./_version");
var logger = new logger_1.Logger(_version_1.version);
var utils_1 = require("./utils");
var CrowdsaleAccount = /** @class */ (function (_super) {
    __extends(CrowdsaleAccount, _super);
    function CrowdsaleAccount() {
        return _super !== null && _super.apply(this, arguments) || this;
    }
    CrowdsaleAccount.prototype.isCrowdsaleAccount = function (value) {
        return !!(value && value._isCrowdsaleAccount);
    };
    return CrowdsaleAccount;
}(properties_1.Description));
exports.CrowdsaleAccount = CrowdsaleAccount;
// See: https://github.com/ethereum/pyethsaletool
function decrypt(json, password) {
    var data = JSON.parse(json);
    password = (0, utils_1.getPassword)(password);
    // Ethereum Address
    var ethaddr = (0, address_1.getAddress)((0, utils_1.searchPath)(data, "ethaddr"));
    // Encrypted Seed
    var encseed = (0, utils_1.looseArrayify)((0, utils_1.searchPath)(data, "encseed"));
    if (!encseed || (encseed.length % 16) !== 0) {
        logger.throwArgumentError("invalid encseed", "json", json);
    }
    var key = (0, bytes_1.arrayify)((0, pbkdf2_1.pbkdf2)(password, password, 2000, 32, "sha256")).slice(0, 16);
    var iv = encseed.slice(0, 16);
    var encryptedSeed = encseed.slice(16);
    // Decrypt the seed
    var aesCbc = new aes_js_1.default.ModeOfOperation.cbc(key, iv);
    var seed = aes_js_1.default.padding.pkcs7.strip((0, bytes_1.arrayify)(aesCbc.decrypt(encryptedSeed)));
    // This wallet format is weird... Convert the binary encoded hex to a string.
    var seedHex = "";
    for (var i = 0; i < seed.length; i++) {
        seedHex += String.fromCharCode(seed[i]);
    }
    var seedHexBytes = (0, strings_1.toUtf8Bytes)(seedHex);
    var privateKey = (0, keccak256_1.keccak256)(seedHexBytes);
    return new CrowdsaleAccount({
        _isCrowdsaleAccount: true,
        address: ethaddr,
        privateKey: privateKey
    });
}
exports.decrypt = decrypt;
//# sourceMappingURL=crowdsale.js.map