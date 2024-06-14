import { ProviderWrapper } from "hardhat/internal/core/providers/wrapper"
import { EthereumProvider, EIP1193Provider, RequestArguments } from "hardhat/types";

/**
 * Wrapped provider which collects tx data
 */
export class EGRDataCollectionProvider extends ProviderWrapper {
  private mochaConfig: any;

  constructor(provider: EIP1193Provider, mochaConfig) {
    super(provider);
    this.mochaConfig = mochaConfig
  }

  public async request(args: RequestArguments): Promise<unknown> {
    // Truffle
    if (args.method === "eth_getTransactionReceipt") {
      const receipt: any = await this._wrappedProvider.request(args);
      if (receipt?.status && receipt?.transactionHash){
        const tx = await this._wrappedProvider.request({
          method: "eth_getTransactionByHash",
          params: [receipt.transactionHash]
        });
        await this.mochaConfig.attachments.recordTransaction(receipt, tx);
      }
      return receipt;

    // Ethers: will get run twice for deployments (e.g both receipt and txhash are fetched)
    } else if (args.method === 'eth_getTransactionByHash'){
      const receipt: any = await this._wrappedProvider.request({
        method: "eth_getTransactionReceipt",
        params: args.params
      })
      const tx = await this._wrappedProvider.request(args)
      if (receipt?.status){
        await this.mochaConfig.attachments.recordTransaction(receipt, tx)
      }
      return tx;

    // Waffle: This is necessary when using Waffle wallets. eth_sendTransaction fetches the
    // transactionHash as part of its flow, eth_sendRawTransaction *does not*.
    } else if (args.method === 'eth_sendRawTransaction') {
      const txHash = await this._wrappedProvider.request(args);

      if (typeof txHash === 'string'){
        const tx = await this._wrappedProvider.request({
          method: "eth_getTransactionByHash",
          params: [txHash]
        });
        const receipt : any = await this._wrappedProvider.request({
            method: "eth_getTransactionReceipt",
            params: [txHash]
        });

        if (receipt?.status){
          await this.mochaConfig.attachments.recordTransaction(receipt, tx)
        }
      }
      return txHash;
    }
    return this._wrappedProvider.request(args);
  }
}

/**
 * A set of async RPC calls which substitute the sync calls made by the core reporter
 * These allow us to use HardhatEVM or another in-process provider.
 */
export class EGRAsyncApiProvider {
  public provider: EthereumProvider;

  constructor(provider: EthereumProvider) {
    this.provider = provider;
  }

  async getNetworkId() {
    return this.provider.send("net_version", []);
  }

  async getCode(address: string) {
    return this.provider.send("eth_getCode", [address, "latest"]);
  }

  async getLatestBlock() {
    return this.provider.send("eth_getBlockByNumber", ["latest", false]);
  }

  async getBlockByNumber(num: number) {
    const hexNumber = `0x${num.toString(16)}`;
    return this.provider.send("eth_getBlockByNumber", [hexNumber, true]);
  }

  async blockNumber() {
    const block = await this.getLatestBlock();
    return parseInt(block.number, 16);
  }

  async getTransactionByHash(tx) {
    return this.provider.send("eth_getTransactionByHash", [tx]);
  }

  async call(payload, blockNumber) {
    return this.provider.send("eth_call", [payload, blockNumber]);
  }
}
