package p2p

import (
	"fmt"
	"runtime"
	"testing"
)

func TestClientIdentity(t *testing.T) {
	clientIdentity := NewSimpleClientIdentity("Ethereum(G)", "0.5.16", "test", []byte("pubkey"))
	clientString := clientIdentity.String()
	expected := fmt.Sprintf("Ethereum(G)/v0.5.16/test/%s/%s", runtime.GOOS, runtime.Version())
	if clientString != expected {
		t.Errorf("Expected clientIdentity to be %v, got %v", expected, clientString)
	}
	customIdentifier := clientIdentity.GetCustomIdentifier()
	if customIdentifier != "test" {
		t.Errorf("Expected clientIdentity.GetCustomIdentifier() to be 'test', got %v", customIdentifier)
	}
	clientIdentity.SetCustomIdentifier("test2")
	customIdentifier = clientIdentity.GetCustomIdentifier()
	if customIdentifier != "test2" {
		t.Errorf("Expected clientIdentity.GetCustomIdentifier() to be 'test2', got %v", customIdentifier)
	}
	clientString = clientIdentity.String()
	expected = fmt.Sprintf("Ethereum(G)/v0.5.16/test2/%s/%s", runtime.GOOS, runtime.Version())
	if clientString != expected {
		t.Errorf("Expected clientIdentity to be %v, got %v", expected, clientString)
	}
}
