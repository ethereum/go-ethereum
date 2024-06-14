import { Common } from "@nomicfoundation/ethereumjs-common";

import { LocalNodeConfig } from "../node-types";
import { HardforkName } from "../../../util/hardforks";

export function makeCommon({ chainId, networkId, hardfork }: LocalNodeConfig) {
  const common = Common.custom(
    {
      chainId,
      networkId,
    },
    {
      // ethereumjs uses this name for the merge hardfork
      hardfork:
        hardfork === HardforkName.MERGE ? "mergeForkIdTransition" : hardfork,
    }
  );

  return common;
}
