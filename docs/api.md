# SimpleFS API Reference

This document provides a comprehensive reference for the SimpleFS API.

## Table of Contents

- [Core Types](#core-types)
- [Initialization](#initialization)
- [File Operations](#file-operations)
- [Directory Operations](#directory-operations)
- [Path Operations](#path-operations)
- [Attributes](#attributes)
- [Versioning](#versioning)
- [Hooks](#hooks)
- [Explicit Locking](#explicit-locking)
- [Error Handling](#error-handling)

## Core Types

### SimpleFS

The main filesystem type that provides all operations.

```go
type SimpleFS struct {
    // Private fields not exposed
}
```

### FileInfo

Contains metadata about a file.

```go
type FileInfo struct {
    Name       string            // The base name of the file
    Size       int64             // Length in bytes
    ModTime    time.Time         // Modification time
    IsDir      bool              // Is this a directory?
    Mode       os.FileMode       // File mode bits
    Attributes map[string]string // Extended attributes
}
```

### Options

Configuration options for the filesystem.

```go
type Options struct {
    EnableJournaling bool // Whether to enable journaling
    EnableVersioning bool // Whether to enable versioning
    MaxVersions      int  // Maximum number of versions to keep (0 = unlimited)
}
```

### VersionInfo

Information about a file version.

```go
type VersionInfo struct {
    VersionID   string            // Unique ID for this version
    Path        string            // Original file path
    CreatedAt   time.Time         // When this version was created
    Size        int64             // Size in bytes
    Attributes  map[string]string // Attributes at time of versioning
    Description string            // Optional description
}
```

### HookContext

Context provided to hook functions.

```go
type HookContext struct {
    Operation OperationType    // Type of operation
    Path      string           // Path of the file or directory
    SrcPath   string           // Source path for copy/move operations
    Data      []byte           // Data for write operations
    Mode      os.FileMode      // File mode for write operations
    Key       string           // Key for attribute operations
    Value     string           // Value for attribute operations
    Error     error            // Error from the operation (in post hooks)
    FS        *SimpleFS        // Reference to the filesystem
    Custom    map[string]interface{} // Custom data for hooks
}
```

### LockInfo

Information about a file lock.

```go
type LockInfo struct {
    Path      string    // Path being locked
    Type      LockType  // Type of lock
    Owner     string    // Identifier of the lock owner
    CreatedAt time.Time // When the lock was created
    Timeout   time.Duration // How long until the lock expires
}
```

## Initialization

### NewSimpleFS

Creates a new filesystem with the given root path and options.

```go
func NewSimpleFS(rootPath string, opts *Options) (*SimpleFS, error)
```

**Parameters:**
- `rootPath`: The root directory for the filesystem
- `opts`: Configuration options (can be `nil` for defaults)

**Returns:**
- A new filesystem instance
- An error if initialization fails

**Example:**
```go
opts := fs.DefaultOptions()
fileSystem, err := fs.NewSimpleFS("./myfs", opts)
if err != nil {
    log.Fatalf("Error creating filesystem: %v", err)
}
defer fileSystem.Close()
```

### DefaultOptions

Returns the default options.

```go
func DefaultOptions() *Options
```

**Returns:**
- Default options with journaling enabled and versioning disabled

## File Operations

### WriteFile

Writes data to a file, creating it if it doesn't exist.

```go
func (fs *SimpleFS) WriteFile(path string, data []byte) error
```

**Parameters:**
- `path`: The path to the file
- `data`: The data to write

**Returns:**
- An error if the operation fails

### WriteFileWithMode

Writes data to a file with specific permissions.

```go
func (fs *SimpleFS) WriteFileWithMode(path string, data []byte, mode os.FileMode) error
```

**Parameters:**
- `path`: The path to the file
- `data`: The data to write
- `mode`: The file mode (permissions)

**Returns:**
- An error if the operation fails

### ReadFile

Reads the content of a file.

```go
func (fs *SimpleFS) ReadFile(path string) ([]byte, error)
```

**Parameters:**
- `path`: The path to the file

**Returns:**
- The file content
- An error if the operation fails

### CopyFile

Copies a file from one path to another.

```go
func (fs *SimpleFS) CopyFile(src, dst string) error
```

**Parameters:**
- `src`: The source file path
- `dst`: The destination file path

**Returns:**
- An error if the operation fails

### MoveFile

Moves a file from one path to another.

```go
func (fs *SimpleFS) MoveFile(src, dst string) error
```

**Parameters:**
- `src`: The source file path
- `dst`: The destination file path

**Returns:**
- An error if the operation fails

### DeleteFile

Deletes a file.

```go
func (fs *SimpleFS) DeleteFile(path string) error
```

**Parameters:**
- `path`: The path to the file

**Returns:**
- An error if the operation fails

### FileExists

Checks if a file exists.

```go
func (fs *SimpleFS) FileExists(path string) bool
```

**Parameters:**
- `path`: The path to check

**Returns:**
- `true` if the file exists, `false` otherwise

### Stat

Returns file information.

```go
func (fs *SimpleFS) Stat(path string) (*FileInfo, error)
```

**Parameters:**
- `path`: The path to the file

**Returns:**
- File information
- An error if the operation fails

## Directory Operations

### CreateDir

Creates a new directory.

```go
func (fs *SimpleFS) CreateDir(path string) error
```

**Parameters:**
- `path`: The path to the directory

**Returns:**
- An error if the operation fails

### ListDir

Lists the contents of a directory.

```go
func (fs *SimpleFS) ListDir(path string) ([]FileInfo, error)
```

**Parameters:**
- `path`: The path to the directory

**Returns:**
- A list of file information
- An error if the operation fails

### DeleteDir

Deletes a directory and all its contents.

```go
func (fs *SimpleFS) DeleteDir(path string) error
```

**Parameters:**
- `path`: The path to the directory

**Returns:**
- An error if the operation fails

### IsDir

Checks if a path is a directory.

```go
func (fs *SimpleFS) IsDir(path string) bool
```

**Parameters:**
- `path`: The path to check

**Returns:**
- `true` if the path is a directory, `false` otherwise

## Path Operations

### IsValidPath

Checks if a path is valid and doesn't attempt to escape the root.

```go
func (fs *SimpleFS) IsValidPath(path string) (bool, error)
```

**Parameters:**
- `path`: The path to check

**Returns:**
- `true` if the path is valid, `false` otherwise
- An error if the path is invalid

### GetPathInfo

Returns information about a path.

```go
func (fs *SimpleFS) GetPathInfo(path string) (*PathInfo, error)
```

**Parameters:**
- `path`: The path to check

**Returns:**
- Path information
- An error if the operation fails

### GetRelativePath

Gets the relative path from the root.

```go
func (fs *SimpleFS) GetRelativePath(path string) (string, error)
```

**Parameters:**
- `path`: The path to convert

**Returns:**
- The relative path
- An error if the operation fails

### GetAbsolutePath

Gets the absolute path including the root.

```go
func (fs *SimpleFS) GetAbsolutePath(path string) (string, error)
```

**Parameters:**
- `path`: The path to convert

**Returns:**
- The absolute path
- An error if the operation fails

### PathExists

Checks if a path exists.

```go
func (fs *SimpleFS) PathExists(path string) bool
```

**Parameters:**
- `path`: The path to check

**Returns:**
- `true` if the path exists, `false` otherwise

## Attributes

### SetAttribute

Sets an extended attribute on a file.

```go
func (fs *SimpleFS) SetAttribute(path, key, value string) error
```

**Parameters:**
- `path`: The path to the file
- `key`: The attribute key
- `value`: The attribute value

**Returns:**
- An error if the operation fails

### GetAttribute

Gets an extended attribute from a file.

```go
func (fs *SimpleFS) GetAttribute(path, key string) (string, error)
```

**Parameters:**
- `path`: The path to the file
- `key`: The attribute key

**Returns:**
- The attribute value
- An error if the operation fails

### GetAllAttributes

Gets all extended attributes from a file.

```go
func (fs *SimpleFS) GetAllAttributes(path string) (map[string]string, error)
```

**Parameters:**
- `path`: The path to the file

**Returns:**
- A map of attribute keys to values
- An error if the operation fails

### DeleteAttribute

Deletes an extended attribute from a file.

```go
func (fs *SimpleFS) DeleteAttribute(path, key string) error
```

**Parameters:**
- `path`: The path to the file
- `key`: The attribute key

**Returns:**
- An error if the operation fails

## Versioning

### ListVersions

Lists all versions of a file.

```go
func (fs *SimpleFS) ListVersions(path string) (*VersionListing, error)
```

**Parameters:**
- `path`: The path to the file

**Returns:**
- A list of versions
- An error if the operation fails

### GetVersion

Gets a specific version of a file.

```go
func (fs *SimpleFS) GetVersion(path, versionID string) ([]byte, *VersionInfo, error)
```

**Parameters:**
- `path`: The path to the file
- `versionID`: The version ID

**Returns:**
- The file content
- Version information
- An error if the operation fails

### RestoreVersion

Restores a file to a specific version.

```go
func (fs *SimpleFS) RestoreVersion(path, versionID string) error
```

**Parameters:**
- `path`: The path to the file
- `versionID`: The version ID

**Returns:**
- An error if the operation fails

### DeleteVersion

Deletes a specific version of a file.

```go
func (fs *SimpleFS) DeleteVersion(path, versionID string) error
```

**Parameters:**
- `path`: The path to the file
- `versionID`: The version ID

**Returns:**
- An error if the operation fails

### SetVersionDescription

Sets a description for a specific version.

```go
func (fs *SimpleFS) SetVersionDescription(path, versionID, description string) error
```

**Parameters:**
- `path`: The path to the file
- `versionID`: The version ID
- `description`: The description

**Returns:**
- An error if the operation fails

## Hooks

### RegisterHook

Registers a hook function for a specific operation and hook type.

```go
func (fs *SimpleFS) RegisterHook(op OperationType, typ HookType, hook HookFunc)
```

**Parameters:**
- `op`: The operation type
- `typ`: The hook type (pre or post)
- `hook`: The hook function

### UnregisterHook

Unregisters all hooks for a specific operation and hook type.

```go
func (fs *SimpleFS) UnregisterHook(op OperationType, typ HookType)
```

**Parameters:**
- `op`: The operation type
- `typ`: The hook type (pre or post)

### UnregisterAllHooks

Unregisters all hooks.

```go
func (fs *SimpleFS) UnregisterAllHooks()
```

### LoggingHook

Creates a hook that logs operations to a file.

```go
func LoggingHook(logPath string) (HookFunc, error)
```

**Parameters:**
- `logPath`: The path to the log file

**Returns:**
- A hook function
- An error if the operation fails

### ReadOnlyHook

Creates a hook that prevents write operations.

```go
func ReadOnlyHook() HookFunc
```

**Returns:**
- A hook function

### BackupHook

Creates a hook that backs up files before modification.

```go
func BackupHook(backupDir string) (HookFunc, error)
```

**Parameters:**
- `backupDir`: The directory to store backups

**Returns:**
- A hook function
- An error if the operation fails

## Explicit Locking

### WithExplicitLocking

Adds explicit locking capability to the filesystem.

```go
func (fs *SimpleFS) WithExplicitLocking() *SimpleFS
```

**Returns:**
- The filesystem with explicit locking enabled

### LockFile

Acquires an explicit lock on a file.

```go
func (fs *SimpleFS) LockFile(path, owner string, lockType LockType, timeout time.Duration) (*LockInfo, error)
```

**Parameters:**
- `path`: The path to the file
- `owner`: The lock owner identifier
- `lockType`: The type of lock (read or write)
- `timeout`: How long the lock should last (0 for no timeout)

**Returns:**
- Lock information
- An error if the operation fails

### UnlockFile

Releases an explicit lock on a file.

```go
func (fs *SimpleFS) UnlockFile(path, owner string) error
```

**Parameters:**
- `path`: The path to the file
- `owner`: The lock owner identifier

**Returns:**
- An error if the operation fails

### IsFileLocked

Checks if a file has an explicit lock.

```go
func (fs *SimpleFS) IsFileLocked(path string) bool
```

**Parameters:**
- `path`: The path to the file

**Returns:**
- `true` if the file is locked, `false` otherwise

### GetFileLockInfo

Gets information about a file's lock.

```go
func (fs *SimpleFS) GetFileLockInfo(path string) (*LockInfo, bool)
```

**Parameters:**
- `path`: The path to the file

**Returns:**
- Lock information
- `true` if the file is locked, `false` otherwise

### WaitForFileLock

Waits for a file's lock to be released.

```go
func (fs *SimpleFS) WaitForFileLock(path string, waitTime time.Duration) bool
```

**Parameters:**
- `path`: The path to the file
- `waitTime`: How long to wait

**Returns:**
- `true` if the lock was released, `false` if timed out

## Error Handling

Most methods return an error as the last return value. These errors should be checked to ensure operations complete successfully.

Common error types:
- Path validation errors: file not found, invalid path, path escaping root
- Permission errors: unable to read/write due to OS permissions
- Concurrency errors: file locked by another operation
- Hook errors: operation rejected by a pre-hook
- Version errors: version not found or versioning not enabled
