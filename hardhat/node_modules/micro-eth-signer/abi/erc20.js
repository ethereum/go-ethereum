"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.hints = exports.ABI = void 0;
const utils_ts_1 = require("../utils.js");
const common_ts_1 = require("./common.js");
// prettier-ignore
exports.ABI = [
    { type: "function", name: "name", outputs: [{ type: "string" }] }, { type: "function", name: "totalSupply", outputs: [{ type: "uint256" }] }, { type: "function", name: "decimals", outputs: [{ type: "uint8" }] }, { type: "function", name: "symbol", outputs: [{ type: "string" }] }, { type: "function", name: "approve", inputs: [{ name: "spender", type: "address" }, { name: "value", type: "uint256" }], outputs: [{ name: "success", type: "bool" }] }, { type: "function", name: "transferFrom", inputs: [{ name: "from", type: "address" }, { name: "to", type: "address" }, { name: "value", type: "uint256" }], outputs: [{ name: "success", type: "bool" }] }, { type: "function", name: "balances", inputs: [{ type: "address" }], outputs: [{ type: "uint256" }] }, { type: "function", name: "allowed", inputs: [{ type: "address" }, { type: "address" }], outputs: [{ type: "uint256" }] }, { type: "function", name: "balanceOf", inputs: [{ name: "owner", type: "address" }], outputs: [{ name: "balance", type: "uint256" }] }, { type: "function", name: "transfer", inputs: [{ name: "to", type: "address" }, { name: "value", type: "uint256" }], outputs: [{ name: "success", type: "bool" }] }, { type: "function", name: "allowance", inputs: [{ name: "owner", type: "address" }, { name: "spender", type: "address" }], outputs: [{ name: "remaining", type: "uint256" }] }, { name: "Approval", type: "event", anonymous: false, inputs: [{ indexed: true, name: "owner", type: "address" }, { indexed: true, name: "spender", type: "address" }, { indexed: false, name: "value", type: "uint256" }] }, { name: "Transfer", type: "event", anonymous: false, inputs: [{ indexed: true, name: "from", type: "address" }, { indexed: true, name: "to", type: "address" }, { indexed: false, name: "value", type: "uint256" }] }
];
// https://eips.ethereum.org/EIPS/eip-20
exports.hints = {
    approve(v, opt) {
        if (!opt.contractInfo || !opt.contractInfo.decimals || !opt.contractInfo.symbol)
            throw new Error('Not enough info');
        return `Allow spending ${(0, utils_ts_1.createDecimal)(opt.contractInfo.decimals).encode(v.value)} ${opt.contractInfo.symbol} by ${v.spender}`;
    },
    transferFrom(v, opt) {
        if (!opt.contractInfo || !opt.contractInfo.decimals || !opt.contractInfo.symbol)
            throw new Error('Not enough info');
        return `Transfer ${(0, utils_ts_1.createDecimal)(opt.contractInfo.decimals).encode(v.value)} ${opt.contractInfo.symbol} from ${v.from} to ${v.to}`;
    },
    transfer(v, opt) {
        if (!opt.contractInfo || !opt.contractInfo.decimals || !opt.contractInfo.symbol)
            throw new Error('Not enough info');
        return `Transfer ${(0, utils_ts_1.createDecimal)(opt.contractInfo.decimals).encode(v.value)} ${opt.contractInfo.symbol} to ${v.to}`;
    },
    Approval(v, opt) {
        if (!opt.contractInfo || !opt.contractInfo.decimals || !opt.contractInfo.symbol)
            throw new Error('Not enough info');
        return `Allow ${v.spender} spending up to ${(0, utils_ts_1.createDecimal)(opt.contractInfo.decimals).encode(v.value)} ${opt.contractInfo.symbol} from ${v.owner}`;
    },
    Transfer(v, opt) {
        if (!opt.contractInfo || !opt.contractInfo.decimals || !opt.contractInfo.symbol)
            throw new Error('Not enough info');
        return `Transfer ${(0, utils_ts_1.createDecimal)(opt.contractInfo.decimals).encode(v.value)} ${opt.contractInfo.symbol} from ${v.from} to ${v.to}`;
    },
};
const ERC20ABI = /* @__PURE__ */ (0, common_ts_1.addHints)(exports.ABI, exports.hints);
exports.default = ERC20ABI;
//# sourceMappingURL=erc20.js.map