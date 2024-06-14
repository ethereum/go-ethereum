export function hardhatWaffleIncompatibilityCheck() {
  if ((global as any).__HARDHAT_WAFFLE_IS_LOADED === true) {
    throw new Error(
      `You are using both @nomicfoundation/hardhat-chai-matchers and @nomiclabs/hardhat-waffle. They don't work correctly together, so please make sure you only use one.

We recommend you migrate to @nomicfoundation/hardhat-chai-matchers. Learn how to do it here: https://hardhat.org/migrate-from-waffle`
    );
  }

  (global as any).__HARDHAT_CHAI_MATCHERS_IS_LOADED = true;
}
