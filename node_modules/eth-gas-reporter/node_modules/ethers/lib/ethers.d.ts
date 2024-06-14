import { BaseContract, Contract, ContractFactory } from "@ethersproject/contracts";
import { BigNumber, FixedNumber } from "@ethersproject/bignumber";
import { Signer, VoidSigner } from "@ethersproject/abstract-signer";
import { Wallet } from "@ethersproject/wallet";
import * as constants from "@ethersproject/constants";
import * as providers from "@ethersproject/providers";
import { getDefaultProvider } from "@ethersproject/providers";
import { Wordlist, wordlists } from "@ethersproject/wordlists";
import * as utils from "./utils";
import { ErrorCode as errors } from "@ethersproject/logger";
import type { TypedDataDomain, TypedDataField } from "@ethersproject/abstract-signer";
import { BigNumberish } from "@ethersproject/bignumber";
import { Bytes, BytesLike, Signature } from "@ethersproject/bytes";
import { Transaction, UnsignedTransaction } from "@ethersproject/transactions";
import { version } from "./_version";
declare const logger: utils.Logger;
import { ContractFunction, ContractReceipt, ContractTransaction, Event, EventFilter, Overrides, PayableOverrides, CallOverrides, PopulatedTransaction, ContractInterface } from "@ethersproject/contracts";
export { Signer, Wallet, VoidSigner, getDefaultProvider, providers, BaseContract, Contract, ContractFactory, BigNumber, FixedNumber, constants, errors, logger, utils, wordlists, version, ContractFunction, ContractReceipt, ContractTransaction, Event, EventFilter, Overrides, PayableOverrides, CallOverrides, PopulatedTransaction, ContractInterface, TypedDataDomain, TypedDataField, BigNumberish, Bytes, BytesLike, Signature, Transaction, UnsignedTransaction, Wordlist };
//# sourceMappingURL=ethers.d.ts.map