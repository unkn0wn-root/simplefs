package fs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// IsValidPath checks if a path is valid and doesn't attempt to escape the root
func (fs *SimpleFS) IsValidPath(path string) (bool, error) {
	// Check for empty path
	if path == "" {
		return false, errors.New("empty path")
	}

	// Get the full path
	fullPath, err := fs.fullPath(path)
	if err != nil {
		return false, err
	}

	// Check if the path is inside the root
	rel, err := filepath.Rel(fs.rootPath, fullPath)
	if err != nil {
		return false, fmt.Errorf("invalid path: %w", err)
	}

	// Check if path attempts to escape root
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return false, errors.New("path attempts to escape the root directory")
	}

	return true, nil
}

// Sanitize sanitizes a path to ensure it's safe to use
func SanitizePath(path string) string {
	// Clean the path to remove any ".." components
	path = filepath.Clean(path)

	// Remove leading slash if present
	path = strings.TrimPrefix(path, "/")

	return path
}

// JoinPath safely joins path components
func JoinPath(components ...string) string {
	// Join components
	path := filepath.Join(components...)

	// Sanitize the result
	return SanitizePath(path)
}

// Split splits a path into its components
func SplitPath(path string) []string {
	// Sanitize the path first
	path = SanitizePath(path)

	// Split the path
	if path == "" {
		return []string{}
	}

	return strings.Split(path, string(filepath.Separator))
}

// GetRelativePath gets the relative path from the root
func (fs *SimpleFS) GetRelativePath(path string) (string, error) {
	// Get the full path
	fullPath, err := fs.fullPath(path)
	if err != nil {
		return "", err
	}

	// Get the relative path
	rel, err := filepath.Rel(fs.rootPath, fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}

	return rel, nil
}

// GetAbsolutePath gets the absolute path including the root
func (fs *SimpleFS) GetAbsolutePath(path string) (string, error) {
	return fs.fullPath(path)
}

// GetPathInfo returns information about a path
type PathInfo struct {
	Path       string   // Original path
	Absolute   string   // Absolute path
	Relative   string   // Relative path from root
	Exists     bool     // Whether the path exists
	IsDir      bool     // Whether the path is a directory
	Size       int64    // Size in bytes (if a file)
	Components []string // Path components
}

// GetPathInfo returns information about a path
func (fs *SimpleFS) GetPathInfo(path string) (*PathInfo, error) {
	// Sanitize the path
	path = SanitizePath(path)

	// Get the absolute path
	abs, err := fs.GetAbsolutePath(path)
	if err != nil {
		return nil, err
	}

	// Get the relative path
	rel, err := fs.GetRelativePath(path)
	if err != nil {
		return nil, err
	}

	// Get components
	components := SplitPath(path)

	// Check if the path exists
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			// Path doesn't exist
			return &PathInfo{
				Path:       path,
				Absolute:   abs,
				Relative:   rel,
				Exists:     false,
				Components: components,
			}, nil
		}
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	// Path exists
	return &PathInfo{
		Path:       path,
		Absolute:   abs,
		Relative:   rel,
		Exists:     true,
		IsDir:      info.IsDir(),
		Size:       info.Size(),
		Components: components,
	}, nil
}

// GetParentPath gets the parent path of a path
func (fs *SimpleFS) GetParentPath(path string) string {
	// Sanitize the path
	path = SanitizePath(path)

	// Get the parent
	return filepath.Dir(path)
}

// GetBasename gets the basename of a path
func (fs *SimpleFS) GetBasename(path string) string {
	// Sanitize the path
	path = SanitizePath(path)

	// Get the basename
	return filepath.Base(path)
}

// PathExists checks if a path exists
func (fs *SimpleFS) PathExists(path string) bool {
	// Get the full path
	fullPath, err := fs.fullPath(path)
	if err != nil {
		return false
	}

	// Check if the path exists
	_, err = os.Stat(fullPath)
	return err == nil
}

// IsDir checks if a path is a directory
func (fs *SimpleFS) IsDir(path string) bool {
	// Get the full path
	fullPath, err := fs.fullPath(path)
	if err != nil {
		return false
	}

	// Check if the path is a directory
	info, err := os.Stat(fullPath)
	if err != nil {
		return false
	}

	return info.IsDir()
}

// IsFile checks if a path is a file
func (fs *SimpleFS) IsFile(path string) bool {
	// Get the full path
	fullPath, err := fs.fullPath(path)
	if err != nil {
		return false
	}

	// Check if the path is a file
	info, err := os.Stat(fullPath)
	if err != nil {
		return false
	}

	return !info.IsDir()
}
