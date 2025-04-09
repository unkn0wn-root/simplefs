# SimpleFS

## Overview

SimpleFS is a file system implementation in Go with API for file operations like journaling, versioning, extended attributes, hooks, etc.

## Key Features

- **Core File Operations**: Create, read, write, delete, copy, and move files
- **Directory Management**: Create, list, and delete directories
- **Journaling**: Transaction-based operations with crash recovery
- **Versioning**: Automatic file versioning with history tracking
- **Extended Attributes**: Store metadata alongside files
- **Hooks System**: Lifecycle hooks for file operations
- **CLI Tool**: Command-line interface for interacting with the file system

## Project Structure

```
gosimplefs/
├── cmd/
│   └── simplefs/         # CLI tool for the file system
├── docs/
│   ├── api.md            # API documentation
│   └── advanced-usage.md # Advanced usage guides
├── examples/             # Example code
├── attributes.go     # Extended attributes implementation
├── fs.go             # Main file system implementation
├── hooks.go          # Hooks system
├── journal.go        # Journaling implementation
├── locks.go          # Concurrency control
├── path.go           # Path manipulation utilities
├── versioning.go     # File versioning
├── internal/
│   └── utils/            # Utility functions
│       ├── hash.go       # Hash functions
│       └── strings.go    # String utilities
├── LICENSE               # MIT License
└── README.md             # Project README
```

### Basic Usage

```go
// Create a new file system
opts := fs.DefaultOptions()
fileSystem, err := fs.NewSimpleFS("./myfs", opts)
if err != nil {
    log.Fatalf("Error creating file system: %v", err)
}
defer fileSystem.Close()

// Write a file
err = fileSystem.WriteFile("hello.txt", []byte("Hello, SimpleFS!"))
if err != nil {
    log.Fatalf("Error writing file: %v", err)
}

// Read a file
data, err := fileSystem.ReadFile("hello.txt")
if err != nil {
    log.Fatalf("Error reading file: %v", err)
}
fmt.Println(string(data))
```

### Enabling Advanced Features

```go
// Enable versioning and journaling
opts := fs.DefaultOptions()
opts.EnableVersioning = true
opts.EnableJournaling = true
opts.MaxVersions = 10

fileSystem, err := fs.NewSimpleFS("./myfs", opts)
if err != nil {
    log.Fatalf("Error creating file system: %v", err)
}
defer fileSystem.Close()

// Add hooks for custom behavior
fileSystem.RegisterHook(fs.OpWriteFile, fs.HookTypePre, myValidationHook)

// Enable explicit locking
fileSystem = fileSystem.WithExplicitLocking()
```

## CLI Tool

The CLI tool provides a command-line interface for interacting with the file system. Here are some example commands:

```bash
# Create a directory
simplefs mkdir documents

# Write a file
simplefs write documents/hello.txt "Hello, SimpleFS!"

# List directory contents
simplefs ls documents

# Read a file
simplefs read documents/hello.txt

# Set attributes
simplefs attr set documents/hello.txt author "Johnny Be Goode"

# Work with versions
simplefs versions list documents/hello.txt
```

## Installation

```bash
go get github.com/unkn0wn-root/simplefs
```
