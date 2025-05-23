/**
 * Construct the future id for a contract, contractAt or library, namespaced by the
 * moduleId.
 *
 * This method supports both bare contract names (e.g. `MyContract`) and fully
 * qualified names (e.g. `contracts/MyModule.sol:MyContract`).
 *
 * If a fully qualified name is used, the id is only direvied from its contract
 * name, ignoring its source name part. The reason is that ids need to be
 * compatible with most common file systems (including Windows!), and the source
 * name may have incompatible characters.
 *
 * @param moduleId - the id of the module the future is part of
 * @param userProvidedId - the overriding id provided by the user (it will still
 * be namespaced)
 * @param contractOrLibraryName - the contract or library name, either a bare name
 * or a fully qualified name.
 * @returns the future id
 */
export declare function toContractFutureId(moduleId: string, userProvidedId: string | undefined, contractOrLibraryName: string): string;
/**
 * Construct the future id for a call or static call, namespaced by the moduleId.
 *
 * @param moduleId - the id of the module the future is part of
 * @param userProvidedId - the overriding id provided by the user (it will still
 * be namespaced)
 * @param contractName - the contract or library name that forms part of the
 * fallback
 * @param functionName - the function name that forms part of the fallback
 * @returns the future id
 */
export declare function toCallFutureId(moduleId: string, userProvidedId: string | undefined, contractModuleId: string, contractId: string, functionName: string): string;
/**
 * Construct the future id for an encoded function call, namespaced by the moduleId.
 *
 * @param moduleId - the id of the module the future is part of
 * @param userProvidedId - the overriding id provided by the user (it will still
 * be namespaced)
 * @param contractName - the contract or library name that forms part of the
 * fallback
 * @param functionName - the function name that forms part of the fallback
 * @returns the future id
 */
export declare function toEncodeFunctionCallFutureId(moduleId: string, userProvidedId: string | undefined, contractModuleId: string, contractId: string, functionName: string): string;
/**
 * Construct the future id for a read event argument future, namespaced by
 * the moduleId.
 *
 * @param moduleId - the id of the module the future is part of
 * @param userProvidedId - the overriding id provided by the user (it will still
 * be namespaced)
 * @param contractName - the contract or library name that forms part of the
 * fallback
 * @param eventName - the event name that forms part of the fallback
 * @param nameOrIndex - the argument name or argumentindex that forms part
 * of the fallback
 * @param eventIndex - the event index that forms part of the fallback
 * @returns the future id
 */
export declare function toReadEventArgumentFutureId(moduleId: string, userProvidedId: string | undefined, contractName: string, eventName: string, nameOrIndex: string | number, eventIndex: number): string;
/**
 * Construct the future id for a send data future, namespaced by the moduleId.
 *
 * @param moduleId - the id of the module the future is part of
 * @param userProvidedId - the overriding id provided by the user (it will still
 * be namespaced)
 * @returns the future id
 */
export declare function toSendDataFutureId(moduleId: string, userProvidedId: string): string;
//# sourceMappingURL=future-id-builders.d.ts.map