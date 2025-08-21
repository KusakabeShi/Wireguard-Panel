package main

import (
	"fmt"
	"log"
	"os/exec"

	"wg-panel/internal/utils"
)

func main() {
	fmt.Println("=== Command Error Handling Demo ===\n")

	// Demonstrate the difference between old and new error handling

	fmt.Println("1. Testing successful command:")
	fmt.Println("   Command: echo 'Hello World'")
	
	// New way
	if err := utils.RunCommand("echo", "Hello World"); err != nil {
		log.Printf("New method error: %v", err)
	} else {
		fmt.Println("   ✓ Success with new method")
	}

	fmt.Println("\n2. Testing command with output:")
	fmt.Println("   Command: date")
	
	// New way with output
	if output, err := utils.RunCommandWithOutput("date"); err != nil {
		log.Printf("New method error: %v", err)
	} else {
		fmt.Printf("   ✓ Output: %s", output)
	}

	fmt.Println("\n3. Testing failed command (old vs new error handling):")
	fmt.Println("   Command: ls /nonexistent-directory-12345")

	// Old way (basic error handling)
	fmt.Println("\n   Old method:")
	cmd := exec.Command("ls", "/nonexistent-directory-12345")
	if err := cmd.Run(); err != nil {
		fmt.Printf("   ✗ Basic error: %v\n", err)
	}

	// New way (detailed error handling)
	fmt.Println("\n   New method:")
	if err := utils.RunCommand("ls", "/nonexistent-directory-12345"); err != nil {
		fmt.Printf("   ✗ Detailed error:\n%v\n", err)
	}

	fmt.Println("\n4. Testing command with stderr output:")
	fmt.Println("   Command: sh -c 'echo error >&2; exit 1'")

	// Old way
	fmt.Println("\n   Old method:")
	cmd = exec.Command("sh", "-c", "echo 'Permission denied' >&2; exit 1")
	if err := cmd.Run(); err != nil {
		fmt.Printf("   ✗ Basic error: %v\n", err)
	}

	// New way
	fmt.Println("\n   New method:")
	if err := utils.RunCommand("sh", "-c", "echo 'Permission denied' >&2; exit 1"); err != nil {
		fmt.Printf("   ✗ Detailed error:\n%v\n", err)
	}

	fmt.Println("\n5. Testing non-existent command:")
	fmt.Println("   Command: nonexistent-command-xyz")

	// Old way
	fmt.Println("\n   Old method:")
	cmd = exec.Command("nonexistent-command-xyz")
	if err := cmd.Run(); err != nil {
		fmt.Printf("   ✗ Basic error: %v\n", err)
	}

	// New way
	fmt.Println("\n   New method:")
	if err := utils.RunCommand("nonexistent-command-xyz"); err != nil {
		fmt.Printf("   ✗ Detailed error:\n%v\n", err)
	}

	fmt.Println("\n6. Testing ignore error functionality:")
	fmt.Println("   Command: rm /nonexistent-file (errors ignored)")
	
	// This won't fail the program even if the file doesn't exist
	utils.RunCommandIgnoreError("rm", "/nonexistent-file-12345")
	fmt.Println("   ✓ Command completed (errors ignored for cleanup operations)")

	fmt.Println("\n=== Demo Complete ===")
	fmt.Println("\nKey improvements:")
	fmt.Println("• See exact command that failed")
	fmt.Println("• Get stdout and stderr output")  
	fmt.Println("• Know the exit code")
	fmt.Println("• See execution duration")
	fmt.Println("• Better debugging information")
}