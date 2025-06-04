export interface HardhatMetadata {
  // A string identifying the version of Hardhat, for debugging purposes,
  // not meant to be displayed to users.
  clientVersion: string;

  // The chain's id. Used to sign transactions.
  chainId: number;

  // A 0x-prefixed hex-encoded 32 bytes id which uniquely identifies an instance/run
  // of Hardhat Network. Running Hardhat Network more than once (even with the same version
  // and parameters) will always result in different `instanceId`s.
  // Running `hardhat_reset` will change the `instanceId` of an existing Hardhat Network.
  instanceId: string;

  // The latest block's number in Hardhat Network
  latestBlockNumber: number;

  // The latest block's hash in Hardhat Network
  latestBlockHash: string;

  // This field is only present when Hardhat Network is forking another chain.
  forkedNetwork?: {
    // The chainId of the network that is being forked
    chainId: number;

    // The number of the block that the network forked from.
    forkBlockNumber: number;

    // The hash of the block that the network forked from.
    forkBlockHash: string;
  };
}
