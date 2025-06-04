use std::rc::Rc;

use edr_solidity::build_model::ContractMetadata;
use napi_derive::napi;
use serde::Serialize;

/// Opaque handle to the `Bytecode` struct.
/// Only used on the JS side by the `VmTraceDecoder` class.
// NOTE: Needed, because we store the resolved `Bytecode` in the MessageTrace
// JS plain objects and those need a dedicated (class) type.
#[napi]
pub struct BytecodeWrapper(pub(crate) Rc<ContractMetadata>);

impl BytecodeWrapper {
    pub fn new(bytecode: Rc<ContractMetadata>) -> Self {
        Self(bytecode)
    }

    pub fn inner(&self) -> &Rc<ContractMetadata> {
        &self.0
    }
}

impl std::ops::Deref for BytecodeWrapper {
    type Target = ContractMetadata;

    fn deref(&self) -> &Self::Target {
        &self.0
    }
}

#[derive(Debug, PartialEq, Eq, Serialize)]
#[allow(non_camel_case_types)] // intentionally mimicks the original case in TS
#[allow(clippy::upper_case_acronyms)]
#[napi]
// Mimicks [`edr_solidity::build_model::ContractFunctionType`].
pub enum ContractFunctionType {
    CONSTRUCTOR,
    FUNCTION,
    FALLBACK,
    RECEIVE,
    GETTER,
    MODIFIER,
    FREE_FUNCTION,
}

impl From<edr_solidity::build_model::ContractFunctionType> for ContractFunctionType {
    fn from(value: edr_solidity::build_model::ContractFunctionType) -> Self {
        match value {
            edr_solidity::build_model::ContractFunctionType::Constructor => Self::CONSTRUCTOR,
            edr_solidity::build_model::ContractFunctionType::Function => Self::FUNCTION,
            edr_solidity::build_model::ContractFunctionType::Fallback => Self::FALLBACK,
            edr_solidity::build_model::ContractFunctionType::Receive => Self::RECEIVE,
            edr_solidity::build_model::ContractFunctionType::Getter => Self::GETTER,
            edr_solidity::build_model::ContractFunctionType::Modifier => Self::MODIFIER,
            edr_solidity::build_model::ContractFunctionType::FreeFunction => Self::FREE_FUNCTION,
        }
    }
}
