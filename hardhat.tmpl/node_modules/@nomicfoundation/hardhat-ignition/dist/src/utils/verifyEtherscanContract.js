"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.verifyEtherscanContract = void 0;
async function verifyEtherscanContract(etherscanInstance, { address, compilerVersion, sourceCode, name, args }) {
    try {
        const { message: guid } = await etherscanInstance.verify(address, sourceCode, name, compilerVersion, args);
        const verificationStatus = await etherscanInstance.getVerificationStatus(guid);
        if (verificationStatus.isSuccess()) {
            const contractURL = etherscanInstance.getContractUrl(address);
            return { type: "success", contractURL };
        }
        else {
            // todo: what case would cause verification status not to succeed without throwing?
            return { type: "failure", reason: new Error(verificationStatus.message) };
        }
    }
    catch (e) {
        if (e instanceof Error) {
            return { type: "failure", reason: e };
        }
        else {
            throw e;
        }
    }
}
exports.verifyEtherscanContract = verifyEtherscanContract;
//# sourceMappingURL=verifyEtherscanContract.js.map