package utils

import (
	"strings"
	"testing"
	"time"
)

func TestRunCommand_Success(t *testing.T) {
	err := RunCommand("echo", "hello world")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestRunCommandWithOutput_Success(t *testing.T) {
	output, err := RunCommandWithOutput("echo", "hello world")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	
	if !strings.Contains(output, "hello world") {
		t.Errorf("Expected output to contain 'hello world', got: %s", output)
	}
}

func TestRunCommand_Failure(t *testing.T) {
	err := RunCommand("false")
	if err == nil {
		t.Error("Expected error, got nil")
	}
	
	cmdErr, ok := err.(*CommandError)
	if !ok {
		t.Errorf("Expected CommandError, got: %T", err)
	}
	
	if cmdErr.ExitCode != 1 {
		t.Errorf("Expected exit code 1, got: %d", cmdErr.ExitCode)
	}
}

func TestRunCommand_NonExistentCommand(t *testing.T) {
	err := RunCommand("nonexistent-command-12345")
	if err == nil {
		t.Error("Expected error for non-existent command, got nil")
	}
	
	cmdErr, ok := err.(*CommandError)
	if !ok {
		t.Errorf("Expected CommandError, got: %T", err)
	}
	
	// Should contain information about the failed command
	errorStr := cmdErr.Error()
	if !strings.Contains(errorStr, "nonexistent-command-12345") {
		t.Errorf("Expected error to contain command name, got: %s", errorStr)
	}
}

func TestRunCommandWithTimeout_Success(t *testing.T) {
	output, err := RunCommandWithTimeout(5*time.Second, "echo", "hello")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	
	if !strings.Contains(output, "hello") {
		t.Errorf("Expected output to contain 'hello', got: %s", output)
	}
}

func TestRunCommandWithTimeout_Timeout(t *testing.T) {
	// This test might be slow, so we'll make it fast
	_, err := RunCommandWithTimeout(100*time.Millisecond, "sleep", "1")
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
	
	cmdErr, ok := err.(*CommandError)
	if !ok {
		t.Errorf("Expected CommandError, got: %T", err)
	}
	
	errorStr := cmdErr.Error()
	if !strings.Contains(errorStr, "timed out") {
		t.Errorf("Expected timeout error message, got: %s", errorStr)
	}
}

func TestCommandError_FormattedOutput(t *testing.T) {
	// Test stderr capture
	err := RunCommand("sh", "-c", "echo 'error message' >&2; exit 1")
	if err == nil {
		t.Error("Expected error, got nil")
	}
	
	cmdErr, ok := err.(*CommandError)
	if !ok {
		t.Errorf("Expected CommandError, got: %T", err)
	}
	
	errorStr := cmdErr.Error()
	if !strings.Contains(errorStr, "stderr: error message") {
		t.Errorf("Expected stderr in error message, got: %s", errorStr)
	}
}