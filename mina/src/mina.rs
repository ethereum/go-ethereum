use mina_hasher::{DomainParameter, Hashable, Hasher, ROInput};
use mina_signer::{BaseField, PubKey, Signature, Signer};
use o1_utils::{field_helpers::FieldHelpersError, FieldHelpers};

#[derive(Debug, Clone)]
#[repr(C)]
pub enum HashParameter {
    Mainnet = 0x00,
    Testnet = 0x01,
    Empty = 0x02,
    TransactionCommitment = 0x03,
}

impl From<HashParameter> for u8 {
    fn from(id: HashParameter) -> u8 {
        id as u8
    }
}

impl DomainParameter for HashParameter {
    fn into_bytes(self) -> Vec<u8> {
        vec![self as u8]
    }
}

#[derive(Clone)]
pub struct Message {
    pub fields: Vec<BaseField>,
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
    type D = HashParameter;

    fn to_roinput(&self) -> ROInput {
        self.fields
            .iter()
            .fold(ROInput::new(), |roi, field| roi.append_field(*field))
    }

    fn domain_string(network_id: HashParameter) -> Option<String> {
        match network_id {
            HashParameter::Mainnet => "MinaSignatureMainnet".to_string().into(),
            HashParameter::Testnet => "CodaSignature".to_string().into(),
            HashParameter::TransactionCommitment => "MinaAcctUpdateCons".to_string().into(),
            HashParameter::Empty => None,
        }
    }
}

pub fn poseidon(msg: &Message, network_id: HashParameter) -> BaseField {
    let mut hasher = mina_hasher::create_kimchi::<Message>(network_id);

    hasher.hash(msg)
}

pub fn verify(
    signature: &Signature,
    pubkey: &PubKey,
    msg: &Message,
    network_id: HashParameter,
) -> bool {
    let mut signer = mina_signer::create_kimchi::<Message>(network_id);

    signer.verify(signature, pubkey, msg)
}
