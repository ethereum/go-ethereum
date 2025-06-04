"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.deriveKeyFromMnemonicAndPath = void 0;
function deriveKeyFromMnemonicAndPath(mnemonic, hdPath, passphrase) {
    const { mnemonicToSeedSync, } = require("ethereum-cryptography/bip39");
    // NOTE: If mnemonic has space or newline at the beginning or end, it will be trimmed.
    // This is because mnemonic containing them may generate different private keys.
    const trimmedMnemonic = mnemonic.trim();
    const seed = mnemonicToSeedSync(trimmedMnemonic, passphrase);
    const { HDKey, } = require("ethereum-cryptography/hdkey");
    const masterKey = HDKey.fromMasterSeed(seed);
    const derived = masterKey.derive(hdPath);
    return derived.privateKey === null
        ? undefined
        : Buffer.from(derived.privateKey);
}
exports.deriveKeyFromMnemonicAndPath = deriveKeyFromMnemonicAndPath;
//# sourceMappingURL=keys-derivation.js.map