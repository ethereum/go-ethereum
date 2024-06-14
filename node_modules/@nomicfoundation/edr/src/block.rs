use edr_eth::{Address, Bytes, B256, B64};
use napi::bindgen_prelude::{BigInt, Buffer};
use napi_derive::napi;

use crate::{cast::TryCast, withdrawal::Withdrawal};

#[napi(object)]
pub struct BlockOptions {
    /// The parent block's hash
    pub parent_hash: Option<Buffer>,
    /// The block's beneficiary
    pub beneficiary: Option<Buffer>,
    /// The state's root hash
    pub state_root: Option<Buffer>,
    /// The block's difficulty
    pub difficulty: Option<BigInt>,
    /// The block's number
    pub number: Option<BigInt>,
    /// The block's gas limit
    pub gas_limit: Option<BigInt>,
    /// The block's timestamp
    pub timestamp: Option<BigInt>,
    /// The block's extra data
    pub extra_data: Option<Buffer>,
    /// The block's mix hash (or prevrandao)
    pub mix_hash: Option<Buffer>,
    /// The block's nonce
    pub nonce: Option<Buffer>,
    /// The block's base gas fee
    pub base_fee: Option<BigInt>,
    /// The block's withdrawals
    pub withdrawals: Option<Vec<Withdrawal>>,
    /// Blob gas was added by EIP-4844 and is ignored in older headers.
    pub blob_gas: Option<BlobGas>,
    /// The hash tree root of the parent beacon block for the given execution
    /// block (EIP-4788).
    pub parent_beacon_block_root: Option<Buffer>,
}

impl TryFrom<BlockOptions> for edr_eth::block::BlockOptions {
    type Error = napi::Error;

    #[cfg_attr(feature = "tracing", tracing::instrument(skip_all))]
    fn try_from(value: BlockOptions) -> Result<Self, Self::Error> {
        Ok(Self {
            parent_hash: value
                .parent_hash
                .map(TryCast::<B256>::try_cast)
                .transpose()?,
            beneficiary: value
                .beneficiary
                .map(TryCast::<Address>::try_cast)
                .transpose()?,
            state_root: value
                .state_root
                .map(TryCast::<B256>::try_cast)
                .transpose()?,
            difficulty: value
                .difficulty
                .map_or(Ok(None), |difficulty| difficulty.try_cast().map(Some))?,
            number: value
                .number
                .map_or(Ok(None), |number| number.try_cast().map(Some))?,
            gas_limit: value
                .gas_limit
                .map_or(Ok(None), |gas_limit| gas_limit.try_cast().map(Some))?,
            timestamp: value
                .timestamp
                .map_or(Ok(None), |timestamp| timestamp.try_cast().map(Some))?,
            extra_data: value
                .extra_data
                .map(|extra_data| Bytes::copy_from_slice(&extra_data)),
            mix_hash: value.mix_hash.map(TryCast::<B256>::try_cast).transpose()?,
            nonce: value.nonce.map(TryCast::<B64>::try_cast).transpose()?,
            base_fee: value
                .base_fee
                .map_or(Ok(None), |basefee| basefee.try_cast().map(Some))?,
            withdrawals: value
                .withdrawals
                .map(|withdrawals| {
                    withdrawals
                        .into_iter()
                        .map(edr_eth::withdrawal::Withdrawal::try_from)
                        .collect()
                })
                .transpose()?,
            blob_gas: value
                .blob_gas
                .map(edr_eth::block::BlobGas::try_from)
                .transpose()?,
            parent_beacon_block_root: value
                .parent_beacon_block_root
                .map(TryCast::<B256>::try_cast)
                .transpose()?,
        })
    }
}

/// Information about the blob gas used in a block.
#[napi(object)]
pub struct BlobGas {
    /// The total amount of blob gas consumed by the transactions within the
    /// block.
    pub gas_used: BigInt,
    /// The running total of blob gas consumed in excess of the target, prior to
    /// the block. Blocks with above-target blob gas consumption increase this
    /// value, blocks with below-target blob gas consumption decrease it
    /// (bounded at 0).
    pub excess_gas: BigInt,
}

impl TryFrom<BlobGas> for edr_eth::block::BlobGas {
    type Error = napi::Error;

    fn try_from(value: BlobGas) -> Result<Self, Self::Error> {
        Ok(Self {
            gas_used: BigInt::try_cast(value.gas_used)?,
            excess_gas: BigInt::try_cast(value.excess_gas)?,
        })
    }
}
