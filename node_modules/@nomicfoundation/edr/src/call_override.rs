use std::sync::mpsc::channel;

use edr_eth::{Address, Bytes};
use napi::{
    bindgen_prelude::{Buffer, Promise},
    threadsafe_function::{
        ErrorStrategy, ThreadSafeCallContext, ThreadsafeFunction, ThreadsafeFunctionCallMode,
    },
    tokio::runtime,
    Env, JsFunction, Status,
};
use napi_derive::napi;

use crate::cast::TryCast;

/// The result of executing a call override.
#[napi(object)]
pub struct CallOverrideResult {
    pub result: Buffer,
    pub should_revert: bool,
}

impl TryCast<Option<edr_provider::CallOverrideResult>> for Option<CallOverrideResult> {
    type Error = napi::Error;

    fn try_cast(self) -> Result<Option<edr_provider::CallOverrideResult>, Self::Error> {
        match self {
            None => Ok(None),
            Some(result) => Ok(Some(edr_provider::CallOverrideResult {
                output: result.result.try_cast()?,
                should_revert: result.should_revert,
            })),
        }
    }
}

struct CallOverrideCall {
    contract_address: Address,
    data: Bytes,
}

#[derive(Clone)]
pub struct CallOverrideCallback {
    call_override_callback_fn: ThreadsafeFunction<CallOverrideCall, ErrorStrategy::Fatal>,
    runtime: runtime::Handle,
}

impl CallOverrideCallback {
    pub fn new(
        env: &Env,
        call_override_callback: JsFunction,
        runtime: runtime::Handle,
    ) -> napi::Result<Self> {
        let mut call_override_callback_fn = call_override_callback.create_threadsafe_function(
            0,
            |ctx: ThreadSafeCallContext<CallOverrideCall>| {
                let address = ctx
                    .env
                    .create_buffer_with_data(ctx.value.contract_address.to_vec())?
                    .into_raw();

                let data = ctx
                    .env
                    .create_buffer_with_data(ctx.value.data.to_vec())?
                    .into_raw();

                Ok(vec![address, data])
            },
        )?;

        // Maintain a weak reference to the function to avoid the event loop from
        // exiting.
        call_override_callback_fn.unref(env)?;

        Ok(Self {
            call_override_callback_fn,
            runtime,
        })
    }

    pub fn call_override(
        &self,
        contract_address: Address,
        data: Bytes,
    ) -> Option<edr_provider::CallOverrideResult> {
        let (sender, receiver) = channel();

        let runtime = self.runtime.clone();
        let status = self.call_override_callback_fn.call_with_return_value(
            CallOverrideCall {
                contract_address,
                data,
            },
            ThreadsafeFunctionCallMode::Blocking,
            move |result: Promise<Option<CallOverrideResult>>| {
                runtime.spawn(async move {
                    let result = result.await?.try_cast();
                    sender.send(result).map_err(|_error| {
                        napi::Error::new(
                            Status::GenericFailure,
                            "Failed to send result from call_override_callback",
                        )
                    })
                });
                Ok(())
            },
        );

        assert_eq!(status, Status::Ok, "Call override callback failed");

        receiver
            .recv()
            .unwrap()
            .expect("Failed call to call_override_callback")
    }
}
