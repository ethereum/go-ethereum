const utils = require("./utils");
const GasData = require("./gasData");
const SyncRequest = require("./syncRequest");
const ProxyResolver = require("./proxyResolver");

/**
 * Tracks blocks and cycles across them, extracting gas usage data and
 * associating it with the relevant contracts, methods.
 */
class TransactionWatcher {
  constructor(config) {
    this.itStartBlock = 0; // Tracks within `it` block transactions (gas usage per test)
    this.beforeStartBlock = 0; // Tracks from `before/beforeEach` transactions (methods & deploys)
    this.data = new GasData();
    this.sync = new SyncRequest(config.url);
    this.provider = config.provider;
    this.resolver = new ProxyResolver(this.data, config);
  }

  /**
   * Cycles across a range of blocks, from beforeStartBlock set in the reporter's
   * `test` hook to current block when it's called. Collect deployments and methods
   * gas usage data.
   * @return {Number} Total gas usage for the `it` block
   */
  blocks() {
    let gasUsed = 0;
    const endBlock = this.sync.blockNumber();

    while (this.beforeStartBlock <= endBlock) {
      let block = this.sync.getBlockByNumber(this.beforeStartBlock);

      if (block) {
        // Track gas used within `it` blocks
        if (this.itStartBlock <= this.beforeStartBlock) {
          gasUsed += utils.gas(block.gasUsed);
        }

        // Collect methods and deployments data
        block.transactions.forEach(transaction => {
          const receipt = this.sync.getTransactionReceipt(transaction.hash);

          // Omit transactions that throw
          if (parseInt(receipt.status) === 0) return;

          receipt.contractAddress
            ? this._collectDeploymentsData(transaction, receipt)
            : this._collectMethodsData(transaction, receipt);
        });
      }
      this.beforeStartBlock++;
    }
    return gasUsed;
  }

  async transaction(receipt, transaction) {
    receipt.contractAddress
      ? await this._asyncCollectDeploymentsData(transaction, receipt)
      : await this._asyncCollectMethodsData(transaction, receipt);
  }

  /**
   * Extracts and stores deployments gas usage data for a tx
   * @param  {Object} transaction return value of `getTransactionByHash`
   * @param  {Object} receipt
   */
  _collectDeploymentsData(transaction, receipt) {
    const match = this.data.getContractByDeploymentInput(transaction.input);

    if (match) {
      this.data.trackNameByAddress(match.name, receipt.contractAddress);
      match.gasData.push(utils.gas(receipt.gasUsed));
    }
  }

  /**
   * Extracts and stores deployments gas usage data for a tx
   * @param  {Object} transaction return value of `getTransactionByHash`
   * @param  {Object} receipt
   */
  async _asyncCollectDeploymentsData(transaction, receipt) {
    const match = this.data.getContractByDeploymentInput(transaction.input);

    if (match) {
      await this.data.asyncTrackNameByAddress(
        match.name,
        receipt.contractAddress
      );
      match.gasData.push(utils.gas(receipt.gasUsed));
    }
  }

  /**
   * Extracts and stores methods gas usage data for a tx
   * @param  {Object} transaction return value of `getTransactionByHash`
   * @param  {Object} receipt
   */
  _collectMethodsData(transaction, receipt) {
    let contractName = this.data.getNameByAddress(transaction.to);

    // Case: proxied call
    if (this._isProxied(contractName, transaction.input)) {
      contractName = this.resolver.resolve(transaction);

      // Case: hidden contract factory deployment
    } else if (!contractName) {
      contractName = this.resolver.resolveByDeployedBytecode(transaction.to);
    }

    // Case: all else fails, use first match strategy
    if (!contractName) {
      contractName = this.resolver.resolveByMethodSignature(transaction);
    }

    const id = utils.getMethodID(contractName, transaction.input);

    if (this.data.methods[id]) {
      this.data.methods[id].gasData.push(utils.gas(receipt.gasUsed));
      this.data.methods[id].numberOfCalls += 1;
    } else {
      this.resolver.unresolvedCalls++;
    }
  }

  /**
   * Extracts and stores methods gas usage data for a tx
   * @param  {Object} transaction return value of `getTransactionByHash`
   * @param  {Object} receipt
   */
  async _asyncCollectMethodsData(transaction, receipt) {
    let contractName = await this.data.asyncGetNameByAddress(transaction.to);

    // Case: proxied call
    if (this._isProxied(contractName, transaction.input)) {
      contractName = this.resolver.resolve(transaction);

      // Case: hidden contract factory deployment
    } else if (!contractName) {
      contractName = await this.resolver.asyncResolveByDeployedBytecode(
        transaction.to
      );
    }

    // Case: all else fails, use first match strategy
    if (!contractName) {
      contractName = this.resolver.resolveByMethodSignature(transaction);
    }

    const id = utils.getMethodID(contractName, transaction.input);

    if (this.data.methods[id]) {
      this.data.methods[id].gasData.push(utils.gas(receipt.gasUsed));
      this.data.methods[id].numberOfCalls += 1;
    } else {
      this.resolver.unresolvedCalls++;
    }
  }

  /**
   * Returns true if there is a contract name associated with an address
   * but method can't be matched to it
   * @param  {String}  name  contract name
   * @param  {String}  input code
   * @return {Boolean}
   */
  _isProxied(name, input) {
    return name && !this.data.methods[utils.getMethodID(name, input)];
  }
}

module.exports = TransactionWatcher;
