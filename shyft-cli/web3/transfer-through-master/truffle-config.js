module.exports = {
  // See <http://truffleframework.com/docs/advanced/configuration>
  // to customize your Truffle configuration!

  networks: {
    geth_testnet: {
      host: "127.0.0.1",
      port: 8545,
      netword_id: "*"
    }
  }
};
