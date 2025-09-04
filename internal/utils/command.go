package utils

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// CommandError represents a detailed command execution error
type CommandError struct {
	Command   string
	Args      []string
	ExitCode  int
	Stdout    string
	Stderr    string
	SystemErr error
	Duration  time.Duration
}

func (e *CommandError) Error() string {
	var buf strings.Builder

	// Basic command info
	buf.WriteString(fmt.Sprintf("Command failed: %s", e.Command))
	if len(e.Args) > 0 {
		buf.WriteString(fmt.Sprintf(" %s", strings.Join(e.Args, " ")))
	}
	buf.WriteString(fmt.Sprintf(" (exit code: %d)", e.ExitCode))

	// Add duration if available
	if e.Duration > 0 {
		buf.WriteString(fmt.Sprintf(" [took %v]", e.Duration))
	}

	// Add stdout if present and not empty
	if strings.TrimSpace(e.Stdout) != "" {
		buf.WriteString(fmt.Sprintf("\n  stdout: %s", strings.TrimSpace(e.Stdout)))
	}

	// Add stderr if present and not empty
	if strings.TrimSpace(e.Stderr) != "" {
		buf.WriteString(fmt.Sprintf("\n  stderr: %s", strings.TrimSpace(e.Stderr)))
	}

	// Add system error if different from exit code
	if e.SystemErr != nil {
		buf.WriteString(fmt.Sprintf("\n  system error: %v", e.SystemErr))
	}

	return buf.String()
}

// RunCommand executes a command with detailed error reporting
func RunCommand(name string, args ...string) error {
	_, err := RunCommandWithOutput(name, args...)
	return err
}

// RunCommandWithOutput executes a command and returns both output and detailed errors
func RunCommandWithOutput(name string, args ...string) (string, error) {
	start := time.Now()

	cmd := exec.Command(name, args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start)

	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	if err != nil {
		cmdErr := &CommandError{
			Command:   name,
			Args:      args,
			Stdout:    stdoutStr,
			Stderr:    stderrStr,
			SystemErr: err,
			Duration:  duration,
		}

		// Try to get exit code
		if exitError, ok := err.(*exec.ExitError); ok {
			cmdErr.ExitCode = exitError.ExitCode()
		} else {
			cmdErr.ExitCode = -1
		}

		return stdoutStr, cmdErr
	}

	return stdoutStr, nil
}

// RunCommandIgnoreError executes a command but ignores errors (useful for cleanup operations)
func RunCommandIgnoreError(name string, args ...string) (string, error) {
	start := time.Now()

	cmd := exec.Command(name, args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start)

	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	// Even if we ignore the error, we might want to log it for debugging
	if err != nil {
		// You could add logging here if needed
		_ = &CommandError{
			Command:   name,
			Args:      args,
			Stdout:    stdoutStr,
			Stderr:    stderrStr,
			SystemErr: err,
			Duration:  duration,
		}
	}

	return stdoutStr, err // Return original error for caller to decide
}

// RunCommandWithTimeout executes a command with a timeout
func RunCommandWithTimeout(timeout time.Duration, name string, args ...string) (string, error) {
	start := time.Now()

	cmd := exec.Command(name, args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start the command
	err := cmd.Start()
	if err != nil {
		return "", &CommandError{
			Command:   name,
			Args:      args,
			SystemErr: fmt.Errorf("failed to start command:-> %v", err),
			Duration:  time.Since(start),
		}
	}

	// Wait for completion or timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err = <-done:
		duration := time.Since(start)
		stdoutStr := stdout.String()
		stderrStr := stderr.String()

		if err != nil {
			cmdErr := &CommandError{
				Command:   name,
				Args:      args,
				Stdout:    stdoutStr,
				Stderr:    stderrStr,
				SystemErr: err,
				Duration:  duration,
			}

			if exitError, ok := err.(*exec.ExitError); ok {
				cmdErr.ExitCode = exitError.ExitCode()
			} else {
				cmdErr.ExitCode = -1
			}

			return stdoutStr, cmdErr
		}

		return stdoutStr, nil

	case <-time.After(timeout):
		// Kill the process
		cmd.Process.Kill()
		cmd.Wait() // Clean up

		return stdout.String(), &CommandError{
			Command:   name,
			Args:      args,
			Stdout:    stdout.String(),
			Stderr:    stderr.String(),
			SystemErr: fmt.Errorf("command timed out after %v", timeout),
			Duration:  timeout,
			ExitCode:  -1,
		}
	}
}

func If[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}
