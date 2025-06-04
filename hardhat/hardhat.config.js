require("@nomicfoundation/hardhat-toolbox");

module.exports = {
  solidity: "0.8.28",
  networks: {
    hell: {
      url: "http://127.0.0.1:8545",
      accounts: [process.env.PRIVATE_KEY] // Set in GitHub Secrets
    }
  }
};
