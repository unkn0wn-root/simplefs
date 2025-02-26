# Advanced Usage Guide for SimpleFS

This guide covers advanced usage patterns and features of SimpleFS.

## Table of Contents

- [Journaling and Recovery](#journaling-and-recovery)
- [File Versioning](#file-versioning)
- [Using Hooks](#using-hooks)
- [Extended Attributes](#extended-attributes)
- [Concurrent Access](#concurrent-access)
- [Explicit Locking](#explicit-locking)
- [Path Manipulation](#path-manipulation)
- [Security Considerations](#security-considerations)
- [Performance Optimization](#performance-optimization)
- [Integration Patterns](#integration-patterns)

## Journaling and Recovery

SimpleFS provides a journaling mechanism that logs all file operations for crash recovery.

### How Journaling Works

1. Before each write operation, details are written to a journal file
2. If the system crashes during an operation, the journal can be replayed on next startup
3. This ensures file system consistency even after unexpected shutdowns

### Enabling Journaling

Journaling is enabled by default, but you can explicitly configure it:

```go
opts := fs.DefaultOptions()
opts.EnableJournaling = true
fileSystem, err := fs.NewSimpleFS("./myfs", opts)
```

### Performing Recovery

To recover from a crash:

```go
fileSystem, err := fs.NewSimpleFS("./myfs", nil)
if err != nil {
    log.Fatalf("Error opening filesystem: %v", err)
}

// Recover from any previous crash
err = fileSystem.Recover()
if err != nil {
    log.Printf("Warning: Recovery failed: %v", err)
}
```

### Journal Maintenance

The journal file can grow over time. You can perform maintenance operations:

```go
// Rotate the journal file
err = fileSystem.journal.Rotate()

// Truncate the journal file
err = fileSystem.journal.Truncate()
```

## File Versioning

SimpleFS can maintain multiple versions of files as they change.

### Enabling Versioning

```go
opts := fs.DefaultOptions()
opts.EnableVersioning = true
opts.MaxVersions = 10 // Keep up to 10 versions per file
fileSystem, err := fs.NewSimpleFS("./myfs", opts)
```

### Working with Versions

List all versions of a file:

```go
versions, err := fileSystem.ListVersions("document.txt")
for _, version := range versions.Versions {
    fmt.Printf("Version: %s, Created: %s\n", version.VersionID, version.CreatedAt)
}
```

Get a specific version:

```go
data, versionInfo, err := fileSystem.GetVersion("document.txt", versionID)
```

Restore to a previous version:

```go
err = fileSystem.RestoreVersion("document.txt", versionID)
```

Add description to a version:

```go
err = fileSystem.SetVersionDescription("document.txt", versionID, "Final draft")
```

## Using Hooks

Hooks allow you to intercept and modify file system operations.

### Types of Hooks

- **Pre-hooks**: Executed before an operation, can prevent the operation
- **Post-hooks**: Executed after an operation, cannot prevent the operation

### Registering Hooks

```go
// Create a logging hook
loggingHook, err := fs.LoggingHook("./operations.log")
if err != nil {
    log.Fatalf("Error creating logging hook: %v", err)
}

// Register for write operations
fileSystem.RegisterHook(fs.OpWriteFile, fs.HookTypePre, loggingHook)
```

### Custom Hooks

You can create custom hooks for specialized behavior:

```go
validationHook := func(ctx *fs.HookContext) error {
    // Only allow .txt files
    if ctx.Operation == fs.OpWriteFile && !strings.HasSuffix(ctx.Path, ".txt") {
        return errors.New("only .txt files are allowed")
    }
    return nil
}

fileSystem.RegisterHook(fs.OpWriteFile, fs.HookTypePre, validationHook)
```

### Built-in Hooks

SimpleFS provides several built-in hooks:

```go
// Backup files before modification
backupHook, err := fs.BackupHook("./backups")

// Make the filesystem read-only
readOnlyHook := fs.ReadOnlyHook()
```

### Unregistering Hooks

```go
// Unregister a specific hook type
fileSystem.UnregisterHook(fs.OpWriteFile, fs.HookTypePre)

// Unregister all hooks
fileSystem.UnregisterAllHooks()
```

## Extended Attributes

Extended attributes allow storing metadata alongside files.

### Managing Attributes

```go
// Set attributes
fileSystem.SetAttribute("document.txt", "author", "John Doe")
fileSystem.SetAttribute("document.txt", "created", time.Now().Format(time.RFC3339))

// Get an attribute
author, err := fileSystem.GetAttribute("document.txt", "author")

// Get all attributes
attrs, err := fileSystem.GetAllAttributes("document.txt")
for key, value := range attrs {
    fmt.Printf("%s: %s\n", key, value)
}

// Delete an attribute
err = fileSystem.DeleteAttribute("document.txt", "temporary")
```

### Using Attributes for Metadata

Attributes can store any string data and are useful for:

- File classifications (tags, categories)
- User information (author, owner)
- Timestamps (created, last accessed)
- Custom metadata (version, status)

## Concurrent Access

SimpleFS handles concurrent access with file-level locking.

### Automatic Locking

The filesystem automatically uses read/write locks for operations:

- Multiple read operations can occur simultaneously
- Write operations get exclusive access
- Operations block until locks are available

### Transaction-like Operations

For complex operations that need atomic behavior:

```go
// Get locks for multiple files
file1Lock := fileSystem.getFileLock(file1Path)
file2Lock := fileSystem.getFileLock(file2Path)

// Lock all resources
file1Lock.Lock()
defer file1Lock.Unlock()
file2Lock.Lock()
defer file2Lock.Unlock()

// Perform operations atomically
// ...
```

## Explicit Locking

Beyond the automatic internal locking, SimpleFS supports explicit application-level locking.

### Enabling Explicit Locking

```go
fileSystem = fileSystem.WithExplicitLocking()
```

### Lock Types

- **Read Lock**: Multiple readers can access the file simultaneously
- **Write Lock**: Exclusive access, blocks all other access

### Acquiring Locks

```go
// Acquire a write lock with 30-second timeout
lockInfo, err := fileSystem.LockFile("document.txt", "user-123", fs.WriteLock, 30*time.Second)
if err != nil {
    // Handle locking error
    return
}

// Perform operations on the locked file
// ...

// Release the lock when done
err = fileSystem.UnlockFile("document.txt", "user-123")
```

### Checking Lock Status

```go
if fileSystem.IsFileLocked("document.txt") {
    lockInfo, exists := fileSystem.GetFileLockInfo("document.txt")
    if exists {
        fmt.Printf("File is locked by %s\n", lockInfo.Owner)
    }
}
```

### Waiting for Locks

```go
// Wait up to 10 seconds for a lock to be released
if fileSystem.WaitForFileLock("document.txt", 10*time.Second) {
    // Lock was released, proceed
} else {
    // Timed out waiting for lock
}
```

## Path Manipulation

SimpleFS provides utilities for safe path handling.

### Path Validation

```go
// Check if a path is valid
valid, err := fileSystem.IsValidPath(userProvidedPath)
if !valid {
    fmt.Printf("Invalid path: %v\n", err)
    return
}
```

### Path Information

```go
// Get detailed path information
pathInfo, err := fileSystem.GetPathInfo("documents/projects/report.txt")
if err != nil {
    // Handle error
    return
}

fmt.Printf("Absolute path: %s\n", pathInfo.Absolute)
fmt.Printf("Components: %v\n", pathInfo.Components)
fmt.Printf("Is directory: %t\n", pathInfo.IsDir)
```

### Path Utilities

```go
// Get parent directory
parent := fileSystem.GetParentPath("documents/projects/report.txt")
// Returns "documents/projects"

// Get basename
name := fileSystem.GetBasename("documents/projects/report.txt")
// Returns "report.txt"
```

## Security Considerations

### Path Traversal Protection

SimpleFS prevents path traversal attacks by validating and sanitizing all paths:

```go
// This will fail with an error about escaping the root directory
data, err := fileSystem.ReadFile("../outside/secret.txt")
```

### Permissions

When using `WriteFileWithMode`, you can control file permissions:

```go
// Create a file readable only by the owner
err := fileSystem.WriteFileWithMode("private.txt", data, 0600)
```

### Secure Configurations

For security-sensitive applications:

```go
// Use explicit locking for application-level access control
fileSystem = fileSystem.WithExplicitLocking()

// Register validation hooks to enforce security policies
fileSystem.RegisterHook(fs.OpWriteFile, fs.HookTypePre, securityPolicyHook)
```

## Performance Optimization

### Reducing Lock Contention

- Use the most specific path possible to minimize lock conflicts
- Keep operations small and focused
- Avoid long-running operations while holding locks

### Journaling Considerations

Journaling adds overhead but provides safety. For performance-critical applications:

```go
// Disable journaling if crash recovery isn't required
opts := fs.DefaultOptions()
opts.EnableJournaling = false
fileSystem, err := fs.NewSimpleFS("./myfs", opts)
```

### Journal Maintenance

Regularly maintain the journal to prevent it from growing too large:

```go
// In a background goroutine:
for {
    fileSystem.journal.Rotate()
    time.Sleep(24 * time.Hour)
}
```

## Integration Patterns

### Web Server Integration

```go
http.HandleFunc("/files/", func(w http.ResponseWriter, r *http.Request) {
    path := strings.TrimPrefix(r.URL.Path, "/files/")

    // Validate and sanitize path
    valid, err := fileSystem.IsValidPath(path)
    if !valid || err != nil {
        http.Error(w, "Invalid path", http.StatusBadRequest)
        return
    }

    switch r.Method {
    case "GET":
        data, err := fileSystem.ReadFile(path)
        if err != nil {
            http.Error(w, err.Error(), http.StatusNotFound)
            return
        }
        w.Write(data)

    case "PUT":
        data, err := io.ReadAll(r.Body)
        if err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }

        err = fileSystem.WriteFile(path, data)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        w.WriteHeader(http.StatusOK)

    case "DELETE":
        if fileSystem.IsDir(path) {
            err = fileSystem.DeleteDir(path)
        } else {
            err = fileSystem.DeleteFile(path)
        }

        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        w.WriteHeader(http.StatusOK)
    }
})
```

### Distributed Systems

For distributed systems, use explicit locking with owner identification:

```go
nodeID := "node-" + uuid.New().String()

// Acquire locks with the node ID as the owner
lockInfo, err := fileSystem.LockFile(path, nodeID, fs.WriteLock, 30*time.Second)

// Implement a heartbeat system to extend lock timeouts
go func() {
    for {
        // Release and re-acquire lock to extend timeout
        fileSystem.UnlockFile(path, nodeID)
        fileSystem.LockFile(path, nodeID, fs.WriteLock, 30*time.Second)
        time.Sleep(10 * time.Second)
    }
}()
```

### Content Management Systems

Create specialized hooks for content management:

```go
// Auto-versioning hook
autoVersionHook := func(ctx *fs.HookContext) error {
    if ctx.Operation == fs.OpWriteFile {
        ctx.Custom["create_version"] = true
    }
    return nil
}

// Content validation hook
contentValidationHook := func(ctx *fs.HookContext) error {
    if ctx.Operation == fs.OpWriteFile {
        // Validate content (e.g., check for malicious scripts)
        if containsMaliciousContent(ctx.Data) {
            return errors.New("content contains potentially harmful code")
        }
    }
    return nil
}

fileSystem.RegisterHook(fs.OpWriteFile, fs.HookTypePre, autoVersionHook)
fileSystem.RegisterHook(fs.OpWriteFile, fs.HookTypePre, contentValidationHook)
```
