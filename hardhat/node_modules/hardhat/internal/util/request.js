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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.requestJson = void 0;
const debug_1 = __importDefault(require("debug"));
const log = (0, debug_1.default)("hardhat:util:request");
async function requestJson(url, timeout) {
    const { request } = await Promise.resolve().then(() => __importStar(require("undici")));
    const controller = new AbortController();
    const requestAborted = new Error("Request aborted: timeout reached");
    let timeoutId;
    if (timeout !== undefined) {
        timeoutId = setTimeout(() => {
            controller.abort(requestAborted);
        }, timeout);
    }
    try {
        const response = await request(url, {
            method: "GET",
            signal: controller.signal,
        });
        if (response.statusCode !== 200) {
            /* eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
            -- this is going to be captured by the catch block and logged */
            throw new Error(`Request failed with status code ${response.statusCode}`);
        }
        const jsonResponse = await response.body.json();
        return jsonResponse;
    }
    catch (error) {
        if (error === requestAborted) {
            log(`Request to ${url} aborted after ${timeout}ms`);
        }
        else {
            log(`Request to ${url} failed: ${error instanceof Error ? error.message : JSON.stringify(error)}`);
        }
    }
    finally {
        if (timeoutId !== undefined) {
            clearTimeout(timeoutId);
        }
    }
}
exports.requestJson = requestJson;
//# sourceMappingURL=request.js.map