# Command Error Handling

This document describes the enhanced command error handling system implemented in WG-Panel.

## Overview

All `exec.Command` calls in the codebase have been replaced with utility functions that provide comprehensive error reporting, including both stdout and stderr output in human-readable format.

## Key Features

### 1. Detailed Error Information
- **Command and Arguments**: Shows the exact command that failed
- **Exit Code**: Displays the process exit code
- **Stdout/Stderr**: Captures and displays both output streams
- **Duration**: Shows how long the command took to run
- **System Error**: Includes underlying system error details

### 2. Multiple Execution Modes
- `RunCommand()`: Execute and return only errors
- `RunCommandWithOutput()`: Execute and return both output and errors
- `RunCommandIgnoreError()`: Execute but don't fail on errors (useful for cleanup)
- `RunCommandWithTimeout()`: Execute with timeout protection

## Usage Examples

### Basic Command Execution
```go
// Old way (basic error handling)
cmd := exec.Command("ip", "link", "show", "wg-test")
if err := cmd.Run(); err != nil {
    return fmt.Errorf("failed to check interface: %v", err)
}

// New way (detailed error handling)
if err := utils.RunCommand("ip", "link", "show", "wg-test"); err != nil {
    return fmt.Errorf("failed to check interface: %v", err)
}
```

### Command with Output
```go
// Old way
cmd := exec.Command("wg", "show", "wg-test", "dump")
output, err := cmd.Output()
if err != nil {
    return nil, fmt.Errorf("failed to get stats:-> %v", err)
}

// New way
output, err := utils.RunCommandWithOutput("wg", "show", "wg-test", "dump")
if err != nil {
    return nil, fmt.Errorf("failed to get stats:-> %v", err)
}
```

### Cleanup Operations (Ignore Errors)
```go
// Old way
cmd := exec.Command("ip", "addr", "del", "192.168.1.1/24", "dev", "wg-test")
cmd.Run() // Ignore errors

// New way
utils.RunCommandIgnoreError("ip", "addr", "del", "192.168.1.1/24", "dev", "wg-test")
```

### Commands with Timeout
```go
// Prevent hanging on problematic commands
output, err := utils.RunCommandWithTimeout(30*time.Second, "wg-quick", "up", "/etc/wireguard/wg-test.conf")
if err != nil {
    return fmt.Errorf("interface setup timed out:-> %v", err)
}
```

## Error Message Format

When a command fails, you'll get detailed error messages like:

```
Command failed: iptables -t nat -A POSTROUTING -s 192.168.1.0/24 -j MASQUERADE (exit code: 1) [took 45ms]
  stdout: 
  stderr: iptables: No chain/target/match by that name.
  system error: exit status 1
```

This provides:
1. **What command failed** with all arguments
2. **Exit code** for debugging
3. **Execution time** to identify performance issues  
4. **Stdout** if the command produced output
5. **Stderr** showing the actual error message from the command
6. **System error** for additional context

## Implementation Details

### Files Modified

1. **`internal/utils/command.go`** - New command execution utilities
2. **`internal/services/wireguard_service.go`** - Updated all WireGuard commands
3. **`internal/services/firewall_service.go`** - Updated all iptables/ip commands

### Error Types

The `CommandError` struct provides structured access to all error details:

```go
type CommandError struct {
    Command    string        // Command name
    Args       []string      // Command arguments  
    ExitCode   int          // Process exit code
    Stdout     string       // Standard output
    Stderr     string       // Standard error
    SystemErr  error        // Underlying system error
    Duration   time.Duration // Execution time
}
```

## Benefits

### 1. Better Debugging
- See exact command that failed
- Get actual error messages from tools like `iptables`, `wg`, `ip`
- Understand timing issues with duration logging

### 2. Improved User Experience
- Clear error messages instead of generic "exit status 1"
- Users can understand what went wrong
- Support teams get actionable information

### 3. Enhanced Monitoring
- Log detailed command failures for analysis
- Track command execution times
- Identify problematic network operations

### 4. Robust Error Handling
- Timeout protection prevents hanging
- Structured errors for programmatic handling
- Consistent error format across all commands

## Testing

The command utility includes comprehensive tests covering:
- Successful command execution
- Failed commands with proper error details
- Non-existent commands
- Timeout scenarios  
- Stdout/stderr capture

Run tests with:
```bash
go test ./internal/utils/ -v
```

## Migration Notes

When migrating existing code:

1. Replace `exec.Command(...).Run()` with `utils.RunCommand(...)`
2. Replace `exec.Command(...).Output()` with `utils.RunCommandWithOutput(...)`
3. For cleanup operations, use `utils.RunCommandIgnoreError(...)`
4. Consider timeouts for potentially slow operations

The new functions maintain the same error semantics but provide much richer error information.