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
exports.download = void 0;
const fs_extra_1 = __importDefault(require("fs-extra"));
const path_1 = __importDefault(require("path"));
const packageInfo_1 = require("./packageInfo");
const proxy_1 = require("./proxy");
const TEMP_FILE_PREFIX = "tmp-";
function resolveTempFileName(filePath) {
    const { dir, ext, name } = path_1.default.parse(filePath);
    return path_1.default.format({
        dir,
        ext,
        name: `${TEMP_FILE_PREFIX}${name}`,
    });
}
async function download(url, filePath, timeoutMillis = 10000, extraHeaders = {}) {
    const { getGlobalDispatcher, ProxyAgent, request } = await Promise.resolve().then(() => __importStar(require("undici")));
    let dispatcher;
    if (process.env.http_proxy !== undefined && (0, proxy_1.shouldUseProxy)(url)) {
        dispatcher = new ProxyAgent(process.env.http_proxy);
    }
    else {
        dispatcher = getGlobalDispatcher();
    }
    const hardhatVersion = (0, packageInfo_1.getHardhatVersion)();
    // Fetch the url
    const response = await request(url, {
        dispatcher,
        headersTimeout: timeoutMillis,
        maxRedirections: 10,
        method: "GET",
        headers: {
            ...extraHeaders,
            "User-Agent": `hardhat ${hardhatVersion}`,
        },
    });
    if (response.statusCode >= 200 && response.statusCode <= 299) {
        const responseBody = Buffer.from(await response.body.arrayBuffer());
        const tmpFilePath = resolveTempFileName(filePath);
        await fs_extra_1.default.ensureDir(path_1.default.dirname(filePath));
        await fs_extra_1.default.writeFile(tmpFilePath, responseBody);
        return fs_extra_1.default.move(tmpFilePath, filePath, { overwrite: true });
    }
    // undici's response bodies must always be consumed to prevent leaks
    const text = await response.body.text();
    // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
    throw new Error(`Failed to download ${url} - ${response.statusCode} received. ${text}`);
}
exports.download = download;
//# sourceMappingURL=download.js.map