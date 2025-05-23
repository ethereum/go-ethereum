import { CustomError } from "hardhat/common";
export declare class HardhatNetworkHelpersError extends CustomError {
    constructor(message: string);
}
export declare class InvalidSnapshotError extends CustomError {
    constructor();
}
export declare class FixtureSnapshotError extends CustomError {
    constructor(parent: InvalidSnapshotError);
}
export declare class FixtureAnonymousFunctionError extends CustomError {
    constructor();
}
export declare class OnlyHardhatNetworkError extends CustomError {
    constructor(networkName: string, version?: string);
}
//# sourceMappingURL=errors.d.ts.map