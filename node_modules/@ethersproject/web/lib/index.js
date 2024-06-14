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
Object.defineProperty(exports, "__esModule", { value: true });
exports.poll = exports.fetchJson = exports._fetchData = void 0;
var base64_1 = require("@ethersproject/base64");
var bytes_1 = require("@ethersproject/bytes");
var properties_1 = require("@ethersproject/properties");
var strings_1 = require("@ethersproject/strings");
var logger_1 = require("@ethersproject/logger");
var _version_1 = require("./_version");
var logger = new logger_1.Logger(_version_1.version);
var geturl_1 = require("./geturl");
function staller(duration) {
    return new Promise(function (resolve) {
        setTimeout(resolve, duration);
    });
}
function bodyify(value, type) {
    if (value == null) {
        return null;
    }
    if (typeof (value) === "string") {
        return value;
    }
    if ((0, bytes_1.isBytesLike)(value)) {
        if (type && (type.split("/")[0] === "text" || type.split(";")[0].trim() === "application/json")) {
            try {
                return (0, strings_1.toUtf8String)(value);
            }
            catch (error) { }
            ;
        }
        return (0, bytes_1.hexlify)(value);
    }
    return value;
}
function unpercent(value) {
    return (0, strings_1.toUtf8Bytes)(value.replace(/%([0-9a-f][0-9a-f])/gi, function (all, code) {
        return String.fromCharCode(parseInt(code, 16));
    }));
}
// This API is still a work in progress; the future changes will likely be:
// - ConnectionInfo => FetchDataRequest<T = any>
// - FetchDataRequest.body? = string | Uint8Array | { contentType: string, data: string | Uint8Array }
//   - If string => text/plain, Uint8Array => application/octet-stream (if content-type unspecified)
// - FetchDataRequest.processFunc = (body: Uint8Array, response: FetchDataResponse) => T
// For this reason, it should be considered internal until the API is finalized
function _fetchData(connection, body, processFunc) {
    // How many times to retry in the event of a throttle
    var attemptLimit = (typeof (connection) === "object" && connection.throttleLimit != null) ? connection.throttleLimit : 12;
    logger.assertArgument((attemptLimit > 0 && (attemptLimit % 1) === 0), "invalid connection throttle limit", "connection.throttleLimit", attemptLimit);
    var throttleCallback = ((typeof (connection) === "object") ? connection.throttleCallback : null);
    var throttleSlotInterval = ((typeof (connection) === "object" && typeof (connection.throttleSlotInterval) === "number") ? connection.throttleSlotInterval : 100);
    logger.assertArgument((throttleSlotInterval > 0 && (throttleSlotInterval % 1) === 0), "invalid connection throttle slot interval", "connection.throttleSlotInterval", throttleSlotInterval);
    var errorPassThrough = ((typeof (connection) === "object") ? !!(connection.errorPassThrough) : false);
    var headers = {};
    var url = null;
    // @TODO: Allow ConnectionInfo to override some of these values
    var options = {
        method: "GET",
    };
    var allow304 = false;
    var timeout = 2 * 60 * 1000;
    if (typeof (connection) === "string") {
        url = connection;
    }
    else if (typeof (connection) === "object") {
        if (connection == null || connection.url == null) {
            logger.throwArgumentError("missing URL", "connection.url", connection);
        }
        url = connection.url;
        if (typeof (connection.timeout) === "number" && connection.timeout > 0) {
            timeout = connection.timeout;
        }
        if (connection.headers) {
            for (var key in connection.headers) {
                headers[key.toLowerCase()] = { key: key, value: String(connection.headers[key]) };
                if (["if-none-match", "if-modified-since"].indexOf(key.toLowerCase()) >= 0) {
                    allow304 = true;
                }
            }
        }
        options.allowGzip = !!connection.allowGzip;
        if (connection.user != null && connection.password != null) {
            if (url.substring(0, 6) !== "https:" && connection.allowInsecureAuthentication !== true) {
                logger.throwError("basic authentication requires a secure https url", logger_1.Logger.errors.INVALID_ARGUMENT, { argument: "url", url: url, user: connection.user, password: "[REDACTED]" });
            }
            var authorization = connection.user + ":" + connection.password;
            headers["authorization"] = {
                key: "Authorization",
                value: "Basic " + (0, base64_1.encode)((0, strings_1.toUtf8Bytes)(authorization))
            };
        }
        if (connection.skipFetchSetup != null) {
            options.skipFetchSetup = !!connection.skipFetchSetup;
        }
        if (connection.fetchOptions != null) {
            options.fetchOptions = (0, properties_1.shallowCopy)(connection.fetchOptions);
        }
    }
    var reData = new RegExp("^data:([^;:]*)?(;base64)?,(.*)$", "i");
    var dataMatch = ((url) ? url.match(reData) : null);
    if (dataMatch) {
        try {
            var response = {
                statusCode: 200,
                statusMessage: "OK",
                headers: { "content-type": (dataMatch[1] || "text/plain") },
                body: (dataMatch[2] ? (0, base64_1.decode)(dataMatch[3]) : unpercent(dataMatch[3]))
            };
            var result = response.body;
            if (processFunc) {
                result = processFunc(response.body, response);
            }
            return Promise.resolve(result);
        }
        catch (error) {
            logger.throwError("processing response error", logger_1.Logger.errors.SERVER_ERROR, {
                body: bodyify(dataMatch[1], dataMatch[2]),
                error: error,
                requestBody: null,
                requestMethod: "GET",
                url: url
            });
        }
    }
    if (body) {
        options.method = "POST";
        options.body = body;
        if (headers["content-type"] == null) {
            headers["content-type"] = { key: "Content-Type", value: "application/octet-stream" };
        }
        if (headers["content-length"] == null) {
            headers["content-length"] = { key: "Content-Length", value: String(body.length) };
        }
    }
    var flatHeaders = {};
    Object.keys(headers).forEach(function (key) {
        var header = headers[key];
        flatHeaders[header.key] = header.value;
    });
    options.headers = flatHeaders;
    var runningTimeout = (function () {
        var timer = null;
        var promise = new Promise(function (resolve, reject) {
            if (timeout) {
                timer = setTimeout(function () {
                    if (timer == null) {
                        return;
                    }
                    timer = null;
                    reject(logger.makeError("timeout", logger_1.Logger.errors.TIMEOUT, {
                        requestBody: bodyify(options.body, flatHeaders["content-type"]),
                        requestMethod: options.method,
                        timeout: timeout,
                        url: url
                    }));
                }, timeout);
            }
        });
        var cancel = function () {
            if (timer == null) {
                return;
            }
            clearTimeout(timer);
            timer = null;
        };
        return { promise: promise, cancel: cancel };
    })();
    var runningFetch = (function () {
        return __awaiter(this, void 0, void 0, function () {
            var attempt, response, location_1, tryAgain, stall, retryAfter, error_1, body_1, result, error_2, tryAgain, timeout_1;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        attempt = 0;
                        _a.label = 1;
                    case 1:
                        if (!(attempt < attemptLimit)) return [3 /*break*/, 20];
                        response = null;
                        _a.label = 2;
                    case 2:
                        _a.trys.push([2, 9, , 10]);
                        return [4 /*yield*/, (0, geturl_1.getUrl)(url, options)];
                    case 3:
                        response = _a.sent();
                        if (!(attempt < attemptLimit)) return [3 /*break*/, 8];
                        if (!(response.statusCode === 301 || response.statusCode === 302)) return [3 /*break*/, 4];
                        location_1 = response.headers.location || "";
                        if (options.method === "GET" && location_1.match(/^https:/)) {
                            url = response.headers.location;
                            return [3 /*break*/, 19];
                        }
                        return [3 /*break*/, 8];
                    case 4:
                        if (!(response.statusCode === 429)) return [3 /*break*/, 8];
                        tryAgain = true;
                        if (!throttleCallback) return [3 /*break*/, 6];
                        return [4 /*yield*/, throttleCallback(attempt, url)];
                    case 5:
                        tryAgain = _a.sent();
                        _a.label = 6;
                    case 6:
                        if (!tryAgain) return [3 /*break*/, 8];
                        stall = 0;
                        retryAfter = response.headers["retry-after"];
                        if (typeof (retryAfter) === "string" && retryAfter.match(/^[1-9][0-9]*$/)) {
                            stall = parseInt(retryAfter) * 1000;
                        }
                        else {
                            stall = throttleSlotInterval * parseInt(String(Math.random() * Math.pow(2, attempt)));
                        }
                        //console.log("Stalling 429");
                        return [4 /*yield*/, staller(stall)];
                    case 7:
                        //console.log("Stalling 429");
                        _a.sent();
                        return [3 /*break*/, 19];
                    case 8: return [3 /*break*/, 10];
                    case 9:
                        error_1 = _a.sent();
                        response = error_1.response;
                        if (response == null) {
                            runningTimeout.cancel();
                            logger.throwError("missing response", logger_1.Logger.errors.SERVER_ERROR, {
                                requestBody: bodyify(options.body, flatHeaders["content-type"]),
                                requestMethod: options.method,
                                serverError: error_1,
                                url: url
                            });
                        }
                        return [3 /*break*/, 10];
                    case 10:
                        body_1 = response.body;
                        if (allow304 && response.statusCode === 304) {
                            body_1 = null;
                        }
                        else if (!errorPassThrough && (response.statusCode < 200 || response.statusCode >= 300)) {
                            runningTimeout.cancel();
                            logger.throwError("bad response", logger_1.Logger.errors.SERVER_ERROR, {
                                status: response.statusCode,
                                headers: response.headers,
                                body: bodyify(body_1, ((response.headers) ? response.headers["content-type"] : null)),
                                requestBody: bodyify(options.body, flatHeaders["content-type"]),
                                requestMethod: options.method,
                                url: url
                            });
                        }
                        if (!processFunc) return [3 /*break*/, 18];
                        _a.label = 11;
                    case 11:
                        _a.trys.push([11, 13, , 18]);
                        return [4 /*yield*/, processFunc(body_1, response)];
                    case 12:
                        result = _a.sent();
                        runningTimeout.cancel();
                        return [2 /*return*/, result];
                    case 13:
                        error_2 = _a.sent();
                        if (!(error_2.throttleRetry && attempt < attemptLimit)) return [3 /*break*/, 17];
                        tryAgain = true;
                        if (!throttleCallback) return [3 /*break*/, 15];
                        return [4 /*yield*/, throttleCallback(attempt, url)];
                    case 14:
                        tryAgain = _a.sent();
                        _a.label = 15;
                    case 15:
                        if (!tryAgain) return [3 /*break*/, 17];
                        timeout_1 = throttleSlotInterval * parseInt(String(Math.random() * Math.pow(2, attempt)));
                        //console.log("Stalling callback");
                        return [4 /*yield*/, staller(timeout_1)];
                    case 16:
                        //console.log("Stalling callback");
                        _a.sent();
                        return [3 /*break*/, 19];
                    case 17:
                        runningTimeout.cancel();
                        logger.throwError("processing response error", logger_1.Logger.errors.SERVER_ERROR, {
                            body: bodyify(body_1, ((response.headers) ? response.headers["content-type"] : null)),
                            error: error_2,
                            requestBody: bodyify(options.body, flatHeaders["content-type"]),
                            requestMethod: options.method,
                            url: url
                        });
                        return [3 /*break*/, 18];
                    case 18:
                        runningTimeout.cancel();
                        // If we had a processFunc, it either returned a T or threw above.
                        // The "body" is now a Uint8Array.
                        return [2 /*return*/, body_1];
                    case 19:
                        attempt++;
                        return [3 /*break*/, 1];
                    case 20: return [2 /*return*/, logger.throwError("failed response", logger_1.Logger.errors.SERVER_ERROR, {
                            requestBody: bodyify(options.body, flatHeaders["content-type"]),
                            requestMethod: options.method,
                            url: url
                        })];
                }
            });
        });
    })();
    return Promise.race([runningTimeout.promise, runningFetch]);
}
exports._fetchData = _fetchData;
function fetchJson(connection, json, processFunc) {
    var processJsonFunc = function (value, response) {
        var result = null;
        if (value != null) {
            try {
                result = JSON.parse((0, strings_1.toUtf8String)(value));
            }
            catch (error) {
                logger.throwError("invalid JSON", logger_1.Logger.errors.SERVER_ERROR, {
                    body: value,
                    error: error
                });
            }
        }
        if (processFunc) {
            result = processFunc(result, response);
        }
        return result;
    };
    // If we have json to send, we must
    // - add content-type of application/json (unless already overridden)
    // - convert the json to bytes
    var body = null;
    if (json != null) {
        body = (0, strings_1.toUtf8Bytes)(json);
        // Create a connection with the content-type set for JSON
        var updated = (typeof (connection) === "string") ? ({ url: connection }) : (0, properties_1.shallowCopy)(connection);
        if (updated.headers) {
            var hasContentType = (Object.keys(updated.headers).filter(function (k) { return (k.toLowerCase() === "content-type"); }).length) !== 0;
            if (!hasContentType) {
                updated.headers = (0, properties_1.shallowCopy)(updated.headers);
                updated.headers["content-type"] = "application/json";
            }
        }
        else {
            updated.headers = { "content-type": "application/json" };
        }
        connection = updated;
    }
    return _fetchData(connection, body, processJsonFunc);
}
exports.fetchJson = fetchJson;
function poll(func, options) {
    if (!options) {
        options = {};
    }
    options = (0, properties_1.shallowCopy)(options);
    if (options.floor == null) {
        options.floor = 0;
    }
    if (options.ceiling == null) {
        options.ceiling = 10000;
    }
    if (options.interval == null) {
        options.interval = 250;
    }
    return new Promise(function (resolve, reject) {
        var timer = null;
        var done = false;
        // Returns true if cancel was successful. Unsuccessful cancel means we're already done.
        var cancel = function () {
            if (done) {
                return false;
            }
            done = true;
            if (timer) {
                clearTimeout(timer);
            }
            return true;
        };
        if (options.timeout) {
            timer = setTimeout(function () {
                if (cancel()) {
                    reject(new Error("timeout"));
                }
            }, options.timeout);
        }
        var retryLimit = options.retryLimit;
        var attempt = 0;
        function check() {
            return func().then(function (result) {
                // If we have a result, or are allowed null then we're done
                if (result !== undefined) {
                    if (cancel()) {
                        resolve(result);
                    }
                }
                else if (options.oncePoll) {
                    options.oncePoll.once("poll", check);
                }
                else if (options.onceBlock) {
                    options.onceBlock.once("block", check);
                    // Otherwise, exponential back-off (up to 10s) our next request
                }
                else if (!done) {
                    attempt++;
                    if (attempt > retryLimit) {
                        if (cancel()) {
                            reject(new Error("retry limit reached"));
                        }
                        return;
                    }
                    var timeout = options.interval * parseInt(String(Math.random() * Math.pow(2, attempt)));
                    if (timeout < options.floor) {
                        timeout = options.floor;
                    }
                    if (timeout > options.ceiling) {
                        timeout = options.ceiling;
                    }
                    setTimeout(check, timeout);
                }
                return null;
            }, function (error) {
                if (cancel()) {
                    reject(error);
                }
            });
        }
        check();
    });
}
exports.poll = poll;
//# sourceMappingURL=index.js.map