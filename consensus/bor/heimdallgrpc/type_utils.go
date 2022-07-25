package heimdallgrpc

import (
	"encoding/binary"

	proto "github.com/maticnetwork/polyproto/heimdall"
)

func ConvertH160toAddress(h160 *proto.H160) [20]byte {
	var addr [20]byte

	binary.BigEndian.PutUint64(addr[0:], h160.Hi.Hi)
	binary.BigEndian.PutUint64(addr[8:], h160.Hi.Lo)
	binary.BigEndian.PutUint32(addr[16:], h160.Lo)

	return addr
}

func ConvertAddressToH160(addr [20]byte) *proto.H160 {
	return &proto.H160{
		Lo: binary.BigEndian.Uint32(addr[16:]),
		Hi: &proto.H128{Lo: binary.BigEndian.Uint64(addr[8:]), Hi: binary.BigEndian.Uint64(addr[0:])},
	}
}

func ConvertH256ToHash(h256 *proto.H256) [32]byte {
	var hash [32]byte

	binary.BigEndian.PutUint64(hash[0:], h256.Hi.Hi)
	binary.BigEndian.PutUint64(hash[8:], h256.Hi.Lo)
	binary.BigEndian.PutUint64(hash[16:], h256.Lo.Hi)
	binary.BigEndian.PutUint64(hash[24:], h256.Lo.Lo)

	return hash
}

func ConvertHashToH256(hash [32]byte) *proto.H256 {
	return &proto.H256{
		Lo: &proto.H128{Lo: binary.BigEndian.Uint64(hash[24:]), Hi: binary.BigEndian.Uint64(hash[16:])},
		Hi: &proto.H128{Lo: binary.BigEndian.Uint64(hash[8:]), Hi: binary.BigEndian.Uint64(hash[0:])},
	}
}
