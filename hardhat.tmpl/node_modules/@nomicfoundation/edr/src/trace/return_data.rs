//! Rewrite of `hardhat-network/provider/return-data.ts` from Hardhat.

use alloy_sol_types::SolError;
use napi::bindgen_prelude::{BigInt, Uint8Array};
use napi_derive::napi;

// Built-in error types
// See <https://docs.soliditylang.org/en/v0.8.26/control-structures.html#error-handling-assert-require-revert-and-exceptions>
alloy_sol_types::sol! {
  error Error(string);
  error Panic(uint256);
}

#[napi]
pub struct ReturnData {
    #[napi(readonly)]
    pub value: Uint8Array,
    selector: Option<[u8; 4]>,
}

#[napi]
impl ReturnData {
    #[napi(constructor)]
    pub fn new(value: Uint8Array) -> Self {
        let selector = if value.len() >= 4 {
            Some(value[0..4].try_into().unwrap())
        } else {
            None
        };

        Self { value, selector }
    }

    #[napi]
    pub fn is_empty(&self) -> bool {
        self.value.is_empty()
    }

    pub fn matches_selector(&self, selector: impl AsRef<[u8]>) -> bool {
        self.selector
            .map_or(false, |value| value == selector.as_ref())
    }

    #[napi]
    pub fn is_error_return_data(&self) -> bool {
        self.selector == Some(Error::SELECTOR)
    }

    #[napi]
    pub fn is_panic_return_data(&self) -> bool {
        self.selector == Some(Panic::SELECTOR)
    }

    #[napi]
    pub fn decode_error(&self) -> napi::Result<String> {
        if self.is_empty() {
            return Ok(String::new());
        }

        let result = Error::abi_decode(&self.value[..], false).map_err(|_err| {
            napi::Error::new(
                napi::Status::InvalidArg,
                "Expected return data to be a Error(string) and contain a valid string",
            )
        })?;

        Ok(result._0)
    }

    #[napi]
    pub fn decode_panic(&self) -> napi::Result<BigInt> {
        let result = Panic::abi_decode(&self.value[..], false).map_err(|_err| {
            napi::Error::new(
                napi::Status::InvalidArg,
                "Expected return data to be a Error(string) and contain a valid string",
            )
        })?;

        Ok(BigInt {
            sign_bit: false,
            words: result._0.as_limbs().to_vec(),
        })
    }
}
