/**
 * A mapping between libraries and the addresses to which they were deployed.
 *
 * Containing support for two level configuration, These two level
 * configurations can be seen below.
 *
 * {
 *     "lib.sol:L1": "0x...",
 *     "lib.sol:L2": "0x...",
 *     "lib.sol": {"L3": "0x..."}
 * }
 */
export interface LibraryAddresses {
    [qualifiedNameOrSourceUnit: string]: string | {
        [unqualifiedLibraryName: string]: string;
    };
}
/**
 * A mapping between libraries and lists of placeholder instances present in their hex-encoded bytecode.
 * For each placeholder its length and the position of the first character is stored.
 *
 * Each start and length entry will always directly refer to the position in
 * binary and not hex-encoded bytecode.
 */
export interface LinkReferences {
    [libraryLabel: string]: Array<{
        start: number;
        length: number;
    }>;
}
