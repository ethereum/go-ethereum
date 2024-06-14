import { NomicLabsHardhatPluginError } from "hardhat/plugins";
export declare class HardhatChaiMatchersError extends NomicLabsHardhatPluginError {
    constructor(message: string, parent?: Error);
}
export declare class HardhatChaiMatchersDecodingError extends HardhatChaiMatchersError {
    constructor(encodedData: string, type: string, parent: Error);
}
/**
 * This class is used to assert assumptions in our implementation. Chai's
 * AssertionError should be used for user assertions.
 */
export declare class HardhatChaiMatchersAssertionError extends HardhatChaiMatchersError {
    constructor(message: string);
}
export declare class HardhatChaiMatchersNonChainableMatcherError extends HardhatChaiMatchersError {
    constructor(matcherName: string, previousMatcherName: string);
}
//# sourceMappingURL=errors.d.ts.map