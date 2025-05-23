import type { BlockTag, TransactionRequest } from "ethers";
import {
  assertArgument,
  ethers,
  getAddress,
  hexlify,
  resolveAddress,
  toUtf8Bytes,
  TransactionLike,
  TypedDataEncoder,
} from "ethers";
import { HardhatEthersProvider } from "./internal/hardhat-ethers-provider";
import {
  copyRequest,
  getRpcTransaction,
  resolveProperties,
} from "./internal/ethers-utils";
import { HardhatEthersError, NotImplementedError } from "./internal/errors";

export class HardhatEthersSigner implements ethers.Signer {
  public readonly address: string;
  public readonly provider: ethers.JsonRpcProvider | HardhatEthersProvider;

  public static async create(provider: HardhatEthersProvider, address: string) {
    const hre = await import("hardhat");

    // depending on the config, we set a fixed gas limit for all transactions
    let gasLimit: number | undefined;

    if (hre.network.name === "hardhat") {
      // If we are connected to the in-process hardhat network and the config
      // has a fixed number as the gas config, we use that.
      // Hardhat core already sets this value to the block gas limit when the
      // user doesn't specify a number.
      if (hre.network.config.gas !== "auto") {
        gasLimit = hre.network.config.gas;
      }
    } else if (hre.network.name === "localhost") {
      const configuredGasLimit = hre.config.networks.localhost.gas;

      if (configuredGasLimit !== "auto") {
        // if the resolved gas config is a number, we use that
        gasLimit = configuredGasLimit;
      } else {
        // if the resolved gas config is "auto", we need to check that
        // the user config is undefined, because that's the default value;
        // otherwise explicitly setting the gas to "auto" would have no effect
        if (hre.userConfig.networks?.localhost?.gas === undefined) {
          // finally, we check if we are connected to a hardhat network
          let isHardhatNetwork = false;
          try {
            await hre.network.provider.send("hardhat_metadata");
            isHardhatNetwork = true;
          } catch {}

          if (isHardhatNetwork) {
            // WARNING: this assumes that the hardhat node is being run in the
            // same project which might be wrong
            gasLimit = hre.config.networks.hardhat.blockGasLimit;
          }
        }
      }
    }

    return new HardhatEthersSigner(address, provider, gasLimit);
  }

  private constructor(
    address: string,
    _provider: ethers.JsonRpcProvider | HardhatEthersProvider,
    private readonly _gasLimit?: number
  ) {
    this.address = getAddress(address);
    this.provider = _provider;
  }

  public connect(
    provider: ethers.JsonRpcProvider | HardhatEthersProvider
  ): ethers.Signer {
    return new HardhatEthersSigner(this.address, provider);
  }

  public getNonce(blockTag?: BlockTag | undefined): Promise<number> {
    return this.provider.getTransactionCount(this.address, blockTag);
  }

  public populateCall(
    tx: TransactionRequest
  ): Promise<ethers.TransactionLike<string>> {
    return populate(this, tx);
  }

  public populateTransaction(
    tx: TransactionRequest
  ): Promise<ethers.TransactionLike<string>> {
    return this.populateCall(tx);
  }

  public async estimateGas(tx: TransactionRequest): Promise<bigint> {
    return this.provider.estimateGas(await this.populateCall(tx));
  }

  public async call(tx: TransactionRequest): Promise<string> {
    return this.provider.call(await this.populateCall(tx));
  }

  public resolveName(name: string): Promise<string | null> {
    return this.provider.resolveName(name);
  }

  public async signTransaction(_tx: TransactionRequest): Promise<string> {
    // TODO if we split the signer for the in-process and json-rpc networks,
    // we can enable this method when using the in-process network or when the
    // json-rpc network has a private key
    throw new NotImplementedError("HardhatEthersSigner.signTransaction");
  }

  public async sendTransaction(
    tx: TransactionRequest
  ): Promise<ethers.TransactionResponse> {
    // This cannot be mined any earlier than any recent block
    const blockNumber = await this.provider.getBlockNumber();

    // Send the transaction
    const hash = await this._sendUncheckedTransaction(tx);

    // Unfortunately, JSON-RPC only provides and opaque transaction hash
    // for a response, and we need the actual transaction, so we poll
    // for it; it should show up very quickly

    return new Promise((resolve) => {
      const timeouts = [1000, 100];
      const checkTx = async () => {
        // Try getting the transaction
        const txPolled = await this.provider.getTransaction(hash);
        if (txPolled !== null) {
          resolve(txPolled.replaceableTransaction(blockNumber));
          return;
        }

        // Wait another 4 seconds
        setTimeout(() => {
          // eslint-disable-next-line @typescript-eslint/no-floating-promises
          checkTx();
        }, timeouts.pop() ?? 4000);
      };
      // eslint-disable-next-line @typescript-eslint/no-floating-promises
      checkTx();
    });
  }

  public signMessage(message: string | Uint8Array): Promise<string> {
    const resolvedMessage =
      typeof message === "string" ? toUtf8Bytes(message) : message;
    return this.provider.send("personal_sign", [
      hexlify(resolvedMessage),
      this.address.toLowerCase(),
    ]);
  }

  public async signTypedData(
    domain: ethers.TypedDataDomain,
    types: Record<string, ethers.TypedDataField[]>,
    value: Record<string, any>
  ): Promise<string> {
    const copiedValue = deepCopy(value);

    // Populate any ENS names (in-place)
    const populated = await TypedDataEncoder.resolveNames(
      domain,
      types,
      copiedValue,
      async (v: string) => {
        return v;
      }
    );

    return this.provider.send("eth_signTypedData_v4", [
      this.address.toLowerCase(),
      JSON.stringify(
        TypedDataEncoder.getPayload(populated.domain, types, populated.value),
        (_k, v) => {
          if (typeof v === "bigint") {
            return v.toString();
          }

          return v;
        }
      ),
    ]);
  }

  public async getAddress(): Promise<string> {
    return this.address;
  }

  public toJSON() {
    return `<SignerWithAddress ${this.address}>`;
  }

  private async _sendUncheckedTransaction(
    tx: TransactionRequest
  ): Promise<string> {
    const resolvedTx = deepCopy(tx);

    const promises: Array<Promise<void>> = [];

    // Make sure the from matches the sender
    if (resolvedTx.from !== null && resolvedTx.from !== undefined) {
      const _from = resolvedTx.from;
      promises.push(
        (async () => {
          const from = await resolveAddress(_from, this.provider);
          assertArgument(
            from !== null &&
              from !== undefined &&
              from.toLowerCase() === this.address.toLowerCase(),
            "from address mismatch",
            "transaction",
            tx
          );
          resolvedTx.from = from;
        })()
      );
    } else {
      resolvedTx.from = this.address;
    }

    if (resolvedTx.gasLimit === null || resolvedTx.gasLimit === undefined) {
      if (this._gasLimit !== undefined) {
        resolvedTx.gasLimit = this._gasLimit;
      } else {
        promises.push(
          (async () => {
            resolvedTx.gasLimit = await this.provider.estimateGas({
              ...resolvedTx,
              from: this.address,
            });
          })()
        );
      }
    }

    // The address may be an ENS name or Addressable
    if (resolvedTx.to !== null && resolvedTx.to !== undefined) {
      const _to = resolvedTx.to;
      promises.push(
        (async () => {
          resolvedTx.to = await resolveAddress(_to, this.provider);
        })()
      );
    }

    // Wait until all of our properties are filled in
    if (promises.length > 0) {
      await Promise.all(promises);
    }

    const hexTx = getRpcTransaction(resolvedTx);

    return this.provider.send("eth_sendTransaction", [hexTx]);
  }
}

// exported as an alias to make migration easier
export { HardhatEthersSigner as SignerWithAddress };

async function populate(
  signer: ethers.Signer,
  tx: TransactionRequest
): Promise<TransactionLike<string>> {
  const pop: any = copyRequest(tx);

  if (pop.to !== null && pop.to !== undefined) {
    pop.to = resolveAddress(pop.to, signer);
  }

  if (pop.from !== null && pop.from !== undefined) {
    const from = pop.from;
    pop.from = Promise.all([
      signer.getAddress(),
      resolveAddress(from, signer),
    ]).then(([address, resolvedFrom]) => {
      assertArgument(
        address.toLowerCase() === resolvedFrom.toLowerCase(),
        "transaction from mismatch",
        "tx.from",
        resolvedFrom
      );
      return address;
    });
  } else {
    pop.from = signer.getAddress();
  }

  return resolveProperties(pop);
}

const Primitive = "bigint,boolean,function,number,string,symbol".split(/,/g);
function deepCopy<T = any>(value: T): T {
  if (
    value === null ||
    value === undefined ||
    Primitive.indexOf(typeof value) >= 0
  ) {
    return value;
  }

  // Keep any Addressable
  if (typeof (value as any).getAddress === "function") {
    return value;
  }

  if (Array.isArray(value)) {
    return (value as any).map(deepCopy);
  }

  if (typeof value === "object") {
    return Object.keys(value).reduce((accum, key) => {
      accum[key] = (value as any)[key];
      return accum;
    }, {} as any);
  }

  throw new HardhatEthersError(
    `Assertion error: ${value as any} (${typeof value})`
  );
}
