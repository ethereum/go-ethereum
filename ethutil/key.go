package ethutil

type Key struct {
	PrivateKey []byte
	PublicKey  []byte
}

func NewKeyFromBytes(data []byte) *Key {
	val := NewValueFromBytes(data)
	return &Key{val.Get(0).Bytes(), val.Get(1).Bytes()}
}

func (k *Key) Address() []byte {
	return Sha3Bin(k.PublicKey[1:])[12:]
}

func (k *Key) RlpEncode() []byte {
	return EmptyValue().Append(k.PrivateKey).Append(k.PublicKey).Encode()
}
