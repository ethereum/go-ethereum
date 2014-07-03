package ethwire

import (
	"fmt"
	"runtime"
	"testing"
)

func TestClientIdentity(t *testing.T) {
	clientIdentity := NewSimpleClientIdentity("Ethereum(G)", "0.5.16", "test")
	clientString := clientIdentity.String()
	expected := fmt.Sprintf("Ethereum(G)/v0.5.16/test/%s/Go", runtime.GOOS)
	if clientString != expected {
		t.Error("Expected clientIdentity to be %v, got %v", expected, clientString)
	}
	customIdentifier := clientIdentity.GetCustomIdentifier()
	if customIdentifier != "test" {
		t.Error("Expected clientIdentity.GetCustomIdentifier() to be 'test', got %v", customIdentifier)
	}
	clientIdentity.SetCustomIdentifier("test2")
	customIdentifier = clientIdentity.GetCustomIdentifier()
	if customIdentifier != "test2" {
		t.Error("Expected clientIdentity.GetCustomIdentifier() to be 'test2', got %v", customIdentifier)
	}
	clientString = clientIdentity.String()
	expected = fmt.Sprintf("Ethereum(G)/v0.5.16/test2/%s/Go", runtime.GOOS)
	if clientString != expected {
		t.Error("Expected clientIdentity to be %v, got %v", expected, clientString)
	}
}
