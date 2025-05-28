package fs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/unkn0wn-root/simplefs/internal/utils"
)

// SetAttribute sets an extended attribute on a file
func (fs *SimpleFS) SetAttribute(path, key, value string) error {
	fullPath, err := fs.fullPath(path)
	if err != nil {
		return err
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", path)
	}

	// Create attributes directory if it doesn't exist
	attrDir := filepath.Join(fs.rootPath, ".attributes")
	if _, err := os.Stat(attrDir); os.IsNotExist(err) {
		if err := os.MkdirAll(attrDir, 0755); err != nil {
			return fmt.Errorf("failed to create attributes directory: %w", err)
		}
	}

	// Create a hashed path for the attributes file
	// Using a hash to avoid issues with special characters in filenames
	hashedName := utils.HashString(path)
	hashedPath := filepath.Join(attrDir, hashedName+".json")

	attrLock := fs.getFileLock(hashedPath)
	attrLock.Lock()
	defer attrLock.Unlock()

	ctx := &HookContext{
		Operation: OpSetAttribute,
		Path:      path,
		Key:       key,
		Value:     value,
	}
	if err := fs.executeHooks(HookTypePre, ctx); err != nil {
		return err
	}

	attrs := make(map[string]string)
	if data, err := os.ReadFile(hashedPath); err == nil {
		if err := json.Unmarshal(data, &attrs); err != nil {
			// If file exists but is corrupted, start with empty map
			attrs = make(map[string]string)
		}
	}
	attrs[key] = value

	// Write back to file
	data, err := json.MarshalIndent(attrs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal attributes: %w", err)
	}

	if fs.journal != nil {
		if err := fs.journal.Log(JournalEntry{
			Operation: "setattr",
			Path:      path,
			Timestamp: getNow(),
			Attributes: map[string]string{
				key: value,
			},
		}); err != nil {
			return fmt.Errorf("failed to log attribute set: %w", err)
		}
	}

	if err := os.WriteFile(hashedPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write attributes file: %w", err)
	}

	return fs.executeHooks(HookTypePost, ctx)
}

// GetAttribute gets an extended attribute from a file
func (fs *SimpleFS) GetAttribute(path, key string) (string, error) {
	attrDir := filepath.Join(fs.rootPath, ".attributes")
	hashedName := utils.HashString(path)
	hashedPath := filepath.Join(attrDir, hashedName+".json")

	// Lock the attributes file for reading
	attrLock := fs.getFileLock(hashedPath)
	attrLock.RLock()
	defer attrLock.RUnlock()

	ctx := &HookContext{
		Operation: OpGetAttribute,
		Path:      path,
		Key:       key,
	}
	if err := fs.executeHooks(HookTypePre, ctx); err != nil {
		return "", err
	}

	data, err := os.ReadFile(hashedPath)
	if err != nil {
		return "", fmt.Errorf("failed to read attributes: %w", err)
	}

	attrs := make(map[string]string)
	if err := json.Unmarshal(data, &attrs); err != nil {
		return "", fmt.Errorf("failed to unmarshal attributes: %w", err)
	}

	value, exists := attrs[key]
	if !exists {
		return "", fmt.Errorf("attribute %s does not exist for %s", key, path)
	}

	if err := fs.executeHooks(HookTypePost, ctx); err != nil {
		return "", err
	}

	return value, nil
}

// GetAllAttributes gets all extended attributes from a file
func (fs *SimpleFS) GetAllAttributes(path string) (map[string]string, error) {
	attrDir := filepath.Join(fs.rootPath, ".attributes")
	hashedName := utils.HashString(path)
	hashedPath := filepath.Join(attrDir, hashedName+".json")

	if _, err := os.Stat(hashedPath); os.IsNotExist(err) {
		return make(map[string]string), nil
	}

	attrLock := fs.getFileLock(hashedPath)
	attrLock.RLock()
	defer attrLock.RUnlock()

	ctx := &HookContext{
		Operation: OpGetAllAttributes,
		Path:      path,
	}
	if err := fs.executeHooks(HookTypePre, ctx); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(hashedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read attributes: %w", err)
	}

	attrs := make(map[string]string)
	if err := json.Unmarshal(data, &attrs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
	}

	if err := fs.executeHooks(HookTypePost, ctx); err != nil {
		return nil, err
	}

	return attrs, nil
}

// DeleteAttribute deletes an extended attribute from a file
func (fs *SimpleFS) DeleteAttribute(path, key string) error {
	attrDir := filepath.Join(fs.rootPath, ".attributes")
	hashedName := utils.HashString(path)
	hashedPath := filepath.Join(attrDir, hashedName+".json")

	attrLock := fs.getFileLock(hashedPath)
	attrLock.Lock()
	defer attrLock.Unlock()

	if _, err := os.Stat(hashedPath); os.IsNotExist(err) {
		return fmt.Errorf("no attributes found for %s", path)
	}

	ctx := &HookContext{
		Operation: OpDeleteAttribute,
		Path:      path,
		Key:       key,
	}
	if err := fs.executeHooks(HookTypePre, ctx); err != nil {
		return err
	}

	data, err := os.ReadFile(hashedPath)
	if err != nil {
		return fmt.Errorf("failed to read attributes: %w", err)
	}

	attrs := make(map[string]string)
	if err := json.Unmarshal(data, &attrs); err != nil {
		return fmt.Errorf("failed to unmarshal attributes: %w", err)
	}

	if _, exists := attrs[key]; !exists {
		return fmt.Errorf("attribute %s does not exist for %s", key, path)
	}

	delete(attrs, key)

	// If no attributes left, delete the file
	if len(attrs) == 0 {
		if err := os.Remove(hashedPath); err != nil {
			return fmt.Errorf("failed to delete empty attributes file: %w", err)
		}

		return fs.executeHooks(HookTypePost, ctx)
	}

	newData, err := json.MarshalIndent(attrs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal attributes: %w", err)
	}

	if fs.journal != nil {
		if err := fs.journal.Log(JournalEntry{
			Operation: "deleteattr",
			Path:      path,
			Timestamp: getNow(),
			Attributes: map[string]string{
				key: "",
			},
		}); err != nil {
			return fmt.Errorf("failed to log attribute deletion: %w", err)
		}
	}

	if err := os.WriteFile(hashedPath, newData, 0644); err != nil {
		return fmt.Errorf("failed to write attributes file: %w", err)
	}

	return fs.executeHooks(HookTypePost, ctx)
}

// Helper function to get current time
var getNow = func() time.Time {
	return time.Now()
}
