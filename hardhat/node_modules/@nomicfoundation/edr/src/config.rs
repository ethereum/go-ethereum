use napi_derive::napi;

/// Identifier for the Ethereum spec.
#[napi]
pub enum SpecId {
    /// Frontier
    Frontier = 0,
    /// Frontier Thawing
    FrontierThawing = 1,
    /// Homestead
    Homestead = 2,
    /// DAO Fork
    DaoFork = 3,
    /// Tangerine
    Tangerine = 4,
    /// Spurious Dragon
    SpuriousDragon = 5,
    /// Byzantium
    Byzantium = 6,
    /// Constantinople
    Constantinople = 7,
    /// Petersburg
    Petersburg = 8,
    /// Istanbul
    Istanbul = 9,
    /// Muir Glacier
    MuirGlacier = 10,
    /// Berlin
    Berlin = 11,
    /// London
    London = 12,
    /// Arrow Glacier
    ArrowGlacier = 13,
    /// Gray Glacier
    GrayGlacier = 14,
    /// Merge
    Merge = 15,
    /// Shanghai
    Shanghai = 16,
    /// Cancun
    Cancun = 17,
    /// Prague
    Prague = 18,
    /// Latest
    Latest = 19,
}

impl From<SpecId> for edr_evm::SpecId {
    fn from(value: SpecId) -> Self {
        match value {
            SpecId::Frontier => edr_evm::SpecId::FRONTIER,
            SpecId::FrontierThawing => edr_evm::SpecId::FRONTIER_THAWING,
            SpecId::Homestead => edr_evm::SpecId::HOMESTEAD,
            SpecId::DaoFork => edr_evm::SpecId::DAO_FORK,
            SpecId::Tangerine => edr_evm::SpecId::TANGERINE,
            SpecId::SpuriousDragon => edr_evm::SpecId::SPURIOUS_DRAGON,
            SpecId::Byzantium => edr_evm::SpecId::BYZANTIUM,
            SpecId::Constantinople => edr_evm::SpecId::CONSTANTINOPLE,
            SpecId::Petersburg => edr_evm::SpecId::PETERSBURG,
            SpecId::Istanbul => edr_evm::SpecId::ISTANBUL,
            SpecId::MuirGlacier => edr_evm::SpecId::MUIR_GLACIER,
            SpecId::Berlin => edr_evm::SpecId::BERLIN,
            SpecId::London => edr_evm::SpecId::LONDON,
            SpecId::ArrowGlacier => edr_evm::SpecId::ARROW_GLACIER,
            SpecId::GrayGlacier => edr_evm::SpecId::GRAY_GLACIER,
            SpecId::Merge => edr_evm::SpecId::MERGE,
            SpecId::Shanghai => edr_evm::SpecId::SHANGHAI,
            SpecId::Cancun => edr_evm::SpecId::CANCUN,
            SpecId::Prague => edr_evm::SpecId::PRAGUE,
            SpecId::Latest => edr_evm::SpecId::LATEST,
        }
    }
}
