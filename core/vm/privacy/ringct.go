package privacy

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/XinFinOrg/XDPoSChain/common"
	"math/big"

	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/log"
)

// These constants define the lengths of serialized public keys.
const (
	PubKeyBytesLenCompressed   = 33
	PubKeyBytesLenUncompressed = 65
	PubKeyBytesLenHybrid       = 65
)

const (
	pubkeyCompressed   byte = 0x2 // y_bit + x coord
	pubkeyUncompressed byte = 0x4 // x coord + y coord
	pubkeyHybrid       byte = 0x6 // y_bit + x coord + y coord
)

//The proof contains pretty much stuffs
//The proof contains pretty much stuffs
//Ring size rs: 1 byte => proof[0]
//num input: number of real inputs: 1 byte => proof[1]
//List of inputs/UTXO index typed uint64 => total size = rs * numInput * 8 = proof[0]*proof[1]*8
//List of key images: total size = numInput * 33 = proof[1] * 33
//number of output n: 1 byte
//List of output => n * 130 bytes
//transaction fee: uint256 => 32 byte
//ringCT proof size ctSize: uint16 => 2 byte
//ringCT proof: ctSize bytes
//bulletproofs: bp
type PrivateSendVerifier struct {
	proof []byte
	//ringCT 	RingCT
}

type Ring []*ecdsa.PublicKey

type RingSignature struct {
	NumRing        int
	Size           int                // size of ring
	M              [32]byte           // message
	C              *big.Int           // ring signature value, 1 element
	S              [][]*big.Int       // ring signature values: [NumRing][Size]
	Ring           []Ring             // array of rings of pubkeys: [NumRing]
	I              []*ecdsa.PublicKey // key images, size = the number of rings [NumRing]
	Curve          elliptic.Curve
	SerializedRing []byte //temporary memory stored the raw ring ct used in case of verifying ringCT with message verification
}

func (p *PrivateSendVerifier) verify() bool {
	return false
}

func (p *PrivateSendVerifier) deserialize() {

}

// helper function, returns type of v
func typeof(v interface{}) string {
	return fmt.Sprintf("%T", v)
}

func isOdd(a *big.Int) bool {
	return a.Bit(0) == 1
}

// SerializeCompressed serializes a public key in a 33-byte compressed format.
func SerializeCompressed(p *ecdsa.PublicKey) []byte {
	b := make([]byte, 0, PubKeyBytesLenCompressed)
	format := pubkeyCompressed
	if isOdd(p.Y) {
		format |= 0x1
	}
	b = append(b, format)
	return append(b, PadTo32Bytes(p.X.Bytes())...)
}

func DeserializeCompressed(curve elliptic.Curve, b []byte) *ecdsa.PublicKey {
	x := new(big.Int).SetBytes(b[1:33])
	// Y = +-sqrt(x^3 + B)
	x3 := new(big.Int).Mul(x, x)
	x3.Mul(x3, x)
	x3.Add(x3, curve.Params().B)

	// now calculate sqrt mod p of x2 + B
	// This code used to do a full sqrt based on tonelli/shanks,
	// but this was replaced by the algorithms referenced in
	// https://bitcointalk.org/index.php?topic=162805.msg1712294#msg1712294
	PPlus1Div4 := new(big.Int).Add(curve.Params().P, big.NewInt(1))
	PPlus1Div4 = PPlus1Div4.Div(PPlus1Div4, big.NewInt(4))
	y := new(big.Int).Exp(x3, PPlus1Div4, curve.Params().P)
	ybit := b[0]%2 == 1
	if ybit != isOdd(y) {
		y.Sub(curve.Params().P, y)
	}
	if ybit != isOdd(y) {
		return nil
	}
	return &ecdsa.PublicKey{curve, x, y}
}

// bytes returns the public key ring as a byte slice.
func (r Ring) Bytes() (b []byte) {
	for _, pub := range r {
		b = append(b, PadTo32Bytes(pub.X.Bytes())...)
		b = append(b, PadTo32Bytes(pub.Y.Bytes())...)
	}
	return
}

func PadTo32Bytes(in []byte) (out []byte) {
	out = append(out, in...)
	for {
		if len(out) == 32 {
			return
		}
		out = append([]byte{0}, out...)
	}
}

// converts the signature to a byte array
// this is the format that will be used when passing EVM bytecode
func (r *RingSignature) Serialize() ([]byte, error) {
	sig := []byte{}
	// add size and message
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(r.NumRing))
	sig = append(sig, b[:]...) // 8 bytes

	b = make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(r.Size))
	sig = append(sig, b[:]...) // 8 bytes

	sig = append(sig, PadTo32Bytes(r.M[:])...)      // 32 bytes
	sig = append(sig, PadTo32Bytes(r.C.Bytes())...) // 32 bytes

	for k := 0; k < r.NumRing; k++ {
		// 96 bytes each iteration
		for i := 0; i < r.Size; i++ {
			sig = append(sig, PadTo32Bytes(r.S[k][i].Bytes())...)
		}
	}
	for k := 0; k < r.NumRing; k++ {
		// 96 bytes each iteration
		for i := 0; i < r.Size; i++ {
			rb := SerializeCompressed(r.Ring[k][i])
			sig = append(sig, rb...)
		}
	}

	for k := 0; k < r.NumRing; k++ {
		// 64 bytes
		rb := SerializeCompressed(r.I[k])
		sig = append(sig, rb...)
	}

	if len(sig) != 8+8+32+32+(32+33)*r.NumRing*r.Size+33*r.NumRing {
		return []byte{}, errors.New("Could not serialize ring signature")
	}

	return sig, nil
}

func computeSignatureSize(numRing int, ringSize int) int {
	return 8 + 8 + 32 + 32 + numRing*ringSize*32 + numRing*ringSize*33 + numRing*33
}

// deserializes the byteified signature into a RingSignature struct
func Deserialize(r []byte) (*RingSignature, error) {
	if len(r) < 16 {
		return nil, errors.New("Failed to deserialize ring signature")
	}
	offset := 0
	sig := new(RingSignature)
	numRing := r[offset : offset+8]
	offset += 8
	size := r[offset : offset+8]
	offset += 8

	size_uint := binary.BigEndian.Uint64(size)
	size_int := int(size_uint)
	sig.Size = size_int

	size_uint = binary.BigEndian.Uint64(numRing)
	size_int = int(size_uint)
	sig.NumRing = size_int

	if len(r) != computeSignatureSize(sig.NumRing, sig.Size) {
		return nil, errors.New("incorrect ring size")
	}

	m := r[offset : offset+32]
	offset += 32

	var m_byte [32]byte
	copy(m_byte[:], m)

	sig.M = m_byte
	sig.C = new(big.Int).SetBytes(r[offset : offset+32])
	offset += 32

	sig.S = make([][]*big.Int, sig.NumRing)
	for i := 0; i < sig.NumRing; i++ {
		sig.S[i] = make([]*big.Int, sig.Size)
		for j := 0; j < sig.Size; j++ {
			sig.S[i][j] = new(big.Int).SetBytes(r[offset : offset+32])
			offset += 32
		}
	}

	sig.Curve = crypto.S256()

	sig.Ring = make([]Ring, sig.NumRing)
	for i := 0; i < sig.NumRing; i++ {
		sig.Ring[i] = make([]*ecdsa.PublicKey, sig.Size)
		for j := 0; j < sig.Size; j++ {
			compressedKey := r[offset : offset+33]
			offset += 33
			compressedPubKey := DeserializeCompressed(sig.Curve, compressedKey)
			sig.Ring[i][j] = compressedPubKey
		}
	}

	sig.I = make([]*ecdsa.PublicKey, sig.NumRing)
	for i := 0; i < sig.NumRing; i++ {
		compressedKey := r[offset : offset+33]
		offset += 33
		compressedPubKey := DeserializeCompressed(sig.Curve, compressedKey)
		sig.I[i] = compressedPubKey
	}

	sig.SerializedRing = r

	return sig, nil
}

// takes public key ring and places the public key corresponding to `privkey` in index s of the ring
// returns a key ring of type []*ecdsa.PublicKey
func GenKeyRing(ring []*ecdsa.PublicKey, privkey *ecdsa.PrivateKey, s int) ([]*ecdsa.PublicKey, error) {
	size := len(ring) + 1
	new_ring := make([]*ecdsa.PublicKey, size)
	pubkey := privkey.Public().(*ecdsa.PublicKey)

	if s > len(ring) {
		return nil, errors.New("index s out of bounds")
	}

	new_ring[s] = pubkey
	for i := 1; i < size; i++ {
		idx := (i + s) % size
		new_ring[idx] = ring[i-1]
	}

	return new_ring, nil
}

// creates a ring with size specified by `size` and places the public key corresponding to `privkey` in index s of the ring
// returns a new key ring of type []*ecdsa.PublicKey
func GenNewKeyRing(size int, privkey *ecdsa.PrivateKey, s int) ([]*ecdsa.PublicKey, error) {
	ring := make([]*ecdsa.PublicKey, size)
	pubkey := privkey.Public().(*ecdsa.PublicKey)

	if s > len(ring) {
		return nil, errors.New("index s out of bounds")
	}

	ring[s] = pubkey

	for i := 1; i < size; i++ {
		idx := (i + s) % size
		priv, err := crypto.GenerateKey()
		if err != nil {
			return nil, err
		}

		pub := priv.Public()
		ring[idx] = pub.(*ecdsa.PublicKey)
	}

	return ring, nil
}

// calculate key image I = x * H_p(P) where H_p is a hash function that returns a point
// H_p(P) = sha3(P) * G
func GenKeyImage(privkey *ecdsa.PrivateKey) *ecdsa.PublicKey {
	pubkey := privkey.Public().(*ecdsa.PublicKey)
	image := new(ecdsa.PublicKey)

	// calculate sha3(P)
	h_x, h_y := HashPoint(pubkey)

	// calculate H_p(P) = x * sha3(P) * G
	i_x, i_y := privkey.Curve.ScalarMult(h_x, h_y, privkey.D.Bytes())

	image.X = i_x
	image.Y = i_y
	return image
}

func HashPoint(p *ecdsa.PublicKey) (*big.Int, *big.Int) {
	input := append(PadTo32Bytes(p.X.Bytes()), PadTo32Bytes(p.Y.Bytes())...)
	log.Info("HashPoint", "input ", common.Bytes2Hex(input))
	hash := crypto.Keccak256(input)
	log.Info("HashPoint", "hash ", common.Bytes2Hex(hash))
	return p.Curve.ScalarBaseMult(hash[:])
}

// create ring signature from list of public keys given inputs:
// msg: byte array, message to be signed
// ring: array of *ecdsa.PublicKeys to be included in the ring
// privkey: *ecdsa.PrivateKey of signer
// s: index of signer in ring
func Sign(m [32]byte, rings []Ring, privkeys []*ecdsa.PrivateKey, s int) (*RingSignature, error) {
	numRing := len(rings)
	if numRing < 1 {
		return nil, errors.New("there is no ring to make signature")
	}
	// check ringsize > 1
	ringsize := len(rings[0])
	if ringsize < 2 {
		return nil, errors.New("size of ring less than two")
	} else if s >= ringsize || s < 0 {
		return nil, errors.New("secret index out of range of ring size")
	}

	// setup
	//pubkey := privkey.Public().(*ecdsa.PublicKey)
	pubkeys := make([]*ecdsa.PublicKey, numRing)
	for i := 0; i < numRing; i++ {
		pubkeys[i] = &privkeys[i].PublicKey
	}
	curve := pubkeys[0].Curve
	sig := new(RingSignature)
	sig.Size = ringsize
	sig.NumRing = numRing
	sig.M = m
	sig.Ring = rings
	sig.Curve = curve

	// check that key at index s is indeed the signer
	for i := 0; i < numRing; i++ {
		if rings[i][s] != pubkeys[i] {
			return nil, errors.New("secret index in ring is not signer")
		}
	}

	// generate key image
	images := make([]*ecdsa.PublicKey, numRing)
	for i := 0; i < numRing; i++ {
		images[i] = GenKeyImage(privkeys[i])
	}
	sig.I = images

	// start at c[1]
	// pick random scalar u (glue value), calculate c[1] = H(m, u*G) where H is a hash function and G is the base point of the curve
	C := make([]*big.Int, ringsize)
	S := make([][]*big.Int, numRing)
	for i := 0; i < numRing; i++ {
		S[i] = make([]*big.Int, ringsize)
	}

	//Initialize S except S[..][s]
	for i := 0; i < numRing; i++ {
		for j := 0; j < ringsize; j++ {
			if j != s {
				randomGenerated, err := rand.Int(rand.Reader, curve.Params().P)
				if err != nil {
					return nil, err
				}
				S[i][j] = randomGenerated
			}
		}
	}

	L := make([][]*ecdsa.PublicKey, numRing)
	R := make([][]*ecdsa.PublicKey, numRing)
	for i := 0; i < numRing; i++ {
		L[i] = make([]*ecdsa.PublicKey, ringsize)
		R[i] = make([]*ecdsa.PublicKey, ringsize)
	}
	alpha := make([]*big.Int, numRing)

	var l []byte
	//compute L[i][s], R[i][s], i = 0..numRing
	for i := 0; i < numRing; i++ {
		randomGenerated, err := rand.Int(rand.Reader, curve.Params().P)
		if err != nil {
			return nil, err
		}
		alpha[i] = randomGenerated
		// start at secret index s/PI
		// compute L_s = u*G
		l_x, l_y := curve.ScalarBaseMult(PadTo32Bytes(alpha[i].Bytes()))
		L[i][s] = &ecdsa.PublicKey{curve, l_x, l_y}
		lT := append(PadTo32Bytes(l_x.Bytes()), PadTo32Bytes(l_y.Bytes())...)
		l = append(l, lT...)
		// compute R_s = u*H_p(P[s])
		h_x, h_y := HashPoint(pubkeys[i])
		r_x, r_y := curve.ScalarMult(h_x, h_y, PadTo32Bytes(alpha[i].Bytes()))
		R[i][s] = &ecdsa.PublicKey{curve, r_x, r_y}
		rT := append(PadTo32Bytes(r_x.Bytes()), PadTo32Bytes(r_y.Bytes())...)
		l = append(l, rT...)
	}

	// concatenate m and u*G and calculate c[s+1] = H(m, L_s, R_s)
	C_j := crypto.Keccak256(append(m[:], l...))
	idx := s + 1
	if idx == ringsize {
		idx = 0
	}
	if idx == 0 {
		C[0] = new(big.Int).SetBytes(C_j[:])
	} else {
		C[idx] = new(big.Int).SetBytes(C_j[:])
	}
	for idx != s {
		var l []byte
		for j := 0; j < numRing; j++ {
			// calculate L[j][idx] = s[j][idx]*G + c[idx]*Ring[j][idx]
			px, py := curve.ScalarMult(rings[j][idx].X, rings[j][idx].Y, PadTo32Bytes(C[idx].Bytes())) // px, py = c_i*P_i
			sx, sy := curve.ScalarBaseMult(PadTo32Bytes(S[j][idx].Bytes()))                            // sx, sy = s[n-1]*G
			l_x, l_y := curve.Add(sx, sy, px, py)
			L[j][idx] = &ecdsa.PublicKey{curve, l_x, l_y}
			lT := append(PadTo32Bytes(l_x.Bytes()), PadTo32Bytes(l_y.Bytes())...)
			l = append(l, lT...)

			// calculate R[j][idx] = s[j][idx]*H_p(Ring[j][idx]) + c[idx]*I[j]
			px, py = curve.ScalarMult(images[j].X, images[j].Y, C[idx].Bytes()) // px, py = c_i*I
			hx, hy := HashPoint(rings[j][idx])
			sx, sy = curve.ScalarMult(hx, hy, S[j][idx].Bytes()) // sx, sy = s[n-1]*H_p(P_i)
			r_x, r_y := curve.Add(sx, sy, px, py)
			R[j][idx] = &ecdsa.PublicKey{curve, r_x, r_y}
			rT := append(PadTo32Bytes(r_x.Bytes()), PadTo32Bytes(r_y.Bytes())...)
			l = append(l, rT...)
		}

		idx++
		if idx == ringsize {
			idx = 0
		}

		var ciIdx int
		if idx == 0 {
			ciIdx = 0
		} else {
			ciIdx = idx
		}
		cSha := crypto.Keccak256(append(PadTo32Bytes(m[:]), l...))
		C[ciIdx] = new(big.Int).SetBytes(cSha[:])
	}

	//compute S[j][s] = alpha[j] - c[s] * privkeys[j], privkeys[j] = private key corresponding to key image I[j]
	for j := 0; j < numRing; j++ {
		cx := C[s]
		// close ring by finding S[j][s] = (alpha[j] - c[s]*privkeys[s] ) mod P where k[s] is the private key and P is the order of the curve
		S[j][s] = new(big.Int).Mod(new(big.Int).Sub(alpha[j], new(big.Int).Mul(cx, privkeys[j].D)), curve.Params().N)
	}

	// everything ok, add values to signature
	sig.S = S
	sig.C = C[0]
	sig.NumRing = numRing
	sig.Size = ringsize
	sig.C = C[0]

	return sig, nil
}

// verify ring signature contained in RingSignature struct
// returns true if a valid signature, false otherwise
func Verify(sig *RingSignature, verifyMes bool) bool {
	// setup
	rings := sig.Ring
	ringsize := sig.Size
	numRing := sig.NumRing
	S := sig.S
	C := make([]*big.Int, ringsize+1)
	C[0] = sig.C
	curve := sig.Curve
	image := sig.I

	// calculate c[i+1] = H(m, s[i]*G + c[i]*P[i])
	// and c[0] = H)(m, s[n-1]*G + c[n-1]*P[n-1]) where n is the ring size
	//log.Info("C", "0", common.Bytes2Hex(C[0].Bytes()))
	for j := 0; j < ringsize; j++ {
		var l []byte
		for i := 0; i < numRing; i++ {
			// calculate L[i][j] = s[i][j]*G + c[j]*Ring[i][j]
			px, py := curve.ScalarMult(rings[i][j].X, rings[i][j].Y, C[j].Bytes()) // px, py = c_i*P_i
			sx, sy := curve.ScalarBaseMult(S[i][j].Bytes())                        // sx, sy = s[i]*G
			l_x, l_y := curve.Add(sx, sy, px, py)
			lT := append(PadTo32Bytes(l_x.Bytes()), PadTo32Bytes(l_y.Bytes())...)
			//log.Info("L[i][j]", "i", i, "j", j, "L", common.Bytes2Hex(lT))
			l = append(l, lT...)

			// calculate R_i = s[i][j]*H_p(Ring[i][j]) + c[j]*I[j]
			px, py = curve.ScalarMult(image[i].X, image[i].Y, C[j].Bytes()) // px, py = c[i]*I
			hx, hy := HashPoint(rings[i][j])
			//log.Info("H[i][j]", "i", i, "j", j, "x.input", common.Bytes2Hex(rings[i][j].X.Bytes()), "y.input", common.Bytes2Hex(rings[i][j].Y.Bytes()))
			//log.Info("H[i][j]", "i", i, "j", j, "x", common.Bytes2Hex(hx.Bytes()), "y", common.Bytes2Hex(hy.Bytes()))
			sx, sy = curve.ScalarMult(hx, hy, S[i][j].Bytes()) // sx, sy = s[i]*H_p(P[i])
			r_x, r_y := curve.Add(sx, sy, px, py)
			rT := append(PadTo32Bytes(r_x.Bytes()), PadTo32Bytes(r_y.Bytes())...)
			//log.Info("R[i][j]", "i", i, "j", j, "L", common.Bytes2Hex(rT))
			l = append(l, rT...)
		}

		// calculate c[i+1] = H(m, L_i, R_i)
		//cj_mes := append(PadTo32Bytes(sig.M[:]), l...)
		C_j := crypto.Keccak256(append(PadTo32Bytes(sig.M[:]), l...))
		//log.Info("C hash input", "j", j + 1, "C_input", common.Bytes2Hex(cj_mes))

		/*if j == ringsize-1 {
			C[0] = new(big.Int).SetBytes(C_j[:])
		} else {*/
		C[j+1] = new(big.Int).SetBytes(C_j[:])
		//log.Info("C", "j", j + 1, "C", common.Bytes2Hex(C[j + 1].Bytes()))
		//}
	}

	return bytes.Equal(sig.C.Bytes(), C[ringsize].Bytes())
}

func Link(sig_a *RingSignature, sig_b *RingSignature) bool {
	for i := 0; i < len(sig_a.I); i++ {
		for j := 0; j < len(sig_b.I); j++ {
			if sig_a.I[i].X == sig_b.I[j].X && sig_a.I[i].Y == sig_b.I[j].Y {
				return true
			}
		}
	}
	return false
}

//function returns(mutiple rings, private keys, message, error)
func GenerateMultiRingParams(numRing int, ringSize int, s int) (rings []Ring, privkeys []*ecdsa.PrivateKey, m [32]byte, err error) {
	for i := 0; i < numRing; i++ {
		privkey, err := crypto.GenerateKey()
		if err != nil {
			return nil, nil, [32]byte{}, err
		}
		privkeys = append(privkeys, privkey)

		ring, err := GenNewKeyRing(ringSize, privkey, s)
		if err != nil {
			return nil, nil, [32]byte{}, err
		}
		rings = append(rings, ring)
	}

	_, err = rand.Read(m[:])
	if err != nil {
		return nil, nil, [32]byte{}, err
	}
	return rings, privkeys, m, nil
}

func TestRingSignature() (bool, []byte) {
	/*for i := 14; i < 15; i++ {
	for j := 14; j < 15; j++ {
		for k := 0; k <= j; k++ {*/
	numRing := 1
	ringSize := 10
	s := 9
	rings, privkeys, m, err := GenerateMultiRingParams(numRing, ringSize, s)
	ringSignature, err := Sign(m, rings, privkeys, s)
	if err != nil {
		log.Error("Failed to create Ring signature")
		return false, []byte{}
	}

	sig, err := ringSignature.Serialize()
	if err != nil {
		return false, []byte{}
	}

	deserializedSig, err := Deserialize(sig)
	if err != nil {
		return false, []byte{}
	}
	verified := Verify(deserializedSig, false)
	if !verified {
		log.Error("Failed to verify Ring signature")
		return false, []byte{}
	}

	return true, []byte{}
}
