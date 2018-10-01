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

package mru

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/ethereum/go-ethereum/swarm/storage/mru/lookup"
)

func areEqualJSON(s1, s2 string) (bool, error) {
	//credit for the trick: turtlemonvh https://gist.github.com/turtlemonvh/e4f7404e28387fadb8ad275a99596f67
	var o1 interface{}
	var o2 interface{}

	err := json.Unmarshal([]byte(s1), &o1)
	if err != nil {
		return false, fmt.Errorf("Error mashalling string 1 :: %s", err.Error())
	}
	err = json.Unmarshal([]byte(s2), &o2)
	if err != nil {
		return false, fmt.Errorf("Error mashalling string 2 :: %s", err.Error())
	}

	return reflect.DeepEqual(o1, o2), nil
}

// TestEncodingDecodingUpdateRequests ensures that requests are serialized properly
// while also checking cryptographically that only the owner of a resource can update it.
func TestEncodingDecodingUpdateRequests(t *testing.T) {

	charlie := newCharlieSigner() //Charlie
	bob := newBobSigner()         //Bob

	// Create a resource to our good guy Charlie's name
	topic, _ := NewTopic("a good resource name", nil)
	createRequest := NewFirstRequest(topic)
	createRequest.User = charlie.Address()

	// We now encode the create message to simulate we send it over the wire
	messageRawData, err := createRequest.MarshalJSON()
	if err != nil {
		t.Fatalf("Error encoding create resource request: %s", err)
	}

	// ... the message arrives and is decoded...
	var recoveredCreateRequest Request
	if err := recoveredCreateRequest.UnmarshalJSON(messageRawData); err != nil {
		t.Fatalf("Error decoding create resource request: %s", err)
	}

	// ... but verification should fail because it is not signed!
	if err := recoveredCreateRequest.Verify(); err == nil {
		t.Fatal("Expected Verify to fail since the message is not signed")
	}

	// We now assume that the resource was created and propagated. With rootAddr we can retrieve the resource metadata
	// and recover the information above. To sign an update, we need the rootAddr and the metaHash to construct
	// proof of ownership

	const expectedSignature = "0x32c2d2c7224e24e4d3ae6a10595fc6e945f1b3ecdf548a04d8247c240a50c9240076aa7730abad6c8a46dfea00cfb8f43b6211f02db5c4cc5ed8584cb0212a4d00"
	const expectedJSON = `{"view":{"topic":"0x6120676f6f64207265736f75726365206e616d65000000000000000000000000","user":"0x876a8936a7cd0b79ef0735ad0896c1afe278781c"},"epoch":{"time":1000,"level":1},"protocolVersion":0,"data":"0x5468697320686f75722773207570646174653a20537761726d2039392e3020686173206265656e2072656c656173656421"}`

	//Put together an unsigned update request that we will serialize to send it to the signer.
	data := []byte("This hour's update: Swarm 99.0 has been released!")
	request := &Request{
		ResourceUpdate: ResourceUpdate{
			ID: ID{
				Epoch: lookup.Epoch{
					Time:  1000,
					Level: 1,
				},
				View: createRequest.ResourceUpdate.View,
			},
			data: data,
		},
	}

	messageRawData, err = request.MarshalJSON()
	if err != nil {
		t.Fatalf("Error encoding update request: %s", err)
	}

	equalJSON, err := areEqualJSON(string(messageRawData), expectedJSON)
	if err != nil {
		t.Fatalf("Error decoding update request JSON: %s", err)
	}
	if !equalJSON {
		t.Fatalf("Received a different JSON message. Expected %s, got %s", expectedJSON, string(messageRawData))
	}

	// now the encoded message messageRawData is sent over the wire and arrives to the signer

	//Attempt to extract an UpdateRequest out of the encoded message
	var recoveredRequest Request
	if err := recoveredRequest.UnmarshalJSON(messageRawData); err != nil {
		t.Fatalf("Error decoding update request: %s", err)
	}

	//sign the request and see if it matches our predefined signature above.
	if err := recoveredRequest.Sign(charlie); err != nil {
		t.Fatalf("Error signing request: %s", err)
	}

	compareByteSliceToExpectedHex(t, "signature", recoveredRequest.Signature[:], expectedSignature)

	// mess with the signature and see what happens. To alter the signature, we briefly decode it as JSON
	// to alter the signature field.
	var j updateRequestJSON
	if err := json.Unmarshal([]byte(expectedJSON), &j); err != nil {
		t.Fatal("Error unmarshalling test json, check expectedJSON constant")
	}
	j.Signature = "Certainly not a signature"
	corruptMessage, _ := json.Marshal(j) // encode the message with the bad signature
	var corruptRequest Request
	if err = corruptRequest.UnmarshalJSON(corruptMessage); err == nil {
		t.Fatal("Expected DecodeUpdateRequest to fail when trying to interpret a corrupt message with an invalid signature")
	}

	// Now imagine Bob wants to create an update of his own about the same resource,
	// signing a message with his private key
	if err := request.Sign(bob); err != nil {
		t.Fatalf("Error signing: %s", err)
	}

	// Now Bob encodes the message to send it over the wire...
	messageRawData, err = request.MarshalJSON()
	if err != nil {
		t.Fatalf("Error encoding message:%s", err)
	}

	// ... the message arrives to our Swarm node and it is decoded.
	recoveredRequest = Request{}
	if err := recoveredRequest.UnmarshalJSON(messageRawData); err != nil {
		t.Fatalf("Error decoding message:%s", err)
	}

	// Before checking what happened with Bob's update, let's see what would happen if we mess
	// with the signature big time to see if Verify catches it
	savedSignature := *recoveredRequest.Signature                               // save the signature for later
	binary.LittleEndian.PutUint64(recoveredRequest.Signature[5:], 556845463424) // write some random data to break the signature
	if err = recoveredRequest.Verify(); err == nil {
		t.Fatal("Expected Verify to fail on corrupt signature")
	}

	// restore the Bob's signature from corruption
	*recoveredRequest.Signature = savedSignature

	// Now the signature is not corrupt
	if err = recoveredRequest.Verify(); err != nil {
		t.Fatal(err)
	}

	// Reuse object and sign with our friend Charlie's private key
	if err := recoveredRequest.Sign(charlie); err != nil {
		t.Fatalf("Error signing with the correct private key: %s", err)
	}

	// And now, Verify should work since this update now belongs to Charlie
	if err = recoveredRequest.Verify(); err != nil {
		t.Fatalf("Error verifying that Charlie, can sign a reused request object:%s", err)
	}

	// mess with the lookup key to make sure Verify fails:
	recoveredRequest.Time = 77999 // this will alter the lookup key
	if err = recoveredRequest.Verify(); err == nil {
		t.Fatalf("Expected Verify to fail since the lookup key has been altered")
	}
}

func getTestRequest() *Request {
	return &Request{
		ResourceUpdate: *getTestResourceUpdate(),
	}
}

func TestUpdateChunkSerializationErrorChecking(t *testing.T) {

	// Test that parseUpdate fails if the chunk is too small
	var r Request
	if err := r.fromChunk(storage.ZeroAddr, make([]byte, minimumUpdateDataLength-1+signatureLength)); err == nil {
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
	recovered.fromChunk(chunk.Address(), chunk.Data())
	if !reflect.DeepEqual(recovered, r) {
		t.Fatal("Expected recovered SignedResource update to equal the original one")
	}
}

// check that signature address matches update signer address
func TestReverse(t *testing.T) {

	epoch := lookup.Epoch{
		Time:  7888,
		Level: 6,
	}

	// make fake timeProvider
	timeProvider := &fakeTimeProvider{
		currentTime: startTime.Time,
	}

	// signer containing private key
	signer := newAliceSigner()

	// set up rpc and create resourcehandler
	_, _, teardownTest, err := setupTest(timeProvider, signer)
	if err != nil {
		t.Fatal(err)
	}
	defer teardownTest()

	topic, _ := NewTopic("Cervantes quotes", nil)
	view := View{
		Topic: topic,
		User:  signer.Address(),
	}

	data := []byte("Donde una puerta se cierra, otra se abre")

	request := new(Request)
	request.View = view
	request.Epoch = epoch
	request.data = data

	// generate a chunk key for this request
	key := request.Addr()

	if err = request.Sign(signer); err != nil {
		t.Fatal(err)
	}

	chunk, err := request.toChunk()
	if err != nil {
		t.Fatal(err)
	}

	// check that we can recover the owner account from the update chunk's signature
	var checkUpdate Request
	if err := checkUpdate.fromChunk(chunk.Address(), chunk.Data()); err != nil {
		t.Fatal(err)
	}
	checkdigest, err := checkUpdate.GetDigest()
	if err != nil {
		t.Fatal(err)
	}
	recoveredaddress, err := getUserAddr(checkdigest, *checkUpdate.Signature)
	if err != nil {
		t.Fatalf("Retrieve address from signature fail: %v", err)
	}
	originaladdress := crypto.PubkeyToAddress(signer.PrivKey.PublicKey)

	// check that the metadata retrieved from the chunk matches what we gave it
	if recoveredaddress != originaladdress {
		t.Fatalf("addresses dont match: %x != %x", originaladdress, recoveredaddress)
	}

	if !bytes.Equal(key[:], chunk.Address()[:]) {
		t.Fatalf("Expected chunk key '%x', was '%x'", key, chunk.Address())
	}
	if epoch != checkUpdate.Epoch {
		t.Fatalf("Expected epoch to be '%s', was '%s'", epoch.String(), checkUpdate.Epoch.String())
	}
	if !bytes.Equal(data, checkUpdate.data) {
		t.Fatalf("Expected data '%x', was '%x'", data, checkUpdate.data)
	}
}
