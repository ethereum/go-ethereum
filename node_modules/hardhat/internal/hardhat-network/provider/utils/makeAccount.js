"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.makeAccount = void 0;
const ethereumjs_util_1 = require("@nomicfoundation/ethereumjs-util");
const isHexPrefixed_1 = require("./isHexPrefixed");
function makeAccount(ga) {
    let balance;
    if (typeof ga.balance === "string" && (0, isHexPrefixed_1.isHexPrefixed)(ga.balance)) {
        balance = BigInt(ga.balance);
    }
    else {
        balance = BigInt(ga.balance);
    }
    const account = ethereumjs_util_1.Account.fromAccountData({ balance });
    const pk = (0, ethereumjs_util_1.toBytes)(ga.privateKey);
    const address = new ethereumjs_util_1.Address((0, ethereumjs_util_1.privateToAddress)(pk));
    return { account, address };
}
exports.makeAccount = makeAccount;
//# sourceMappingURL=makeAccount.js.map