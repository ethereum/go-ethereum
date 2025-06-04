"use strict";
// To modify this file, you must update ./misc/admin/lib/cmds/update-exports.js
import * as ethers from "./ethers";
try {
    const anyGlobal = window;
    if (anyGlobal._ethers == null) {
        anyGlobal._ethers = ethers;
    }
}
catch (error) { }
export { ethers };
export { Signer, Wallet, VoidSigner, getDefaultProvider, providers, BaseContract, Contract, ContractFactory, BigNumber, FixedNumber, constants, errors, logger, utils, wordlists, 
////////////////////////
// Compile-Time Constants
version, Wordlist } from "./ethers";
//# sourceMappingURL=index.js.map