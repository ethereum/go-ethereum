//! Naive rewrite of `hardhat-network/stack-traces/solidity-stack-traces.ts`
//! from Hardhat.

use edr_eth::U256;
use edr_evm::hex;
use napi::bindgen_prelude::{BigInt, Either24, FromNapiValue, ToNapiValue, Uint8Array, Undefined};
use napi_derive::napi;
use serde::{Serialize, Serializer};

use super::model::ContractFunctionType;
use crate::{cast::TryCast, trace::u256_to_bigint};

#[napi]
#[repr(u8)]
#[allow(non_camel_case_types)] // intentionally mimicks the original case in TS
#[allow(clippy::upper_case_acronyms)]
#[derive(PartialEq, Eq, PartialOrd, Ord, strum::FromRepr, strum::IntoStaticStr, Serialize)]
pub enum StackTraceEntryType {
    CALLSTACK_ENTRY = 0,
    UNRECOGNIZED_CREATE_CALLSTACK_ENTRY,
    UNRECOGNIZED_CONTRACT_CALLSTACK_ENTRY,
    PRECOMPILE_ERROR,
    REVERT_ERROR,
    PANIC_ERROR,
    CUSTOM_ERROR,
    FUNCTION_NOT_PAYABLE_ERROR,
    INVALID_PARAMS_ERROR,
    FALLBACK_NOT_PAYABLE_ERROR,
    FALLBACK_NOT_PAYABLE_AND_NO_RECEIVE_ERROR,
    UNRECOGNIZED_FUNCTION_WITHOUT_FALLBACK_ERROR, /* TODO: Should trying to call a
                                                   * private/internal be a special case of
                                                   * this? */
    MISSING_FALLBACK_OR_RECEIVE_ERROR,
    RETURNDATA_SIZE_ERROR,
    NONCONTRACT_ACCOUNT_CALLED_ERROR,
    CALL_FAILED_ERROR,
    DIRECT_LIBRARY_CALL_ERROR,
    UNRECOGNIZED_CREATE_ERROR,
    UNRECOGNIZED_CONTRACT_ERROR,
    OTHER_EXECUTION_ERROR,
    // This is a special case to handle a regression introduced in solc 0.6.3
    // For more info: https://github.com/ethereum/solidity/issues/9006
    UNMAPPED_SOLC_0_6_3_REVERT_ERROR,
    CONTRACT_TOO_LARGE_ERROR,
    INTERNAL_FUNCTION_CALLSTACK_ENTRY,
    CONTRACT_CALL_RUN_OUT_OF_GAS_ERROR,
}

#[napi]
pub fn stack_trace_entry_type_to_string(val: StackTraceEntryType) -> &'static str {
    val.into()
}

#[napi]
pub const FALLBACK_FUNCTION_NAME: &str = "<fallback>";
#[napi]
pub const RECEIVE_FUNCTION_NAME: &str = "<receive>";
#[napi]
pub const CONSTRUCTOR_FUNCTION_NAME: &str = "constructor";
#[napi]
pub const UNRECOGNIZED_FUNCTION_NAME: &str = "<unrecognized-selector>";
#[napi]
pub const UNKNOWN_FUNCTION_NAME: &str = "<unknown>";
#[napi]
pub const PRECOMPILE_FUNCTION_NAME: &str = "<precompile>";
#[napi]
pub const UNRECOGNIZED_CONTRACT_NAME: &str = "<UnrecognizedContract>";

#[napi(object)]
#[derive(Clone, PartialEq, Serialize)]
pub struct SourceReference {
    pub source_name: String,
    pub source_content: String,
    pub contract: Option<String>,
    pub function: Option<String>,
    pub line: u32,
    // [number, number] tuple
    pub range: Vec<u32>,
}

impl From<edr_solidity::solidity_stack_trace::SourceReference> for SourceReference {
    fn from(value: edr_solidity::solidity_stack_trace::SourceReference) -> Self {
        let (range_start, range_end) = value.range;
        Self {
            source_name: value.source_name,
            source_content: value.source_content,
            contract: value.contract,
            function: value.function,
            line: value.line,
            range: vec![range_start, range_end],
        }
    }
}

/// A [`StackTraceEntryType`] constant that is convertible to/from a
/// `napi_value`.
///
/// Since Rust does not allow constants directly as members, we use this wrapper
/// to allow the `StackTraceEntryType` to be used as a member of an interface
/// when defining the N-API bindings.
// NOTE: It's currently not possible to use an enum as const generic parameter,
// so we use the underlying `u8` repr used by the enum.
#[derive(Clone, Copy)]
pub struct StackTraceEntryTypeConst<const ENTRY_TYPE: u8>;
impl<const ENTRY_TYPE: u8> FromNapiValue for StackTraceEntryTypeConst<ENTRY_TYPE> {
    unsafe fn from_napi_value(
        env: napi::sys::napi_env,
        napi_val: napi::sys::napi_value,
    ) -> napi::Result<Self> {
        let inner: u8 = FromNapiValue::from_napi_value(env, napi_val)?;

        if inner != ENTRY_TYPE {
            return Err(napi::Error::new(
                napi::Status::InvalidArg,
                format!("Expected StackTraceEntryType value: {ENTRY_TYPE}, got: {inner}"),
            ));
        }

        Ok(StackTraceEntryTypeConst)
    }
}
impl<const ENTRY_TYPE: u8> ToNapiValue for StackTraceEntryTypeConst<ENTRY_TYPE> {
    unsafe fn to_napi_value(
        env: napi::sys::napi_env,
        _val: Self,
    ) -> napi::Result<napi::sys::napi_value> {
        u8::to_napi_value(env, ENTRY_TYPE)
    }
}

impl<const ENTRY_TYPE: u8> Serialize for StackTraceEntryTypeConst<ENTRY_TYPE> {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        let inner = StackTraceEntryType::from_repr(ENTRY_TYPE).ok_or_else(|| {
            serde::ser::Error::custom(format!("Invalid StackTraceEntryType value: {ENTRY_TYPE}"))
        })?;

        inner.serialize(serializer)
    }
}

#[napi(object)]
#[derive(Clone, Serialize)]
pub struct CallstackEntryStackTraceEntry {
    #[napi(js_name = "type", ts_type = "StackTraceEntryType.CALLSTACK_ENTRY")]
    pub type_: StackTraceEntryTypeConst<{ StackTraceEntryType::CALLSTACK_ENTRY as u8 }>,
    pub source_reference: SourceReference,
    pub function_type: ContractFunctionType,
}

impl From<CallstackEntryStackTraceEntry> for SolidityStackTraceEntry {
    fn from(val: CallstackEntryStackTraceEntry) -> Self {
        Either24::A(val)
    }
}

#[napi(object)]
#[derive(Clone, Serialize)]
pub struct UnrecognizedCreateCallstackEntryStackTraceEntry {
    #[napi(
        js_name = "type",
        ts_type = "StackTraceEntryType.UNRECOGNIZED_CREATE_CALLSTACK_ENTRY"
    )]
    pub type_: StackTraceEntryTypeConst<
        { StackTraceEntryType::UNRECOGNIZED_CREATE_CALLSTACK_ENTRY as u8 },
    >,
    pub source_reference: Option<Undefined>,
}

impl From<UnrecognizedCreateCallstackEntryStackTraceEntry> for SolidityStackTraceEntry {
    fn from(val: UnrecognizedCreateCallstackEntryStackTraceEntry) -> Self {
        Either24::B(val)
    }
}

#[napi(object)]
#[derive(Clone, Serialize)]
pub struct UnrecognizedContractCallstackEntryStackTraceEntry {
    #[napi(
        js_name = "type",
        ts_type = "StackTraceEntryType.UNRECOGNIZED_CONTRACT_CALLSTACK_ENTRY"
    )]
    pub type_: StackTraceEntryTypeConst<
        { StackTraceEntryType::UNRECOGNIZED_CONTRACT_CALLSTACK_ENTRY as u8 },
    >,
    #[serde(serialize_with = "serialize_uint8array_to_hex")]
    pub address: Uint8Array,
    pub source_reference: Option<Undefined>,
}

impl From<UnrecognizedContractCallstackEntryStackTraceEntry> for SolidityStackTraceEntry {
    fn from(val: UnrecognizedContractCallstackEntryStackTraceEntry) -> Self {
        Either24::C(val)
    }
}

#[napi(object)]
#[derive(Clone, Serialize)]
pub struct PrecompileErrorStackTraceEntry {
    #[napi(js_name = "type", ts_type = "StackTraceEntryType.PRECOMPILE_ERROR")]
    pub type_: StackTraceEntryTypeConst<{ StackTraceEntryType::PRECOMPILE_ERROR as u8 }>,
    pub precompile: u32,
    pub source_reference: Option<Undefined>,
}

impl From<PrecompileErrorStackTraceEntry> for SolidityStackTraceEntry {
    fn from(val: PrecompileErrorStackTraceEntry) -> Self {
        Either24::D(val)
    }
}

#[napi(object)]
#[derive(Clone, Serialize)]
pub struct RevertErrorStackTraceEntry {
    #[napi(js_name = "type", ts_type = "StackTraceEntryType.REVERT_ERROR")]
    pub type_: StackTraceEntryTypeConst<{ StackTraceEntryType::REVERT_ERROR as u8 }>,
    #[serde(serialize_with = "serialize_uint8array_to_hex")]
    pub return_data: Uint8Array,
    pub source_reference: SourceReference,
    pub is_invalid_opcode_error: bool,
}

impl From<RevertErrorStackTraceEntry> for SolidityStackTraceEntry {
    fn from(val: RevertErrorStackTraceEntry) -> Self {
        Either24::E(val)
    }
}

#[napi(object)]
#[derive(Clone, Serialize)]
pub struct PanicErrorStackTraceEntry {
    #[napi(js_name = "type", ts_type = "StackTraceEntryType.PANIC_ERROR")]
    pub type_: StackTraceEntryTypeConst<{ StackTraceEntryType::PANIC_ERROR as u8 }>,
    #[serde(serialize_with = "serialize_evm_value_bigint_using_u256")]
    pub error_code: BigInt,
    pub source_reference: Option<SourceReference>,
}

impl From<PanicErrorStackTraceEntry> for SolidityStackTraceEntry {
    fn from(val: PanicErrorStackTraceEntry) -> Self {
        Either24::F(val)
    }
}

#[napi(object)]
#[derive(Clone, Serialize)]
pub struct CustomErrorStackTraceEntry {
    #[napi(js_name = "type", ts_type = "StackTraceEntryType.CUSTOM_ERROR")]
    pub type_: StackTraceEntryTypeConst<{ StackTraceEntryType::CUSTOM_ERROR as u8 }>,
    // unlike RevertErrorStackTraceEntry, this includes the message already parsed
    pub message: String,
    pub source_reference: SourceReference,
}

impl From<CustomErrorStackTraceEntry> for SolidityStackTraceEntry {
    fn from(val: CustomErrorStackTraceEntry) -> Self {
        Either24::G(val)
    }
}

#[napi(object)]
#[derive(Clone, Serialize)]
pub struct FunctionNotPayableErrorStackTraceEntry {
    #[napi(
        js_name = "type",
        ts_type = "StackTraceEntryType.FUNCTION_NOT_PAYABLE_ERROR"
    )]
    pub type_: StackTraceEntryTypeConst<{ StackTraceEntryType::FUNCTION_NOT_PAYABLE_ERROR as u8 }>,
    #[serde(serialize_with = "serialize_evm_value_bigint_using_u256")]
    pub value: BigInt,
    pub source_reference: SourceReference,
}

impl From<FunctionNotPayableErrorStackTraceEntry> for SolidityStackTraceEntry {
    fn from(val: FunctionNotPayableErrorStackTraceEntry) -> Self {
        Either24::H(val)
    }
}

#[napi(object)]
#[derive(Clone, Serialize)]
pub struct InvalidParamsErrorStackTraceEntry {
    #[napi(js_name = "type", ts_type = "StackTraceEntryType.INVALID_PARAMS_ERROR")]
    pub type_: StackTraceEntryTypeConst<{ StackTraceEntryType::INVALID_PARAMS_ERROR as u8 }>,
    pub source_reference: SourceReference,
}

impl From<InvalidParamsErrorStackTraceEntry> for SolidityStackTraceEntry {
    fn from(val: InvalidParamsErrorStackTraceEntry) -> Self {
        Either24::I(val)
    }
}

#[napi(object)]
#[derive(Clone, Serialize)]
pub struct FallbackNotPayableErrorStackTraceEntry {
    #[napi(
        js_name = "type",
        ts_type = "StackTraceEntryType.FALLBACK_NOT_PAYABLE_ERROR"
    )]
    pub type_: StackTraceEntryTypeConst<{ StackTraceEntryType::FALLBACK_NOT_PAYABLE_ERROR as u8 }>,
    #[serde(serialize_with = "serialize_evm_value_bigint_using_u256")]
    pub value: BigInt,
    pub source_reference: SourceReference,
}

impl From<FallbackNotPayableErrorStackTraceEntry> for SolidityStackTraceEntry {
    fn from(val: FallbackNotPayableErrorStackTraceEntry) -> Self {
        Either24::J(val)
    }
}

#[napi(object)]
#[derive(Clone, Serialize)]
pub struct FallbackNotPayableAndNoReceiveErrorStackTraceEntry {
    #[napi(
        js_name = "type",
        ts_type = "StackTraceEntryType.FALLBACK_NOT_PAYABLE_AND_NO_RECEIVE_ERROR"
    )]
    pub type_: StackTraceEntryTypeConst<
        { StackTraceEntryType::FALLBACK_NOT_PAYABLE_AND_NO_RECEIVE_ERROR as u8 },
    >,
    #[serde(serialize_with = "serialize_evm_value_bigint_using_u256")]
    pub value: BigInt,
    pub source_reference: SourceReference,
}

impl From<FallbackNotPayableAndNoReceiveErrorStackTraceEntry> for SolidityStackTraceEntry {
    fn from(val: FallbackNotPayableAndNoReceiveErrorStackTraceEntry) -> Self {
        Either24::K(val)
    }
}

#[napi(object)]
#[derive(Clone, Serialize)]
pub struct UnrecognizedFunctionWithoutFallbackErrorStackTraceEntry {
    #[napi(
        js_name = "type",
        ts_type = "StackTraceEntryType.UNRECOGNIZED_FUNCTION_WITHOUT_FALLBACK_ERROR"
    )]
    pub type_: StackTraceEntryTypeConst<
        { StackTraceEntryType::UNRECOGNIZED_FUNCTION_WITHOUT_FALLBACK_ERROR as u8 },
    >,
    pub source_reference: SourceReference,
}

impl From<UnrecognizedFunctionWithoutFallbackErrorStackTraceEntry> for SolidityStackTraceEntry {
    fn from(val: UnrecognizedFunctionWithoutFallbackErrorStackTraceEntry) -> Self {
        Either24::L(val)
    }
}

#[napi(object)]
#[derive(Clone, Serialize)]
pub struct MissingFallbackOrReceiveErrorStackTraceEntry {
    #[napi(
        js_name = "type",
        ts_type = "StackTraceEntryType.MISSING_FALLBACK_OR_RECEIVE_ERROR"
    )]
    pub type_:
        StackTraceEntryTypeConst<{ StackTraceEntryType::MISSING_FALLBACK_OR_RECEIVE_ERROR as u8 }>,
    pub source_reference: SourceReference,
}

impl From<MissingFallbackOrReceiveErrorStackTraceEntry> for SolidityStackTraceEntry {
    fn from(val: MissingFallbackOrReceiveErrorStackTraceEntry) -> Self {
        Either24::M(val)
    }
}

#[napi(object)]
#[derive(Clone, Serialize)]
pub struct ReturndataSizeErrorStackTraceEntry {
    #[napi(
        js_name = "type",
        ts_type = "StackTraceEntryType.RETURNDATA_SIZE_ERROR"
    )]
    pub type_: StackTraceEntryTypeConst<{ StackTraceEntryType::RETURNDATA_SIZE_ERROR as u8 }>,
    pub source_reference: SourceReference,
}

impl From<ReturndataSizeErrorStackTraceEntry> for SolidityStackTraceEntry {
    fn from(val: ReturndataSizeErrorStackTraceEntry) -> Self {
        Either24::N(val)
    }
}

#[napi(object)]
#[derive(Clone, Serialize)]
pub struct NonContractAccountCalledErrorStackTraceEntry {
    #[napi(
        js_name = "type",
        ts_type = "StackTraceEntryType.NONCONTRACT_ACCOUNT_CALLED_ERROR"
    )]
    pub type_:
        StackTraceEntryTypeConst<{ StackTraceEntryType::NONCONTRACT_ACCOUNT_CALLED_ERROR as u8 }>,
    pub source_reference: SourceReference,
}

impl From<NonContractAccountCalledErrorStackTraceEntry> for SolidityStackTraceEntry {
    fn from(val: NonContractAccountCalledErrorStackTraceEntry) -> Self {
        Either24::O(val)
    }
}

#[napi(object)]
#[derive(Clone, Serialize)]
pub struct CallFailedErrorStackTraceEntry {
    #[napi(js_name = "type", ts_type = "StackTraceEntryType.CALL_FAILED_ERROR")]
    pub type_: StackTraceEntryTypeConst<{ StackTraceEntryType::CALL_FAILED_ERROR as u8 }>,
    pub source_reference: SourceReference,
}

impl From<CallFailedErrorStackTraceEntry> for SolidityStackTraceEntry {
    fn from(val: CallFailedErrorStackTraceEntry) -> Self {
        Either24::P(val)
    }
}

#[napi(object)]
#[derive(Clone, Serialize)]
pub struct DirectLibraryCallErrorStackTraceEntry {
    #[napi(
        js_name = "type",
        ts_type = "StackTraceEntryType.DIRECT_LIBRARY_CALL_ERROR"
    )]
    pub type_: StackTraceEntryTypeConst<{ StackTraceEntryType::DIRECT_LIBRARY_CALL_ERROR as u8 }>,
    pub source_reference: SourceReference,
}

impl From<DirectLibraryCallErrorStackTraceEntry> for SolidityStackTraceEntry {
    fn from(val: DirectLibraryCallErrorStackTraceEntry) -> Self {
        Either24::Q(val)
    }
}

#[napi(object)]
#[derive(Clone, Serialize)]
pub struct UnrecognizedCreateErrorStackTraceEntry {
    #[napi(
        js_name = "type",
        ts_type = "StackTraceEntryType.UNRECOGNIZED_CREATE_ERROR"
    )]
    pub type_: StackTraceEntryTypeConst<{ StackTraceEntryType::UNRECOGNIZED_CREATE_ERROR as u8 }>,
    #[serde(serialize_with = "serialize_uint8array_to_hex")]
    pub return_data: Uint8Array,
    pub source_reference: Option<Undefined>,
    pub is_invalid_opcode_error: bool,
}

impl From<UnrecognizedCreateErrorStackTraceEntry> for SolidityStackTraceEntry {
    fn from(val: UnrecognizedCreateErrorStackTraceEntry) -> Self {
        Either24::R(val)
    }
}

#[napi(object)]
#[derive(Clone, Serialize)]
pub struct UnrecognizedContractErrorStackTraceEntry {
    #[napi(
        js_name = "type",
        ts_type = "StackTraceEntryType.UNRECOGNIZED_CONTRACT_ERROR"
    )]
    pub type_: StackTraceEntryTypeConst<{ StackTraceEntryType::UNRECOGNIZED_CONTRACT_ERROR as u8 }>,
    #[serde(serialize_with = "serialize_uint8array_to_hex")]
    pub address: Uint8Array,
    #[serde(serialize_with = "serialize_uint8array_to_hex")]
    pub return_data: Uint8Array,
    pub source_reference: Option<Undefined>,
    pub is_invalid_opcode_error: bool,
}

impl From<UnrecognizedContractErrorStackTraceEntry> for SolidityStackTraceEntry {
    fn from(val: UnrecognizedContractErrorStackTraceEntry) -> Self {
        Either24::S(val)
    }
}

#[napi(object)]
#[derive(Clone, Serialize)]
pub struct OtherExecutionErrorStackTraceEntry {
    #[napi(
        js_name = "type",
        ts_type = "StackTraceEntryType.OTHER_EXECUTION_ERROR"
    )]
    pub type_: StackTraceEntryTypeConst<{ StackTraceEntryType::OTHER_EXECUTION_ERROR as u8 }>,
    pub source_reference: Option<SourceReference>,
}

impl From<OtherExecutionErrorStackTraceEntry> for SolidityStackTraceEntry {
    fn from(val: OtherExecutionErrorStackTraceEntry) -> Self {
        Either24::T(val)
    }
}

#[napi(object)]
#[derive(Clone, Serialize)]
pub struct UnmappedSolc063RevertErrorStackTraceEntry {
    #[napi(
        js_name = "type",
        ts_type = "StackTraceEntryType.UNMAPPED_SOLC_0_6_3_REVERT_ERROR"
    )]
    pub type_:
        StackTraceEntryTypeConst<{ StackTraceEntryType::UNMAPPED_SOLC_0_6_3_REVERT_ERROR as u8 }>,
    pub source_reference: Option<SourceReference>,
}

impl From<UnmappedSolc063RevertErrorStackTraceEntry> for SolidityStackTraceEntry {
    fn from(val: UnmappedSolc063RevertErrorStackTraceEntry) -> Self {
        Either24::U(val)
    }
}

#[napi(object)]
#[derive(Clone, Serialize)]
pub struct ContractTooLargeErrorStackTraceEntry {
    #[napi(
        js_name = "type",
        ts_type = "StackTraceEntryType.CONTRACT_TOO_LARGE_ERROR"
    )]
    pub type_: StackTraceEntryTypeConst<{ StackTraceEntryType::CONTRACT_TOO_LARGE_ERROR as u8 }>,
    pub source_reference: Option<SourceReference>,
}

impl From<ContractTooLargeErrorStackTraceEntry> for SolidityStackTraceEntry {
    fn from(val: ContractTooLargeErrorStackTraceEntry) -> Self {
        Either24::V(val)
    }
}

#[napi(object)]
#[derive(Clone, Serialize)]
pub struct InternalFunctionCallStackEntry {
    #[napi(
        js_name = "type",
        ts_type = "StackTraceEntryType.INTERNAL_FUNCTION_CALLSTACK_ENTRY"
    )]
    pub type_:
        StackTraceEntryTypeConst<{ StackTraceEntryType::INTERNAL_FUNCTION_CALLSTACK_ENTRY as u8 }>,
    pub pc: u32,
    pub source_reference: SourceReference,
}

impl From<InternalFunctionCallStackEntry> for SolidityStackTraceEntry {
    fn from(val: InternalFunctionCallStackEntry) -> Self {
        Either24::W(val)
    }
}

#[napi(object)]
#[derive(Clone, Serialize)]
pub struct ContractCallRunOutOfGasError {
    #[napi(
        js_name = "type",
        ts_type = "StackTraceEntryType.CONTRACT_CALL_RUN_OUT_OF_GAS_ERROR"
    )]
    pub type_:
        StackTraceEntryTypeConst<{ StackTraceEntryType::CONTRACT_CALL_RUN_OUT_OF_GAS_ERROR as u8 }>,
    pub source_reference: Option<SourceReference>,
}

impl From<ContractCallRunOutOfGasError> for SolidityStackTraceEntry {
    fn from(val: ContractCallRunOutOfGasError) -> Self {
        Either24::X(val)
    }
}

#[allow(dead_code)]
// NOTE: This ported directly from JS for completeness and is used in the Rust
// side of the bindings. However, napi-rs does not support exporting Rust type
// aliases to the index.d.ts file, and it does not store the type definitions
// when expanding the macros, so to use it we would have to specify this type
// literally (all 26 lines of it) at every #[napi]-exported function, which is
// not ideal.
// Rather, we just bite the bullet for now and use the type alias directly
// (which falls back to `any` as it's not recognized in the context of the
// index.d.ts file) until we finish the porting work.
pub type SolidityStackTraceEntry = Either24<
    CallstackEntryStackTraceEntry,
    UnrecognizedCreateCallstackEntryStackTraceEntry,
    UnrecognizedContractCallstackEntryStackTraceEntry,
    PrecompileErrorStackTraceEntry,
    RevertErrorStackTraceEntry,
    PanicErrorStackTraceEntry,
    CustomErrorStackTraceEntry,
    FunctionNotPayableErrorStackTraceEntry,
    InvalidParamsErrorStackTraceEntry,
    FallbackNotPayableErrorStackTraceEntry,
    FallbackNotPayableAndNoReceiveErrorStackTraceEntry,
    UnrecognizedFunctionWithoutFallbackErrorStackTraceEntry,
    MissingFallbackOrReceiveErrorStackTraceEntry,
    ReturndataSizeErrorStackTraceEntry,
    NonContractAccountCalledErrorStackTraceEntry,
    CallFailedErrorStackTraceEntry,
    DirectLibraryCallErrorStackTraceEntry,
    UnrecognizedCreateErrorStackTraceEntry,
    UnrecognizedContractErrorStackTraceEntry,
    OtherExecutionErrorStackTraceEntry,
    UnmappedSolc063RevertErrorStackTraceEntry,
    ContractTooLargeErrorStackTraceEntry,
    InternalFunctionCallStackEntry,
    ContractCallRunOutOfGasError,
>;

impl TryCast<SolidityStackTraceEntry> for edr_solidity::solidity_stack_trace::StackTraceEntry {
    type Error = napi::Error;

    fn try_cast(self) -> Result<SolidityStackTraceEntry, Self::Error> {
        use edr_solidity::solidity_stack_trace::StackTraceEntry;
        let result = match self {
            StackTraceEntry::CallstackEntry {
                source_reference,
                function_type,
            } => CallstackEntryStackTraceEntry {
                type_: StackTraceEntryTypeConst,
                source_reference: source_reference.into(),
                function_type: function_type.into(),
            }
            .into(),
            StackTraceEntry::UnrecognizedCreateCallstackEntry => {
                UnrecognizedCreateCallstackEntryStackTraceEntry {
                    type_: StackTraceEntryTypeConst,
                    source_reference: None,
                }
                .into()
            }
            StackTraceEntry::UnrecognizedContractCallstackEntry { address } => {
                UnrecognizedContractCallstackEntryStackTraceEntry {
                    type_: StackTraceEntryTypeConst,
                    address: Uint8Array::from(address.as_slice()),
                    source_reference: None,
                }
                .into()
            }
            StackTraceEntry::PrecompileError { precompile } => PrecompileErrorStackTraceEntry {
                type_: StackTraceEntryTypeConst,
                precompile,
                source_reference: None,
            }
            .into(),
            StackTraceEntry::RevertError {
                return_data,
                source_reference,
                is_invalid_opcode_error,
            } => RevertErrorStackTraceEntry {
                type_: StackTraceEntryTypeConst,
                return_data: return_data.into(),
                source_reference: source_reference.into(),
                is_invalid_opcode_error,
            }
            .into(),
            StackTraceEntry::PanicError {
                error_code,
                source_reference,
            } => PanicErrorStackTraceEntry {
                type_: StackTraceEntryTypeConst,
                error_code: u256_to_bigint(&error_code),
                source_reference: source_reference.map(std::convert::Into::into),
            }
            .into(),
            StackTraceEntry::CustomError {
                message,
                source_reference,
            } => CustomErrorStackTraceEntry {
                type_: StackTraceEntryTypeConst,
                message,
                source_reference: source_reference.into(),
            }
            .into(),
            StackTraceEntry::FunctionNotPayableError {
                value,
                source_reference,
            } => FunctionNotPayableErrorStackTraceEntry {
                type_: StackTraceEntryTypeConst,
                value: u256_to_bigint(&value),
                source_reference: source_reference.into(),
            }
            .into(),
            StackTraceEntry::InvalidParamsError { source_reference } => {
                InvalidParamsErrorStackTraceEntry {
                    type_: StackTraceEntryTypeConst,
                    source_reference: source_reference.into(),
                }
                .into()
            }
            StackTraceEntry::FallbackNotPayableError {
                value,
                source_reference,
            } => FallbackNotPayableErrorStackTraceEntry {
                type_: StackTraceEntryTypeConst,
                value: u256_to_bigint(&value),
                source_reference: source_reference.into(),
            }
            .into(),
            StackTraceEntry::FallbackNotPayableAndNoReceiveError {
                value,
                source_reference,
            } => FallbackNotPayableAndNoReceiveErrorStackTraceEntry {
                type_: StackTraceEntryTypeConst,
                value: u256_to_bigint(&value),
                source_reference: source_reference.into(),
            }
            .into(),
            StackTraceEntry::UnrecognizedFunctionWithoutFallbackError { source_reference } => {
                UnrecognizedFunctionWithoutFallbackErrorStackTraceEntry {
                    type_: StackTraceEntryTypeConst,
                    source_reference: source_reference.into(),
                }
                .into()
            }
            StackTraceEntry::MissingFallbackOrReceiveError { source_reference } => {
                MissingFallbackOrReceiveErrorStackTraceEntry {
                    type_: StackTraceEntryTypeConst,
                    source_reference: source_reference.into(),
                }
                .into()
            }
            StackTraceEntry::ReturndataSizeError { source_reference } => {
                ReturndataSizeErrorStackTraceEntry {
                    type_: StackTraceEntryTypeConst,
                    source_reference: source_reference.into(),
                }
                .into()
            }
            StackTraceEntry::NoncontractAccountCalledError { source_reference } => {
                NonContractAccountCalledErrorStackTraceEntry {
                    type_: StackTraceEntryTypeConst,
                    source_reference: source_reference.into(),
                }
                .into()
            }
            StackTraceEntry::CallFailedError { source_reference } => {
                CallFailedErrorStackTraceEntry {
                    type_: StackTraceEntryTypeConst,
                    source_reference: source_reference.into(),
                }
                .into()
            }
            StackTraceEntry::DirectLibraryCallError { source_reference } => {
                DirectLibraryCallErrorStackTraceEntry {
                    type_: StackTraceEntryTypeConst,
                    source_reference: source_reference.into(),
                }
                .into()
            }
            StackTraceEntry::UnrecognizedCreateError {
                return_data,
                is_invalid_opcode_error,
            } => UnrecognizedCreateErrorStackTraceEntry {
                type_: StackTraceEntryTypeConst,
                return_data: return_data.into(),
                is_invalid_opcode_error,
                source_reference: None,
            }
            .into(),
            StackTraceEntry::UnrecognizedContractError {
                address,
                return_data,
                is_invalid_opcode_error,
            } => UnrecognizedContractErrorStackTraceEntry {
                type_: StackTraceEntryTypeConst,
                address: Uint8Array::from(address.as_slice()),
                return_data: return_data.into(),
                is_invalid_opcode_error,
                source_reference: None,
            }
            .into(),
            StackTraceEntry::OtherExecutionError { source_reference } => {
                OtherExecutionErrorStackTraceEntry {
                    type_: StackTraceEntryTypeConst,
                    source_reference: source_reference.map(std::convert::Into::into),
                }
                .into()
            }
            StackTraceEntry::UnmappedSolc0_6_3RevertError { source_reference } => {
                UnmappedSolc063RevertErrorStackTraceEntry {
                    type_: StackTraceEntryTypeConst,
                    source_reference: source_reference.map(std::convert::Into::into),
                }
                .into()
            }
            StackTraceEntry::ContractTooLargeError { source_reference } => {
                ContractTooLargeErrorStackTraceEntry {
                    type_: StackTraceEntryTypeConst,
                    source_reference: source_reference.map(std::convert::Into::into),
                }
                .into()
            }
            StackTraceEntry::InternalFunctionCallstackEntry {
                pc,
                source_reference,
            } => InternalFunctionCallStackEntry {
                type_: StackTraceEntryTypeConst,
                pc,
                source_reference: source_reference.into(),
            }
            .into(),
            StackTraceEntry::ContractCallRunOutOfGasError { source_reference } => {
                ContractCallRunOutOfGasError {
                    type_: StackTraceEntryTypeConst,
                    source_reference: source_reference.map(std::convert::Into::into),
                }
                .into()
            }
        };
        Ok(result)
    }
}

#[allow(dead_code)]
// Same as above, but for the `SolidityStackTrace` type.
pub type SolidityStackTrace = Vec<SolidityStackTraceEntry>;

const _: () = {
    const fn assert_to_from_napi_value<T: FromNapiValue + ToNapiValue>() {}
    assert_to_from_napi_value::<SolidityStackTraceEntry>();
};

/// Serializes a [`BigInt`] that represents an EVM value as a [`edr_eth::U256`].
fn serialize_evm_value_bigint_using_u256<S>(bigint: &BigInt, s: S) -> Result<S::Ok, S::Error>
where
    S: Serializer,
{
    let val = U256::from_limbs_slice(&bigint.words);

    val.serialize(s)
}

fn serialize_uint8array_to_hex<S>(uint8array: &Uint8Array, s: S) -> Result<S::Ok, S::Error>
where
    S: Serializer,
{
    let hex = hex::encode(uint8array.as_ref());

    hex.serialize(s)
}
