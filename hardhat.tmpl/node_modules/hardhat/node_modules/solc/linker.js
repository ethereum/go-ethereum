"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
const assert_1 = __importDefault(require("assert"));
const js_sha3_1 = require("js-sha3");
const helpers_1 = require("./common/helpers");
/**
 * Generates a new-style library placeholder from a fully-qualified library name.
 *
 * Newer versions of the compiler use hashed names instead of just truncating the name
 * before putting it in a placeholder.
 *
 * @param fullyQualifiedLibraryName Fully qualified library name.
 */
function libraryHashPlaceholder(fullyQualifiedLibraryName) {
    return `$${(0, js_sha3_1.keccak256)(fullyQualifiedLibraryName).slice(0, 34)}$`;
}
/**
 * Finds all placeholders corresponding to the specified library label and replaces them
 * with a concrete address. Works with both hex-encoded and binary bytecode as long as
 * the address is in the same format.
 *
 * @param bytecode Bytecode string.
 *
 * @param label Library label, either old- or new-style. Must exactly match the part between `__` markers in the
 *     placeholders. Will be padded with `_` characters if too short or truncated if too long.
 *
 * @param address Address to replace placeholders with. Must be the right length.
 *     It will **not** be padded with zeros if too short.
 */
function replacePlaceholder(bytecode, label, address) {
    // truncate to 36 characters
    const truncatedName = label.slice(0, 36);
    const libLabel = `__${truncatedName.padEnd(36, '_')}__`;
    while (bytecode.indexOf(libLabel) >= 0) {
        bytecode = bytecode.replace(libLabel, address);
    }
    return bytecode;
}
/**
 * Finds and all library placeholders in the provided bytecode and replaces them with actual addresses.
 * Supports both old- and new-style placeholders (even both in the same file).
 * See [Library Linking](https://docs.soliditylang.org/en/latest/using-the-compiler.html#library-linking)
 * for a full explanation of the linking process.
 *
 * Example of a legacy placeholder: `__lib.sol:L_____________________________`
 * Example of a new-style placeholder: `__$cb901161e812ceb78cfe30ca65050c4337$__`
 *
 * @param bytecode Hex-encoded bytecode string. All 40-byte substrings starting and ending with
 *     `__` will be interpreted as placeholders.
 *
 * @param libraries Mapping between fully qualified library names and the hex-encoded
 *     addresses they should be replaced with. Addresses shorter than 40 characters are automatically padded with zeros.
 *
 * @returns bytecode Hex-encoded bytecode string with placeholders replaced with addresses.
 *    Note that some placeholders may remain in the bytecode if `libraries` does not provide addresses for all of them.
 */
function linkBytecode(bytecode, libraries) {
    (0, assert_1.default)(typeof bytecode === 'string');
    (0, assert_1.default)(typeof libraries === 'object');
    // NOTE: for backwards compatibility support old compiler which didn't use file names
    const librariesComplete = {};
    for (const [fullyQualifiedLibraryName, libraryObjectOrAddress] of Object.entries(libraries)) {
        if ((0, helpers_1.isNil)(libraryObjectOrAddress)) {
            throw new Error(`No address provided for library ${fullyQualifiedLibraryName}`);
        }
        // API compatible with the standard JSON i/o
        // {"lib.sol": {"L": "0x..."}}
        if ((0, helpers_1.isObject)(libraryObjectOrAddress)) {
            for (const [unqualifiedLibraryName, address] of Object.entries(libraryObjectOrAddress)) {
                librariesComplete[unqualifiedLibraryName] = address;
                librariesComplete[`${fullyQualifiedLibraryName}:${unqualifiedLibraryName}`] = address;
            }
            continue;
        }
        // backwards compatible API for early solc-js versions
        const parsed = fullyQualifiedLibraryName.match(/^(?<sourceUnitName>[^:]+):(?<unqualifiedLibraryName>.+)$/);
        const libraryAddress = libraryObjectOrAddress;
        if (!(0, helpers_1.isNil)(parsed)) {
            const { unqualifiedLibraryName } = parsed.groups;
            librariesComplete[unqualifiedLibraryName] = libraryAddress;
        }
        librariesComplete[fullyQualifiedLibraryName] = libraryAddress;
    }
    for (const libraryName in librariesComplete) {
        let hexAddress = librariesComplete[libraryName];
        if (!hexAddress.startsWith('0x') || hexAddress.length > 42) {
            throw new Error(`Invalid address specified for ${libraryName}`);
        }
        // remove 0x prefix
        hexAddress = hexAddress.slice(2).padStart(40, '0');
        bytecode = replacePlaceholder(bytecode, libraryName, hexAddress);
        bytecode = replacePlaceholder(bytecode, libraryHashPlaceholder(libraryName), hexAddress);
    }
    return bytecode;
}
/**
 * Finds locations of all library address placeholders in the hex-encoded bytecode.
 * Returns information in a format matching `evm.bytecode.linkReferences` output
 * in Standard JSON.
 *
 * See [Library Linking](https://docs.soliditylang.org/en/latest/using-the-compiler.html#library-linking)
 * for a full explanation of library placeholders and linking process.
 *
 * WARNING: The output matches `evm.bytecode.linkReferences` exactly only in
 * case of old-style placeholders created from fully qualified library names
 * of no more than 36 characters, and even then only if the name does not start
 * or end with an underscore. This is different from
 * `evm.bytecode.linkReferences`, which uses fully qualified library names.
 * This is a limitation of the placeholder format - the fully qualified names
 * are not preserved in the compiled bytecode and cannot be reconstructed
 * without external information.
 *
 * @param bytecode Hex-encoded bytecode string.
 *
 * @returns linkReferences A mapping between library labels and their locations
 * in the bytecode. In case of old-style placeholders the label is a fully
 * qualified library name truncated to 36 characters. For new-style placeholders
 * it's the first 34 characters of the hex-encoded hash of the fully qualified
 * library name, with a leading and trailing $ character added. Note that the
 * offsets and lengths refer to the *binary* (not hex-encoded) bytecode, just
 * like in `evm.bytecode.linkReferences`.
 */
function findLinkReferences(bytecode) {
    (0, assert_1.default)(typeof bytecode === 'string');
    // find 40 bytes in the pattern of __...<36 digits>...__
    // e.g. __Lib.sol:L_____________________________
    const linkReferences = {};
    let offset = 0;
    while (true) {
        const found = bytecode.match(/__(.{36})__/);
        if (!found) {
            break;
        }
        const start = found.index;
        // trim trailing underscores
        // NOTE: this has no way of knowing if the trailing underscore was part of the name
        const libraryName = found[1].replace(/_+$/gm, '');
        if (!linkReferences[libraryName]) {
            linkReferences[libraryName] = [];
        }
        // offsets are in bytes in binary representation (and not hex)
        linkReferences[libraryName].push({
            start: (offset + start) / 2,
            length: 20
        });
        offset += start + 20;
        bytecode = bytecode.slice(start + 20);
    }
    return linkReferences;
}
module.exports = {
    linkBytecode,
    findLinkReferences
};
