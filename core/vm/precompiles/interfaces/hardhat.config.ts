import "hardhat-abi-exporter";
import { HardhatUserConfig } from "hardhat/types";

const config: HardhatUserConfig = {
    paths: {
        artifacts: "build/artifacts",
        cache: "build/cache",
        sources: "contracts",
    },
    solidity: {
        compilers: [
            {
                version: "0.8.20",
                settings: {
                    evmVersion: "istanbul",
                    optimizer: {
                        enabled: true,
                        runs: 200,
                    },
                },
            },
        ],
    },
    abiExporter: {
        path: "./abis",
        runOnCompile: true,
        clear: true,
        flat: true,
        format: "json",
    },
};

export default config;
