---
title: Gas price oracle
---

The gas price oracle is a helper function of the Geth client that tries to find an
appropriate default gas price when sending transactions. It can be parametrized with the
following command line options:

- `gpomin`: lower limit of suggested gas price. This should be set at least as high as the
  `gasprice` setting usually used by miners so that your transactions will not be rejected
  automatically because of a too low price.

- `gpomax`: higher limit of suggested gas price. During load peaks when there is a
  competition between transactions to get into the blocks, the price needs to be limited,
  otherwise the oracle would eventually try to overbid everyone else at any price.

- `gpofull`: a block is considered "full" when a certain percentage of the block gas limit
  (specified in percents) is used up by transactions. If a block is not "full", that means
  that a transaction could have been accepted even with a minimal price offered.

- `gpobasedown`: an exponential ratio (specified in `1/1000ths`) by which the base price
  decreases when the lowest acceptable price of the last block is below the last base
  price.

- `gpobaseup`: an exponential ratio (specified in `1/1000ths`) by which the base price
  increases when the lowest acceptable price of the last block is over the last base
  price.

- `gpobasecf`: a correction factor (specified in percents) of the base price. The
  suggested price is the corrected base price, limited by `gpomin` and `gpomax`.

The lowest acceptable price is defined as a price that could have been enough to insert a
transaction into a certain block. Although this value varies slightly with the gas used by
the particular transaction, it is aproximated as follows: if the block is full, it is the
lowest transaction gas price found in that block. If the block is not full, it equals to
gpomin.

The base price is a moving value that is adjusted from block to block, up if it was lower
than the lowest acceptable price, down otherwise. Note that there is a slight amount of
randomness added to the correction factors so that your client will not behave absolutely
predictable on the market.

If you want to specify a constant for the default gas price and not use the oracle, set
both `gpomin` and `gpomax` to the same value.
