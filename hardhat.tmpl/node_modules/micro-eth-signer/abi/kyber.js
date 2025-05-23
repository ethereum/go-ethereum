"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.KYBER_NETWORK_PROXY_CONTRACT = void 0;
const utils_ts_1 = require("../utils.js");
const common_ts_1 = require("./common.js");
// prettier-ignore
const _ABI = [
    { type: "function", name: "getExpectedRate", inputs: [{ name: "src", type: "address" }, { name: "dest", type: "address" }, { name: "srcQty", type: "uint256" }], outputs: [{ name: "expectedRate", type: "uint256" }, { name: "worstRate", type: "uint256" }] }, { type: "function", name: "getExpectedRateAfterFee", inputs: [{ name: "src", type: "address" }, { name: "dest", type: "address" }, { name: "srcQty", type: "uint256" }, { name: "platformFeeBps", type: "uint256" }, { name: "hint", type: "bytes" }], outputs: [{ name: "expectedRate", type: "uint256" }] }, { type: "function", name: "trade", inputs: [{ name: "src", type: "address" }, { name: "srcAmount", type: "uint256" }, { name: "dest", type: "address" }, { name: "destAddress", type: "address" }, { name: "maxDestAmount", type: "uint256" }, { name: "minConversionRate", type: "uint256" }, { name: "platformWallet", type: "address" }], outputs: [{ type: "uint256" }] }, { type: "function", name: "tradeWithHint", inputs: [{ name: "src", type: "address" }, { name: "srcAmount", type: "uint256" }, { name: "dest", type: "address" }, { name: "destAddress", type: "address" }, { name: "maxDestAmount", type: "uint256" }, { name: "minConversionRate", type: "uint256" }, { name: "walletId", type: "address" }, { name: "hint", type: "bytes" }], outputs: [{ type: "uint256" }] }, { type: "function", name: "tradeWithHintAndFee", inputs: [{ name: "src", type: "address" }, { name: "srcAmount", type: "uint256" }, { name: "dest", type: "address" }, { name: "destAddress", type: "address" }, { name: "maxDestAmount", type: "uint256" }, { name: "minConversionRate", type: "uint256" }, { name: "platformWallet", type: "address" }, { name: "platformFeeBps", type: "uint256" }, { name: "hint", type: "bytes" }], outputs: [{ name: "destAmount", type: "uint256" }] }
];
const _10n = BigInt(10);
const hints = {
    tradeWithHintAndFee(v, opt) {
        if (!opt.contracts)
            throw Error('Not enough info');
        const tokenInfo = (c) => c === '0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee'
            ? { symbol: 'ETH', decimals: 18 }
            : opt.contracts[c];
        const formatToken = (amount, info) => `${(0, utils_ts_1.createDecimal)(info.decimals).encode(amount)} ${info.symbol}`;
        const [srcInfo, destInfo] = [tokenInfo(v.src), tokenInfo(v.dest)];
        if (!srcInfo || !destInfo)
            throw Error('Not enough info');
        const destAmount = (v.srcAmount *
            v.minConversionRate *
            _10n ** BigInt(destInfo.decimals)) /
            _10n ** (BigInt(srcInfo.decimals) + BigInt(18));
        const fee = formatToken((BigInt(v.platformFeeBps) * BigInt(v.srcAmount)) / BigInt(10000), srcInfo);
        return `Swap ${formatToken(v.srcAmount, srcInfo)} For ${formatToken(destAmount, destInfo)} (with platform fee: ${fee})`;
    },
};
const ABI = /* @__PURE__ */ (0, common_ts_1.addHints)(_ABI, hints);
exports.default = ABI;
exports.KYBER_NETWORK_PROXY_CONTRACT = '0x9aab3f75489902f3a48495025729a0af77d4b11e';
//# sourceMappingURL=kyber.js.map