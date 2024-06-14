/**
 * Configuration defaults
 */

class Config {
  constructor(options = {}) {
    this.token = options.token || "ETH";
    this.blockLimit = options.blockLimit || 6718946;
    this.defaultGasPrice = 5;

    this.currency = options.currency || "eur";
    this.gasPriceApi =
      options.gasPriceApi ||
      "https://api.etherscan.io/api?module=proxy&action=eth_gasPrice";
    this.coinmarketcap = options.coinmarketcap || null;
    this.ethPrice = options.ethPrice || null;
    this.gasPrice = options.gasPrice || null;
    this.outputFile = options.outputFile || null;
    this.forceConsoleOutput = options.forceConsoleOutput || false;
    this.rst = options.rst || false;
    this.rstTitle = options.rstTitle || "";
    this.showTimeSpent = options.showTimeSpent || false;
    this.srcPath = options.src || "contracts";
    this.artifactType = options.artifactType || "truffle-v5";
    this.getContracts = options.getContracts || null;
    this.noColors = options.noColors;
    this.proxyResolver = options.proxyResolver || null;
    this.metadata = options.metadata || null;
    this.showMethodSig = options.showMethodSig || false;
    this.provider = options.provider || null;
    this.maxMethodDiff = options.maxMethodDiff;
    this.maxDeploymentDiff = options.maxDeploymentDiff;

    this.excludeContracts = Array.isArray(options.excludeContracts)
      ? options.excludeContracts
      : [];

    this.onlyCalledMethods = options.onlyCalledMethods === false ? false : true;

    this.url = options.url
      ? this._normalizeUrl(options.url)
      : this.resolveClientUrl();
  }

  /**
   * Tries to obtain the client url reporter's sync-requests will target.
   * @return {String}         url e.g http://localhost:8545
   */
  resolveClientUrl() {
    // Case: web3 globally available in mocha test context
    try {
      if (web3 && web3.currentProvider) {
        const cp = web3.currentProvider;

        // Truffle/Web3 http
        if (cp.host) return cp.host;

        // Truffle/Web3 websockets
        if (cp.connection) return this._normalizeUrl(cp.connection.url);
      }
    } catch (err) {
      // Web3 undefined
    }

    // Case: Failure
    const message =
      `ERROR: eth-gas-reporter was unable to resolve a client url ` +
      `from the provider available in your test context. Try setting the ` +
      `url as a mocha reporter option (ex: url='http://localhost:8545')`;

    console.log(message);
    process.exit(1);
  }

  /**
   * Forces websockets to http
   * @param  {String} url e.g web3.provider.connection.url
   * @return {String}     http:// prefixed url
   */
  _normalizeUrl(url) {
    return url.replace("ws://", "http://");
  }
}

module.exports = Config;
