module.exports = {
  networks: {
    development: {
      host: "localhost",
      port: 8545,
      network_id: "*",
      websockets: process.env.TEST === "integration" ? true : false
    }
  },
  mocha: {
    reporter: "eth-gas-reporter",
    reporterOptions: {
      currency: "chf",
      token: "ETH",
      coinmarketcap: process.env.COINMARKETCAP_API_KEY || null,
      gasPriceApi:
        "https://api.etherscan.io/api?module=proxy&action=eth_gasPrice",
      onlyCalledMethods: false,
      noColors: true,
      rst: true,
      rstTitle: "Gas Usage",
      showTimeSpent: true,
      excludeContracts: ["Migrations"],
      proxyResolver: "EtherRouter",
      codechecks: true,
      showMethodSig: true
    }
  }
};
