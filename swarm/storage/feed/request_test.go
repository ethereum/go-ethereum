// Copyright 2018 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package feed

import (
	"encoding/binary"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/ethereum/go-ethereum/swarm/storage/feed/lookup"
	"github.com/ethereum/go-ethereum/swarm/testutil"
)

// TestEncodingDecodingUpdateRequests ensures that requests are serialized properly
// while also checking cryptographically that only the owner of a feed can update it.
func TestEncodingDecodingUpdateRequests(tx *testing.T) {
	t := testutil.BeginTest(tx, false) // set to true to generate results
	defer t.FinishTest()

	TimestampProvider = &fakeTimeProvider{
		currentTime: startTime.Time, // clock starts at t=4200
	}

	charlie := newCharlieSigner() //Charlie
	bob := newBobSigner()         //Bob

	// Create a feed to our good guy Charlie's name
	topic, _ := NewTopic("a good topic name", nil)
	firstRequest := NewFirstRequest(topic)
	firstRequest.User = charlie.Address()

	t.TestJSONMarshaller("firstrequest.json", firstRequest)

	// but verification should fail because it is not signed!
	t.MustFail(firstRequest.Verify(), "Expected Verify to fail since the message is not signed")

	// We now assume that the feed update was created and propagated.
	//Put together an unsigned update request that we will serialize to send it to the signer.
	data := []byte("This hour's update: Swarm 99.0 has been released!")
	request := &Request{
		Update: Update{
			ID: ID{
				Epoch: lookup.Epoch{
					Time:  1000,
					Level: 1,
				},
				Feed: firstRequest.Update.Feed,
			},
			data: data,
		},
	}

	messageRawData, err := request.MarshalJSON()
	t.Ok(err)
	t.JSONBytesEqualsFile("secondrequest.json", messageRawData)

	// now the encoded message messageRawData is sent over the wire and arrives to the signer

	//Attempt to extract an UpdateRequest out of the encoded message
	var recoveredRequest Request
	err = recoveredRequest.UnmarshalJSON(messageRawData)
	t.Ok(err)

	//sign the request and see if it matches our predefined signature above.
	err = recoveredRequest.Sign(charlie)
	t.Ok(err)
	t.EqualsKey("signature", hexutil.Encode(recoveredRequest.Signature[:]))

	// mess with the signature and see what happens. To alter the signature, we briefly decode it as JSON
	// to alter the signature field.
	var j updateRequestJSON
	err = json.Unmarshal([]byte(messageRawData), &j)
	t.Ok(err)

	j.Signature = "Certainly not a signature"
	corruptMessage, _ := json.Marshal(j) // encode the message with the bad signature
	var corruptRequest Request
	err = corruptRequest.UnmarshalJSON(corruptMessage)
	t.MustFail(err, "Expected DecodeUpdateRequest to fail when trying to interpret a corrupt message with an invalid signature")

	// Now imagine Bob wants to create an update of his own about the same feed,
	// signing a message with his private key
	err = request.Sign(bob)
	t.Ok(err)

	// Now Bob encodes the message to send it over the wire...
	messageRawData, err = request.MarshalJSON()
	t.Ok(err)

	// ... the message arrives to our Swarm node and it is decoded.
	recoveredRequest = Request{}
	err = recoveredRequest.UnmarshalJSON(messageRawData)
	t.Ok(err)

	// Before checking what happened with Bob's update, let's see what would happen if we mess
	// with the signature big time to see if Verify catches it
	savedSignature := *recoveredRequest.Signature                               // save the signature for later
	binary.LittleEndian.PutUint64(recoveredRequest.Signature[5:], 556845463424) // write some random data to break the signature
	err = recoveredRequest.Verify()
	t.MustFail(err, "Expected Verify to fail on corrupt signature")

	// restore the Bob's signature from corruption
	*recoveredRequest.Signature = savedSignature

	// Now the signature is not corrupt
	err = recoveredRequest.Verify()
	t.Ok(err)

	// Reuse object and sign with our friend Charlie's private key
	err = recoveredRequest.Sign(charlie)
	t.Ok(err)

	// And now, Verify should work since this update now belongs to Charlie
	err = recoveredRequest.Verify()
	t.Ok(err)

	// mess with the lookup key to make sure Verify fails:
	recoveredRequest.Time = 77999 // this will alter the lookup key
	err = recoveredRequest.Verify()
	t.MustFail(err, "Expected Verify to fail since the lookup key has been altered")
}

func getTestRequest() *Request {
	return &Request{
		Update: *getTestFeedUpdate(),
	}
}

func TestUpdateChunkSerializationErrorChecking(t *testing.T) {

	// Test that parseUpdate fails if the chunk is too small
	var r Request
	if err := r.fromChunk(storage.NewChunk(storage.ZeroAddr, make([]byte, minimumUpdateDataLength-1+signatureLength))); err == nil {
		t.Fatalf("Expected request.fromChunk to fail when chunkData contains less than %d bytes", minimumUpdateDataLength)
	}

	r = *getTestRequest()

	_, err := r.toChunk()
	if err == nil {
		t.Fatal("Expected request.toChunk to fail when there is no data")
	}
	r.data = []byte("Al bien hacer jamás le falta premio") // put some arbitrary length data
	_, err = r.toChunk()
	if err == nil {
		t.Fatal("expected request.toChunk to fail when there is no signature")
	}

	charlie := newCharlieSigner()
	if err := r.Sign(charlie); err != nil {
		t.Fatalf("error signing:%s", err)
	}

	chunk, err := r.toChunk()
	if err != nil {
		t.Fatalf("error creating update chunk:%s", err)
	}

	compareByteSliceToExpectedHex(t, "chunk", chunk.Data(), "0x0000000000000000776f726c64206e657773207265706f72742c20657665727920686f7572000000876a8936a7cd0b79ef0735ad0896c1afe278781ce803000000000019416c206269656e206861636572206a616dc3a173206c652066616c7461207072656d696f5a0ffe0bc27f207cd5b00944c8b9cee93e08b89b5ada777f123ac535189333f174a6a4ca2f43a92c4a477a49d774813c36ce8288552c58e6205b0ac35d0507eb00")

	var recovered Request
	recovered.fromChunk(chunk)
	if !reflect.DeepEqual(recovered, r) {
		t.Fatal("Expected recovered feed update request to equal the original one")
	}
}

// check that signature address matches update signer address
func TestReverse(tx *testing.T) {
	t := testutil.BeginTest(tx, false) // set to true to generate results
	defer t.FinishTest()

	epoch := lookup.Epoch{
		Time:  7888,
		Level: 6,
	}

	// signer containing private key
	signer := newAliceSigner()

	topic, _ := NewTopic("Cervantes quotes", nil)
	fd := Feed{
		Topic: topic,
		User:  signer.Address(),
	}

	data := []byte("Donde una puerta se cierra, otra se abre")

	request := new(Request)
	request.Feed = fd
	request.Epoch = epoch
	request.data = data

	// generate a chunk key for this request
	key := request.Addr()

	err := request.Sign(signer)
	t.Ok(err)

	chunk, err := request.toChunk()
	t.Ok(err)

	// check that we can recover the owner account from the update chunk's signature
	var checkUpdate Request
	err = checkUpdate.fromChunk(chunk)
	t.Ok(err)

	checkdigest, err := checkUpdate.GetDigest()
	t.Ok(err)

	recoveredAddr, err := getUserAddr(checkdigest, *checkUpdate.Signature) //Retrieve address from signature
	t.Ok(err)

	originalAddr := crypto.PubkeyToAddress(signer.PrivKey.PublicKey)

	// check that the metadata retrieved from the chunk matches what we gave it
	t.Assert(recoveredAddr == originalAddr, "addresses dont match: %x != %x", originalAddr, recoveredAddr)
	t.Equals(chunk.Address()[:], key[:])

	t.Assert(epoch == checkUpdate.Epoch, "Expected epoch to be '%s', was '%s'", epoch.String(), checkUpdate.Epoch.String())

	t.Equals(checkUpdate.data, data)
}
