mod config;

use std::sync::Arc;

use edr_provider::{time::CurrentTime, InvalidRequestReason};
use edr_rpc_eth::jsonrpc;
use edr_solidity::contract_decoder::ContractDecoder;
use napi::{
    bindgen_prelude::Uint8Array, tokio::runtime, Either, Env, JsFunction, JsObject, Status,
};
use napi_derive::napi;

use self::config::ProviderConfig;
use crate::{
    call_override::CallOverrideCallback,
    context::EdrContext,
    logger::{Logger, LoggerConfig, LoggerError},
    subscribe::SubscriberCallback,
    trace::{solidity_stack_trace::SolidityStackTrace, RawTrace},
};

/// A JSON-RPC provider for Ethereum.
#[napi]
pub struct Provider {
    provider: Arc<edr_provider::Provider<LoggerError>>,
    runtime: runtime::Handle,
    contract_decoder: Arc<ContractDecoder>,
    #[cfg(feature = "scenarios")]
    scenario_file: Option<napi::tokio::sync::Mutex<napi::tokio::fs::File>>,
}

#[napi]
impl Provider {
    #[doc = "Constructs a new provider with the provided configuration."]
    #[napi(ts_return_type = "Promise<Provider>")]
    pub fn with_config(
        env: Env,
        // We take the context as argument to ensure that tracing is initialized properly.
        _context: &EdrContext,
        config: ProviderConfig,
        logger_config: LoggerConfig,
        tracing_config: TracingConfigWithBuffers,
        #[napi(ts_arg_type = "(event: SubscriptionEvent) => void")] subscriber_callback: JsFunction,
    ) -> napi::Result<JsObject> {
        let runtime = runtime::Handle::current();

        let config = edr_provider::ProviderConfig::try_from(config)?;

        // TODO https://github.com/NomicFoundation/edr/issues/760
        let build_info_config =
            edr_solidity::artifacts::BuildInfoConfig::parse_from_buffers((&tracing_config).into())
                .map_err(|err| napi::Error::from_reason(err.to_string()))?;
        let contract_decoder = ContractDecoder::new(&build_info_config)
            .map_err(|error| napi::Error::from_reason(error.to_string()))?;
        let contract_decoder = Arc::new(contract_decoder);

        let logger = Box::new(Logger::new(
            &env,
            logger_config,
            Arc::clone(&contract_decoder),
        )?);
        let subscriber_callback = SubscriberCallback::new(&env, subscriber_callback)?;
        let subscriber_callback = Box::new(move |event| subscriber_callback.call(event));

        let (deferred, promise) = env.create_deferred()?;
        runtime.clone().spawn_blocking(move || {
            #[cfg(feature = "scenarios")]
            let scenario_file =
                runtime::Handle::current().block_on(crate::scenarios::scenario_file(
                    &config,
                    edr_provider::Logger::is_enabled(&*logger),
                ))?;

            let result = edr_provider::Provider::new(
                runtime.clone(),
                logger,
                subscriber_callback,
                config,
                Arc::clone(&contract_decoder),
                CurrentTime,
            )
            .map_or_else(
                |error| Err(napi::Error::new(Status::GenericFailure, error.to_string())),
                |provider| {
                    Ok(Provider {
                        provider: Arc::new(provider),
                        runtime,
                        contract_decoder,
                        #[cfg(feature = "scenarios")]
                        scenario_file,
                    })
                },
            );

            deferred.resolve(|_env| result);
            Ok::<_, napi::Error>(())
        });

        Ok(promise)
    }

    #[doc = "Handles a JSON-RPC request and returns a JSON-RPC response."]
    #[napi]
    pub async fn handle_request(&self, json_request: String) -> napi::Result<Response> {
        let provider = self.provider.clone();
        let request = match serde_json::from_str(&json_request) {
            Ok(request) => request,
            Err(error) => {
                let message = error.to_string();
                let reason = InvalidRequestReason::new(&json_request, &message);

                // HACK: We need to log failed deserialization attempts when they concern input
                // validation.
                if let Some((method_name, provider_error)) = reason.provider_error() {
                    // Ignore potential failure of logging, as returning the original error is more
                    // important
                    let _result = runtime::Handle::current()
                        .spawn_blocking(move || {
                            provider.log_failed_deserialization(&method_name, &provider_error)
                        })
                        .await
                        .map_err(|error| {
                            napi::Error::new(Status::GenericFailure, error.to_string())
                        })?;
                }

                let data = serde_json::from_str(&json_request).ok();
                let response = jsonrpc::ResponseData::<()>::Error {
                    error: jsonrpc::Error {
                        code: reason.error_code(),
                        message: reason.error_message(),
                        data,
                    },
                };

                return serde_json::to_string(&response)
                    .map_err(|error| {
                        napi::Error::new(
                            Status::InvalidArg,
                            format!("Invalid JSON `{json_request}` due to: {error}"),
                        )
                    })
                    .map(|json| Response {
                        solidity_trace: None,
                        data: Either::A(json),
                        traces: Vec::new(),
                    });
            }
        };

        #[cfg(feature = "scenarios")]
        if let Some(scenario_file) = &self.scenario_file {
            crate::scenarios::write_request(scenario_file, &request).await?;
        }

        let mut response = runtime::Handle::current()
            .spawn_blocking(move || provider.handle_request(request))
            .await
            .map_err(|e| napi::Error::new(Status::GenericFailure, e.to_string()))?;

        // We can take the solidity trace as it won't be used for anything else
        let solidity_trace = response.as_mut().err().and_then(|error| {
            if let edr_provider::ProviderError::TransactionFailed(failure) = error {
                if matches!(
                    failure.failure.reason,
                    edr_provider::TransactionFailureReason::OutOfGas(_)
                ) {
                    None
                } else {
                    Some(Arc::new(std::mem::take(
                        &mut failure.failure.solidity_trace,
                    )))
                }
            } else {
                None
            }
        });

        // We can take the traces as they won't be used for anything else
        let traces = match &mut response {
            Ok(response) => std::mem::take(&mut response.traces),
            Err(edr_provider::ProviderError::TransactionFailed(failure)) => {
                std::mem::take(&mut failure.traces)
            }
            Err(_) => Vec::new(),
        };

        let response = jsonrpc::ResponseData::from(response.map(|response| response.result));

        serde_json::to_string(&response)
            .and_then(|json| {
                // We experimentally determined that 500_000_000 was the maximum string length
                // that can be returned without causing the error:
                //
                // > Failed to convert rust `String` into napi `string`
                //
                // To be safe, we're limiting string lengths to half of that.
                const MAX_STRING_LENGTH: usize = 250_000_000;

                if json.len() <= MAX_STRING_LENGTH {
                    Ok(Either::A(json))
                } else {
                    serde_json::to_value(response).map(Either::B)
                }
            })
            .map_err(|error| napi::Error::new(Status::GenericFailure, error.to_string()))
            .map(|data| {
                let solidity_trace = solidity_trace.map(|trace| SolidityTraceData {
                    trace,
                    contract_decoder: Arc::clone(&self.contract_decoder),
                });
                Response {
                    solidity_trace,
                    data,
                    traces: traces.into_iter().map(Arc::new).collect(),
                }
            })
    }

    #[napi(ts_return_type = "void")]
    pub fn set_call_override_callback(
        &self,
        env: Env,
        #[napi(
            ts_arg_type = "(contract_address: Buffer, data: Buffer) => Promise<CallOverrideResult | undefined>"
        )]
        call_override_callback: JsFunction,
    ) -> napi::Result<()> {
        let provider = self.provider.clone();

        let call_override_callback =
            CallOverrideCallback::new(&env, call_override_callback, self.runtime.clone())?;
        let call_override_callback =
            Arc::new(move |address, data| call_override_callback.call_override(address, data));

        provider.set_call_override_callback(Some(call_override_callback));

        Ok(())
    }

    /// Set to `true` to make the traces returned with `eth_call`,
    /// `eth_estimateGas`, `eth_sendRawTransaction`, `eth_sendTransaction`,
    /// `evm_mine`, `hardhat_mine` include the full stack and memory. Set to
    /// `false` to disable this.
    #[napi(ts_return_type = "void")]
    pub fn set_verbose_tracing(&self, verbose_tracing: bool) {
        self.provider.set_verbose_tracing(verbose_tracing);
    }
}

/// Tracing config for Solidity stack trace generation.
#[napi(object)]
pub struct TracingConfigWithBuffers {
    /// Build information to use for decoding contracts. Either a Hardhat v2
    /// build info file that contains both input and output or a Hardhat v3
    /// build info file that doesn't contain output and a separate output file.
    pub build_infos: Option<Either<Vec<Uint8Array>, Vec<BuildInfoAndOutput>>>,
    /// Whether to ignore contracts whose name starts with "Ignored".
    pub ignore_contracts: Option<bool>,
}

/// Hardhat V3 build info where the compiler output is not part of the build
/// info file.
#[napi(object)]
pub struct BuildInfoAndOutput {
    /// The build info input file
    pub build_info: Uint8Array,
    /// The build info output file
    pub output: Uint8Array,
}

impl<'a> From<&'a BuildInfoAndOutput>
    for edr_solidity::artifacts::BuildInfoBufferSeparateOutput<'a>
{
    fn from(value: &'a BuildInfoAndOutput) -> Self {
        Self {
            build_info: value.build_info.as_ref(),
            output: value.output.as_ref(),
        }
    }
}

impl<'a> From<&'a TracingConfigWithBuffers>
    for edr_solidity::artifacts::BuildInfoConfigWithBuffers<'a>
{
    fn from(value: &'a TracingConfigWithBuffers) -> Self {
        use edr_solidity::artifacts::{BuildInfoBufferSeparateOutput, BuildInfoBuffers};

        let build_infos = value.build_infos.as_ref().map(|infos| match infos {
            Either::A(with_output) => BuildInfoBuffers::WithOutput(
                with_output
                    .iter()
                    .map(std::convert::AsRef::as_ref)
                    .collect(),
            ),
            Either::B(separate_output) => BuildInfoBuffers::SeparateInputOutput(
                separate_output
                    .iter()
                    .map(BuildInfoBufferSeparateOutput::from)
                    .collect(),
            ),
        });

        Self {
            build_infos,
            ignore_contracts: value.ignore_contracts,
        }
    }
}

#[derive(Debug)]
struct SolidityTraceData {
    trace: Arc<edr_evm::trace::Trace>,
    contract_decoder: Arc<ContractDecoder>,
}

#[napi]
pub struct Response {
    // N-API is known to be slow when marshalling `serde_json::Value`s, so we try to return a
    // `String`. If the object is too large to be represented as a `String`, we return a `Buffer`
    // instead.
    data: Either<String, serde_json::Value>,
    /// When a transaction fails to execute, the provider returns a trace of the
    /// transaction.
    solidity_trace: Option<SolidityTraceData>,
    /// This may contain zero or more traces, depending on the (batch) request
    traces: Vec<Arc<edr_evm::trace::Trace>>,
}

#[napi]
impl Response {
    /// Returns the response data as a JSON string or a JSON object.
    #[napi(getter)]
    pub fn data(&self) -> Either<String, serde_json::Value> {
        self.data.clone()
    }

    #[napi(getter)]
    pub fn traces(&self) -> Vec<RawTrace> {
        self.traces
            .iter()
            .map(|trace| RawTrace::new(trace.clone()))
            .collect()
    }

    // Rust port of https://github.com/NomicFoundation/hardhat/blob/c20bf195a6efdc2d74e778b7a4a7799aac224841/packages/hardhat-core/src/internal/hardhat-network/provider/provider.ts#L590
    #[doc = "Compute the error stack trace. Return the stack trace if it can be decoded, otherwise returns none. Throws if there was an error computing the stack trace."]
    #[napi]
    pub fn stack_trace(&self) -> napi::Result<Option<SolidityStackTrace>> {
        let Some(SolidityTraceData {
            trace,
            contract_decoder,
        }) = &self.solidity_trace
        else {
            return Ok(None);
        };
        let nested_trace = edr_solidity::nested_tracer::convert_trace_messages_to_nested_trace(
            trace.as_ref().clone(),
        )
        .map_err(|err| napi::Error::from_reason(err.to_string()))?;

        if let Some(vm_trace) = nested_trace {
            let decoded_trace = contract_decoder.try_to_decode_message_trace(vm_trace);
            let stack_trace = edr_solidity::solidity_tracer::get_stack_trace(decoded_trace)
                .map_err(|err| napi::Error::from_reason(err.to_string()))?;
            let stack_trace = stack_trace
                .into_iter()
                .map(super::cast::TryCast::try_cast)
                .collect::<Result<Vec<_>, _>>()?;

            Ok(Some(stack_trace))
        } else {
            Ok(None)
        }
    }
}
