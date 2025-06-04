use napi_derive::napi;

#[napi]
pub fn link_hex_string_bytecode(code: String, address: String, position: u32) -> String {
    edr_solidity::library_utils::link_hex_string_bytecode(code, &address, position)
}
