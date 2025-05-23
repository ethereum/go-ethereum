/// <reference types="mocha" />
import { HardhatNetworkConfig } from "../../../types";
export declare const DEFAULT_SOLC_VERSION = "0.7.3";
export declare const HARDHAT_NETWORK_DEFAULT_GAS_PRICE = "auto";
export declare const HARDHAT_NETWORK_DEFAULT_MAX_PRIORITY_FEE_PER_GAS = 1000000000;
export declare const HARDHAT_NETWORK_DEFAULT_INITIAL_BASE_FEE_PER_GAS = 1000000000;
export declare const HARDHAT_NETWORK_MNEMONIC = "test test test test test test test test test test test junk";
export declare const DEFAULT_HARDHAT_NETWORK_BALANCE = "10000000000000000000000";
export declare const defaultDefaultNetwork = "hardhat";
export declare const defaultLocalhostNetworkParams: {
    url: string;
    timeout: number;
};
export declare const defaultHdAccountsConfigParams: {
    initialIndex: number;
    count: number;
    path: string;
    passphrase: string;
};
export declare const defaultHardhatNetworkHdAccountsConfigParams: {
    mnemonic: string;
    accountsBalance: string;
    initialIndex: number;
    count: number;
    path: string;
    passphrase: string;
};
export declare const DEFAULT_GAS_MULTIPLIER = 1;
export declare const defaultHardhatNetworkParams: Omit<HardhatNetworkConfig, "gas" | "initialDate">;
export declare const defaultHttpNetworkParams: {
    accounts: "remote";
    gas: "auto";
    gasPrice: "auto";
    gasMultiplier: number;
    httpHeaders: {};
    timeout: number;
};
export declare const defaultMochaOptions: Mocha.MochaOptions;
export declare const defaultSolcOutputSelection: {
    "*": {
        "*": string[];
        "": string[];
    };
};
//# sourceMappingURL=default-config.d.ts.map