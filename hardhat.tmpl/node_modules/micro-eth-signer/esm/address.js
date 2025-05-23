/*! micro-eth-signer - MIT License (c) 2021 Paul Miller (paulmillr.com) */
import { secp256k1 } from '@noble/curves/secp256k1';
import { keccak_256 } from '@noble/hashes/sha3';
import { bytesToHex } from '@noble/hashes/utils';
import { add0x, astr, ethHex, strip0x } from "./utils.js";
export const addr = {
    RE: /^(0[xX])?([0-9a-fA-F]{40})?$/,
    parse: (address, allowEmpty = false) => {
        astr(address);
        // NOTE: empty address allowed for 'to', but would be mistake for other address fields.
        // '0x' instead of null/undefined because we don't want to send contract creation tx if user
        // accidentally missed 'to' field.
        if (allowEmpty && address === '0x')
            return { hasPrefix: true, data: '' };
        const res = address.match(addr.RE) || [];
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
        const low = addr.parse(nonChecksummedAddress, allowEmpty).data.toLowerCase();
        const hash = bytesToHex(keccak_256(low));
        let checksummed = '';
        for (let i = 0; i < low.length; i++) {
            const hi = Number.parseInt(hash[i], 16);
            const li = low[i];
            checksummed += hi <= 7 ? li : li.toUpperCase(); // if char is 9-f, upcase it
        }
        return add0x(checksummed);
    },
    /**
     * Creates address from secp256k1 public key.
     */
    fromPublicKey: (key) => {
        if (!key)
            throw new Error('invalid public key: ' + key);
        const pub65b = secp256k1.ProjectivePoint.fromHex(key).toRawBytes(false);
        const hashed = keccak_256(pub65b.subarray(1, 65));
        const address = bytesToHex(hashed).slice(24); // slice 24..64
        return addr.addChecksum(address);
    },
    /**
     * Creates address from ETH private key in hex or ui8a format.
     */
    fromPrivateKey: (key) => {
        if (typeof key === 'string')
            key = strip0x(key);
        return addr.fromPublicKey(secp256k1.getPublicKey(key, false));
    },
    /**
     * Generates hex string with new random private key and address. Uses CSPRNG internally.
     */
    random() {
        const privateKey = ethHex.encode(secp256k1.utils.randomPrivateKey());
        return { privateKey, address: addr.fromPrivateKey(privateKey) };
    },
    /**
     * Verifies checksum if the address is checksummed.
     * Always returns true when the address is not checksummed.
     * @param allowEmpty - allows '0x'
     */
    isValid: (checksummedAddress, allowEmpty = false) => {
        let parsed;
        try {
            parsed = addr.parse(checksummedAddress, allowEmpty);
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
        return addr.addChecksum(low, allowEmpty) === checksummedAddress;
    },
};
//# sourceMappingURL=address.js.map