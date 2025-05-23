"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getProvider = exports.fetchFromProvider = void 0;
const micro_ftch_1 = require("micro-ftch");
const fetchFromProvider = async (url, params) => {
    const res = await (0, micro_ftch_1.default)(url, {
        headers: {
            'content-type': 'application/json',
        },
        type: 'json',
        data: {
            method: params.method,
            params: params.params,
            jsonrpc: '2.0',
            id: 1,
        },
    });
    return res.result;
};
exports.fetchFromProvider = fetchFromProvider;
const getProvider = (provider) => {
    if (typeof provider === 'string') {
        return provider;
    }
    else if (provider?.connection?.url !== undefined) {
        return provider.connection.url;
    }
    else {
        throw new Error('Must provide valid provider URL or Web3Provider');
    }
};
exports.getProvider = getProvider;
//# sourceMappingURL=provider.js.map