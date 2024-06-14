import "hardhat/types/config";
import "hardhat/types/runtime";
import { DeployConfig, StrategyConfig } from "@nomicfoundation/ignition-core";
declare module "hardhat/types/config" {
    interface ProjectPathsUserConfig {
        ignition?: string;
    }
    interface ProjectPathsConfig {
        ignition: string;
    }
    interface HardhatNetworkUserConfig {
        ignition?: {
            maxFeePerGasLimit?: bigint;
            maxPriorityFeePerGas?: bigint;
        };
    }
    interface HardhatNetworkConfig {
        ignition: {
            maxFeePerGasLimit?: bigint;
            maxPriorityFeePerGas?: bigint;
        };
    }
    interface HttpNetworkUserConfig {
        ignition?: {
            maxFeePerGasLimit?: bigint;
            maxPriorityFeePerGas?: bigint;
        };
    }
    interface HttpNetworkConfig {
        ignition: {
            maxFeePerGasLimit?: bigint;
            maxPriorityFeePerGas?: bigint;
        };
    }
    interface HardhatUserConfig {
        ignition?: Partial<DeployConfig> & {
            strategyConfig?: Partial<StrategyConfig>;
        };
    }
    interface HardhatConfig {
        ignition: Partial<DeployConfig> & {
            strategyConfig?: Partial<StrategyConfig>;
        };
    }
}
//# sourceMappingURL=type-extensions.d.ts.map