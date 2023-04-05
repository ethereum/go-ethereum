mod mina;

use std::array::TryFromSliceError;

use mina::{Message, NetworkId};
use mina_signer::{BaseField, CurvePoint, PubKey, Signature, ScalarField};
use o1_utils::FieldHelpers;

pub const FIELD_SIZE: usize = 32;

#[no_mangle]
pub extern "C" fn poseidon_hash(
    network_id: u8,
    field_ptr: *const u8,
    field_len: usize,
    output_ptr: *mut u8, // 32 bytes
) -> bool {
    let network_id = match network_id {
        0x00 => NetworkId::TESTNET,
        0x01 => NetworkId::MAINNET,
        0xff => NetworkId::NULLNET,
        _ => return false,
    };

    let fields = unsafe { std::slice::from_raw_parts(field_ptr, field_len * FIELD_SIZE) };

    let fields = match fields
        .chunks(FIELD_SIZE)
        .map(|chunk| chunk[..32].try_into())
        .collect::<Result<Vec<[u8; 32]>, TryFromSliceError>>()
    {
        Ok(fields) => fields,
        Err(_) => return false,
    };

    let msg = match Message::from_bytes_slice(&fields) {
        Ok(msg) => msg,
        Err(_) => return false,
    };

    let hash = mina::poseidon(&msg, network_id);

    let output = unsafe { std::slice::from_raw_parts_mut(output_ptr, FIELD_SIZE) };

    output.copy_from_slice(&hash.to_bytes());

    true
}

#[cfg(test)]
mod tests {
    // use super::*;

    #[test]
    fn it_works() {}
}
