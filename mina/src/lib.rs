mod mina;

use std::array::TryFromSliceError;

use mina::{HashParameter, Message};
use mina_signer::{BaseField, CurvePoint, PubKey, ScalarField, Signature};
use o1_utils::FieldHelpers;

pub const FIELD_SIZE: usize = 32;

/**
 * # Safety
 * this functions accepts raw pointer from golang
 */
#[no_mangle]
pub unsafe extern "C" fn poseidon(
    network_id: u8,
    field_ptr: *const u8,
    field_len: usize,
    output_ptr: *mut u8, // 32 bytes
) -> bool {
    if (field_ptr.is_null() && field_len != 0) || output_ptr.is_null() {
        return false;
    }

    let network_id = match network_id {
        0x00 => HashParameter::Mainnet,
        0x01 => HashParameter::Testnet,
        0x02 => HashParameter::Empty,
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

/**
 * # Safety
 * this functions accepts raw pointer from golang
 */
#[no_mangle]
pub unsafe extern "C" fn verify(
    network_id: u8,
    pubkey_x: *const u8,
    pubkey_y: *const u8,
    sig_rx: *const u8,
    sig_s: *const u8,
    field_ptr: *const u8,
    field_len: usize,
    output_ptr: *mut bool,
) -> bool {
    if pubkey_x.is_null()
        || pubkey_y.is_null()
        || sig_rx.is_null()
        || sig_s.is_null()
        || (field_ptr.is_null() && field_len != 0)
        || output_ptr.is_null()
    {
        return false;
    }

    let network_id = match network_id {
        0x00 => HashParameter::Mainnet,
        0x01 => HashParameter::Testnet,
        0x02 => HashParameter::Empty,
        _ => return false,
    };

    let pubkey_x = unsafe { std::slice::from_raw_parts(pubkey_x, FIELD_SIZE) };
    let pubkey_y = unsafe { std::slice::from_raw_parts(pubkey_y, FIELD_SIZE) };

    let pubkey = PubKey::from_point_unsafe(CurvePoint::new(
        match BaseField::from_bytes(pubkey_x) {
            Ok(x) => x,
            Err(_) => return false,
        },
        match BaseField::from_bytes(pubkey_y) {
            Ok(y) => y,
            Err(_) => return false,
        },
        false,
    ));

    let sig_rx = unsafe { std::slice::from_raw_parts(sig_rx, FIELD_SIZE) };
    let sig_s = unsafe { std::slice::from_raw_parts(sig_s, FIELD_SIZE) };

    let signature = Signature::new(
        match BaseField::from_bytes(sig_rx) {
            Ok(rx) => rx,
            Err(_) => return false,
        },
        match ScalarField::from_bytes(sig_s) {
            Ok(s) => s,
            Err(_) => return false,
        },
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

    let result = mina::verify(&signature, &pubkey, &msg, network_id);

    unsafe { *output_ptr = result };

    true
}

#[cfg(test)]
mod tests {
    use std::str::FromStr;

    use super::*;
    use num_bigint::BigUint;
    use serde::Deserialize;

    #[derive(Debug, Deserialize)]
    struct PoseidonTestVector {
        input: Vec<String>,
        output: String,
    }

    #[derive(Debug, Deserialize)]
    struct PoseidonTestVectors {
        test_vectors: Vec<PoseidonTestVector>,
    }

    #[test]
    fn poseidon_test_vectors() {
        let test_vectors: PoseidonTestVectors =
            serde_json::from_str(include_str!("test/poseidon_test_vectors.json")).unwrap();

        for test_vector in test_vectors.test_vectors {
            let mut output = [0u8; 32];

            let input = test_vector
                .input
                .iter()
                .flat_map(|input| BaseField::from_hex(input).unwrap().to_bytes())
                .collect::<Vec<u8>>();

            unsafe {
                assert!(poseidon(
                    0x02,
                    input.as_ptr(),
                    test_vector.input.len(),
                    output.as_mut_ptr()
                ))
            };

            assert_eq!(
                BaseField::from_bytes(&output).unwrap().to_hex(),
                test_vector.output
            );
        }
    }

    #[derive(Debug, Deserialize)]
    struct SignerTestVector {
        pub_key_x: String,
        pub_key_y: String,
        sig_rx: String,
        sig_s: String,
        fields: Vec<String>,
        output: bool,
    }

    #[derive(Debug, Deserialize)]
    struct SignerTestVectors {
        test_vectors: Vec<SignerTestVector>,
    }

    #[test]
    fn test_signer() {
        let test_vectors: SignerTestVectors =
            serde_json::from_str(include_str!("test/signer_test_vectors.json")).unwrap();

        for test_vector in test_vectors.test_vectors {
            let mut output = false;

            let pub_key_x =
                BaseField::from_biguint(&BigUint::from_str(&test_vector.pub_key_x).unwrap())
                    .unwrap()
                    .to_bytes();
            let pub_key_y =
                BaseField::from_biguint(&BigUint::from_str(&test_vector.pub_key_y).unwrap())
                    .unwrap()
                    .to_bytes();
            let sig_rx = BaseField::from_biguint(&BigUint::from_str(&test_vector.sig_rx).unwrap())
                .unwrap()
                .to_bytes();
            let sig_s = ScalarField::from_biguint(&BigUint::from_str(&test_vector.sig_s).unwrap())
                .unwrap()
                .to_bytes();
            let fields = test_vector
                .fields
                .iter()
                .flat_map(|input| {
                    BaseField::from_biguint(&BigUint::from_str(input).unwrap())
                        .unwrap()
                        .to_bytes()
                })
                .collect::<Vec<u8>>();

            unsafe {
                assert!(verify(
                    0x01,
                    pub_key_x.as_ptr(),
                    pub_key_y.as_ptr(),
                    sig_rx.as_ptr(),
                    sig_s.as_ptr(),
                    fields.as_ptr(),
                    test_vector.fields.len(),
                    &mut output
                ))
            };

            assert_eq!(output, test_vector.output);
        }
    }

    #[test]
    fn null_pointer() {
        unsafe {
            assert!(!poseidon(0x00, std::ptr::null(), 1, std::ptr::null_mut()));
            assert!(!verify(
                0x00,
                std::ptr::null(),
                std::ptr::null(),
                std::ptr::null(),
                std::ptr::null(),
                std::ptr::null(),
                0,
                std::ptr::null_mut()
            ));
        }
    }
}
