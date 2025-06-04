//! Naive rewrite of `hardhat-network/provider/vm/exit.ts` from Hardhat.
//! Used together with `VmTracer`.

use std::fmt;

use edr_evm::HaltReason;
use napi_derive::napi;

#[napi]
pub struct Exit(pub(crate) ExitCode);

#[napi]
/// Represents the exit code of the EVM.
#[derive(Debug, PartialEq, Eq)]
#[allow(clippy::upper_case_acronyms, non_camel_case_types)] // These are exported and mapped 1:1 to existing JS enum
pub enum ExitCode {
    /// Execution was successful.
    SUCCESS = 0,
    /// Execution was reverted.
    REVERT,
    /// Execution ran out of gas.
    OUT_OF_GAS,
    /// Execution encountered an internal error.
    INTERNAL_ERROR,
    /// Execution encountered an invalid opcode.
    INVALID_OPCODE,
    /// Execution encountered a stack underflow.
    STACK_UNDERFLOW,
    /// Create init code size exceeds limit (runtime).
    CODESIZE_EXCEEDS_MAXIMUM,
    /// Create collision.
    CREATE_COLLISION,
    /// Unknown halt reason.
    UNKNOWN_HALT_REASON,
}

impl fmt::Display for ExitCode {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            ExitCode::SUCCESS => write!(f, "Success"),
            ExitCode::REVERT => write!(f, "Reverted"),
            ExitCode::OUT_OF_GAS => write!(f, "Out of gas"),
            ExitCode::INTERNAL_ERROR => write!(f, "Internal error"),
            ExitCode::INVALID_OPCODE => write!(f, "Invalid opcode"),
            ExitCode::STACK_UNDERFLOW => write!(f, "Stack underflow"),
            ExitCode::CODESIZE_EXCEEDS_MAXIMUM => write!(f, "Codesize exceeds maximum"),
            ExitCode::CREATE_COLLISION => write!(f, "Create collision"),
            ExitCode::UNKNOWN_HALT_REASON => write!(f, "Unknown halt reason"),
        }
    }
}

#[allow(clippy::fallible_impl_from)] // naively ported for now
impl From<edr_solidity::exit_code::ExitCode> for ExitCode {
    fn from(code: edr_solidity::exit_code::ExitCode) -> Self {
        use edr_solidity::exit_code::ExitCode;

        match code {
            ExitCode::Success => Self::SUCCESS,
            ExitCode::Revert => Self::REVERT,
            ExitCode::Halt(HaltReason::OutOfGas(_)) => Self::OUT_OF_GAS,
            ExitCode::Halt(HaltReason::OpcodeNotFound | HaltReason::InvalidFEOpcode
              // Returned when an opcode is not implemented for the hardfork
              | HaltReason::NotActivated) => Self::INVALID_OPCODE,
            ExitCode::Halt(HaltReason::StackUnderflow) => Self::STACK_UNDERFLOW,
            ExitCode::Halt(HaltReason::CreateContractSizeLimit) => Self::CODESIZE_EXCEEDS_MAXIMUM,
            ExitCode::Halt(HaltReason::CreateCollision) => Self::CREATE_COLLISION,
            ExitCode::Halt(_) => Self::UNKNOWN_HALT_REASON,
        }
    }
}

#[napi]
impl Exit {
    #[napi(getter)]
    pub fn kind(&self) -> ExitCode {
        self.0
    }

    #[napi]
    pub fn is_error(&self) -> bool {
        !matches!(self.0, ExitCode::SUCCESS)
    }

    #[napi]
    pub fn get_reason(&self) -> String {
        self.0.to_string()
    }
}
