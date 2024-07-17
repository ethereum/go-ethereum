require("@nomiclabs/hardhat-waffle");

module.exports = {
  solidity: "0.8.4",
  networks: {
    devnet: {
      url: "http://localhost:8545",
      chainId: 1337,
    },
  },
};
