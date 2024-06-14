import 'hardhat/types/config';
import { TypechainConfig, TypechainUserConfig } from './types';
declare module 'hardhat/types/config' {
    interface HardhatUserConfig {
        typechain?: TypechainUserConfig;
    }
    interface HardhatConfig {
        typechain: TypechainConfig;
    }
}
