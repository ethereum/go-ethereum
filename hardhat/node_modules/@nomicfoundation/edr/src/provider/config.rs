use std::{
    num::NonZeroU64,
    path::PathBuf,
    time::{Duration, SystemTime},
};

use edr_eth::HashMap;
use edr_provider::AccountConfig;
use napi::{
    bindgen_prelude::{BigInt, Buffer},
    Either,
};
use napi_derive::napi;

use crate::{account::GenesisAccount, block::BlobGas, cast::TryCast, config::SpecId};

/// Configuration for a chain
#[napi(object)]
pub struct ChainConfig {
    /// The chain ID
    pub chain_id: BigInt,
    /// The chain's supported hardforks
    pub hardforks: Vec<HardforkActivation>,
}

/// Configuration for forking a blockchain
#[napi(object)]
pub struct ForkConfig {
    /// The URL of the JSON-RPC endpoint to fork from
    pub json_rpc_url: String,
    /// The block number to fork from. If not provided, the latest safe block is
    /// used.
    pub block_number: Option<BigInt>,
    /// The HTTP headers to use when making requests to the JSON-RPC endpoint
    pub http_headers: Option<Vec<HttpHeader>>,
}

#[napi(object)]
pub struct HttpHeader {
    pub name: String,
    pub value: String,
}

/// Configuration for a hardfork activation
#[napi(object)]
pub struct HardforkActivation {
    /// The block number at which the hardfork is activated
    pub block_number: BigInt,
    /// The activated hardfork
    pub spec_id: SpecId,
}

#[napi(string_enum)]
#[doc = "The type of ordering to use when selecting blocks to mine."]
pub enum MineOrdering {
    #[doc = "Insertion order"]
    Fifo,
    #[doc = "Effective miner fee"]
    Priority,
}

/// Configuration for the provider's mempool.
#[napi(object)]
pub struct MemPoolConfig {
    pub order: MineOrdering,
}

#[napi(object)]
pub struct IntervalRange {
    pub min: BigInt,
    pub max: BigInt,
}

/// Configuration for the provider's miner.
#[napi(object)]
pub struct MiningConfig {
    pub auto_mine: bool,
    pub interval: Option<Either<BigInt, IntervalRange>>,
    pub mem_pool: MemPoolConfig,
}

/// Configuration for a provider
#[napi(object)]
pub struct ProviderConfig {
    /// Whether to allow blocks with the same timestamp
    pub allow_blocks_with_same_timestamp: bool,
    /// Whether to allow unlimited contract size
    pub allow_unlimited_contract_size: bool,
    /// Whether to return an `Err` when `eth_call` fails
    pub bail_on_call_failure: bool,
    /// Whether to return an `Err` when a `eth_sendTransaction` fails
    pub bail_on_transaction_failure: bool,
    /// The gas limit of each block
    pub block_gas_limit: BigInt,
    /// The directory to cache remote JSON-RPC responses
    pub cache_dir: Option<String>,
    /// The chain ID of the blockchain
    pub chain_id: BigInt,
    /// The configuration for chains
    pub chains: Vec<ChainConfig>,
    /// The address of the coinbase
    pub coinbase: Buffer,
    /// Enables RIP-7212
    pub enable_rip_7212: bool,
    /// The configuration for forking a blockchain. If not provided, a local
    /// blockchain will be created
    pub fork: Option<ForkConfig>,
    /// The genesis accounts of the blockchain
    pub genesis_accounts: Vec<GenesisAccount>,
    /// The hardfork of the blockchain
    pub hardfork: SpecId,
    /// The initial base fee per gas of the blockchain. Required for EIP-1559
    /// transactions and later
    pub initial_base_fee_per_gas: Option<BigInt>,
    /// The initial blob gas of the blockchain. Required for EIP-4844
    pub initial_blob_gas: Option<BlobGas>,
    /// The initial date of the blockchain, in seconds since the Unix epoch
    pub initial_date: Option<BigInt>,
    /// The initial parent beacon block root of the blockchain. Required for
    /// EIP-4788
    pub initial_parent_beacon_block_root: Option<Buffer>,
    /// The minimum gas price of the next block.
    pub min_gas_price: BigInt,
    /// The configuration for the miner
    pub mining: MiningConfig,
    /// The network ID of the blockchain
    pub network_id: BigInt,
}

impl TryFrom<ForkConfig> for edr_provider::hardhat_rpc_types::ForkConfig {
    type Error = napi::Error;

    fn try_from(value: ForkConfig) -> Result<Self, Self::Error> {
        let block_number: Option<u64> = value.block_number.map(TryCast::try_cast).transpose()?;
        let http_headers = value.http_headers.map(|http_headers| {
            http_headers
                .into_iter()
                .map(|HttpHeader { name, value }| (name, value))
                .collect()
        });

        Ok(Self {
            json_rpc_url: value.json_rpc_url,
            block_number,
            http_headers,
        })
    }
}

impl From<MemPoolConfig> for edr_provider::MemPoolConfig {
    fn from(value: MemPoolConfig) -> Self {
        Self {
            order: value.order.into(),
        }
    }
}

impl From<MineOrdering> for edr_evm::MineOrdering {
    fn from(value: MineOrdering) -> Self {
        match value {
            MineOrdering::Fifo => Self::Fifo,
            MineOrdering::Priority => Self::Priority,
        }
    }
}

impl TryFrom<MiningConfig> for edr_provider::MiningConfig {
    type Error = napi::Error;

    fn try_from(value: MiningConfig) -> Result<Self, Self::Error> {
        let mem_pool = value.mem_pool.into();

        let interval = value
            .interval
            .map(|interval| {
                let interval = match interval {
                    Either::A(interval) => {
                        let interval = interval.try_cast()?;
                        let interval = NonZeroU64::new(interval).ok_or_else(|| {
                            napi::Error::new(
                                napi::Status::GenericFailure,
                                "Interval must be greater than 0",
                            )
                        })?;

                        edr_provider::IntervalConfig::Fixed(interval)
                    }
                    Either::B(IntervalRange { min, max }) => edr_provider::IntervalConfig::Range {
                        min: min.try_cast()?,
                        max: max.try_cast()?,
                    },
                };

                napi::Result::Ok(interval)
            })
            .transpose()?;

        Ok(Self {
            auto_mine: value.auto_mine,
            interval,
            mem_pool,
        })
    }
}

impl TryFrom<ProviderConfig> for edr_provider::ProviderConfig {
    type Error = napi::Error;

    fn try_from(value: ProviderConfig) -> Result<Self, Self::Error> {
        let chains = value
            .chains
            .into_iter()
            .map(
                |ChainConfig {
                     chain_id,
                     hardforks,
                 }| {
                    let hardforks = hardforks
                        .into_iter()
                        .map(
                            |HardforkActivation {
                                 block_number,
                                 spec_id,
                             }| {
                                let block_number = block_number.try_cast()?;
                                let spec_id = spec_id.into();

                                Ok((block_number, spec_id))
                            },
                        )
                        .collect::<napi::Result<Vec<_>>>()?;

                    let chain_id = chain_id.try_cast()?;
                    Ok((chain_id, edr_eth::spec::HardforkActivations::new(hardforks)))
                },
            )
            .collect::<napi::Result<_>>()?;

        let block_gas_limit =
            NonZeroU64::new(value.block_gas_limit.try_cast()?).ok_or_else(|| {
                napi::Error::new(
                    napi::Status::GenericFailure,
                    "Block gas limit must be greater than 0",
                )
            })?;

        Ok(Self {
            accounts: value
                .genesis_accounts
                .into_iter()
                .map(AccountConfig::try_from)
                .collect::<napi::Result<Vec<_>>>()?,
            allow_blocks_with_same_timestamp: value.allow_blocks_with_same_timestamp,
            allow_unlimited_contract_size: value.allow_unlimited_contract_size,
            bail_on_call_failure: value.bail_on_call_failure,
            bail_on_transaction_failure: value.bail_on_transaction_failure,
            block_gas_limit,
            cache_dir: PathBuf::from(
                value
                    .cache_dir
                    .unwrap_or(String::from(edr_defaults::CACHE_DIR)),
            ),
            chain_id: value.chain_id.try_cast()?,
            chains,
            coinbase: value.coinbase.try_cast()?,
            enable_rip_7212: value.enable_rip_7212,
            fork: value.fork.map(TryInto::try_into).transpose()?,
            genesis_accounts: HashMap::new(),
            hardfork: value.hardfork.into(),
            initial_base_fee_per_gas: value
                .initial_base_fee_per_gas
                .map(TryCast::try_cast)
                .transpose()?,
            initial_blob_gas: value.initial_blob_gas.map(TryInto::try_into).transpose()?,
            initial_date: value
                .initial_date
                .map(|date| {
                    let elapsed_since_epoch = Duration::from_secs(date.try_cast()?);
                    napi::Result::Ok(SystemTime::UNIX_EPOCH + elapsed_since_epoch)
                })
                .transpose()?,
            initial_parent_beacon_block_root: value
                .initial_parent_beacon_block_root
                .map(TryCast::try_cast)
                .transpose()?,
            mining: value.mining.try_into()?,
            min_gas_price: value.min_gas_price.try_cast()?,
            network_id: value.network_id.try_cast()?,
        })
    }
}
