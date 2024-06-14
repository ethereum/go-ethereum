/**
 * This function returns a number that should be safe to consider as the
 * largest possible reorg in a network.
 *
 * If there's not such a number, or we aren't aware of it, this function
 * returns undefined.
 */
export function getLargestPossibleReorg(networkId: number): bigint | undefined {
  // mainnet
  if (networkId === 1) {
    return 32n;
  }

  // Kovan
  if (networkId === 42) {
    return 32n;
  }

  // Goerli
  if (networkId === 5) {
    return 32n;
  }

  // Rinkeby
  if (networkId === 4) {
    return 32n;
  }

  // Ropsten
  if (networkId === 3) {
    return 100n;
  }

  // xDai
  if (networkId === 100) {
    return 38n;
  }
}

export const FALLBACK_MAX_REORG = 128n;
