import type { EtherscanConfig, SourcifyConfig, BlockscoutConfig } from "../types";
import "hardhat/types/config";
declare module "hardhat/types/config" {
    interface HardhatUserConfig {
        etherscan?: Partial<EtherscanConfig>;
        sourcify?: Partial<SourcifyConfig>;
        blockscout?: Partial<BlockscoutConfig>;
    }
    interface HardhatConfig {
        etherscan: EtherscanConfig;
        sourcify: SourcifyConfig;
        blockscout: BlockscoutConfig;
    }
}
//# sourceMappingURL=type-extensions.d.ts.map