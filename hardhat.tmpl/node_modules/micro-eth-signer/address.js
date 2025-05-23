"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.addr = void 0;
/*! micro-eth-signer - MIT License (c) 2021 Paul Miller (paulmillr.com) */
const secp256k1_1 = require("@noble/curves/secp256k1");
const sha3_1 = require("@noble/hashes/sha3");
const utils_1 = require("@noble/hashes/utils");
const utils_ts_1 = require("./utils.js");
exports.addr = {
    RE: /^(0[xX])?([0-9a-fA-F]{40})?$/,
    parse: (address, allowEmpty = false) => {
        (0, utils_ts_1.astr)(address);
        // NOTE: empty address allowed for 'to', but would be mistake for other address fields.
        // '0x' instead of null/undefined because we don't want to send contract creation tx if user
        // accidentally missed 'to' field.
        if (allowEmpty && address === '0x')
            return { hasPrefix: true, data: '' };
        const res = address.match(exports.addr.RE) || [];
        const hasPrefix = res[1] != null;
        const data = res[2];
        if (!data) {
            const len = hasPrefix ? 42 : 40;
            throw new Error(`address must be ${len}-char hex, got ${address.length}-char ${address}`);
        }
        return { hasPrefix, data };
    },
    /**
     * Address checksum is calculated by hashing with keccak_256.
     * It hashes *string*, not a bytearray: keccak('beef') not keccak([0xbe, 0xef])
     * @param nonChecksummedAddress
     * @param allowEmpty - allows '0x'
     * @returns checksummed address
     */
    addChecksum: (nonChecksummedAddress, allowEmpty = false) => {
        const low = exports.addr.parse(nonChecksummedAddress, allowEmpty).data.toLowerCase();
        const hash = (0, utils_1.bytesToHex)((0, sha3_1.keccak_256)(low));
        let checksummed = '';
        for (let i = 0; i < low.length; i++) {
            const hi = Number.parseInt(hash[i], 16);
            const li = low[i];
            checksummed += hi <= 7 ? li : li.toUpperCase(); // if char is 9-f, upcase it
        }
        return (0, utils_ts_1.add0x)(checksummed);
    },
    /**
     * Creates address from secp256k1 public key.
     */
    fromPublicKey: (key) => {
        if (!key)
            throw new Error('invalid public key: ' + key);
        const pub65b = secp256k1_1.secp256k1.ProjectivePoint.fromHex(key).toRawBytes(false);
        const hashed = (0, sha3_1.keccak_256)(pub65b.subarray(1, 65));
        const address = (0, utils_1.bytesToHex)(hashed).slice(24); // slice 24..64
        return exports.addr.addChecksum(address);
    },
    /**
     * Creates address from ETH private key in hex or ui8a format.
     */
    fromPrivateKey: (key) => {
        if (typeof key === 'string')
            key = (0, utils_ts_1.strip0x)(key);
        return exports.addr.fromPublicKey(secp256k1_1.secp256k1.getPublicKey(key, false));
    },
    /**
     * Generates hex string with new random private key and address. Uses CSPRNG internally.
     */
    random() {
        const privateKey = utils_ts_1.ethHex.encode(secp256k1_1.secp256k1.utils.randomPrivateKey());
        return { privateKey, address: exports.addr.fromPrivateKey(privateKey) };
    },
    /**
     * Verifies checksum if the address is checksummed.
     * Always returns true when the address is not checksummed.
     * @param allowEmpty - allows '0x'
     */
    isValid: (checksummedAddress, allowEmpty = false) => {
        let parsed;
        try {
            parsed = exports.addr.parse(checksummedAddress, allowEmpty);
        }
        catch (error) {
            return false;
        }
        const { data: address, hasPrefix } = parsed;
        if (!hasPrefix)
            return false;
        const low = address.toLowerCase();
        const upp = address.toUpperCase();
        if (address === low || address === upp)
            return true;
        return exports.addr.addChecksum(low, allowEmpty) === checksummedAddress;
    },
};
//# sourceMappingURL=address.js.map