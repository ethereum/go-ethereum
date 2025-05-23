import type { Etherscan } from "@nomicfoundation/hardhat-verify/etherscan";
import type { VerifyInfo } from "@nomicfoundation/ignition-core";
export declare function verifyEtherscanContract(etherscanInstance: Etherscan, { address, compilerVersion, sourceCode, name, args }: VerifyInfo): Promise<{
    type: "success";
    contractURL: string;
} | {
    type: "failure";
    reason: Error;
}>;
//# sourceMappingURL=verifyEtherscanContract.d.ts.map