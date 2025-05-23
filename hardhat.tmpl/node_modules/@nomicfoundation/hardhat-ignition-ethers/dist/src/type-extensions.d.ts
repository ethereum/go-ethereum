import { EthersIgnitionHelper } from "./ethers-ignition-helper";
declare module "hardhat/types/runtime" {
    interface HardhatRuntimeEnvironment {
        ignition: EthersIgnitionHelper;
    }
}
//# sourceMappingURL=type-extensions.d.ts.map