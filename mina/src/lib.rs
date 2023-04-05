mod mina;

use std::array::TryFromSliceError;

use mina::{Message, NetworkId};
use mina_hasher::Hashable;
use mina_signer::{BaseField, CurvePoint, PubKey, ScalarField, Signature};
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
        0x00 => NetworkId::MAINNET,
        0x01 => NetworkId::TESTNET,
        0x02 => NetworkId::NULLNET,
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

#[no_mangle]
pub extern "C" fn verify(
    network_id: u8,
    pubkey_x: *const u8,
    pubkey_y: *const u8,
    sig_rx: *const u8,
    sig_s: *const u8,
    field_ptr: *const u8,
    field_len: usize,
    output_ptr: *mut bool,
) -> bool {
    let network_id = match network_id {
        0x00 => NetworkId::MAINNET,
        0x01 => NetworkId::TESTNET,
        0x02 => NetworkId::NULLNET,
        _ => return false,
    };

    let pubkey_x = unsafe { std::slice::from_raw_parts(pubkey_x, FIELD_SIZE) };
    let pubkey_y = unsafe { std::slice::from_raw_parts(pubkey_y, FIELD_SIZE) };

    let pubkey = PubKey::from_point_unsafe(CurvePoint::new(
        BaseField::from_bytes(pubkey_x).unwrap(),
        BaseField::from_bytes(pubkey_y).unwrap(),
        false,
    ));

    let sig_rx = unsafe { std::slice::from_raw_parts(sig_rx, FIELD_SIZE) };
    let sig_s = unsafe { std::slice::from_raw_parts(sig_s, FIELD_SIZE) };

    let signature = Signature::new(
        BaseField::from_bytes(sig_rx).unwrap(),
        ScalarField::from_bytes(sig_s).unwrap(),
    );

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

    println!("{:?}", msg.to_roinput().to_fields()[0].to_biguint());

    let result = mina::verify(&signature, &pubkey, &msg, network_id);

    unsafe { *output_ptr = result };

    true
}

#[cfg(test)]
mod tests {
    // use super::*;

    #[test]
    fn it_works() {}
}
