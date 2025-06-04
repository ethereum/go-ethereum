"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.makeEmptyAccountState = exports.makeAccountState = void 0;
const immutable_1 = require("immutable");
exports.makeAccountState = (0, immutable_1.Record)({
    nonce: undefined,
    balance: undefined,
    storage: (0, immutable_1.Map)(),
    code: undefined,
    storageCleared: false,
});
// used for deleted accounts
// they need real values to avoid fetching the data from the remote node
exports.makeEmptyAccountState = (0, immutable_1.Record)({
    nonce: "0x0",
    balance: "0x0",
    storage: (0, immutable_1.Map)(),
    code: "0x",
    storageCleared: true,
});
//# sourceMappingURL=AccountState.js.map