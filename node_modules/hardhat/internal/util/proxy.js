"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.shouldUseProxy = void 0;
function shouldUseProxy(url) {
    const { hostname } = new URL(url);
    const noProxy = process.env.NO_PROXY;
    if (hostname === "localhost" || hostname === "127.0.0.1" || noProxy === "*") {
        return false;
    }
    if (noProxy !== undefined && noProxy !== "") {
        const noProxyList = noProxy.split(",");
        if (noProxyList.includes(hostname)) {
            return false;
        }
    }
    return true;
}
exports.shouldUseProxy = shouldUseProxy;
//# sourceMappingURL=proxy.js.map