import { LibraryAddresses, LinkReferences } from './common/types';
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
declare function linkBytecode(bytecode: string, libraries: LibraryAddresses): string;
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
declare function findLinkReferences(bytecode: string): LinkReferences;
declare const _default: {
    linkBytecode: typeof linkBytecode;
    findLinkReferences: typeof findLinkReferences;
};
export = _default;
