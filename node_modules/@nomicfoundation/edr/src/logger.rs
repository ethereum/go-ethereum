use std::{fmt::Display, sync::mpsc::channel};

use ansi_term::{Color, Style};
use edr_eth::{transaction::Transaction, Bytes, B256, U256};
use edr_evm::{
    blockchain::BlockchainError,
    precompile::{self, Precompiles},
    trace::{AfterMessage, TraceMessage},
    ExecutableTransaction, ExecutionResult, SyncBlock,
};
use edr_provider::{ProviderError, TransactionFailure};
use itertools::izip;
use napi::{
    threadsafe_function::{
        ErrorStrategy, ThreadSafeCallContext, ThreadsafeFunction, ThreadsafeFunctionCallMode,
    },
    Env, JsFunction, Status,
};
use napi_derive::napi;

use crate::cast::TryCast;

#[napi(object)]
pub struct ContractAndFunctionName {
    /// The contract name.
    pub contract_name: String,
    /// The function name. Only present for calls.
    pub function_name: Option<String>,
}

impl TryCast<(String, Option<String>)> for ContractAndFunctionName {
    type Error = napi::Error;

    fn try_cast(self) -> std::result::Result<(String, Option<String>), Self::Error> {
        Ok((self.contract_name, self.function_name))
    }
}

struct ContractAndFunctionNameCall {
    code: Bytes,
    /// Only present for calls.
    calldata: Option<Bytes>,
}

#[napi(object)]
pub struct LoggerConfig {
    /// Whether to enable the logger.
    pub enable: bool,
    #[napi(ts_type = "(inputs: Buffer[]) => string[]")]
    pub decode_console_log_inputs_callback: JsFunction,
    #[napi(ts_type = "(code: Buffer, calldata?: Buffer) => ContractAndFunctionName")]
    pub get_contract_and_function_name_callback: JsFunction,
    #[napi(ts_type = "(message: string, replace: boolean) => void")]
    pub print_line_callback: JsFunction,
}

#[derive(Clone)]
pub enum LoggingState {
    CollapsingMethod(CollapsedMethod),
    HardhatMinining {
        empty_blocks_range_start: Option<u64>,
    },
    IntervalMining {
        empty_blocks_range_start: Option<u64>,
    },
    Empty,
}

impl LoggingState {
    /// Converts the state into a hardhat mining state.
    pub fn into_hardhat_mining(self) -> Option<u64> {
        match self {
            Self::HardhatMinining {
                empty_blocks_range_start,
            } => empty_blocks_range_start,
            _ => None,
        }
    }

    /// Converts the state into an interval mining state.
    pub fn into_interval_mining(self) -> Option<u64> {
        match self {
            Self::IntervalMining {
                empty_blocks_range_start,
            } => empty_blocks_range_start,
            _ => None,
        }
    }
}

impl Default for LoggingState {
    fn default() -> Self {
        Self::Empty
    }
}

#[derive(Clone)]
enum LogLine {
    Single(String),
    WithTitle(String, String),
}

#[derive(Debug, thiserror::Error)]
pub enum LoggerError {
    #[error("Failed to print line")]
    PrintLine,
}

#[derive(Clone)]
pub struct Logger {
    collector: LogCollector,
}

impl Logger {
    pub fn new(env: &Env, config: LoggerConfig) -> napi::Result<Self> {
        Ok(Self {
            collector: LogCollector::new(env, config)?,
        })
    }
}

impl edr_provider::Logger for Logger {
    type BlockchainError = BlockchainError;

    type LoggerError = LoggerError;

    fn is_enabled(&self) -> bool {
        self.collector.is_enabled
    }

    fn set_is_enabled(&mut self, is_enabled: bool) {
        self.collector.is_enabled = is_enabled;
    }

    fn log_call(
        &mut self,
        spec_id: edr_eth::SpecId,
        transaction: &ExecutableTransaction,
        result: &edr_provider::CallResult,
    ) -> Result<(), Self::LoggerError> {
        self.collector.log_call(spec_id, transaction, result);

        Ok(())
    }

    fn log_estimate_gas_failure(
        &mut self,
        spec_id: edr_eth::SpecId,
        transaction: &ExecutableTransaction,
        failure: &edr_provider::EstimateGasFailure,
    ) -> Result<(), Self::LoggerError> {
        self.collector
            .log_estimate_gas(spec_id, transaction, failure);

        Ok(())
    }

    fn log_interval_mined(
        &mut self,
        spec_id: edr_eth::SpecId,
        mining_result: &edr_provider::DebugMineBlockResult<Self::BlockchainError>,
    ) -> Result<(), Self::LoggerError> {
        self.collector.log_interval_mined(spec_id, mining_result)
    }

    fn log_mined_block(
        &mut self,
        spec_id: edr_eth::SpecId,
        mining_results: &[edr_provider::DebugMineBlockResult<Self::BlockchainError>],
    ) -> Result<(), Self::LoggerError> {
        self.collector.log_mined_blocks(spec_id, mining_results);

        Ok(())
    }

    fn log_send_transaction(
        &mut self,
        spec_id: edr_eth::SpecId,
        transaction: &edr_evm::ExecutableTransaction,
        mining_results: &[edr_provider::DebugMineBlockResult<Self::BlockchainError>],
    ) -> Result<(), Self::LoggerError> {
        self.collector
            .log_send_transaction(spec_id, transaction, mining_results);

        Ok(())
    }

    fn print_method_logs(
        &mut self,
        method: &str,
        error: Option<&ProviderError<LoggerError>>,
    ) -> Result<(), Self::LoggerError> {
        if let Some(error) = error {
            self.collector.state = LoggingState::Empty;

            if matches!(error, ProviderError::UnsupportedMethod { .. }) {
                self.collector
                    .print::<false>(Color::Red.paint(error.to_string()))?;
            } else {
                self.collector.print::<false>(Color::Red.paint(method))?;
                self.collector.print_logs()?;

                if !matches!(error, ProviderError::TransactionFailed(_)) {
                    self.collector.print_empty_line()?;

                    let error_message = error.to_string();
                    self.collector
                        .try_indented(|logger| logger.print::<false>(&error_message))?;

                    if matches!(error, ProviderError::InvalidEip155TransactionChainId) {
                        self.collector.try_indented(|logger| {
                            logger.print::<false>(Color::Yellow.paint(
                                "If you are using MetaMask, you can learn how to fix this error here: https://hardhat.org/metamask-issue"
                            ))
                        })?;
                    }
                }

                self.collector.print_empty_line()?;
            }
        } else {
            self.collector.print_method(method)?;

            let printed = self.collector.print_logs()?;
            if printed {
                self.collector.print_empty_line()?;
            }
        }

        Ok(())
    }
}

#[derive(Clone)]
pub struct CollapsedMethod {
    count: usize,
    method: String,
}

#[derive(Clone)]
struct LogCollector {
    decode_console_log_inputs_fn: ThreadsafeFunction<Vec<Bytes>, ErrorStrategy::Fatal>,
    get_contract_and_function_name_fn:
        ThreadsafeFunction<ContractAndFunctionNameCall, ErrorStrategy::Fatal>,
    indentation: usize,
    is_enabled: bool,
    logs: Vec<LogLine>,
    print_line_fn: ThreadsafeFunction<(String, bool), ErrorStrategy::Fatal>,
    state: LoggingState,
    title_length: usize,
}

impl LogCollector {
    pub fn new(env: &Env, config: LoggerConfig) -> napi::Result<Self> {
        let mut decode_console_log_inputs_fn = config
            .decode_console_log_inputs_callback
            .create_threadsafe_function(0, |ctx: ThreadSafeCallContext<Vec<Bytes>>| {
                let inputs =
                    ctx.env
                        .create_array_with_length(ctx.value.len())
                        .and_then(|mut inputs| {
                            for (idx, input) in ctx.value.into_iter().enumerate() {
                                ctx.env.create_buffer_with_data(input.to_vec()).and_then(
                                    |input| inputs.set_element(idx as u32, input.into_raw()),
                                )?;
                            }

                            Ok(inputs)
                        })?;

                Ok(vec![inputs])
            })?;

        // Maintain a weak reference to the function to avoid the event loop from
        // exiting.
        decode_console_log_inputs_fn.unref(env)?;

        let mut get_contract_and_function_name_fn = config
            .get_contract_and_function_name_callback
            .create_threadsafe_function(
                0,
                |ctx: ThreadSafeCallContext<ContractAndFunctionNameCall>| {
                    // Buffer
                    let code = ctx
                        .env
                        .create_buffer_with_data(ctx.value.code.to_vec())?
                        .into_unknown();

                    // Option<Buffer>
                    let calldata = if let Some(calldata) = ctx.value.calldata {
                        ctx.env
                            .create_buffer_with_data(calldata.to_vec())?
                            .into_unknown()
                    } else {
                        ctx.env.get_undefined()?.into_unknown()
                    };

                    Ok(vec![code, calldata])
                },
            )?;

        // Maintain a weak reference to the function to avoid the event loop from
        // exiting.
        get_contract_and_function_name_fn.unref(env)?;

        let mut print_line_fn = config.print_line_callback.create_threadsafe_function(
            0,
            |ctx: ThreadSafeCallContext<(String, bool)>| {
                // String
                let message = ctx.env.create_string_from_std(ctx.value.0)?;

                // bool
                let replace = ctx.env.get_boolean(ctx.value.1)?;

                Ok(vec![message.into_unknown(), replace.into_unknown()])
            },
        )?;

        // Maintain a weak reference to the function to avoid the event loop from
        // exiting.
        print_line_fn.unref(env)?;

        Ok(Self {
            decode_console_log_inputs_fn,
            get_contract_and_function_name_fn,
            indentation: 0,
            is_enabled: config.enable,
            logs: Vec::new(),
            print_line_fn,
            state: LoggingState::default(),
            title_length: 0,
        })
    }

    pub fn log_call(
        &mut self,
        spec_id: edr_eth::SpecId,
        transaction: &ExecutableTransaction,
        result: &edr_provider::CallResult,
    ) {
        let edr_provider::CallResult {
            console_log_inputs,
            execution_result,
            trace,
        } = result;

        self.state = LoggingState::Empty;

        self.indented(|logger| {
            logger.log_contract_and_function_name::<true>(spec_id, trace);

            logger.log_with_title("From", format!("0x{:x}", transaction.caller()));
            if let Some(to) = transaction.to() {
                logger.log_with_title("To", format!("0x{to:x}"));
            }
            if transaction.value() > U256::ZERO {
                logger.log_with_title("Value", wei_to_human_readable(transaction.value()));
            }

            logger.log_console_log_messages(console_log_inputs);

            if let Some(transaction_failure) =
                TransactionFailure::from_execution_result(execution_result, None, trace)
            {
                logger.log_transaction_failure(&transaction_failure);
            }
        });
    }

    pub fn log_estimate_gas(
        &mut self,
        spec_id: edr_eth::SpecId,
        transaction: &ExecutableTransaction,
        result: &edr_provider::EstimateGasFailure,
    ) {
        let edr_provider::EstimateGasFailure {
            console_log_inputs,
            transaction_failure,
        } = result;

        self.state = LoggingState::Empty;

        self.indented(|logger| {
            logger.log_contract_and_function_name::<true>(
                spec_id,
                &transaction_failure.failure.solidity_trace,
            );

            logger.log_with_title("From", format!("0x{:x}", transaction.caller()));
            if let Some(to) = transaction.to() {
                logger.log_with_title("To", format!("0x{to:x}"));
            }
            logger.log_with_title("Value", wei_to_human_readable(transaction.value()));

            logger.log_console_log_messages(console_log_inputs);

            logger.log_transaction_failure(&transaction_failure.failure);
        });
    }

    fn log_transaction_failure(&mut self, failure: &edr_provider::TransactionFailure) {
        let is_revert_error = matches!(
            failure.reason,
            edr_provider::TransactionFailureReason::Revert(_)
        );

        let error_type = if is_revert_error {
            "Error"
        } else {
            "TransactionExecutionError"
        };

        self.log_empty_line();
        self.log(format!("{error_type}: {failure}"));
    }

    pub fn log_mined_blocks(
        &mut self,
        spec_id: edr_eth::SpecId,
        mining_results: &[edr_provider::DebugMineBlockResult<BlockchainError>],
    ) {
        let num_results = mining_results.len();
        for (idx, mining_result) in mining_results.iter().enumerate() {
            let state = std::mem::take(&mut self.state);
            let empty_blocks_range_start = state.into_hardhat_mining();

            if mining_result.block.transactions().is_empty() {
                self.log_hardhat_mined_empty_block(&mining_result.block, empty_blocks_range_start);

                let block_number = mining_result.block.header().number;
                self.state = LoggingState::HardhatMinining {
                    empty_blocks_range_start: Some(
                        empty_blocks_range_start.unwrap_or(block_number),
                    ),
                };
            } else {
                self.log_hardhat_mined_block(spec_id, mining_result);

                if idx < num_results - 1 {
                    self.log_empty_line();
                }
            }
        }
    }

    pub fn log_interval_mined(
        &mut self,
        spec_id: edr_eth::SpecId,
        mining_result: &edr_provider::DebugMineBlockResult<BlockchainError>,
    ) -> Result<(), LoggerError> {
        let block_header = mining_result.block.header();
        let block_number = block_header.number;

        if mining_result.block.transactions().is_empty() {
            let state = std::mem::take(&mut self.state);
            let empty_blocks_range_start = state.into_interval_mining();

            if let Some(empty_blocks_range_start) = empty_blocks_range_start {
                self.print::<true>(format!(
                    "Mined empty block range #{empty_blocks_range_start} to #{block_number}"
                ))?;
            } else {
                let base_fee = if let Some(base_fee) = block_header.base_fee_per_gas.as_ref() {
                    format!(" with base fee {base_fee}")
                } else {
                    String::new()
                };

                self.print::<false>(format!("Mined empty block #{block_number}{base_fee}"))?;
            }

            self.state = LoggingState::IntervalMining {
                empty_blocks_range_start: Some(
                    empty_blocks_range_start.unwrap_or(block_header.number),
                ),
            };
        } else {
            self.log_interval_mined_block(spec_id, mining_result);

            self.print::<false>(format!("Mined block #{block_number}"))?;

            let printed = self.print_logs()?;
            if printed {
                self.print_empty_line()?;
            }
        }

        Ok(())
    }

    pub fn log_send_transaction(
        &mut self,
        spec_id: edr_eth::SpecId,
        transaction: &edr_evm::ExecutableTransaction,
        mining_results: &[edr_provider::DebugMineBlockResult<BlockchainError>],
    ) {
        if !mining_results.is_empty() {
            self.state = LoggingState::Empty;

            let (sent_block_result, sent_transaction_result, sent_trace) = mining_results
                .iter()
                .find_map(|result| {
                    izip!(
                        result.block.transactions(),
                        result.transaction_results.iter(),
                        result.transaction_traces.iter()
                    )
                    .find(|(block_transaction, _, _)| {
                        *block_transaction.transaction_hash() == *transaction.transaction_hash()
                    })
                    .map(|(_, transaction_result, trace)| (result, transaction_result, trace))
                })
                .expect("Transaction result not found");

            if mining_results.len() > 1 {
                self.log_multiple_blocks_warning();
                self.log_auto_mined_block_results(
                    spec_id,
                    mining_results,
                    transaction.transaction_hash(),
                );
                self.log_currently_sent_transaction(
                    spec_id,
                    sent_block_result,
                    transaction,
                    sent_transaction_result,
                    sent_trace,
                );
            } else if let Some(result) = mining_results.first() {
                let transactions = result.block.transactions();
                if transactions.len() > 1 {
                    self.log_multiple_transactions_warning();
                    self.log_auto_mined_block_results(
                        spec_id,
                        mining_results,
                        transaction.transaction_hash(),
                    );
                    self.log_currently_sent_transaction(
                        spec_id,
                        sent_block_result,
                        transaction,
                        sent_transaction_result,
                        sent_trace,
                    );
                } else if let Some(transaction) = transactions.first() {
                    self.log_single_transaction_mining_result(spec_id, result, transaction);
                }
            }
        }
    }

    fn contract_and_function_name(
        &self,
        code: Bytes,
        calldata: Option<Bytes>,
    ) -> (String, Option<String>) {
        let (sender, receiver) = channel();

        let status = self
            .get_contract_and_function_name_fn
            .call_with_return_value(
                ContractAndFunctionNameCall { code, calldata },
                ThreadsafeFunctionCallMode::Blocking,
                move |result: ContractAndFunctionName| {
                    let contract_and_function_name = result.try_cast();
                    sender.send(contract_and_function_name).map_err(|_error| {
                        napi::Error::new(
                            Status::GenericFailure,
                            "Failed to send result from get_contract_and_function_name",
                        )
                    })
                },
            );
        assert_eq!(status, Status::Ok);

        receiver
            .recv()
            .unwrap()
            .expect("Failed call to get_contract_and_function_name")
    }

    fn format(&self, message: impl ToString) -> String {
        let message = message.to_string();

        if message.is_empty() {
            message
        } else {
            message
                .split('\n')
                .map(|line| format!("{:indent$}{line}", "", indent = self.indentation))
                .collect::<Vec<_>>()
                .join("\n")
        }
    }

    fn indented(&mut self, display_fn: impl FnOnce(&mut Self)) {
        self.indentation += 2;
        display_fn(self);
        self.indentation -= 2;
    }

    fn try_indented(
        &mut self,
        display_fn: impl FnOnce(&mut Self) -> Result<(), LoggerError>,
    ) -> Result<(), LoggerError> {
        self.indentation += 2;
        let result = display_fn(self);
        self.indentation -= 2;

        result
    }

    fn log(&mut self, message: impl ToString) {
        let formatted = self.format(message);

        self.logs.push(LogLine::Single(formatted));
    }

    fn log_auto_mined_block_results(
        &mut self,
        spec_id: edr_eth::SpecId,
        results: &[edr_provider::DebugMineBlockResult<BlockchainError>],
        sent_transaction_hash: &B256,
    ) {
        for result in results {
            self.log_block_from_auto_mine(spec_id, result, sent_transaction_hash);
        }
    }

    fn log_base_fee(&mut self, base_fee: Option<&U256>) {
        if let Some(base_fee) = base_fee {
            self.log(format!("Base fee: {base_fee}"));
        }
    }

    fn log_block_from_auto_mine(
        &mut self,
        spec_id: edr_eth::SpecId,
        result: &edr_provider::DebugMineBlockResult<BlockchainError>,
        transaction_hash_to_highlight: &edr_eth::B256,
    ) {
        let edr_provider::DebugMineBlockResult {
            block,
            transaction_results,
            transaction_traces,
            console_log_inputs,
        } = result;

        let transactions = block.transactions();
        let num_transactions = transactions.len();

        debug_assert_eq!(num_transactions, transaction_results.len());
        debug_assert_eq!(num_transactions, transaction_traces.len());

        let block_header = block.header();

        self.indented(|logger| {
            logger.log_block_id(block);

            logger.indented(|logger| {
                logger.log_base_fee(block_header.base_fee_per_gas.as_ref());

                for (idx, transaction, result, trace) in izip!(
                    0..num_transactions,
                    transactions,
                    transaction_results,
                    transaction_traces
                ) {
                    let should_highlight_hash =
                        *transaction.transaction_hash() == *transaction_hash_to_highlight;
                    logger.log_block_transaction(
                        spec_id,
                        transaction,
                        result,
                        trace,
                        console_log_inputs,
                        should_highlight_hash,
                    );

                    logger.log_empty_line_between_transactions(idx, num_transactions);
                }
            });
        });

        self.log_empty_line();
    }

    fn log_block_hash(&mut self, block: &dyn SyncBlock<Error = BlockchainError>) {
        let block_hash = block.hash();

        self.log(format!("Block: {block_hash}"));
    }

    fn log_block_id(&mut self, block: &dyn SyncBlock<Error = BlockchainError>) {
        let block_number = block.header().number;
        let block_hash = block.hash();

        self.log(format!("Block #{block_number}: {block_hash}"));
    }

    fn log_block_number(&mut self, block: &dyn SyncBlock<Error = BlockchainError>) {
        let block_number = block.header().number;

        self.log(format!("Mined block #{block_number}"));
    }

    /// Logs a transaction that's part of a block.
    fn log_block_transaction(
        &mut self,
        spec_id: edr_eth::SpecId,
        transaction: &edr_evm::ExecutableTransaction,
        result: &edr_evm::ExecutionResult,
        trace: &edr_evm::trace::Trace,
        console_log_inputs: &[Bytes],
        should_highlight_hash: bool,
    ) {
        let transaction_hash = transaction.transaction_hash();
        if should_highlight_hash {
            self.log_with_title(
                "Transaction",
                Style::new().bold().paint(transaction_hash.to_string()),
            );
        } else {
            self.log_with_title("Transaction", transaction_hash.to_string());
        }

        self.indented(|logger| {
            logger.log_contract_and_function_name::<false>(spec_id, trace);
            logger.log_with_title("From", format!("0x{:x}", transaction.caller()));
            if let Some(to) = transaction.to() {
                logger.log_with_title("To", format!("0x{to:x}"));
            }
            logger.log_with_title("Value", wei_to_human_readable(transaction.value()));
            logger.log_with_title(
                "Gas used",
                format!(
                    "{gas_used} of {gas_limit}",
                    gas_used = result.gas_used(),
                    gas_limit = transaction.gas_limit()
                ),
            );

            logger.log_console_log_messages(console_log_inputs);

            let transaction_failure = edr_provider::TransactionFailure::from_execution_result(
                result,
                Some(transaction_hash),
                trace,
            );

            if let Some(transaction_failure) = transaction_failure {
                logger.log_transaction_failure(&transaction_failure);
            }
        });
    }

    fn log_console_log_messages(&mut self, console_log_inputs: &[Bytes]) {
        let (sender, receiver) = channel();

        let status = self.decode_console_log_inputs_fn.call_with_return_value(
            console_log_inputs.to_vec(),
            ThreadsafeFunctionCallMode::Blocking,
            move |decoded_inputs: Vec<String>| {
                sender.send(decoded_inputs).map_err(|_error| {
                    napi::Error::new(
                        Status::GenericFailure,
                        "Failed to send result from decode_console_log_inputs",
                    )
                })
            },
        );
        assert_eq!(status, Status::Ok);

        let console_log_inputs = receiver.recv().unwrap();
        // This is a special case, as we always want to print the console.log messages.
        // The difference is how. If we have a logger, we should use that, so that logs
        // are printed in order. If we don't, we just print the messages here.
        if self.is_enabled {
            if !console_log_inputs.is_empty() {
                self.log_empty_line();
                self.log("console.log:");

                self.indented(|logger| {
                    for input in console_log_inputs {
                        logger.log(input);
                    }
                });
            }
        } else {
            for input in console_log_inputs {
                let status = self
                    .print_line_fn
                    .call((input, false), ThreadsafeFunctionCallMode::Blocking);

                assert_eq!(status, napi::Status::Ok);
            }
        }
    }

    fn log_contract_and_function_name<const PRINT_INVALID_CONTRACT_WARNING: bool>(
        &mut self,
        spec_id: edr_eth::SpecId,
        trace: &edr_evm::trace::Trace,
    ) {
        if let Some(TraceMessage::Before(before_message)) = trace.messages.first() {
            if let Some(to) = before_message.to {
                // Call
                let is_precompile = {
                    let precompiles =
                        Precompiles::new(precompile::PrecompileSpecId::from_spec_id(spec_id));
                    precompiles.contains(&to)
                };

                if is_precompile {
                    let precompile = u16::from_be_bytes([to[18], to[19]]);
                    self.log_with_title(
                        "Precompile call",
                        format!("<PrecompileContract {precompile}>"),
                    );
                } else {
                    let is_code_empty = before_message
                        .code
                        .as_ref()
                        .map_or(true, edr_evm::Bytecode::is_empty);

                    if is_code_empty {
                        if PRINT_INVALID_CONTRACT_WARNING {
                            self.log("WARNING: Calling an account which is not a contract");
                        }
                    } else {
                        let (contract_name, function_name) = self.contract_and_function_name(
                            before_message
                                .code
                                .as_ref()
                                .map(edr_evm::Bytecode::original_bytes)
                                .expect("Call must be defined"),
                            Some(before_message.data.clone()),
                        );

                        let function_name = function_name.expect("Function name must be defined");
                        self.log_with_title(
                            "Contract call",
                            if function_name.is_empty() {
                                contract_name
                            } else {
                                format!("{contract_name}#{function_name}")
                            },
                        );
                    }
                }
            } else {
                let result = if let Some(TraceMessage::After(AfterMessage {
                    execution_result,
                    ..
                })) = trace.messages.last()
                {
                    execution_result
                } else {
                    unreachable!("Before messages must have an after message")
                };

                // Create
                let (contract_name, _) =
                    self.contract_and_function_name(before_message.data.clone(), None);

                self.log_with_title("Contract deployment", contract_name);

                if let ExecutionResult::Success { output, .. } = result {
                    if let edr_evm::Output::Create(_, address) = output {
                        if let Some(deployed_address) = address {
                            self.log_with_title(
                                "Contract address",
                                format!("0x{deployed_address:x}"),
                            );
                        }
                    } else {
                        unreachable!("Create calls must return a Create output")
                    }
                }
            }
        }
    }

    fn log_empty_block(&mut self, block: &dyn SyncBlock<Error = BlockchainError>) {
        let block_header = block.header();
        let block_number = block_header.number;

        let base_fee = if let Some(base_fee) = block_header.base_fee_per_gas.as_ref() {
            format!(" with base fee {base_fee}")
        } else {
            String::new()
        };

        self.log(format!("Mined empty block #{block_number}{base_fee}",));
    }

    fn log_empty_line(&mut self) {
        self.log("");
    }

    fn log_empty_line_between_transactions(&mut self, idx: usize, num_transactions: usize) {
        if num_transactions > 1 && idx < num_transactions - 1 {
            self.log_empty_line();
        }
    }

    fn log_hardhat_mined_empty_block(
        &mut self,
        block: &dyn SyncBlock<Error = BlockchainError>,
        empty_blocks_range_start: Option<u64>,
    ) {
        self.indented(|logger| {
            if let Some(empty_blocks_range_start) = empty_blocks_range_start {
                logger.replace_last_log_line(format!(
                    "Mined empty block range #{empty_blocks_range_start} to #{block_number}",
                    block_number = block.header().number
                ));
            } else {
                logger.log_empty_block(block);
            }
        });
    }

    /// Logs the result of interval mining a block.
    fn log_interval_mined_block(
        &mut self,
        spec_id: edr_eth::SpecId,
        result: &edr_provider::DebugMineBlockResult<BlockchainError>,
    ) {
        let edr_provider::DebugMineBlockResult {
            block,
            transaction_results,
            transaction_traces,
            console_log_inputs,
        } = result;

        let transactions = block.transactions();
        let num_transactions = transactions.len();

        debug_assert_eq!(num_transactions, transaction_results.len());
        debug_assert_eq!(num_transactions, transaction_traces.len());

        let block_header = block.header();

        self.indented(|logger| {
            logger.log_block_hash(block);

            logger.indented(|logger| {
                logger.log_base_fee(block_header.base_fee_per_gas.as_ref());

                for (idx, transaction, result, trace) in izip!(
                    0..num_transactions,
                    transactions,
                    transaction_results,
                    transaction_traces
                ) {
                    logger.log_block_transaction(
                        spec_id,
                        transaction,
                        result,
                        trace,
                        console_log_inputs,
                        false,
                    );

                    logger.log_empty_line_between_transactions(idx, num_transactions);
                }
            });
        });
    }

    fn log_hardhat_mined_block(
        &mut self,
        spec_id: edr_eth::SpecId,
        result: &edr_provider::DebugMineBlockResult<BlockchainError>,
    ) {
        let edr_provider::DebugMineBlockResult {
            block,
            transaction_results,
            transaction_traces,
            console_log_inputs,
        } = result;

        let transactions = block.transactions();
        let num_transactions = transactions.len();

        debug_assert_eq!(num_transactions, transaction_results.len());
        debug_assert_eq!(num_transactions, transaction_traces.len());

        self.indented(|logger| {
            if transactions.is_empty() {
                logger.log_empty_block(block);
            } else {
                logger.log_block_number(block);

                logger.indented(|logger| {
                    logger.log_block_hash(block);

                    logger.indented(|logger| {
                        logger.log_base_fee(block.header().base_fee_per_gas.as_ref());

                        for (idx, transaction, result, trace) in izip!(
                            0..num_transactions,
                            transactions,
                            transaction_results,
                            transaction_traces
                        ) {
                            logger.log_block_transaction(
                                spec_id,
                                transaction,
                                result,
                                trace,
                                console_log_inputs,
                                false,
                            );

                            logger.log_empty_line_between_transactions(idx, num_transactions);
                        }
                    });
                });
            }
        });
    }

    /// Logs a warning about multiple blocks being mined.
    fn log_multiple_blocks_warning(&mut self) {
        self.indented(|logger| {
            logger
                .log("There were other pending transactions. More than one block had to be mined:");
        });
        self.log_empty_line();
    }

    /// Logs a warning about multiple transactions being mined.
    fn log_multiple_transactions_warning(&mut self) {
        self.indented(|logger| {
            logger.log("There were other pending transactions mined in the same block:");
        });
        self.log_empty_line();
    }

    fn log_with_title(&mut self, title: impl Into<String>, message: impl Display) {
        // repeat whitespace self.indentation times and concatenate with title
        let title = format!("{:indent$}{}", "", title.into(), indent = self.indentation);
        if title.len() > self.title_length {
            self.title_length = title.len();
        }

        let message = format!("{message}");
        self.logs.push(LogLine::WithTitle(title, message));
    }

    fn log_currently_sent_transaction(
        &mut self,
        spec_id: edr_eth::SpecId,
        block_result: &edr_provider::DebugMineBlockResult<BlockchainError>,
        transaction: &ExecutableTransaction,
        transaction_result: &edr_evm::ExecutionResult,
        trace: &edr_evm::trace::Trace,
    ) {
        self.indented(|logger| {
            logger.log("Currently sent transaction:");
            logger.log("");
        });

        self.log_transaction(
            spec_id,
            block_result,
            transaction,
            transaction_result,
            trace,
        );
    }

    fn log_single_transaction_mining_result(
        &mut self,
        spec_id: edr_eth::SpecId,
        result: &edr_provider::DebugMineBlockResult<BlockchainError>,
        transaction: &ExecutableTransaction,
    ) {
        let trace = result
            .transaction_traces
            .first()
            .expect("A transaction exists, so the trace must exist as well.");

        let transaction_result = result
            .transaction_results
            .first()
            .expect("A transaction exists, so the result must exist as well.");

        self.log_transaction(spec_id, result, transaction, transaction_result, trace);
    }

    fn log_transaction(
        &mut self,
        spec_id: edr_eth::SpecId,
        block_result: &edr_provider::DebugMineBlockResult<BlockchainError>,
        transaction: &ExecutableTransaction,
        transaction_result: &edr_evm::ExecutionResult,
        trace: &edr_evm::trace::Trace,
    ) {
        self.indented(|logger| {
            logger.log_contract_and_function_name::<false>(spec_id, trace);

            let transaction_hash = transaction.transaction_hash();
            logger.log_with_title("Transaction", transaction_hash);

            logger.log_with_title("From", format!("0x{:x}", transaction.caller()));
            if let Some(to) = transaction.to() {
                logger.log_with_title("To", format!("0x{to:x}"));
            }
            logger.log_with_title("Value", wei_to_human_readable(transaction.value()));
            logger.log_with_title(
                "Gas used",
                format!(
                    "{gas_used} of {gas_limit}",
                    gas_used = transaction_result.gas_used(),
                    gas_limit = transaction.gas_limit()
                ),
            );

            let block_number = block_result.block.header().number;
            logger.log_with_title(format!("Block #{block_number}"), block_result.block.hash());

            logger.log_console_log_messages(&block_result.console_log_inputs);

            let transaction_failure = edr_provider::TransactionFailure::from_execution_result(
                transaction_result,
                Some(transaction_hash),
                trace,
            );

            if let Some(transaction_failure) = transaction_failure {
                logger.log_transaction_failure(&transaction_failure);
            }
        });
    }

    fn print<const REPLACE: bool>(&mut self, message: impl ToString) -> Result<(), LoggerError> {
        if !self.is_enabled {
            return Ok(());
        }

        let formatted = self.format(message);

        let status = self
            .print_line_fn
            .call((formatted, REPLACE), ThreadsafeFunctionCallMode::Blocking);

        if status == napi::Status::Ok {
            Ok(())
        } else {
            Err(LoggerError::PrintLine)
        }
    }

    fn print_empty_line(&mut self) -> Result<(), LoggerError> {
        self.print::<false>("")
    }

    fn print_logs(&mut self) -> Result<bool, LoggerError> {
        let logs = std::mem::take(&mut self.logs);
        if logs.is_empty() {
            return Ok(false);
        }

        for log in logs {
            let line = match log {
                LogLine::Single(message) => message,
                LogLine::WithTitle(title, message) => {
                    let title = format!("{title}:");
                    format!("{title:indent$} {message}", indent = self.title_length + 1)
                }
            };

            self.print::<false>(line)?;
        }

        Ok(true)
    }

    fn print_method(&mut self, method: &str) -> Result<(), LoggerError> {
        if let Some(collapsed_method) = self.collapsed_method(method) {
            collapsed_method.count += 1;

            let line = format!("{method} ({count})", count = collapsed_method.count);
            self.print::<true>(Color::Green.paint(line))
        } else {
            self.state = LoggingState::CollapsingMethod(CollapsedMethod {
                count: 1,
                method: method.to_string(),
            });

            self.print::<false>(Color::Green.paint(method))
        }
    }

    /// Retrieves the collapsed method with the provided name, if it exists.
    fn collapsed_method(&mut self, method: &str) -> Option<&mut CollapsedMethod> {
        if let LoggingState::CollapsingMethod(collapsed_method) = &mut self.state {
            if collapsed_method.method == method {
                return Some(collapsed_method);
            }
        }

        None
    }

    fn replace_last_log_line(&mut self, message: impl ToString) {
        let formatted = self.format(message);

        *self.logs.last_mut().expect("There must be a log line") = LogLine::Single(formatted);
    }
}

fn wei_to_human_readable(wei: U256) -> String {
    if wei == U256::ZERO {
        "0 ETH".to_string()
    } else if wei < U256::from(100_000u64) {
        format!("{wei} wei")
    } else if wei < U256::from(100_000_000_000_000u64) {
        let mut decimal = to_decimal_string(wei, 9);
        decimal.push_str(" gwei");
        decimal
    } else {
        let mut decimal = to_decimal_string(wei, 18);
        decimal.push_str(" ETH");
        decimal
    }
}

/// Converts the provided `value` to a decimal string after dividing it by
/// `10^exponent`. The returned string will have at most `MAX_DECIMALS`
/// decimals.
fn to_decimal_string(value: U256, exponent: u8) -> String {
    const MAX_DECIMALS: u8 = 4;

    let (integer, remainder) = value.div_rem(U256::from(10).pow(U256::from(exponent)));
    let decimal = remainder / U256::from(10).pow(U256::from(exponent - MAX_DECIMALS));

    // Remove trailing zeros
    let decimal = decimal.to_string().trim_end_matches('0').to_string();

    format!("{integer}.{decimal}")
}
