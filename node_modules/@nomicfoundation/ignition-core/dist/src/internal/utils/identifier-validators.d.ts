/**
 * Does the identifier match Ignition's rules for ids. Specifically that they
 * started with a letter and only contain alphanumerics and underscores.
 *
 * @param identifier - the id to test
 * @returns true if the identifier is valid
 */
export declare function isValidIgnitionIdentifier(identifier: string): boolean;
/**
 * Does the identifier match Solidity's rules for ids. See the Solidity
 * language spec for more details.
 *
 * @param identifier - the id to test
 * @returns true if the identifier is a valid Solidity identifier
 */
export declare function isValidSolidityIdentifier(identifier: string): boolean;
/**
 * Does the function or event name match Ignition's rules. This is
 * looser than Solidity's rules, but allows Ethers style `myfun(uint256,bool)`
 * function/event specifications.
 *
 * @param functionName - the function name to test
 * @returns true if the function name is valid
 */
export declare function isValidFunctionOrEventName(functionName: string): boolean;
/**
 * Returns true if a contract name (either bare - e.g. `MyContract` - or fully
 * qualified - e.g. `contracts/MyContract.sol:MyContract`) is valid.
 *
 * In the case of FQNs, we only validate the contract name part.
 *
 * The reason to validate the contract name is that we want to use them in
 * future ids, and those need to be compatible with most common file systems
 * (including Windows!).
 *
 * We don't validate the entire FQN, as we'll only use its bare name to
 * derive ids.
 *
 * @param contractName A bare or FQN contract name to validate.
 * @returns true if the contract name is valid.
 */
export declare function isValidContractName(contractName: string): boolean;
//# sourceMappingURL=identifier-validators.d.ts.map