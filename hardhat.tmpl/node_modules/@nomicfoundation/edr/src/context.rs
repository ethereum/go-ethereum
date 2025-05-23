use std::{ops::Deref, sync::Arc};

#[cfg(feature = "tracing")]
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
        let context = Context::new()?;

        Ok(Self {
            inner: Arc::new(context),
        })
    }
}

#[derive(Debug)]
pub struct Context {
    #[cfg(feature = "tracing")]
    _tracing_write_guard: tracing_flame::FlushGuard<std::io::BufWriter<std::fs::File>>,
}

impl Context {
    /// Creates a new [`Context`] instance. Should only be called once!
    pub fn new() -> napi::Result<Self> {
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
            let (flame_layer, guard) = tracing_flame::FlameLayer::with_file("tracing.folded")
                .map_err(|err| {
                    napi::Error::new(
                        Status::GenericFailure,
                        format!("Failed to create tracing.folded file with error: {err:?}"),
                    )
                })?;

            let flame_layer = flame_layer.with_empty_samples(false);
            (flame_layer, guard)
        };

        #[cfg(feature = "tracing")]
        let subscriber = subscriber.with(flame_layer);

        if let Err(error) = tracing::subscriber::set_global_default(subscriber) {
            println!(
                "Failed to set global tracing subscriber with error: {error}\n\
                Please only initialize EdrContext once per process to avoid this error."
            );
        }

        Ok(Self {
            #[cfg(feature = "tracing")]
            _tracing_write_guard: guard,
        })
    }
}
