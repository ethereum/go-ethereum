require("@nomicfoundation/hardhat-toolbox");
require("@nomicfoundation/hardhat-chai-matchers");
require("@nomiclabs/hardhat-ethers");
require("@nomicfoundation/hardhat-ignition-ethers");

/** @type import('hardhat/config').HardhatUserConfig */
module.exports = {
  solidity: "0.8.24",
  networks: {
    localhost: {
      url: "http://127.0.0.1:8545"
    },
    gethDev: {
      url: "http://127.0.0.1:8552",
      accounts: ["0x10db3a848996cd5b190ab083a2a3836c16ea2e16bc30447f3964e76dc5aa8594"]
    }
  },
  ignition: {
    defaultNetwork: "gethDev"
  }
};
