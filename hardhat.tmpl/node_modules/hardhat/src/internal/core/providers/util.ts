import {
  HardhatNetworkAccountConfig,
  HardhatNetworkAccountsConfig,
} from "../../../types";
import { deriveKeyFromMnemonicAndPath } from "../../util/keys-derivation";
import { DEFAULT_HARDHAT_NETWORK_BALANCE } from "../config/default-config";
import { HardhatError } from "../errors";
import { ERRORS } from "../errors-list";

const HD_PATH_REGEX = /^m(:?\/\d+'?)+\/?$/;

export function derivePrivateKeys(
  mnemonic: string,
  hdpath: string,
  initialIndex: number,
  count: number,
  passphrase: string
): Buffer[] {
  if (hdpath.match(HD_PATH_REGEX) === null) {
    throw new HardhatError(ERRORS.NETWORK.INVALID_HD_PATH, { path: hdpath });
  }

  if (!hdpath.endsWith("/")) {
    hdpath += "/";
  }

  const privateKeys: Buffer[] = [];

  for (let i = initialIndex; i < initialIndex + count; i++) {
    const privateKey = deriveKeyFromMnemonicAndPath(
      mnemonic,
      hdpath + i.toString(),
      passphrase
    );

    if (privateKey === undefined) {
      throw new HardhatError(ERRORS.NETWORK.CANT_DERIVE_KEY, {
        mnemonic,
        path: hdpath,
      });
    }

    privateKeys.push(privateKey);
  }

  return privateKeys;
}

export function normalizeHardhatNetworkAccountsConfig(
  accountsConfig: HardhatNetworkAccountsConfig
): HardhatNetworkAccountConfig[] {
  if (Array.isArray(accountsConfig)) {
    return accountsConfig;
  }

  const { bytesToHex } = require("@ethereumjs/util");

  return derivePrivateKeys(
    accountsConfig.mnemonic,
    accountsConfig.path,
    accountsConfig.initialIndex,
    accountsConfig.count,
    accountsConfig.passphrase
  ).map((pk) => ({
    privateKey: bytesToHex(pk),
    balance: accountsConfig.accountsBalance ?? DEFAULT_HARDHAT_NETWORK_BALANCE,
  }));
}
