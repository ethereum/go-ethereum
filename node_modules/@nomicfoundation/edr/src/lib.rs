#![warn(missing_docs)]

//! NAPI bindings for the EDR EVM

#[global_allocator]
static ALLOC: mimalloc::MiMalloc = mimalloc::MiMalloc;

mod account;
mod block;
mod call_override;
mod cast;
mod config;
mod context;
mod debug_trace;
mod log;
mod logger;
mod provider;
mod result;
#[cfg(feature = "scenarios")]
mod scenarios;
mod subscribe;
mod trace;
mod withdrawal;
