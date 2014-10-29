package ethwire

import (
	"fmt"
	"runtime"
	"testing"
)

func TestClientIdentity(t *testing.T) {
	clientIdentity := NewSimpleClientIdentity("Ethereum(G)", "0.5.16", "test")
	clientString := clientIdentity.String()
	expected := fmt.Sprintf("Ethereum(G)/v0.5.16/test/%s/%s", runtime.GOOS, runtime.Version())
	if clientString != expected {
		t.Errorf("Expected clientIdentity to be %q, got %q", expected, clientString)
	}
	customIdentifier := clientIdentity.GetCustomIdentifier()
	if customIdentifier != "test" {
		t.Errorf("Expected clientIdentity.GetCustomIdentifier() to be 'test', got %q", customIdentifier)
	}
	clientIdentity.SetCustomIdentifier("test2")
	customIdentifier = clientIdentity.GetCustomIdentifier()
	if customIdentifier != "test2" {
		t.Errorf("Expected clientIdentity.GetCustomIdentifier() to be 'test2', got %q", customIdentifier)
	}
	clientString = clientIdentity.String()
	expected = fmt.Sprintf("Ethereum(G)/v0.5.16/test2/%s/%s", runtime.GOOS, runtime.Version())
	if clientString != expected {
		t.Errorf("Expected clientIdentity to be %q, got %q", expected, clientString)
	}
}
