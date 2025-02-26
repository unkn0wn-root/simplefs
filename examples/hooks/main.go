package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/unkn0wn-root/simplefs/fs"
)

func main() {
	// Create a temporary directory for our example
	tempDir := "./tmp/hooks-example"

	// Create a new filesystem
	opts := fs.DefaultOptions()
	fileSystem, err := fs.NewSimpleFS(tempDir, opts)
	if err != nil {
		fmt.Printf("Error creating filesystem: %v\n", err)
		os.Exit(1)
	}
	defer fileSystem.Close()

	fmt.Println("==== Hooks Example ====")

	// Create a log file for our hooks
	logPath := "./tmp/hooks-example/operations.log"
	loggingHook, err := fs.LoggingHook(logPath)
	if err != nil {
		fmt.Printf("Error creating logging hook: %v\n", err)
		os.Exit(1)
	}

	// Register logging hook for all operations
	fileSystem.RegisterHook(fs.OpCreateDir, fs.HookTypePre, loggingHook)
	fileSystem.RegisterHook(fs.OpWriteFile, fs.HookTypePre, loggingHook)
	fileSystem.RegisterHook(fs.OpReadFile, fs.HookTypePre, loggingHook)
	fileSystem.RegisterHook(fs.OpDeleteFile, fs.HookTypePre, loggingHook)
	fileSystem.RegisterHook(fs.OpCopyFile, fs.HookTypePre, loggingHook)
	fileSystem.RegisterHook(fs.OpMoveFile, fs.HookTypePre, loggingHook)

	// Create a custom validation hook
	validationHook := func(ctx *fs.HookContext) error {
		// Only allow files with .txt extension to be written
		if ctx.Operation == fs.OpWriteFile {
			if !strings.HasSuffix(ctx.Path, ".txt") {
				return fmt.Errorf("only .txt files are allowed")
			}
		}
		return nil
	}

	// Register the validation hook
	fileSystem.RegisterHook(fs.OpWriteFile, fs.HookTypePre, validationHook)

	// Create a profiling hook to measure operation time
	profilingHook := func(ctx *fs.HookContext) error {
		// For pre-hook, store the start time
		if ctx.Custom == nil {
			ctx.Custom = make(map[string]interface{})
		}
		ctx.Custom["start_time"] = time.Now()
		return nil
	}

	profilingPostHook := func(ctx *fs.HookContext) error {
		// For post-hook, calculate and print the duration
		startTime, ok := ctx.Custom["start_time"].(time.Time)
		if !ok {
			return nil
		}

		duration := time.Since(startTime)
		fmt.Printf("Operation %s on %s took %v\n", ctx.Operation, ctx.Path, duration)
		return nil
	}

	// Register profiling hooks
	fileSystem.RegisterHook(fs.OpWriteFile, fs.HookTypePre, profilingHook)
	fileSystem.RegisterHook(fs.OpWriteFile, fs.HookTypePost, profilingPostHook)
	fileSystem.RegisterHook(fs.OpReadFile, fs.HookTypePre, profilingHook)
	fileSystem.RegisterHook(fs.OpReadFile, fs.HookTypePost, profilingPostHook)

	// Create a directory
	fmt.Println("Creating directory...")
	err = fileSystem.CreateDir("documents")
	if err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		os.Exit(1)
	}

	// Write a valid file
	fmt.Println("\nWriting a .txt file...")
	err = fileSystem.WriteFile("documents/hello.txt", []byte("Hello, Hooks!"))
	if err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		os.Exit(1)
	}

	// Try to write an invalid file (should fail due to our validation hook)
	fmt.Println("\nTrying to write a .dat file (should fail)...")
	err = fileSystem.WriteFile("documents/data.dat", []byte("This should fail"))
	if err != nil {
		fmt.Printf("Expected error: %v\n", err)
	} else {
		fmt.Println("Error: validation hook didn't work!")
		os.Exit(1)
	}

	// Read the file
	fmt.Println("\nReading the file...")
	data, err := fileSystem.ReadFile("documents/hello.txt")
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("File content: %s\n", string(data))

	// Create a backup hook
	backupDir := "./tmp/hooks-example/backups"
	backupHook, err := fs.BackupHook(backupDir)
	if err != nil {
		fmt.Printf("Error creating backup hook: %v\n", err)
		os.Exit(1)
	}

	// Register the backup hook
	fileSystem.RegisterHook(fs.OpWriteFile, fs.HookTypePre, backupHook)

	// Modify the file to trigger the backup
	fmt.Println("\nModifying the file to trigger backup...")
	err = fileSystem.WriteFile("documents/hello.txt", []byte("Hello, Hooks! (modified)"))
	if err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		os.Exit(1)
	}

	// Check if backup was created
	backupFiles, err := os.ReadDir(backupDir)
	if err != nil {
		fmt.Printf("Error reading backup directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nBackup files:")
	for _, file := range backupFiles {
		fmt.Printf("- %s\n", file.Name())
	}

	// Create a read-only mode hook
	readOnlyHook := fs.ReadOnlyHook()

	// Register the read-only hook (temporarily)
	fmt.Println("\nEnabling read-only mode...")
	fileSystem.RegisterHook(fs.OpWriteFile, fs.HookTypePre, readOnlyHook)
	fileSystem.RegisterHook(fs.OpDeleteFile, fs.HookTypePre, readOnlyHook)
	fileSystem.RegisterHook(fs.OpCreateDir, fs.HookTypePre, readOnlyHook)

	// Try to modify the file (should fail due to read-only hook)
	fmt.Println("Trying to modify file in read-only mode (should fail)...")
	err = fileSystem.WriteFile("documents/hello.txt", []byte("This should fail"))
	if err != nil {
		fmt.Printf("Expected error: %v\n", err)
	} else {
		fmt.Println("Error: read-only hook didn't work!")
		os.Exit(1)
	}

	// Disable read-only mode
	fmt.Println("\nDisabling read-only mode...")
	fileSystem.UnregisterHook(fs.OpWriteFile, fs.HookTypePre)
	fileSystem.UnregisterHook(fs.OpDeleteFile, fs.HookTypePre)
	fileSystem.UnregisterHook(fs.OpCreateDir, fs.HookTypePre)

	// Re-register other hooks
	fileSystem.RegisterHook(fs.OpWriteFile, fs.HookTypePre, loggingHook)
	fileSystem.RegisterHook(fs.OpWriteFile, fs.HookTypePre, validationHook)
	fileSystem.RegisterHook(fs.OpWriteFile, fs.HookTypePre, profilingHook)
	fileSystem.RegisterHook(fs.OpWriteFile, fs.HookTypePost, profilingPostHook)
	fileSystem.RegisterHook(fs.OpWriteFile, fs.HookTypePre, backupHook)

	// Now we should be able to modify the file again
	fmt.Println("Modifying file after disabling read-only mode...")
	err = fileSystem.WriteFile("documents/hello.txt", []byte("Hello, Hooks! (modified again)"))
	if err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		os.Exit(1)
	}

	// Read the log file to see all operations
	fmt.Println("\nReading operation log...")
	logData, err := os.ReadFile(logPath)
	if err != nil {
		fmt.Printf("Error reading log file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nOperation log contents:")
	fmt.Println(string(logData))

	fmt.Println("\nHooks example completed successfully!")
}
