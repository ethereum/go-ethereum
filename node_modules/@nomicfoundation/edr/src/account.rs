use edr_eth::signature::secret_key_from_str;
use napi::{bindgen_prelude::BigInt, Status};
use napi_derive::napi;

use crate::cast::TryCast;

/// An account that needs to be created during the genesis block.
#[napi(object)]
pub struct GenesisAccount {
    /// Account secret key
    pub secret_key: String,
    /// Account balance
    pub balance: BigInt,
}

impl TryFrom<GenesisAccount> for edr_provider::AccountConfig {
    type Error = napi::Error;

    fn try_from(value: GenesisAccount) -> Result<Self, Self::Error> {
        let secret_key = secret_key_from_str(&value.secret_key)
            .map_err(|e| napi::Error::new(Status::InvalidArg, e.to_string()))?;

        Ok(Self {
            secret_key,
            balance: value.balance.try_cast()?,
        })
    }
}
