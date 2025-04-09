package fs

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileInfo represents metadata about a file
type FileInfo struct {
	Name       string            // The base name of the file
	Size       int64             // Length in bytes
	ModTime    time.Time         // Modification time
	IsDir      bool              // Is this a directory?
	Mode       os.FileMode       // File mode bits
	Attributes map[string]string // Extended attributes
}

// SimpleFS represents our file system
type SimpleFS struct {
	rootPath    string                   // Root directory of the file system
	journal     *Journal                 // Journal for crash recovery
	locks       map[string]*sync.RWMutex // File-level locks for concurrency control
	locksGuard  sync.Mutex               // Guard for the locks map
	hooks       map[HookKey][]HookFunc   // Registered hooks
	hooksGuard  sync.RWMutex             // Guard for the hooks map
	versioning  bool                     // Whether versioning is enabled
	versionPath string                   // Path to store versions
	maxVersions int                      // Maximum number of versions to keep
	lockManager *ExplicitLockManager
}

// Options configures the file system
type Options struct {
	EnableJournaling bool // Whether to enable journaling
	EnableVersioning bool // Whether to enable versioning
	MaxVersions      int  // Maximum number of versions to keep (0 = unlimited)
}

// DefaultOptions returns the default options
func DefaultOptions() *Options {
	return &Options{
		EnableJournaling: true,
		EnableVersioning: false,
		MaxVersions:      10,
	}
}

// NewSimpleFS creates a new file system with the given root path
func NewSimpleFS(rootPath string, opts *Options) (*SimpleFS, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	// Validate root path
	absRootPath, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, fmt.Errorf("invalid root path: %w", err)
	}

	// Check if root path exists, if not create it
	if _, err := os.Stat(absRootPath); os.IsNotExist(err) {
		err = os.MkdirAll(absRootPath, 0755)
		if err != nil {
			return nil, fmt.Errorf("failed to create root directory: %w", err)
		}
	}

	fs := &SimpleFS{
		rootPath:    absRootPath,
		locks:       make(map[string]*sync.RWMutex),
		hooks:       make(map[HookKey][]HookFunc),
		versioning:  opts.EnableVersioning,
		maxVersions: opts.MaxVersions,
	}

	// Setup journaling if enabled
	if opts.EnableJournaling {
		// Create journal directory
		journalPath := filepath.Join(absRootPath, ".journal")
		if _, err := os.Stat(journalPath); os.IsNotExist(err) {
			err = os.MkdirAll(journalPath, 0755)
			if err != nil {
				return nil, fmt.Errorf("failed to create journal directory: %w", err)
			}
		}

		// Initialize journal
		journal, err := NewJournal(filepath.Join(journalPath, "fs.log"))
		if err != nil {
			return nil, fmt.Errorf("failed to initialize journal: %w", err)
		}
		fs.journal = journal
	}

	// Setup versioning if enabled
	if opts.EnableVersioning {
		// Create versions directory
		versionPath := filepath.Join(absRootPath, ".versions")
		if _, err := os.Stat(versionPath); os.IsNotExist(err) {
			err = os.MkdirAll(versionPath, 0755)
			if err != nil {
				return nil, fmt.Errorf("failed to create versions directory: %w", err)
			}
		}
		fs.versionPath = versionPath
	}

	return fs, nil
}

// Close properly closes the file system
func (fs *SimpleFS) Close() error {
	if fs.journal != nil {
		return fs.journal.Close()
	}
	return nil
}

// getFileLock returns a lock for the given path, creating one if it doesn't exist
func (fs *SimpleFS) getFileLock(path string) *sync.RWMutex {
	fs.locksGuard.Lock()
	defer fs.locksGuard.Unlock()

	if lock, exists := fs.locks[path]; exists {
		return lock
	}

	lock := &sync.RWMutex{}
	fs.locks[path] = lock
	return lock
}

// fullPath returns the absolute path for a given relative path
func (fs *SimpleFS) fullPath(path string) (string, error) {
	// Clean the path to remove any ../ components
	cleanPath := filepath.Clean(path)

	// Ensure the path doesn't escape the root
	fullPath := filepath.Join(fs.rootPath, cleanPath)
	rel, err := filepath.Rel(fs.rootPath, fullPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// Check if the path attempts to escape the root
	if rel == ".." || filepath.HasPrefix(rel, "../") {
		return "", errors.New("path attempts to escape the root directory")
	}

	return fullPath, nil
}

// CreateDir creates a new directory
func (fs *SimpleFS) CreateDir(path string) error {
	fullPath, err := fs.fullPath(path)
	if err != nil {
		return err
	}

	// Lock the directory
	dirLock := fs.getFileLock(filepath.Dir(fullPath))
	dirLock.Lock()
	defer dirLock.Unlock()

	// Execute pre-hooks
	ctx := &HookContext{
		Operation: OpCreateDir,
		Path:      path,
	}
	if err := fs.executeHooks(HookTypePre, ctx); err != nil {
		return err
	}

	if fs.journal != nil {
		err := fs.journal.Log(JournalEntry{
			Operation: "mkdir",
			Path:      path,
			Timestamp: time.Now(),
		})
		if err != nil {
			return fmt.Errorf("failed to log directory creation: %w", err)
		}
	}

	err = os.MkdirAll(fullPath, 0755)
	if err != nil {
		return err
	}

	return fs.executeHooks(HookTypePost, ctx)
}

// WriteFile writes data to a file, creating it if it doesn't exist
func (fs *SimpleFS) WriteFile(path string, data []byte) error {
	return fs.WriteFileWithMode(path, data, 0644)
}

// WriteFileWithMode writes data to a file with specific permissions
func (fs *SimpleFS) WriteFileWithMode(path string, data []byte, mode os.FileMode) error {
	// Get full path
	fullPath, err := fs.fullPath(path)
	if err != nil {
		return err
	}

	// Create parent directories if they don't exist
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directories: %w", err)
	}

	// Lock the file for writing
	fileLock := fs.getFileLock(fullPath)
	fileLock.Lock()
	defer fileLock.Unlock()

	ctx := &HookContext{
		Operation: OpWriteFile,
		Path:      path,
		Data:      data,
		Mode:      mode,
	}
	if err := fs.executeHooks(HookTypePre, ctx); err != nil {
		return err
	}

	// Version the file if enabled and it exists
	if fs.versioning && fs.FileExists(path) {
		if err := fs.createVersion(path); err != nil {
			return fmt.Errorf("failed to create version: %w", err)
		}
	}

	if fs.journal != nil {
		err := fs.journal.Log(JournalEntry{
			Operation: "write",
			Path:      path,
			Data:      data,
			Timestamp: time.Now(),
			Attributes: map[string]string{
				"mode": fmt.Sprintf("%d", mode),
			},
		})
		if err != nil {
			return fmt.Errorf("failed to log file write: %w", err)
		}
	}

	// Write the file
	err = os.WriteFile(fullPath, data, mode)
	if err != nil {
		return err
	}

	return fs.executeHooks(HookTypePost, ctx)
}

// ReadFile reads the content of a file
func (fs *SimpleFS) ReadFile(path string) ([]byte, error) {
	fullPath, err := fs.fullPath(path)
	if err != nil {
		return nil, err
	}

	// Lock the file for reading
	fileLock := fs.getFileLock(fullPath)
	fileLock.RLock()
	defer fileLock.RUnlock()

	ctx := &HookContext{
		Operation: OpReadFile,
		Path:      path,
	}
	if err := fs.executeHooks(HookTypePre, ctx); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}

	if err := fs.executeHooks(HookTypePost, ctx); err != nil {
		return nil, err
	}

	return data, nil
}

// ListDir lists the contents of a directory
func (fs *SimpleFS) ListDir(path string) ([]FileInfo, error) {
	fullPath, err := fs.fullPath(path)
	if err != nil {
		return nil, err
	}

	dirLock := fs.getFileLock(fullPath)
	dirLock.RLock()
	defer dirLock.RUnlock()

	ctx := &HookContext{
		Operation: OpListDir,
		Path:      path,
	}
	if err := fs.executeHooks(HookTypePre, ctx); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}

	infos := make([]FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue // Skip entries we can't get info for
		}

		entryPath := filepath.Join(path, info.Name())

		// Skip hidden files/directories
		if info.Name() == ".journal" ||
			info.Name() == ".attributes" ||
			info.Name() == ".versions" {
			continue
		}

		// Read extended attributes if any
		attrs, _ := fs.GetAllAttributes(entryPath)

		infos = append(infos, FileInfo{
			Name:       info.Name(),
			Size:       info.Size(),
			ModTime:    info.ModTime(),
			IsDir:      info.IsDir(),
			Mode:       info.Mode(),
			Attributes: attrs,
		})
	}

	if err := fs.executeHooks(HookTypePost, ctx); err != nil {
		return nil, err
	}

	return infos, nil
}

// DeleteFile removes a file
func (fs *SimpleFS) DeleteFile(path string) error {
	fullPath, err := fs.fullPath(path)
	if err != nil {
		return err
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return errors.New("cannot delete directory with DeleteFile, use DeleteDir instead")
	}

	// Lock the file for writing
	fileLock := fs.getFileLock(fullPath)
	fileLock.Lock()
	defer fileLock.Unlock()

	// Lock the parent directory too
	dirLock := fs.getFileLock(filepath.Dir(fullPath))
	dirLock.Lock()
	defer dirLock.Unlock()

	ctx := &HookContext{
		Operation: OpDeleteFile,
		Path:      path,
	}
	if err := fs.executeHooks(HookTypePre, ctx); err != nil {
		return err
	}

	// Version the file if enabled
	if fs.versioning {
		if err := fs.createVersion(path); err != nil {
			return fmt.Errorf("failed to create version before deletion: %w", err)
		}
	}

	if fs.journal != nil {
		err := fs.journal.Log(JournalEntry{
			Operation: "delete",
			Path:      path,
			Timestamp: time.Now(),
		})
		if err != nil {
			return fmt.Errorf("failed to log file deletion: %w", err)
		}
	}

	err = os.Remove(fullPath)
	if err != nil {
		return err
	}

	return fs.executeHooks(HookTypePost, ctx)
}

// DeleteDir removes a directory and all its contents
func (fs *SimpleFS) DeleteDir(path string) error {
	fullPath, err := fs.fullPath(path)
	if err != nil {
		return err
	}

	dirLock := fs.getFileLock(fullPath)
	dirLock.Lock()
	defer dirLock.Unlock()

	parentLock := fs.getFileLock(filepath.Dir(fullPath))
	parentLock.Lock()
	defer parentLock.Unlock()

	ctx := &HookContext{
		Operation: OpDeleteDir,
		Path:      path,
	}
	if err := fs.executeHooks(HookTypePre, ctx); err != nil {
		return err
	}

	if fs.journal != nil {
		err := fs.journal.Log(JournalEntry{
			Operation: "delete",
			Path:      path,
			Timestamp: time.Now(),
		})
		if err != nil {
			return fmt.Errorf("failed to log directory deletion: %w", err)
		}
	}

	err = os.RemoveAll(fullPath)
	if err != nil {
		return err
	}

	return fs.executeHooks(HookTypePost, ctx)
}

// CopyFile copies a file from src to dst
func (fs *SimpleFS) CopyFile(src, dst string) error {
	srcPath, err := fs.fullPath(src)
	if err != nil {
		return err
	}

	dstPath, err := fs.fullPath(dst)
	if err != nil {
		return err
	}

	// Lock both files
	srcLock := fs.getFileLock(srcPath)
	srcLock.RLock()
	defer srcLock.RUnlock()

	dstLock := fs.getFileLock(dstPath)
	dstLock.Lock()
	defer dstLock.Unlock()

	ctx := &HookContext{
		Operation: OpCopyFile,
		Path:      dst,
		SrcPath:   src,
	}
	if err := fs.executeHooks(HookTypePre, ctx); err != nil {
		return err
	}

	// Version the destination file if it exists
	if fs.versioning && fs.FileExists(dst) {
		if err := fs.createVersion(dst); err != nil {
			return fmt.Errorf("failed to create version of destination: %w", err)
		}
	}

	sourceFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directories: %w", err)
	}

	destFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	sourceInfo, err := os.Stat(srcPath)
	if err != nil {
		return err
	}

	// Set permissions on destination file
	if err = os.Chmod(dstPath, sourceInfo.Mode()); err != nil {
		return err
	}

	// Copy extended attributes
	attrs, err := fs.GetAllAttributes(src)
	if err == nil && len(attrs) > 0 {
		for k, v := range attrs {
			fs.SetAttribute(dst, k, v)
		}
	}

	return fs.executeHooks(HookTypePost, ctx)
}

// MoveFile moves a file from src to dst
func (fs *SimpleFS) MoveFile(src, dst string) error {
	srcPath, err := fs.fullPath(src)
	if err != nil {
		return err
	}

	dstPath, err := fs.fullPath(dst)
	if err != nil {
		return err
	}

	// Lock both files and parent directories
	srcLock := fs.getFileLock(srcPath)
	srcLock.Lock()
	defer srcLock.Unlock()

	dstLock := fs.getFileLock(dstPath)
	dstLock.Lock()
	defer dstLock.Unlock()

	srcDirLock := fs.getFileLock(filepath.Dir(srcPath))
	srcDirLock.Lock()
	defer srcDirLock.Unlock()

	dstDirLock := fs.getFileLock(filepath.Dir(dstPath))
	dstDirLock.Lock()
	defer dstDirLock.Unlock()

	ctx := &HookContext{
		Operation: OpMoveFile,
		Path:      dst,
		SrcPath:   src,
	}
	if err := fs.executeHooks(HookTypePre, ctx); err != nil {
		return err
	}

	// Version the destination file if it exists
	if fs.versioning && fs.FileExists(dst) {
		if err := fs.createVersion(dst); err != nil {
			return fmt.Errorf("failed to create version of destination before move: %w", err)
		}
	}

	dstDir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	attrs, _ := fs.GetAllAttributes(src)

	if fs.journal != nil {
		// Log source deletion
		if err := fs.journal.Log(JournalEntry{
			Operation: "delete",
			Path:      src,
			Timestamp: time.Now(),
		}); err != nil {
			return fmt.Errorf("failed to log move source deletion: %w", err)
		}

		// Read the file content to log in case move fails
		data, err := fs.ReadFile(src)
		if err != nil {
			return fmt.Errorf("failed to read source file for move: %w", err)
		}

		if err := fs.journal.Log(JournalEntry{
			Operation:  "write",
			Path:       dst,
			Data:       data,
			Timestamp:  time.Now(),
			Attributes: attrs,
		}); err != nil {
			return fmt.Errorf("failed to log move destination write: %w", err)
		}
	}

	// Move the file
	if err := os.Rename(srcPath, dstPath); err != nil {
		return err
	}

	// Set attributes on the new file
	if len(attrs) > 0 {
		for k, v := range attrs {
			fs.SetAttribute(dst, k, v)
		}
	}

	return fs.executeHooks(HookTypePost, ctx)
}

// FileExists checks if a file exists
func (fs *SimpleFS) FileExists(path string) bool {
	fullPath, err := fs.fullPath(path)
	if err != nil {
		return false
	}

	_, err = os.Stat(fullPath)
	return err == nil
}

// Stat returns file info
func (fs *SimpleFS) Stat(path string) (*FileInfo, error) {
	fullPath, err := fs.fullPath(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, err
	}

	attrs, _ := fs.GetAllAttributes(path)

	return &FileInfo{
		Name:       info.Name(),
		Size:       info.Size(),
		ModTime:    info.ModTime(),
		IsDir:      info.IsDir(),
		Mode:       info.Mode(),
		Attributes: attrs,
	}, nil
}

// Recover attempts to recover from a crash by replaying the journal
func (fs *SimpleFS) Recover() error {
	if fs.journal == nil {
		return errors.New("journaling is not enabled")
	}

	return fs.journal.Recover(fs)
}
