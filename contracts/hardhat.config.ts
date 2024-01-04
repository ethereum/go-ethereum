import { HardhatUserConfig } from "hardhat/config";
import "@nomicfoundation/hardhat-toolbox";

const config: HardhatUserConfig = {
  // defaultNetwork: "geth",
  defaultNetwork: "geth",
  networks: {
    hardhat: {},
    geth: {
      url: "http://localhost:8545",
      // accounts: [privateKey1, privateKey2, ...]
    }
  },
  solidity: "0.8.19",
};

export default config;
