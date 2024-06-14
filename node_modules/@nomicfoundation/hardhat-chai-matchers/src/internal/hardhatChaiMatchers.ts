import { supportAddressable } from "./addressable";
import { supportBigNumber } from "./bigNumber";
import { supportEmit } from "./emit";
import { supportHexEqual } from "./hexEqual";
import { supportProperAddress } from "./properAddress";
import { supportProperHex } from "./properHex";
import { supportProperPrivateKey } from "./properPrivateKey";
import { supportChangeEtherBalance } from "./changeEtherBalance";
import { supportChangeEtherBalances } from "./changeEtherBalances";
import { supportChangeTokenBalance } from "./changeTokenBalance";
import { supportReverted } from "./reverted/reverted";
import { supportRevertedWith } from "./reverted/revertedWith";
import { supportRevertedWithCustomError } from "./reverted/revertedWithCustomError";
import { supportRevertedWithPanic } from "./reverted/revertedWithPanic";
import { supportRevertedWithoutReason } from "./reverted/revertedWithoutReason";
import { supportWithArgs } from "./withArgs";

export function hardhatChaiMatchers(
  chai: Chai.ChaiStatic,
  chaiUtils: Chai.ChaiUtils
) {
  supportAddressable(chai.Assertion, chaiUtils);
  supportBigNumber(chai.Assertion, chaiUtils);
  supportEmit(chai.Assertion, chaiUtils);
  supportHexEqual(chai.Assertion);
  supportProperAddress(chai.Assertion);
  supportProperHex(chai.Assertion);
  supportProperPrivateKey(chai.Assertion);
  supportChangeEtherBalance(chai.Assertion, chaiUtils);
  supportChangeEtherBalances(chai.Assertion, chaiUtils);
  supportChangeTokenBalance(chai.Assertion, chaiUtils);
  supportReverted(chai.Assertion, chaiUtils);
  supportRevertedWith(chai.Assertion, chaiUtils);
  supportRevertedWithCustomError(chai.Assertion, chaiUtils);
  supportRevertedWithPanic(chai.Assertion, chaiUtils);
  supportRevertedWithoutReason(chai.Assertion, chaiUtils);
  supportWithArgs(chai.Assertion, chaiUtils);
}
