package p2p

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/obscuren/ecies"
	"github.com/obscuren/secp256k1-go"
)

var (
	skLen     int = 32                             // ecies.MaxSharedKeyLength(pubKey) / 2
	sigLen    int = 32                             // elliptic S256
	pubKeyLen int = 32                             // ECDSA
	msgLen    int = sigLen + 1 + pubKeyLen + skLen // 97
)

//, aesSecret, macSecret, egressMac, ingress
type secretRW struct {
	aesSecret, macSecret, egressMac, ingressMac []byte
}

type cryptoId struct {
	prvKey  *ecdsa.PrivateKey
	pubKey  *ecdsa.PublicKey
	pubKeyR io.ReaderAt
}

func newCryptoId(id ClientIdentity) (self *cryptoId, err error) {
	// will be at server  init
	var prvKeyDER []byte = id.PrivKey()
	if prvKeyDER == nil {
		err = fmt.Errorf("no private key for client")
		return
	}
	// initialise ecies private key via importing DER encoded keys (known via our own clientIdentity)
	var prvKey = crypto.ToECDSA(prvKeyDER)
	if prvKey == nil {
		err = fmt.Errorf("invalid private key for client")
		return
	}
	self = &cryptoId{
		prvKey: prvKey,
		// initialise public key from the imported private key
		pubKey: &prvKey.PublicKey,
		// to be created at server init shared between peers and sessions
		// for reuse, call wth ReadAt, no reset seek needed
	}
	self.pubKeyR = bytes.NewReader(id.Pubkey())
	return
}

//
func (self *cryptoId) setupAuth(remotePubKeyDER, sessionToken []byte) (auth []byte, nonce []byte, sharedKnowledge []byte, err error) {
	// session init, common to both parties
	var remotePubKey = crypto.ToECDSAPub(remotePubKeyDER)
	if remotePubKey == nil {
		err = fmt.Errorf("invalid remote public key")
		return
	}
	var sharedSecret []byte
	// generate shared key from prv and remote pubkey
	sharedSecret, err = ecies.ImportECDSA(self.prvKey).GenerateShared(ecies.ImportECDSAPublic(remotePubKey), skLen, skLen)
	if err != nil {
		return
	}
	// check previous session token
	if sessionToken == nil {
		err = fmt.Errorf("no session token for peer")
		return
	}
	// allocate msgLen long message
	var msg []byte = make([]byte, msgLen)
	// generate skLen long nonce at the end
	nonce = msg[msgLen-skLen:]
	if _, err = rand.Read(nonce); err != nil {
		return
	}
	// create known message
	// should use
	// cipher.xorBytes from crypto/cipher/xor.go for fast xor
	sharedKnowledge = Xor(sharedSecret, sessionToken)
	var signedMsg = Xor(sharedKnowledge, nonce)

	// generate random keypair to use for signing
	var ecdsaRandomPrvKey *ecdsa.PrivateKey
	if ecdsaRandomPrvKey, err = crypto.GenerateKey(); err != nil {
		return
	}
	// var ecdsaRandomPubKey *ecdsa.PublicKey
	//  ecdsaRandomPubKey= &ecdsaRandomPrvKey.PublicKey

	// message known to both parties ecdh-shared-secret^nonce^token
	var signature []byte
	// signature = sign(ecdhe-random, ecdh-shared-secret^nonce^token)
	// uses secp256k1.Sign
	if signature, err = crypto.Sign(signedMsg, ecdsaRandomPrvKey); err != nil {
		return
	}
	// msg = signature || 0x80 || pubk || nonce
	copy(msg, signature)
	msg[sigLen] = 0x80
	self.pubKeyR.ReadAt(msg[sigLen+1:], int64(pubKeyLen)) // gives pubKeyLen, io.EOF (since we dont read onto the nonce)

	// auth = eciesEncrypt(remote-pubk, msg)
	if auth, err = crypto.Encrypt(remotePubKey, msg); err != nil {
		return
	}
	return
}

func (self *cryptoId) verifyAuth(auth, nonce, sharedKnowledge []byte) (sessionToken []byte, rw *secretRW, err error) {
	var msg []byte
	// they prove that msg is meant for me,
	// I prove I possess private key if i can read it
	if msg, err = crypto.Decrypt(self.prvKey, auth); err != nil {
		return
	}

	var remoteNonce []byte = msg[msgLen-skLen:]
	// I prove that i possess prv key (to derive shared secret, and read nonce off encrypted msg) and that I posessed the earlier one , our shared history
	// they prove they possess their private key to derive the same shared secret, plus the same shared history (previous session token)
	var signedMsg = Xor(sharedKnowledge, remoteNonce)
	var remoteRandomPubKeyDER []byte
	if remoteRandomPubKeyDER, err = secp256k1.RecoverPubkey(signedMsg, msg[:32]); err != nil {
		return
	}
	var remoteRandomPubKey = crypto.ToECDSAPub(remoteRandomPubKeyDER)
	if remoteRandomPubKey == nil {
		err = fmt.Errorf("invalid remote public key")
		return
	}
	// 3) Now we can trust ecdhe-random-pubk to derive keys
	//ecdhe-shared-secret = ecdh.agree(ecdhe-random, remote-ecdhe-random-pubk)
	var dhSharedSecret []byte
	dhSharedSecret, err = ecies.ImportECDSA(self.prvKey).GenerateShared(ecies.ImportECDSAPublic(remoteRandomPubKey), skLen, skLen)
	if err != nil {
		return
	}
	// shared-secret = crypto.Sha3(ecdhe-shared-secret || crypto.Sha3(nonce || initiator-nonce))
	var sharedSecret []byte = crypto.Sha3(append(dhSharedSecret, crypto.Sha3(append(nonce, remoteNonce...))...))
	// token = crypto.Sha3(shared-secret)
	sessionToken = crypto.Sha3(sharedSecret)
	// aes-secret = crypto.Sha3(ecdhe-shared-secret || shared-secret)
	var aesSecret = crypto.Sha3(append(dhSharedSecret, sharedSecret...))
	// # destroy shared-secret
	// mac-secret = crypto.Sha3(ecdhe-shared-secret || aes-secret)
	var macSecret = crypto.Sha3(append(dhSharedSecret, aesSecret...))
	// # destroy ecdhe-shared-secret
	// egress-mac = crypto.Sha3(mac-secret^nonce || auth)
	var egressMac = crypto.Sha3(append(Xor(macSecret, nonce), auth...))
	// # destroy nonce
	// ingress-mac = crypto.Sha3(mac-secret^initiator-nonce || auth),
	var ingressMac = crypto.Sha3(append(Xor(macSecret, remoteNonce), auth...))
	// # destroy remote-nonce
	rw = &secretRW{
		aesSecret:  aesSecret,
		macSecret:  macSecret,
		egressMac:  egressMac,
		ingressMac: ingressMac,
	}
	return
}

func Xor(one, other []byte) (xor []byte) {
	for i := 0; i < len(one); i++ {
		xor[i] = one[i] ^ other[i]
	}
	return
}
