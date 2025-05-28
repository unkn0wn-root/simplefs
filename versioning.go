package fs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/unkn0wn-root/simplefs/internal/utils"
)

// VersionInfo contains information about a file version
type VersionInfo struct {
	VersionID   string            // Unique ID for this version
	Path        string            // Original file path
	CreatedAt   time.Time         // When this version was created
	Size        int64             // Size in bytes
	Attributes  map[string]string // Attributes at time of versioning
	Description string            // Optional description
}

// VersionListing is a list of versions for a file
type VersionListing struct {
	Path     string        // File path
	Versions []VersionInfo // Available versions
}

// createVersion creates a new version of a file
func (fs *SimpleFS) createVersion(path string) error {
	if !fs.versioning {
		return fmt.Errorf("versioning is not enabled")
	}

	// pre-hooks
	ctx := &HookContext{
		Operation: OpCreateVersion,
		Path:      path,
	}
	if err := fs.executeHooks(HookTypePre, ctx); err != nil {
		return err
	}

	fullPath, err := fs.fullPath(path)
	if err != nil {
		return err
	}

	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Only version regular files
	if fileInfo.IsDir() {
		return fmt.Errorf("cannot version a directory")
	}

	data, err := fs.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	attrs, _ := fs.GetAllAttributes(path)
	// Create a unique ID for this version
	versionID := uuid.New().String()
	versionInfo := VersionInfo{
		VersionID:  versionID,
		Path:       path,
		CreatedAt:  getNow(),
		Size:       int64(len(data)),
		Attributes: attrs,
	}

	// Hash the path to create a directory
	hashedPath := utils.HashString(path)
	versionDir := filepath.Join(fs.versionPath, hashedPath)

	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return fmt.Errorf("failed to create version directory: %w", err)
	}

	dataPath := filepath.Join(versionDir, versionID+".data")
	if err := os.WriteFile(dataPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write version data: %w", err)
	}

	metaPath := filepath.Join(versionDir, versionID+".json")
	metaData, err := json.MarshalIndent(versionInfo, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal version metadata: %w", err)
	}

	if err := os.WriteFile(metaPath, metaData, 0644); err != nil {
		return fmt.Errorf("failed to write version metadata: %w", err)
	}

	// Prune old versions if maxVersions is set
	if fs.maxVersions > 0 {
		if err := fs.pruneVersions(path); err != nil {
			return fmt.Errorf("failed to prune old versions: %w", err)
		}
	}

	return fs.executeHooks(HookTypePost, ctx)
}

// ListVersions lists all versions of a file
func (fs *SimpleFS) ListVersions(path string) (*VersionListing, error) {
	if !fs.versioning {
		return nil, fmt.Errorf("versioning is not enabled")
	}

	// pre-hooks
	ctx := &HookContext{
		Operation: OpListVersions,
		Path:      path,
	}
	if err := fs.executeHooks(HookTypePre, ctx); err != nil {
		return nil, err
	}

	// Hash the path to find the version directory
	hashedPath := utils.HashString(path)
	versionDir := filepath.Join(fs.versionPath, hashedPath)

	if _, err := os.Stat(versionDir); os.IsNotExist(err) {
		// No versions found
		return &VersionListing{
			Path:     path,
			Versions: []VersionInfo{},
		}, nil
	}

	entries, err := os.ReadDir(versionDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read version directory: %w", err)
	}

	versions := make([]VersionInfo, 0)
	for _, entry := range entries {
		// Only process metadata files
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		metaPath := filepath.Join(versionDir, entry.Name())
		metaData, err := os.ReadFile(metaPath)
		if err != nil {
			continue // Skip if can't read
		}

		var version VersionInfo
		if err := json.Unmarshal(metaData, &version); err != nil {
			continue // Skip if can't parse
		}

		versions = append(versions, version)
	}

	// Sort versions by creation time (newest first)
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].CreatedAt.After(versions[j].CreatedAt)
	})

	listing := &VersionListing{
		Path:     path,
		Versions: versions,
	}

	if err := fs.executeHooks(HookTypePost, ctx); err != nil {
		return nil, err
	}

	return listing, nil
}

// GetVersion gets a specific version of a file
func (fs *SimpleFS) GetVersion(path, versionID string) ([]byte, *VersionInfo, error) {
	if !fs.versioning {
		return nil, nil, fmt.Errorf("versioning is not enabled")
	}

	ctx := &HookContext{
		Operation: OpGetVersion,
		Path:      path,
	}
	if err := fs.executeHooks(HookTypePre, ctx); err != nil {
		return nil, nil, err
	}

	hashedPath := utils.HashString(path)
	versionDir := filepath.Join(fs.versionPath, hashedPath)

	metaPath := filepath.Join(versionDir, versionID+".json")
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("version not found")
	}

	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read version metadata: %w", err)
	}

	var version VersionInfo
	if err := json.Unmarshal(metaData, &version); err != nil {
		return nil, nil, fmt.Errorf("failed to parse version metadata: %w", err)
	}

	dataPath := filepath.Join(versionDir, versionID+".data")
	data, err := os.ReadFile(dataPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read version data: %w", err)
	}

	if err := fs.executeHooks(HookTypePost, ctx); err != nil {
		return nil, nil, err
	}

	return data, &version, nil
}

// RestoreVersion restores a file to a specific version
func (fs *SimpleFS) RestoreVersion(path, versionID string) error {
	if !fs.versioning {
		return fmt.Errorf("versioning is not enabled")
	}

	data, version, err := fs.GetVersion(path, versionID)
	if err != nil {
		return fmt.Errorf("failed to get version: %w", err)
	}

	if fs.FileExists(path) {
		if err := fs.createVersion(path); err != nil {
			return fmt.Errorf("failed to create version of current file: %w", err)
		}
	}

	// Write the file with the version data
	if err := fs.WriteFile(path, data); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Restore attributes if any
	if len(version.Attributes) > 0 {
		for k, v := range version.Attributes {
			if err := fs.SetAttribute(path, k, v); err != nil {
				return fmt.Errorf("failed to restore attribute %s: %w", k, err)
			}
		}
	}

	return nil
}

// DeleteVersion deletes a specific version of a file
func (fs *SimpleFS) DeleteVersion(path, versionID string) error {
	if !fs.versioning {
		return fmt.Errorf("versioning is not enabled")
	}

	hashedPath := utils.HashString(path)
	versionDir := filepath.Join(fs.versionPath, hashedPath)

	metaPath := filepath.Join(versionDir, versionID+".json")
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		return fmt.Errorf("version not found")
	}

	if err := os.Remove(metaPath); err != nil {
		return fmt.Errorf("failed to delete version metadata: %w", err)
	}

	dataPath := filepath.Join(versionDir, versionID+".data")
	if err := os.Remove(dataPath); err != nil {
		return fmt.Errorf("failed to delete version data: %w", err)
	}

	return nil
}

// pruneVersions removes old versions to keep the version count within limits
func (fs *SimpleFS) pruneVersions(path string) error {
	// Get all versions
	listing, err := fs.ListVersions(path)
	if err != nil {
		return err
	}

	// Check if pruning is needed
	if len(listing.Versions) <= fs.maxVersions {
		return nil
	}

	// Determine how many versions to delete
	toDelete := len(listing.Versions) - fs.maxVersions
	for i := len(listing.Versions) - 1; i >= len(listing.Versions)-toDelete; i-- {
		version := listing.Versions[i]
		if err := fs.DeleteVersion(path, version.VersionID); err != nil {
			return err
		}
	}

	return nil
}

// SetVersionDescription sets a description for a specific version
func (fs *SimpleFS) SetVersionDescription(path, versionID, description string) error {
	if !fs.versioning {
		return fmt.Errorf("versioning is not enabled")
	}

	hashedPath := utils.HashString(path)
	versionDir := filepath.Join(fs.versionPath, hashedPath)

	metaPath := filepath.Join(versionDir, versionID+".json")
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		return fmt.Errorf("version not found")
	}

	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		return fmt.Errorf("failed to read version metadata: %w", err)
	}

	var version VersionInfo
	if err := json.Unmarshal(metaData, &version); err != nil {
		return fmt.Errorf("failed to parse version metadata: %w", err)
	}

	version.Description = description

	// Write back metadata
	updatedMetaData, err := json.MarshalIndent(version, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated metadata: %w", err)
	}

	if err := os.WriteFile(metaPath, updatedMetaData, 0644); err != nil {
		return fmt.Errorf("failed to write updated metadata: %w", err)
	}

	return nil
}
