"use strict";
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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.getUrl = void 0;
var http_1 = __importDefault(require("http"));
var https_1 = __importDefault(require("https"));
var zlib_1 = require("zlib");
var url_1 = require("url");
var bytes_1 = require("@ethersproject/bytes");
var properties_1 = require("@ethersproject/properties");
var logger_1 = require("@ethersproject/logger");
var _version_1 = require("./_version");
var logger = new logger_1.Logger(_version_1.version);
function getResponse(request) {
    return new Promise(function (resolve, reject) {
        request.once("response", function (resp) {
            var response = {
                statusCode: resp.statusCode,
                statusMessage: resp.statusMessage,
                headers: Object.keys(resp.headers).reduce(function (accum, name) {
                    var value = resp.headers[name];
                    if (Array.isArray(value)) {
                        value = value.join(", ");
                    }
                    accum[name] = value;
                    return accum;
                }, {}),
                body: null
            };
            //resp.setEncoding("utf8");
            resp.on("data", function (chunk) {
                if (response.body == null) {
                    response.body = new Uint8Array(0);
                }
                response.body = (0, bytes_1.concat)([response.body, chunk]);
            });
            resp.on("end", function () {
                if (response.headers["content-encoding"] === "gzip") {
                    //const size = response.body.length;
                    response.body = (0, bytes_1.arrayify)((0, zlib_1.gunzipSync)(response.body));
                    //console.log("Delta:", response.body.length - size, Buffer.from(response.body).toString());
                }
                resolve(response);
            });
            resp.on("error", function (error) {
                /* istanbul ignore next */
                error.response = response;
                reject(error);
            });
        });
        request.on("error", function (error) { reject(error); });
    });
}
// The URL.parse uses null instead of the empty string
function nonnull(value) {
    if (value == null) {
        return "";
    }
    return value;
}
function getUrl(href, options) {
    return __awaiter(this, void 0, void 0, function () {
        var url, request, req, response;
        return __generator(this, function (_a) {
            switch (_a.label) {
                case 0:
                    if (options == null) {
                        options = {};
                    }
                    url = (0, url_1.parse)(href);
                    request = {
                        protocol: nonnull(url.protocol),
                        hostname: nonnull(url.hostname),
                        port: nonnull(url.port),
                        path: (nonnull(url.pathname) + nonnull(url.search)),
                        method: (options.method || "GET"),
                        headers: (0, properties_1.shallowCopy)(options.headers || {}),
                    };
                    if (options.allowGzip) {
                        request.headers["accept-encoding"] = "gzip";
                    }
                    req = null;
                    switch (nonnull(url.protocol)) {
                        case "http:":
                            req = http_1.default.request(request);
                            break;
                        case "https:":
                            req = https_1.default.request(request);
                            break;
                        default:
                            /* istanbul ignore next */
                            logger.throwError("unsupported protocol " + url.protocol, logger_1.Logger.errors.UNSUPPORTED_OPERATION, {
                                protocol: url.protocol,
                                operation: "request"
                            });
                    }
                    if (options.body) {
                        req.write(Buffer.from(options.body));
                    }
                    req.end();
                    return [4 /*yield*/, getResponse(req)];
                case 1:
                    response = _a.sent();
                    return [2 /*return*/, response];
            }
        });
    });
}
exports.getUrl = getUrl;
//# sourceMappingURL=geturl.js.map