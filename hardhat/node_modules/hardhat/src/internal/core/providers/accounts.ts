import * as t from "io-ts";

import { EIP1193Provider, RequestArguments } from "../../../types";
import { assertHardhatInvariant, HardhatError } from "../errors";
import { ERRORS } from "../errors-list";
import {
  rpcAddress,
  rpcData,
  rpcQuantityToBigInt,
} from "../jsonrpc/types/base-types";
import {
  RpcTransactionRequest,
  rpcTransactionRequest,
} from "../jsonrpc/types/input/transactionRequest";
import { validateParams } from "../jsonrpc/types/input/validation";

import { normalizeToBigInt } from "../../../common/bigInt";
import { ProviderWrapperWithChainId } from "./chainId";
import { derivePrivateKeys } from "./util";
import { ProviderWrapper } from "./wrapper";

export interface JsonRpcTransactionData {
  from?: string;
  to?: string;
  gas?: string | number;
  gasPrice?: string | number;
  value?: string | number;
  data?: string;
  nonce?: string | number;
}

export class LocalAccountsProvider extends ProviderWrapperWithChainId {
  private _addressToPrivateKey: Map<string, Buffer> = new Map();

  constructor(
    provider: EIP1193Provider,
    localAccountsHexPrivateKeys: string[]
  ) {
    super(provider);

    this._initializePrivateKeys(localAccountsHexPrivateKeys);
  }

  public async request(args: RequestArguments): Promise<unknown> {
    const {
      ecsign,
      hashPersonalMessage,
      toRpcSig,
      toBytes,
      bytesToHex: bufferToHex,
    } = await import("@ethereumjs/util");
    const { signTyped } = await import("micro-eth-signer/typed-data");

    if (
      args.method === "eth_accounts" ||
      args.method === "eth_requestAccounts"
    ) {
      return [...this._addressToPrivateKey.keys()];
    }

    const params = this._getParams(args);

    if (args.method === "eth_sign") {
      if (params.length > 0) {
        const [address, data] = validateParams(params, rpcAddress, rpcData);

        if (address !== undefined) {
          if (data === undefined) {
            throw new HardhatError(ERRORS.NETWORK.ETHSIGN_MISSING_DATA_PARAM);
          }

          const privateKey = this._getPrivateKeyForAddress(address);
          const messageHash = hashPersonalMessage(toBytes(data));
          const signature = ecsign(messageHash, privateKey);
          return toRpcSig(signature.v, signature.r, signature.s);
        }
      }
    }

    if (args.method === "personal_sign") {
      if (params.length > 0) {
        const [data, address] = validateParams(params, rpcData, rpcAddress);

        if (data !== undefined) {
          if (address === undefined) {
            throw new HardhatError(
              ERRORS.NETWORK.PERSONALSIGN_MISSING_ADDRESS_PARAM
            );
          }

          const privateKey = this._getPrivateKeyForAddress(address);
          const messageHash = hashPersonalMessage(toBytes(data));
          const signature = ecsign(messageHash, privateKey);
          return toRpcSig(signature.v, signature.r, signature.s);
        }
      }
    }

    if (args.method === "eth_signTypedData_v4") {
      const [address, data] = validateParams(params, rpcAddress, t.any);

      if (data === undefined) {
        throw new HardhatError(ERRORS.NETWORK.ETHSIGN_MISSING_DATA_PARAM);
      }

      let typedMessage = data;
      if (typeof data === "string") {
        try {
          typedMessage = JSON.parse(data);
        } catch {
          throw new HardhatError(
            ERRORS.NETWORK.ETHSIGN_TYPED_DATA_V4_INVALID_DATA_PARAM
          );
        }
      }

      // if we don't manage the address, the method is forwarded
      const privateKey = this._getPrivateKeyForAddressOrNull(address);
      if (privateKey !== null) {
        // Explicitly set extraEntropy to false to make the signing result deterministic
        return signTyped(typedMessage, privateKey, false);
      }
    }

    if (args.method === "eth_sendTransaction" && params.length > 0) {
      const [txRequest] = validateParams(params, rpcTransactionRequest);

      if (txRequest.gas === undefined) {
        throw new HardhatError(
          ERRORS.NETWORK.MISSING_TX_PARAM_TO_SIGN_LOCALLY,
          { param: "gas" }
        );
      }

      if (txRequest.from === undefined) {
        throw new HardhatError(
          ERRORS.NETWORK.MISSING_TX_PARAM_TO_SIGN_LOCALLY,
          { param: "from" }
        );
      }

      const hasGasPrice = txRequest.gasPrice !== undefined;
      const hasEip1559Fields =
        txRequest.maxFeePerGas !== undefined ||
        txRequest.maxPriorityFeePerGas !== undefined;
      const hasEip7702Fields = txRequest.authorizationList !== undefined;

      if (!hasGasPrice && !hasEip1559Fields) {
        throw new HardhatError(ERRORS.NETWORK.MISSING_FEE_PRICE_FIELDS);
      }

      if (hasGasPrice && hasEip1559Fields) {
        throw new HardhatError(ERRORS.NETWORK.INCOMPATIBLE_FEE_PRICE_FIELDS);
      }

      if (hasGasPrice && hasEip7702Fields) {
        throw new HardhatError(ERRORS.NETWORK.INCOMPATIBLE_EIP7702_FIELDS);
      }

      if (hasEip1559Fields && txRequest.maxFeePerGas === undefined) {
        throw new HardhatError(
          ERRORS.NETWORK.MISSING_TX_PARAM_TO_SIGN_LOCALLY,
          { param: "maxFeePerGas" }
        );
      }

      if (hasEip1559Fields && txRequest.maxPriorityFeePerGas === undefined) {
        throw new HardhatError(
          ERRORS.NETWORK.MISSING_TX_PARAM_TO_SIGN_LOCALLY,
          { param: "maxPriorityFeePerGas" }
        );
      }

      if (txRequest.to === undefined && txRequest.data === undefined) {
        throw new HardhatError(
          ERRORS.NETWORK.DATA_FIELD_CANNOT_BE_NULL_WITH_NULL_ADDRESS
        );
      }

      if (txRequest.nonce === undefined) {
        txRequest.nonce = await this._getNonce(txRequest.from);
      }

      const privateKey = this._getPrivateKeyForAddress(txRequest.from!);

      const chainId = await this._getChainId();

      const rawTransaction = await this._getSignedTransaction(
        txRequest,
        chainId,
        privateKey
      );

      return this._wrappedProvider.request({
        method: "eth_sendRawTransaction",
        params: [bufferToHex(rawTransaction)],
      });
    }

    return this._wrappedProvider.request(args);
  }

  private _initializePrivateKeys(localAccountsHexPrivateKeys: string[]) {
    const {
      bytesToHex: bufferToHex,
      toBytes,
      privateToAddress,
    } = require("@ethereumjs/util");

    const privateKeys: Buffer[] = localAccountsHexPrivateKeys.map((h) =>
      toBytes(h)
    );

    for (const pk of privateKeys) {
      const address: string = bufferToHex(privateToAddress(pk)).toLowerCase();
      this._addressToPrivateKey.set(address, pk);
    }
  }

  private _getPrivateKeyForAddress(address: Buffer): Buffer {
    const { bytesToHex: bufferToHex } = require("@ethereumjs/util");
    const pk = this._addressToPrivateKey.get(bufferToHex(address));
    if (pk === undefined) {
      throw new HardhatError(ERRORS.NETWORK.NOT_LOCAL_ACCOUNT, {
        account: bufferToHex(address),
      });
    }

    return pk;
  }

  private _getPrivateKeyForAddressOrNull(address: Buffer): Buffer | null {
    try {
      return this._getPrivateKeyForAddress(address);
    } catch {
      return null;
    }
  }

  private async _getNonce(address: Buffer): Promise<bigint> {
    const { bytesToHex: bufferToHex } = await import("@ethereumjs/util");

    const response = (await this._wrappedProvider.request({
      method: "eth_getTransactionCount",
      params: [bufferToHex(address), "pending"],
    })) as string;

    return rpcQuantityToBigInt(response);
  }

  private async _getSignedTransaction(
    transactionRequest: RpcTransactionRequest,
    chainId: number,
    privateKey: Buffer
  ): Promise<Uint8Array> {
    const { bytesToHex, bytesToInt, bytesToBigInt } = await import(
      "@ethereumjs/util"
    );
    const { addr, Transaction } = await import("micro-eth-signer");

    const txData = {
      ...transactionRequest,
      gasLimit: transactionRequest.gas,
    };

    const accessList = txData.accessList?.map(({ address, storageKeys }) => {
      return {
        address: addr.addChecksum(bytesToHex(address)),
        storageKeys:
          storageKeys !== null ? storageKeys.map((k) => bytesToHex(k)) : [],
      };
    });

    const authorizationList = txData.authorizationList?.map(
      ({ chainId: authChainId, address, nonce, yParity, r, s }) => {
        return {
          chainId: authChainId,
          address: addr.addChecksum(bytesToHex(address)),
          nonce,
          yParity: bytesToInt(yParity),
          r: bytesToBigInt(r),
          s: bytesToBigInt(s),
        };
      }
    );

    const checksummedAddress = addr.addChecksum(
      bytesToHex(txData.to ?? new Uint8Array()),
      true
    );

    assertHardhatInvariant(
      txData.nonce !== undefined,
      "nonce should be defined"
    );

    let transaction;
    // strict mode is not meant to be used in the context of hardhat
    const strictMode = false;
    const baseTxParams = {
      to: checksummedAddress,
      nonce: txData.nonce,
      chainId: txData.chainId ?? normalizeToBigInt(chainId),
      value: txData.value ?? 0n,
      data: bytesToHex(txData.data ?? new Uint8Array()),
      gasLimit: txData.gasLimit,
    };

    if (authorizationList !== undefined) {
      assertHardhatInvariant(
        txData.maxFeePerGas !== undefined,
        "maxFeePerGas should be defined"
      );

      transaction = Transaction.prepare(
        {
          type: "eip7702",
          ...baseTxParams,
          maxFeePerGas: txData.maxFeePerGas,
          maxPriorityFeePerGas: txData.maxPriorityFeePerGas,
          accessList: accessList ?? [],
          authorizationList: authorizationList ?? [],
        },
        strictMode
      );
    } else if (txData.maxFeePerGas !== undefined) {
      transaction = Transaction.prepare(
        {
          type: "eip1559",
          ...baseTxParams,
          maxFeePerGas: txData.maxFeePerGas,
          maxPriorityFeePerGas: txData.maxPriorityFeePerGas,
          accessList: accessList ?? [],
        },
        strictMode
      );
    } else if (accessList !== undefined) {
      transaction = Transaction.prepare(
        {
          type: "eip2930",
          ...baseTxParams,
          gasPrice: txData.gasPrice ?? 0n,
          accessList,
        },
        strictMode
      );
    } else {
      transaction = Transaction.prepare(
        {
          type: "legacy",
          ...baseTxParams,
          gasPrice: txData.gasPrice ?? 0n,
        },
        strictMode
      );
    }

    // Explicitly set extraEntropy to false to make the signing result deterministic
    const signedTransaction = transaction.signBy(privateKey, false);

    return signedTransaction.toRawBytes();
  }
}

export class HDWalletProvider extends LocalAccountsProvider {
  constructor(
    provider: EIP1193Provider,
    mnemonic: string,
    hdpath: string = "m/44'/60'/0'/0/",
    initialIndex: number = 0,
    count: number = 10,
    passphrase: string = ""
  ) {
    // NOTE: If mnemonic has space or newline at the beginning or end, it will be trimmed.
    // This is because mnemonic containing them may generate different private keys.
    const trimmedMnemonic = mnemonic.trim();
    const privateKeys = derivePrivateKeys(
      trimmedMnemonic,
      hdpath,
      initialIndex,
      count,
      passphrase
    );

    const { bytesToHex: bufferToHex } = require("@ethereumjs/util");
    const privateKeysAsHex = privateKeys.map((pk) => bufferToHex(pk));
    super(provider, privateKeysAsHex);
  }
}

abstract class SenderProvider extends ProviderWrapper {
  public async request(args: RequestArguments): Promise<unknown> {
    const method = args.method;
    const params = this._getParams(args);

    if (
      method === "eth_sendTransaction" ||
      method === "eth_call" ||
      method === "eth_estimateGas"
    ) {
      // TODO: Should we validate this type?
      const tx: JsonRpcTransactionData = params[0];

      if (tx !== undefined && tx.from === undefined) {
        const senderAccount = await this._getSender();

        if (senderAccount !== undefined) {
          tx.from = senderAccount;
        } else if (method === "eth_sendTransaction") {
          throw new HardhatError(ERRORS.NETWORK.NO_REMOTE_ACCOUNT_AVAILABLE);
        }
      }
    }

    return this._wrappedProvider.request(args);
  }

  protected abstract _getSender(): Promise<string | undefined>;
}

export class AutomaticSenderProvider extends SenderProvider {
  private _firstAccount: string | undefined;

  protected async _getSender(): Promise<string | undefined> {
    if (this._firstAccount === undefined) {
      const accounts = (await this._wrappedProvider.request({
        method: "eth_accounts",
      })) as string[];

      this._firstAccount = accounts[0];
    }

    return this._firstAccount;
  }
}

export class FixedSenderProvider extends SenderProvider {
  constructor(provider: EIP1193Provider, private readonly _sender: string) {
    super(provider);
  }

  protected async _getSender(): Promise<string | undefined> {
    return this._sender;
  }
}
