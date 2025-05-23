use std::time::{SystemTime, UNIX_EPOCH};

use edr_provider::ProviderRequest;
use napi::tokio::{fs::File, io::AsyncWriteExt, sync::Mutex};
use rand::{distributions::Alphanumeric, Rng};
use serde::Serialize;

const SCENARIO_FILE_PREFIX: &str = "EDR_SCENARIO_PREFIX";

#[derive(Clone, Debug, Serialize)]
struct ScenarioConfig {
    provider_config: edr_scenarios::ScenarioProviderConfig,
    logger_enabled: bool,
}

pub(crate) async fn scenario_file(
    provider_config: &edr_provider::ProviderConfig,
    logger_enabled: bool,
) -> Result<Option<Mutex<File>>, napi::Error> {
    if let Ok(scenario_prefix) = std::env::var(SCENARIO_FILE_PREFIX) {
        let timestamp = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .expect("Time went backwards")
            .as_secs();
        let suffix = rand::thread_rng()
            .sample_iter(&Alphanumeric)
            .take(4)
            .map(char::from)
            .collect::<String>();

        let mut scenario_file =
            File::create(format!("{scenario_prefix}_{timestamp}_{suffix}.json")).await?;

        let config = ScenarioConfig {
            provider_config: provider_config.clone().into(),
            logger_enabled,
        };
        let mut line = serde_json::to_string(&config)?;
        line.push('\n');
        scenario_file.write_all(line.as_bytes()).await?;

        Ok(Some(Mutex::new(scenario_file)))
    } else {
        Ok(None)
    }
}

pub(crate) async fn write_request(
    scenario_file: &Mutex<File>,
    request: &ProviderRequest,
) -> napi::Result<()> {
    let mut line = serde_json::to_string(request)?;
    line.push('\n');
    {
        let mut scenario_file = scenario_file.lock().await;
        scenario_file.write_all(line.as_bytes()).await?;
    }
    Ok(())
}
