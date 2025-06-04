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
exports.UrlJsonRpcProvider = exports.StaticJsonRpcProvider = void 0;
var properties_1 = require("@ethersproject/properties");
var logger_1 = require("@ethersproject/logger");
var _version_1 = require("./_version");
var logger = new logger_1.Logger(_version_1.version);
var json_rpc_provider_1 = require("./json-rpc-provider");
// A StaticJsonRpcProvider is useful when you *know* for certain that
// the backend will never change, as it never calls eth_chainId to
// verify its backend. However, if the backend does change, the effects
// are undefined and may include:
// - inconsistent results
// - locking up the UI
// - block skew warnings
// - wrong results
// If the network is not explicit (i.e. auto-detection is expected), the
// node MUST be running and available to respond to requests BEFORE this
// is instantiated.
var StaticJsonRpcProvider = /** @class */ (function (_super) {
    __extends(StaticJsonRpcProvider, _super);
    function StaticJsonRpcProvider() {
        return _super !== null && _super.apply(this, arguments) || this;
    }
    StaticJsonRpcProvider.prototype.detectNetwork = function () {
        return __awaiter(this, void 0, void 0, function () {
            var network;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        network = this.network;
                        if (!(network == null)) return [3 /*break*/, 2];
                        return [4 /*yield*/, _super.prototype.detectNetwork.call(this)];
                    case 1:
                        network = _a.sent();
                        if (!network) {
                            logger.throwError("no network detected", logger_1.Logger.errors.UNKNOWN_ERROR, {});
                        }
                        // If still not set, set it
                        if (this._network == null) {
                            // A static network does not support "any"
                            (0, properties_1.defineReadOnly)(this, "_network", network);
                            this.emit("network", network, null);
                        }
                        _a.label = 2;
                    case 2: return [2 /*return*/, network];
                }
            });
        });
    };
    return StaticJsonRpcProvider;
}(json_rpc_provider_1.JsonRpcProvider));
exports.StaticJsonRpcProvider = StaticJsonRpcProvider;
var UrlJsonRpcProvider = /** @class */ (function (_super) {
    __extends(UrlJsonRpcProvider, _super);
    function UrlJsonRpcProvider(network, apiKey) {
        var _newTarget = this.constructor;
        var _this = this;
        logger.checkAbstract(_newTarget, UrlJsonRpcProvider);
        // Normalize the Network and API Key
        network = (0, properties_1.getStatic)(_newTarget, "getNetwork")(network);
        apiKey = (0, properties_1.getStatic)(_newTarget, "getApiKey")(apiKey);
        var connection = (0, properties_1.getStatic)(_newTarget, "getUrl")(network, apiKey);
        _this = _super.call(this, connection, network) || this;
        if (typeof (apiKey) === "string") {
            (0, properties_1.defineReadOnly)(_this, "apiKey", apiKey);
        }
        else if (apiKey != null) {
            Object.keys(apiKey).forEach(function (key) {
                (0, properties_1.defineReadOnly)(_this, key, apiKey[key]);
            });
        }
        return _this;
    }
    UrlJsonRpcProvider.prototype._startPending = function () {
        logger.warn("WARNING: API provider does not support pending filters");
    };
    UrlJsonRpcProvider.prototype.isCommunityResource = function () {
        return false;
    };
    UrlJsonRpcProvider.prototype.getSigner = function (address) {
        return logger.throwError("API provider does not support signing", logger_1.Logger.errors.UNSUPPORTED_OPERATION, { operation: "getSigner" });
    };
    UrlJsonRpcProvider.prototype.listAccounts = function () {
        return Promise.resolve([]);
    };
    // Return a defaultApiKey if null, otherwise validate the API key
    UrlJsonRpcProvider.getApiKey = function (apiKey) {
        return apiKey;
    };
    // Returns the url or connection for the given network and API key. The
    // API key will have been sanitized by the getApiKey first, so any validation
    // or transformations can be done there.
    UrlJsonRpcProvider.getUrl = function (network, apiKey) {
        return logger.throwError("not implemented; sub-classes must override getUrl", logger_1.Logger.errors.NOT_IMPLEMENTED, {
            operation: "getUrl"
        });
    };
    return UrlJsonRpcProvider;
}(StaticJsonRpcProvider));
exports.UrlJsonRpcProvider = UrlJsonRpcProvider;
//# sourceMappingURL=url-json-rpc-provider.js.map