package fs

import (
	"errors"
	"os"
	"path/filepath"
)

// OperationType defines the type of filesystem operation
type OperationType string

// HookType defines when a hook should be executed
type HookType string

// HookKey is a composite key for hooks
type HookKey struct {
	Op  OperationType
	Typ HookType
}

// HookFunc is a function that is called before or after an operation
type HookFunc func(ctx *HookContext) error

// HookContext provides context to hook functions
type HookContext struct {
	Operation OperationType          // Type of operation
	Path      string                 // Path of the file or directory
	SrcPath   string                 // Source path for copy/move operations
	Data      []byte                 // Data for write operations
	Mode      os.FileMode            // File mode for write operations
	Key       string                 // Key for attribute operations
	Value     string                 // Value for attribute operations
	Error     error                  // Error from the operation (in post hooks)
	FS        *SimpleFS              // Reference to the filesystem
	Custom    map[string]interface{} // Custom data for hooks
}

// Operation types
const (
	OpCreateDir        OperationType = "createDir"
	OpWriteFile        OperationType = "writeFile"
	OpReadFile         OperationType = "readFile"
	OpListDir          OperationType = "listDir"
	OpDeleteFile       OperationType = "deleteFile"
	OpDeleteDir        OperationType = "deleteDir"
	OpCopyFile         OperationType = "copyFile"
	OpMoveFile         OperationType = "moveFile"
	OpSetAttribute     OperationType = "setAttribute"
	OpGetAttribute     OperationType = "getAttribute"
	OpGetAllAttributes OperationType = "getAllAttributes"
	OpDeleteAttribute  OperationType = "deleteAttribute"
	OpCreateVersion    OperationType = "createVersion"
	OpGetVersion       OperationType = "getVersion"
	OpListVersions     OperationType = "listVersions"
)

// Hook types
const (
	HookTypePre  HookType = "pre"  // Executed before an operation
	HookTypePost HookType = "post" // Executed after an operation
)

// RegisterHook registers a hook function for a specific operation and hook type
func (fs *SimpleFS) RegisterHook(op OperationType, typ HookType, hook HookFunc) {
	fs.hooksGuard.Lock()
	defer fs.hooksGuard.Unlock()

	key := HookKey{Op: op, Typ: typ}

	if fs.hooks == nil {
		fs.hooks = make(map[HookKey][]HookFunc)
	}

	fs.hooks[key] = append(fs.hooks[key], hook)
}

// UnregisterHook unregisters all hooks for a specific operation and hook type
func (fs *SimpleFS) UnregisterHook(op OperationType, typ HookType) {
	fs.hooksGuard.Lock()
	defer fs.hooksGuard.Unlock()

	key := HookKey{Op: op, Typ: typ}
	delete(fs.hooks, key)
}

// UnregisterAllHooks unregisters all hooks
func (fs *SimpleFS) UnregisterAllHooks() {
	fs.hooksGuard.Lock()
	defer fs.hooksGuard.Unlock()

	fs.hooks = make(map[HookKey][]HookFunc)
}

// executeHooks executes all hooks for a specific operation and hook type
func (fs *SimpleFS) executeHooks(typ HookType, ctx *HookContext) error {
	fs.hooksGuard.RLock()
	defer fs.hooksGuard.RUnlock()

	if fs.hooks == nil {
		return nil
	}

	// Set filesystem reference in context
	ctx.FS = fs

	if ctx.Custom == nil {
		ctx.Custom = make(map[string]interface{})
	}

	key := HookKey{Op: ctx.Operation, Typ: typ}
	hooks, ok := fs.hooks[key]
	if !ok {
		// No hooks registered for this operation and type
		return nil
	}

	for _, hook := range hooks {
		if err := hook(ctx); err != nil {
			return err
		}
	}

	return nil
}

// LoggingHook creates a hook that logs operations to a file
func LoggingHook(logPath string) (HookFunc, error) {
	// Create or open log file
	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	// Close file when done
	// This means the file will remain open as long as the hook exists

	return func(ctx *HookContext) error {
		var message string
		switch ctx.Operation {
		case OpCreateDir:
			message = "CREATE_DIR " + ctx.Path
		case OpWriteFile:
			message = "WRITE_FILE " + ctx.Path + " (" + string(len(ctx.Data)) + " bytes)"
		case OpReadFile:
			message = "READ_FILE " + ctx.Path
		case OpListDir:
			message = "LIST_DIR " + ctx.Path
		case OpDeleteFile:
			message = "DELETE_FILE " + ctx.Path
		case OpDeleteDir:
			message = "DELETE_DIR " + ctx.Path
		case OpCopyFile:
			message = "COPY_FILE " + ctx.SrcPath + " -> " + ctx.Path
		case OpMoveFile:
			message = "MOVE_FILE " + ctx.SrcPath + " -> " + ctx.Path
		case OpSetAttribute:
			message = "SET_ATTR " + ctx.Path + " [" + ctx.Key + "=" + ctx.Value + "]"
		case OpGetAttribute:
			message = "GET_ATTR " + ctx.Path + " [" + ctx.Key + "]"
		case OpGetAllAttributes:
			message = "GET_ALL_ATTRS " + ctx.Path
		case OpDeleteAttribute:
			message = "DELETE_ATTR " + ctx.Path + " [" + ctx.Key + "]"
		case OpCreateVersion:
			message = "CREATE_VERSION " + ctx.Path
		case OpGetVersion:
			message = "GET_VERSION " + ctx.Path
		case OpListVersions:
			message = "LIST_VERSIONS " + ctx.Path
		default:
			message = string(ctx.Operation) + " " + ctx.Path
		}

		message = getNow().Format("2006-01-02 15:04:05") + " " + message + "\n"
		_, err := logFile.WriteString(message)
		return err
	}, nil
}

// ReadOnlyHook creates a hook that prevents write operations
func ReadOnlyHook() HookFunc {
	return func(ctx *HookContext) error {
		// Allow read operations
		switch ctx.Operation {
		case OpReadFile, OpListDir, OpGetAttribute, OpGetAllAttributes, OpGetVersion, OpListVersions:
			return nil
		}

		// Deny write operations
		return errors.New("filesystem is read-only")
	}
}

// BackupHook creates a hook that backs up files before modification
func BackupHook(backupDir string) (HookFunc, error) {
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, err
	}

	return func(ctx *HookContext) error {
		// Only handle pre-write operations
		switch ctx.Operation {
		case OpWriteFile, OpDeleteFile, OpMoveFile:
			// Only if file exists
			if !ctx.FS.FileExists(ctx.Path) {
				return nil
			}

			data, err := ctx.FS.ReadFile(ctx.Path)
			if err != nil {
				return nil // Skip if can't read
			}

			backupPath := filepath.Join(backupDir, filepath.Base(ctx.Path)+"."+getNow().Format("20060102-150405"))
			return os.WriteFile(backupPath, data, 0644)
		}

		return nil
	}, nil
}
