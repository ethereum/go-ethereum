use std::fmt::{Debug, Display};

#[allow(deprecated)]
// This is the only source file in production code where it's allowed to create
// `DangerousSecretKeyStr`.
use edr_eth::signature::{secret_key_from_str, DangerousSecretKeyStr};
use napi::{bindgen_prelude::BigInt, JsString, Status};
use napi_derive::napi;
use serde::Serialize;

use crate::cast::TryCast;

/// An account that needs to be created during the genesis block.
#[napi(object)]
pub struct GenesisAccount {
    // Using JsString here as it doesn't have `Debug`, `Display` and `Serialize` implementation
    // which prevents accidentally leaking the secret keys to error messages and logs.
    /// Account secret key
    pub secret_key: JsString,
    /// Account balance
    pub balance: BigInt,
}

impl TryFrom<GenesisAccount> for edr_provider::AccountConfig {
    type Error = napi::Error;

    fn try_from(value: GenesisAccount) -> Result<Self, Self::Error> {
        static_assertions::assert_not_impl_all!(JsString: Debug, Display, Serialize);
        // `k256::SecretKey` has `Debug` implementation, but it's opaque (only shows the
        // type name)
        static_assertions::assert_not_impl_any!(k256::SecretKey: Display, Serialize);

        let secret_key = value.secret_key.into_utf8()?;
        // This is the only place in production code where it's allowed to use
        // `DangerousSecretKeyStr`.
        #[allow(deprecated)]
        let secret_key_str = DangerousSecretKeyStr(secret_key.as_str()?);
        let secret_key: k256::SecretKey = secret_key_from_str(secret_key_str)
            .map_err(|e| napi::Error::new(Status::InvalidArg, e.to_string()))?;

        Ok(Self {
            secret_key,
            balance: value.balance.try_cast()?,
        })
    }
}
