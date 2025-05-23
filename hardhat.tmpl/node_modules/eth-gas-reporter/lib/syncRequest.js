const syncRequest = require("sync-request");
/**
 * A set of sync RPC calls. Synchronicity is necessary to handle build tools that
 * revert between test runner blocks (like `suite`). Mocha doesn't support async methods
 * in the reporter hook and no modern ethereum providers (web3, ethers) support sync methods
 * either so we need to execute them ourselves
 *
 * @author: Alex Rea, <github.com/area>
 */

class Sync {
  constructor(url) {
    this.url = url;
  }

  getNetworkId() {
    return this.request("net_version", []);
  }

  getCode(address) {
    return this.request("eth_getCode", [address, "latest"]);
  }

  blockNumber() {
    const val = this.request("eth_blockNumber", []);
    return parseInt(val, 16);
  }

  getLatestBlock() {
    return this.request("eth_getBlockByNumber", ["latest", false]);
  }

  getBlockByNumber(number) {
    const hexNumber = `0x${number.toString(16)}`;
    return this.request("eth_getBlockByNumber", [hexNumber, true]);
  }

  blockNumber() {
    const block = this.getLatestBlock();
    return parseInt(block.number, 16);
  }

  getTransactionByHash(tx) {
    return this.request("eth_getTransactionByHash", [tx]);
  }

  getTransactionReceipt(tx) {
    return this.request("eth_getTransactionReceipt", [tx]);
  }

  call(payload, blockNumber) {
    return this.request("eth_call", [payload, blockNumber]);
  }

  request(method, params) {
    const payload = {
      json: {
        jsonrpc: "2.0",
        method: method,
        params: params,
        id: 1
      }
    };

    const res = syncRequest("POST", this.url, payload);
    return JSON.parse(res.getBody("utf8")).result;
  }
}

module.exports = Sync;
