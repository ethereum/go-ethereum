use edr_evm::trace::AfterMessage;
use napi::{
    bindgen_prelude::{BigInt, Buffer, Either3},
    Either, Env, JsBuffer, JsBufferValue,
};
use napi_derive::napi;

use crate::log::ExecutionLog;

/// The possible reasons for successful termination of the EVM.
#[napi]
pub enum SuccessReason {
    /// The opcode `STOP` was called
    Stop,
    /// The opcode `RETURN` was called
    Return,
    /// The opcode `SELFDESTRUCT` was called
    SelfDestruct,
}

impl From<edr_evm::SuccessReason> for SuccessReason {
    fn from(eval: edr_evm::SuccessReason) -> Self {
        match eval {
            edr_evm::SuccessReason::Stop => Self::Stop,
            edr_evm::SuccessReason::Return => Self::Return,
            edr_evm::SuccessReason::SelfDestruct => Self::SelfDestruct,
        }
    }
}

impl From<SuccessReason> for edr_evm::SuccessReason {
    fn from(value: SuccessReason) -> Self {
        match value {
            SuccessReason::Stop => Self::Stop,
            SuccessReason::Return => Self::Return,
            SuccessReason::SelfDestruct => Self::SelfDestruct,
        }
    }
}

#[napi(object)]
pub struct CallOutput {
    /// Return value
    pub return_value: JsBuffer,
}

#[napi(object)]
pub struct CreateOutput {
    /// Return value
    pub return_value: JsBuffer,
    /// Optionally, a 160-bit address
    pub address: Option<Buffer>,
}

/// The result when the EVM terminates successfully.
#[napi(object)]
pub struct SuccessResult {
    /// The reason for termination
    pub reason: SuccessReason,
    /// The amount of gas used
    pub gas_used: BigInt,
    /// The amount of gas refunded
    pub gas_refunded: BigInt,
    /// The logs
    pub logs: Vec<ExecutionLog>,
    /// The transaction output
    pub output: Either<CallOutput, CreateOutput>,
}

/// The result when the EVM terminates due to a revert.
#[napi(object)]
pub struct RevertResult {
    /// The amount of gas used
    pub gas_used: BigInt,
    /// The transaction output
    pub output: JsBuffer,
}

/// Indicates that the EVM has experienced an exceptional halt. This causes
/// execution to immediately end with all gas being consumed.
#[napi]
pub enum ExceptionalHalt {
    OutOfGas,
    OpcodeNotFound,
    InvalidFEOpcode,
    InvalidJump,
    NotActivated,
    StackUnderflow,
    StackOverflow,
    OutOfOffset,
    CreateCollision,
    PrecompileError,
    NonceOverflow,
    /// Create init code size exceeds limit (runtime).
    CreateContractSizeLimit,
    /// Error on created contract that begins with EF
    CreateContractStartingWithEF,
    /// EIP-3860: Limit and meter initcode. Initcode size limit exceeded.
    CreateInitCodeSizeLimit,
}

impl From<edr_evm::HaltReason> for ExceptionalHalt {
    fn from(halt: edr_evm::HaltReason) -> Self {
        match halt {
            edr_evm::HaltReason::OutOfGas(..) => ExceptionalHalt::OutOfGas,
            edr_evm::HaltReason::OpcodeNotFound => ExceptionalHalt::OpcodeNotFound,
            edr_evm::HaltReason::InvalidFEOpcode => ExceptionalHalt::InvalidFEOpcode,
            edr_evm::HaltReason::InvalidJump => ExceptionalHalt::InvalidJump,
            edr_evm::HaltReason::NotActivated => ExceptionalHalt::NotActivated,
            edr_evm::HaltReason::StackUnderflow => ExceptionalHalt::StackUnderflow,
            edr_evm::HaltReason::StackOverflow => ExceptionalHalt::StackOverflow,
            edr_evm::HaltReason::OutOfOffset => ExceptionalHalt::OutOfOffset,
            edr_evm::HaltReason::CreateCollision => ExceptionalHalt::CreateCollision,
            edr_evm::HaltReason::PrecompileError => ExceptionalHalt::PrecompileError,
            edr_evm::HaltReason::NonceOverflow => ExceptionalHalt::NonceOverflow,
            edr_evm::HaltReason::CreateContractSizeLimit => {
                ExceptionalHalt::CreateContractSizeLimit
            }
            edr_evm::HaltReason::CreateContractStartingWithEF => {
                ExceptionalHalt::CreateContractStartingWithEF
            }
            edr_evm::HaltReason::CreateInitCodeSizeLimit => {
                ExceptionalHalt::CreateInitCodeSizeLimit
            }
            edr_evm::HaltReason::OverflowPayment
            | edr_evm::HaltReason::StateChangeDuringStaticCall
            | edr_evm::HaltReason::CallNotAllowedInsideStatic
            | edr_evm::HaltReason::OutOfFunds
            | edr_evm::HaltReason::CallTooDeep => {
                unreachable!("Internal halts that can be only found inside Inspector: {halt:?}")
            }
        }
    }
}

impl From<ExceptionalHalt> for edr_evm::HaltReason {
    fn from(value: ExceptionalHalt) -> Self {
        match value {
            ExceptionalHalt::OutOfGas => Self::OutOfGas(edr_evm::OutOfGasError::Basic),
            ExceptionalHalt::OpcodeNotFound => Self::OpcodeNotFound,
            ExceptionalHalt::InvalidFEOpcode => Self::InvalidFEOpcode,
            ExceptionalHalt::InvalidJump => Self::InvalidJump,
            ExceptionalHalt::NotActivated => Self::NotActivated,
            ExceptionalHalt::StackUnderflow => Self::StackUnderflow,
            ExceptionalHalt::StackOverflow => Self::StackOverflow,
            ExceptionalHalt::OutOfOffset => Self::OutOfOffset,
            ExceptionalHalt::CreateCollision => Self::CreateCollision,
            ExceptionalHalt::PrecompileError => Self::PrecompileError,
            ExceptionalHalt::NonceOverflow => Self::NonceOverflow,
            ExceptionalHalt::CreateContractSizeLimit => Self::CreateContractSizeLimit,
            ExceptionalHalt::CreateContractStartingWithEF => Self::CreateContractStartingWithEF,
            ExceptionalHalt::CreateInitCodeSizeLimit => Self::CreateInitCodeSizeLimit,
        }
    }
}

/// The result when the EVM terminates due to an exceptional halt.
#[napi(object)]
pub struct HaltResult {
    /// The exceptional halt that occurred
    pub reason: ExceptionalHalt,
    /// Halting will spend all the gas and will thus be equal to the specified
    /// gas limit
    pub gas_used: BigInt,
}

/// The result of executing a transaction.
#[napi(object)]
pub struct ExecutionResult {
    /// The transaction result
    pub result: Either3<SuccessResult, RevertResult, HaltResult>,
    /// Optional contract address if the transaction created a new contract.
    pub contract_address: Option<Buffer>,
}

impl ExecutionResult {
    pub fn new(env: &Env, message: &AfterMessage) -> napi::Result<Self> {
        let AfterMessage {
            execution_result,
            contract_address,
        } = message;

        let result = match execution_result {
            edr_evm::ExecutionResult::Success {
                reason,
                gas_used,
                gas_refunded,
                logs,
                output,
            } => {
                let logs = logs
                    .iter()
                    .map(|log| ExecutionLog::new(env, log))
                    .collect::<napi::Result<_>>()?;

                Either3::A(SuccessResult {
                    reason: SuccessReason::from(*reason),
                    gas_used: BigInt::from(*gas_used),
                    gas_refunded: BigInt::from(*gas_refunded),
                    logs,
                    output: match output {
                        edr_evm::Output::Call(return_value) => {
                            let return_value = env
                                .create_buffer_with_data(return_value.to_vec())
                                .map(JsBufferValue::into_raw)?;

                            Either::A(CallOutput { return_value })
                        }
                        edr_evm::Output::Create(return_value, address) => {
                            let return_value = env
                                .create_buffer_with_data(return_value.to_vec())
                                .map(JsBufferValue::into_raw)?;

                            Either::B(CreateOutput {
                                return_value,
                                address: address.map(|address| Buffer::from(address.as_slice())),
                            })
                        }
                    },
                })
            }
            edr_evm::ExecutionResult::Revert { gas_used, output } => {
                let output = env
                    .create_buffer_with_data(output.to_vec())
                    .map(JsBufferValue::into_raw)?;

                Either3::B(RevertResult {
                    gas_used: BigInt::from(*gas_used),
                    output,
                })
            }
            edr_evm::ExecutionResult::Halt { reason, gas_used } => Either3::C(HaltResult {
                reason: ExceptionalHalt::from(*reason),
                gas_used: BigInt::from(*gas_used),
            }),
        };

        let contract_address = contract_address.map(|address| Buffer::from(address.as_slice()));

        Ok(Self {
            result,
            contract_address,
        })
    }
}
