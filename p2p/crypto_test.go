package p2p

import (
	// "bytes"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestCryptoHandshake(t *testing.T) {
	var err error
	var sessionToken []byte
	prvInit, _ := crypto.GenerateKey()
	pubInit := &prvInit.PublicKey
	prvResp, _ := crypto.GenerateKey()
	pubResp := &prvResp.PublicKey

	var initiator, responder *cryptoId
	if initiator, err = newCryptoId(&peerId{crypto.FromECDSA(prvInit), crypto.FromECDSAPub(pubInit)}); err != nil {
		return
	}
	if responder, err = newCryptoId(&peerId{crypto.FromECDSA(prvResp), crypto.FromECDSAPub(pubResp)}); err != nil {
		return
	}

	auth, initNonce, _, _ := initiator.initAuth(responder.pubKeyDER, sessionToken)

	response, remoteRespNonce, remoteInitNonce, remoteRandomPubKey, _ := responder.verifyAuth(auth, sessionToken, pubInit)

	respNonce, randomPubKey, _, _ := initiator.verifyAuthResp(response)

	fmt.Printf("%x\n%x\n%x\n%x\n%x\n%x\n%x\n%x\n", auth, initNonce, response, remoteRespNonce, remoteInitNonce, remoteRandomPubKey, respNonce, randomPubKey)
	initSessionToken, initSecretRW, _ := initiator.newSession(initNonce, respNonce, auth, randomPubKey)
	// respSessionToken, respSecretRW, _ := responder.newSession(remoteInitNonce, remoteRespNonce, auth, remoteRandomPubKey)

	// if !bytes.Equal(initSessionToken, respSessionToken) {
	// 	t.Errorf("session tokens do not match")
	// }
	// // aesSecret, macSecret, egressMac, ingressMac
	// if !bytes.Equal(initSecretRW.aesSecret, respSecretRW.aesSecret) {
	// 	t.Errorf("AES secrets do not match")
	// }
	// if !bytes.Equal(initSecretRW.macSecret, respSecretRW.macSecret) {
	// 	t.Errorf("macSecrets do not match")
	// }
	// if !bytes.Equal(initSecretRW.egressMac, respSecretRW.egressMac) {
	// 	t.Errorf("egressMacs do not match")
	// }
	// if !bytes.Equal(initSecretRW.ingressMac, respSecretRW.ingressMac) {
	// 	t.Errorf("ingressMacs do not match")
	// }

}
