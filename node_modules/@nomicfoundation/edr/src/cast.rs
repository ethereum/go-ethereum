use edr_eth::{Address, Bytes, B256, B64, U256};
use napi::{
    bindgen_prelude::{BigInt, Buffer},
    Status,
};

/// An attempted conversion that consumes `self`, which may or may not be
/// expensive. It is identical to [`TryInto`], but it allows us to implement
/// the trait for external types.
pub trait TryCast<T>: Sized {
    /// The type returned in the event of a conversion error.
    type Error;

    /// Performs the conversion.
    fn try_cast(self) -> Result<T, Self::Error>;
}

impl TryCast<Address> for Buffer {
    type Error = napi::Error;

    fn try_cast(self) -> std::result::Result<Address, Self::Error> {
        if self.len() != 20 {
            return Err(napi::Error::new(
                Status::InvalidArg,
                "Buffer was expected to be 20 bytes.".to_string(),
            ));
        }
        Ok(Address::from_slice(&self))
    }
}

impl TryCast<B64> for Buffer {
    type Error = napi::Error;

    fn try_cast(self) -> std::result::Result<B64, Self::Error> {
        if self.len() != 8 {
            return Err(napi::Error::new(
                Status::InvalidArg,
                "Buffer was expected to be 8 bytes.".to_string(),
            ));
        }
        Ok(B64::from_slice(&self))
    }
}

impl TryCast<B256> for Buffer {
    type Error = napi::Error;

    fn try_cast(self) -> std::result::Result<B256, Self::Error> {
        if self.len() != 32 {
            return Err(napi::Error::new(
                Status::InvalidArg,
                "Buffer was expected to be 32 bytes.".to_string(),
            ));
        }
        Ok(B256::from_slice(&self))
    }
}

impl TryCast<u64> for BigInt {
    type Error = napi::Error;

    fn try_cast(self) -> std::result::Result<u64, Self::Error> {
        let (signed, value, lossless) = self.get_u64();

        if signed {
            return Err(napi::Error::new(
                Status::InvalidArg,
                "BigInt was expected to be unsigned.".to_string(),
            ));
        }

        if !lossless {
            return Err(napi::Error::new(
                Status::InvalidArg,
                "BigInt was expected to fit within 64 bits.".to_string(),
            ));
        }

        Ok(value)
    }
}

impl TryCast<usize> for BigInt {
    type Error = napi::Error;

    fn try_cast(self) -> std::result::Result<usize, Self::Error> {
        let size: u64 = BigInt::try_cast(self)?;
        usize::try_from(size).map_err(|e| napi::Error::new(Status::InvalidArg, e.to_string()))
    }
}

impl TryCast<U256> for BigInt {
    type Error = napi::Error;

    fn try_cast(mut self) -> std::result::Result<U256, Self::Error> {
        let num_words = self.words.len();
        match num_words.cmp(&4) {
            std::cmp::Ordering::Less => self.words.append(&mut vec![0u64; 4 - num_words]),
            std::cmp::Ordering::Equal => (),
            std::cmp::Ordering::Greater => {
                return Err(napi::Error::new(
                    Status::InvalidArg,
                    "BigInt cannot have more than 4 words.".to_owned(),
                ));
            }
        }

        Ok(U256::from_limbs(self.words.try_into().unwrap()))
    }
}

impl<T> TryCast<T> for T {
    type Error = napi::Error;

    fn try_cast(self) -> std::result::Result<T, Self::Error> {
        Ok(self)
    }
}

impl TryCast<Bytes> for Buffer {
    type Error = napi::Error;

    fn try_cast(self) -> Result<Bytes, Self::Error> {
        Ok(Bytes::copy_from_slice(&self))
    }
}

impl TryCast<Option<Bytes>> for Option<Buffer> {
    type Error = napi::Error;

    fn try_cast(self) -> Result<Option<Bytes>, Self::Error> {
        Ok(self.map(|buffer| Bytes::copy_from_slice(&buffer)))
    }
}
