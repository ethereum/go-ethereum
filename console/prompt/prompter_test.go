package prompt_test

import (
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/console/prompt"
	"github.com/stretchr/testify/assert"
)

func TestPromptInput(t *testing.T) {
	// Simulate user input
	mockInput := "test input\n"
	r, w, _ := os.Pipe()
	w.WriteString(mockInput)
	w.Close()
	os.Stdin = r // Replace os.Stdin temporarily

	// Create a new prompter
	p := prompt.NewTerminalPrompter()
	defer func() { os.Stdin = os.NewFile(uintptr(0), "/dev/tty") }() // Restore os.Stdin

	// Test PromptInput
	result, err := p.PromptInput("Enter something: ")
	assert.NoError(t, err)
	assert.Equal(t, strings.TrimSpace(mockInput), result)
}

func TestPromptPassword(t *testing.T) {
	// Simulate password input
	mockPassword := "secret\n"
	r, w, _ := os.Pipe()
	w.WriteString(mockPassword)
	w.Close()
	os.Stdin = r // Replace os.Stdin temporarily

	// Create a new prompter
	p := prompt.NewTerminalPrompter()
	defer func() { os.Stdin = os.NewFile(uintptr(0), "/dev/tty") }() // Restore os.Stdin

	// Test PromptPassword
	result, err := p.PromptPassword("Enter password: ")
	assert.NoError(t, err)
	assert.Equal(t, strings.TrimSpace(mockPassword), result)
}

func mockStdin(input string) (restore func()) {
	// Create a pipe to replace stdin
	r, w, _ := os.Pipe()
	w.WriteString(input)
	w.Close()

	// Replace os.Stdin with the pipe
	originalStdin := os.Stdin
	os.Stdin = r

	// Return a function to restore original stdin
	return func() {
		os.Stdin = originalStdin
	}
}

func TestPromptConfirm(t *testing.T) {
	// Test confirmation (yes)
	mockInput := "y\n"
	restore := mockStdin(mockInput)
	defer restore()

	p := prompt.NewTerminalPrompter()
	result, err := p.PromptConfirm("Do you confirm?")
	assert.NoError(t, err, "Expected no error for valid input")
	assert.True(t, result, "Expected confirmation to return true")
}
