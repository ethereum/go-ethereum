use edr_eth::B256;
use napi::{
    bindgen_prelude::BigInt,
    threadsafe_function::{
        ErrorStrategy, ThreadSafeCallContext, ThreadsafeFunction, ThreadsafeFunctionCallMode,
    },
    Env, JsFunction,
};
use napi_derive::napi;

#[derive(Clone)]
pub struct SubscriberCallback {
    inner: ThreadsafeFunction<edr_provider::SubscriptionEvent, ErrorStrategy::Fatal>,
}

impl SubscriberCallback {
    pub fn new(env: &Env, subscription_event_callback: JsFunction) -> napi::Result<Self> {
        let mut callback = subscription_event_callback.create_threadsafe_function(
            0,
            |ctx: ThreadSafeCallContext<edr_provider::SubscriptionEvent>| {
                // SubscriptionEvent
                let mut event = ctx.env.create_object()?;

                ctx.env
                    .create_bigint_from_words(false, ctx.value.filter_id.as_limbs().to_vec())
                    .and_then(|filter_id| event.set_named_property("filterId", filter_id))?;

                let result = match ctx.value.result {
                    edr_provider::SubscriptionEventData::Logs(logs) => ctx.env.to_js_value(&logs),
                    edr_provider::SubscriptionEventData::NewHeads(block) => {
                        let block = edr_rpc_eth::Block::<B256>::from(block);
                        ctx.env.to_js_value(&block)
                    }
                    edr_provider::SubscriptionEventData::NewPendingTransactions(tx_hash) => {
                        ctx.env.to_js_value(&tx_hash)
                    }
                }?;

                event.set_named_property("result", result)?;

                Ok(vec![event])
            },
        )?;

        // Maintain a weak reference to the function to avoid the event loop from
        // exiting.
        callback.unref(env)?;

        Ok(Self { inner: callback })
    }

    pub fn call(&self, event: edr_provider::SubscriptionEvent) {
        // This is blocking because it's important that the subscription events are
        // in-order
        self.inner.call(event, ThreadsafeFunctionCallMode::Blocking);
    }
}

#[napi(object)]
pub struct SubscriptionEvent {
    pub filter_id: BigInt,
    pub result: serde_json::Value,
}
