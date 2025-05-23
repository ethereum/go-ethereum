//! Port of `hardhat-network/stack-traces/debug.ts` from Hardhat.

use napi::bindgen_prelude::Either24;
use napi_derive::napi;

use super::solidity_stack_trace::{RevertErrorStackTraceEntry, SolidityStackTrace};
use crate::trace::return_data::ReturnData;

#[napi]
fn print_stack_trace(trace: SolidityStackTrace) -> napi::Result<()> {
    let entry_values = trace
        .into_iter()
        .map(|entry| match entry {
            Either24::A(entry) => serde_json::to_value(entry),
            Either24::B(entry) => serde_json::to_value(entry),
            Either24::C(entry) => serde_json::to_value(entry),
            Either24::D(entry) => serde_json::to_value(entry),
            Either24::F(entry) => serde_json::to_value(entry),
            Either24::G(entry) => serde_json::to_value(entry),
            Either24::H(entry) => serde_json::to_value(entry),
            Either24::I(entry) => serde_json::to_value(entry),
            Either24::J(entry) => serde_json::to_value(entry),
            Either24::K(entry) => serde_json::to_value(entry),
            Either24::L(entry) => serde_json::to_value(entry),
            Either24::M(entry) => serde_json::to_value(entry),
            Either24::N(entry) => serde_json::to_value(entry),
            Either24::O(entry) => serde_json::to_value(entry),
            Either24::P(entry) => serde_json::to_value(entry),
            Either24::Q(entry) => serde_json::to_value(entry),
            Either24::R(entry) => serde_json::to_value(entry),
            Either24::S(entry) => serde_json::to_value(entry),
            Either24::T(entry) => serde_json::to_value(entry),
            Either24::U(entry) => serde_json::to_value(entry),
            Either24::V(entry) => serde_json::to_value(entry),
            Either24::W(entry) => serde_json::to_value(entry),
            Either24::X(entry) => serde_json::to_value(entry),
            // Decode the error message from the return data
            Either24::E(entry @ RevertErrorStackTraceEntry { .. }) => {
                use serde::de::Error;

                let decoded_error_msg = ReturnData::new(entry.return_data.clone())
                    .decode_error()
                    .map_err(|e| {
                    serde_json::Error::custom(format_args!("Error decoding return data: {e}"))
                })?;

                let mut value = serde_json::to_value(entry)?;
                value["message"] = decoded_error_msg.into();
                Ok(value)
            }
        })
        .collect::<Result<Vec<_>, _>>()
        .map_err(|e| napi::Error::from_reason(format!("Error converting to JSON: {e}")))?;

    println!("{}", serde_json::to_string_pretty(&entry_values)?);

    Ok(())
}
