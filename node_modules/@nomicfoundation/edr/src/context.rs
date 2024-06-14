use std::{io, ops::Deref, sync::Arc};

use napi::Status;
use napi_derive::napi;
use tracing_subscriber::{prelude::*, EnvFilter, Registry};

#[napi]
#[derive(Debug)]
pub struct EdrContext {
    inner: Arc<Context>,
}

impl Deref for EdrContext {
    type Target = Arc<Context>;

    fn deref(&self) -> &Self::Target {
        &self.inner
    }
}

#[napi]
impl EdrContext {
    #[doc = "Creates a new [`EdrContext`] instance. Should only be called once!"]
    #[napi(constructor)]
    pub fn new() -> napi::Result<Self> {
        let context =
            Context::new().map_err(|e| napi::Error::new(Status::GenericFailure, e.to_string()))?;

        Ok(Self {
            inner: Arc::new(context),
        })
    }
}

#[derive(Debug)]
pub struct Context {
    _subscriber_guard: tracing::subscriber::DefaultGuard,
    #[cfg(feature = "tracing")]
    _tracing_write_guard: tracing_flame::FlushGuard<std::io::BufWriter<std::fs::File>>,
}

impl Context {
    /// Creates a new [`Context`] instance. Should only be called once!
    pub fn new() -> io::Result<Self> {
        let fmt_layer = tracing_subscriber::fmt::layer()
            .with_file(true)
            .with_line_number(true)
            .with_thread_ids(true)
            .with_target(false)
            .with_level(true)
            .with_filter(EnvFilter::from_default_env());

        let subscriber = Registry::default().with(fmt_layer);

        #[cfg(feature = "tracing")]
        let (flame_layer, guard) = {
            let (flame_layer, guard) =
                tracing_flame::FlameLayer::with_file("tracing.folded").unwrap();

            let flame_layer = flame_layer.with_empty_samples(false);
            (flame_layer, guard)
        };

        #[cfg(feature = "tracing")]
        let subscriber = subscriber.with(flame_layer);

        let subscriber_guard = tracing::subscriber::set_default(subscriber);

        Ok(Self {
            _subscriber_guard: subscriber_guard,
            #[cfg(feature = "tracing")]
            _tracing_write_guard: guard,
        })
    }
}
