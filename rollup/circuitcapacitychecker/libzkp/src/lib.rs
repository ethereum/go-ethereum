#![feature(once_cell)]

pub mod checker {
    use crate::utils::{c_char_to_str, c_char_to_vec, vec_to_c_char};
    use anyhow::{anyhow, bail, Error};
    use libc::c_char;
    use prover::{
        zkevm::{CircuitCapacityChecker, RowUsage},
        BlockTrace,
    };
    use serde_derive::{Deserialize, Serialize};
    use std::cell::OnceCell;
    use std::collections::HashMap;
    use std::panic;
    use std::ptr::null;

    #[derive(Debug, Clone, Deserialize, Serialize)]
    pub struct CommonResult {
        pub error: Option<String>,
    }

    #[derive(Debug, Clone, Deserialize, Serialize)]
    pub struct RowUsageResult {
        pub acc_row_usage: Option<RowUsage>,
        pub error: Option<String>,
    }

    #[derive(Debug, Clone, Deserialize, Serialize)]
    pub struct TxNumResult {
        pub tx_num: u64,
        pub error: Option<String>,
    }

    static mut CHECKERS: OnceCell<HashMap<u64, CircuitCapacityChecker>> = OnceCell::new();

    /// # Safety
    #[no_mangle]
    pub unsafe extern "C" fn init() {
        env_logger::Builder::from_env(env_logger::Env::default().default_filter_or("debug"))
            .format_timestamp_millis()
            .init();
        let checkers = HashMap::new();
        CHECKERS
            .set(checkers)
            .expect("circuit capacity checker initialized twice");
    }

    /// # Safety
    #[no_mangle]
    pub unsafe extern "C" fn new_circuit_capacity_checker() -> u64 {
        let checkers = CHECKERS
            .get_mut()
            .expect("fail to get circuit capacity checkers map in new_circuit_capacity_checker");
        let id = checkers.len() as u64;
        let checker = CircuitCapacityChecker::new();
        checkers.insert(id, checker);
        id
    }

    /// # Safety
    #[no_mangle]
    pub unsafe extern "C" fn reset_circuit_capacity_checker(id: u64) {
        CHECKERS
            .get_mut()
            .expect("fail to get circuit capacity checkers map in reset_circuit_capacity_checker")
            .get_mut(&id)
            .unwrap_or_else(|| panic!("fail to get circuit capacity checker (id: {id:?}) in reset_circuit_capacity_checker"))
            .reset()
    }

    /// # Safety
    #[no_mangle]
    pub unsafe extern "C" fn apply_tx(id: u64, tx_traces: *const c_char) -> *const c_char {
        let result = apply_tx_inner(id, tx_traces);
        let r = match result {
            Ok(acc_row_usage) => {
                log::debug!(
                    "id: {:?}, acc_row_usage: {:?}",
                    id,
                    acc_row_usage.row_number,
                );
                RowUsageResult {
                    acc_row_usage: Some(acc_row_usage),
                    error: None,
                }
            }
            Err(e) => RowUsageResult {
                acc_row_usage: None,
                error: Some(format!("{e:?}")),
            },
        };
        serde_json::to_vec(&r).map_or(null(), vec_to_c_char)
    }

    unsafe fn apply_tx_inner(id: u64, tx_traces: *const c_char) -> Result<RowUsage, Error> {
        log::debug!(
            "ccc apply_tx raw input, id: {:?}, tx_traces: {:?}",
            id,
            c_char_to_str(tx_traces)?
        );
        let tx_traces_vec = c_char_to_vec(tx_traces);
        let traces = serde_json::from_slice::<BlockTrace>(&tx_traces_vec)?;

        if traces.transactions.len() != 1 {
            bail!("traces.transactions.len() != 1");
        }
        if traces.execution_results.len() != 1 {
            bail!("traces.execution_results.len() != 1");
        }
        if traces.tx_storage_trace.len() != 1 {
            bail!("traces.tx_storage_trace.len() != 1");
        }

        let r = panic::catch_unwind(|| {
            CHECKERS
                .get_mut()
                .ok_or(anyhow!(
                    "fail to get circuit capacity checkers map in apply_tx"
                ))?
                .get_mut(&id)
                .ok_or(anyhow!(
                    "fail to get circuit capacity checker (id: {id:?}) in apply_tx"
                ))?
                .estimate_circuit_capacity(&[traces])
        });
        match r {
            Ok(result) => result,
            Err(e) => {
                bail!("estimate_circuit_capacity (id: {id:?}) error in apply_tx, error: {e:?}")
            }
        }
    }

    /// # Safety
    #[no_mangle]
    pub unsafe extern "C" fn apply_block(id: u64, block_trace: *const c_char) -> *const c_char {
        let result = apply_block_inner(id, block_trace);
        let r = match result {
            Ok(acc_row_usage) => {
                log::debug!(
                    "id: {:?}, acc_row_usage: {:?}",
                    id,
                    acc_row_usage.row_number,
                );
                RowUsageResult {
                    acc_row_usage: Some(acc_row_usage),
                    error: None,
                }
            }
            Err(e) => RowUsageResult {
                acc_row_usage: None,
                error: Some(format!("{e:?}")),
            },
        };
        serde_json::to_vec(&r).map_or(null(), vec_to_c_char)
    }

    unsafe fn apply_block_inner(id: u64, block_trace: *const c_char) -> Result<RowUsage, Error> {
        log::debug!(
            "ccc apply_block raw input, id: {:?}, block_trace: {:?}",
            id,
            c_char_to_str(block_trace)?
        );
        let block_trace = c_char_to_vec(block_trace);
        let traces = serde_json::from_slice::<BlockTrace>(&block_trace)?;

        let r = panic::catch_unwind(|| {
            CHECKERS
                .get_mut()
                .ok_or(anyhow!(
                    "fail to get circuit capacity checkers map in apply_block"
                ))?
                .get_mut(&id)
                .ok_or(anyhow!(
                    "fail to get circuit capacity checker (id: {id:?}) in apply_block"
                ))?
                .estimate_circuit_capacity(&[traces])
        });
        match r {
            Ok(result) => result,
            Err(e) => {
                bail!("estimate_circuit_capacity (id: {id:?}) error in apply_block, error: {e:?}")
            }
        }
    }

    /// # Safety
    #[no_mangle]
    pub unsafe extern "C" fn get_tx_num(id: u64) -> *const c_char {
        let result = get_tx_num_inner(id);
        let r = match result {
            Ok(tx_num) => {
                log::debug!("id: {id}, tx_num: {tx_num}");
                TxNumResult {
                    tx_num,
                    error: None,
                }
            }
            Err(e) => TxNumResult {
                tx_num: 0,
                error: Some(format!("{e:?}")),
            },
        };
        serde_json::to_vec(&r).map_or(null(), vec_to_c_char)
    }

    unsafe fn get_tx_num_inner(id: u64) -> Result<u64, Error> {
        log::debug!("ccc get_tx_num raw input, id: {id}");
        panic::catch_unwind(|| {
            Ok(CHECKERS
                .get_mut()
                .ok_or(anyhow!(
                    "fail to get circuit capacity checkers map in get_tx_num"
                ))?
                .get_mut(&id)
                .ok_or(anyhow!(
                    "fail to get circuit capacity checker (id: {id}) in get_tx_num"
                ))?
                .get_tx_num() as u64)
        })
        .map_or_else(
            |e| bail!("circuit capacity checker (id: {id}) error in get_tx_num: {e:?}"),
            |result| result,
        )
    }

    /// # Safety
    #[no_mangle]
    pub unsafe extern "C" fn set_light_mode(id: u64, light_mode: bool) -> *const c_char {
        let result = set_light_mode_inner(id, light_mode);
        let r = match result {
            Ok(()) => CommonResult { error: None },
            Err(e) => CommonResult {
                error: Some(format!("{e:?}")),
            },
        };
        serde_json::to_vec(&r).map_or(null(), vec_to_c_char)
    }

    unsafe fn set_light_mode_inner(id: u64, light_mode: bool) -> Result<(), Error> {
        log::debug!("ccc set_light_mode raw input, id: {id}");
        panic::catch_unwind(|| {
            CHECKERS
                .get_mut()
                .ok_or(anyhow!(
                    "fail to get circuit capacity checkers map in set_light_mode"
                ))?
                .get_mut(&id)
                .ok_or(anyhow!(
                    "fail to get circuit capacity checker (id: {id}) in set_light_mode"
                ))?
                .set_light_mode(light_mode);
            Ok(())
        })
        .map_or_else(
            |e| bail!("circuit capacity checker (id: {id}) error in set_light_mode: {e:?}"),
            |result| result,
        )
    }
}

pub mod utils {
    use std::ffi::{CStr, CString};
    use std::os::raw::c_char;
    use std::str::Utf8Error;

    /// # Safety
    #[no_mangle]
    pub unsafe extern "C" fn free_c_chars(ptr: *mut c_char) {
        if ptr.is_null() {
            log::warn!("Try to free an empty pointer!");
            return;
        }

        let _ = CString::from_raw(ptr);
    }

    #[allow(dead_code)]
    pub(crate) fn c_char_to_str(c: *const c_char) -> Result<&'static str, Utf8Error> {
        let cstr = unsafe { CStr::from_ptr(c) };
        cstr.to_str()
    }

    #[allow(dead_code)]
    pub(crate) fn c_char_to_vec(c: *const c_char) -> Vec<u8> {
        let cstr = unsafe { CStr::from_ptr(c) };
        cstr.to_bytes().to_vec()
    }

    #[allow(dead_code)]
    pub(crate) fn vec_to_c_char(bytes: Vec<u8>) -> *const c_char {
        CString::new(bytes)
            .expect("fail to create new CString from bytes")
            .into_raw()
    }

    #[allow(dead_code)]
    pub(crate) fn bool_to_int(b: bool) -> u8 {
        match b {
            true => 1,
            false => 0,
        }
    }
}
