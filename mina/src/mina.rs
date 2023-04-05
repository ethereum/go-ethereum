use mina_hasher::{DomainParameter, Hashable, Hasher, ROInput};
use mina_signer::{BaseField, PubKey, Signer, Signature};
use o1_utils::{field_helpers::FieldHelpersError, FieldHelpers};

#[derive(Debug, Clone)]
#[repr(C)]
pub enum NetworkId {
    TESTNET = 0x00,
    MAINNET = 0x01,
    NULLNET = 0xff,
}

impl From<NetworkId> for u8 {
    fn from(id: NetworkId) -> u8 {
        id as u8
    }
}

impl DomainParameter for NetworkId {
    fn into_bytes(self) -> Vec<u8> {
        vec![self as u8]
    }
}

#[derive(Clone)]
pub struct Message {
    fields: Vec<BaseField>,
}

impl Message {
    pub fn from_bytes_slice(fields_bytes: &[[u8; 32]]) -> Result<Self, FieldHelpersError> {
        Ok(Self {
            fields: fields_bytes
                .iter()
                .map(|bytes| BaseField::from_bytes(bytes))
                .collect::<Result<Vec<BaseField>, FieldHelpersError>>()?,
        })
    }
}

impl Hashable for Message {
    type D = NetworkId;

    fn to_roinput(&self) -> ROInput {
        self.fields
            .iter()
            .fold(ROInput::new(), |roi, field| roi.append_field(*field))
    }

    fn domain_string(network_id: NetworkId) -> Option<String> {
        match network_id {
            NetworkId::MAINNET => "MinaSignatureMainnet".to_string().into(),
            NetworkId::TESTNET => "CodaSignature".to_string().into(),
            NetworkId::NULLNET => None,
        }
    }
}

pub fn poseidon(msg: &Message, network_id: NetworkId) -> BaseField {
    let mut hasher = mina_hasher::create_kimchi::<Message>(network_id);

    hasher.hash(msg)
}
