package fs

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// JournalEntry represents a single operation in the journal
type JournalEntry struct {
	Operation  string            // Type of operation (write, delete, mkdir, etc.)
	Path       string            // Relative path within the filesystem
	Data       []byte            // File data for write operations
	Timestamp  time.Time         // When the operation occurred
	Attributes map[string]string // Associated attributes
}

// Journal handles transaction logging for crash recovery
type Journal struct {
	path   string     // Path to the journal file
	mu     sync.Mutex // Mutex for thread safety
	file   *os.File   // Journal file handle
	buffer []byte     // Buffer for journal operations
}

// NewJournal creates a new journal at the specified path
func NewJournal(path string) (*Journal, error) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open journal file: %w", err)
	}

	return &Journal{
		path:   path,
		file:   file,
		buffer: make([]byte, 0, 4096),
	}, nil
}

// Log adds an entry to the journal
func (j *Journal) Log(entry JournalEntry) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal journal entry: %w", err)
	}

	// Add newline for readability
	data = append(data, '\n')

	// Write to the journal file
	_, err = j.file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to journal: %w", err)
	}

	// Ensure data is written to disk
	return j.file.Sync()
}

// Close closes the journal file
func (j *Journal) Close() error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.file != nil {
		err := j.file.Close()
		j.file = nil
		return err
	}
	return nil
}

// IsDir checks if the entry represents a directory
func (e *JournalEntry) IsDir() bool {
	return len(e.Data) == 0 && e.Operation == "mkdir"
}

// Recover attempts to recover from a crash by replaying the journal
func (j *Journal) Recover(fs *SimpleFS) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	// Close the current file if open
	if j.file != nil {
		j.file.Close()
		j.file = nil
	}

	file, err := os.Open(j.path)
	if err != nil {
		return fmt.Errorf("failed to open journal for recovery: %w", err)
	}
	defer file.Close()

	fmt.Println("Starting recovery from journal...")

	// Temporary map to track the latest state of each file
	fileState := make(map[string]JournalEntry)

	// Read all entries and update the state map
	scanner := json.NewDecoder(file)
	entryCount := 0
	for {
		var entry JournalEntry
		if err := scanner.Decode(&entry); err != nil {
			if err == io.EOF {
				break
			}
			// Skip corrupted entries
			fmt.Printf("Warning: Skipping corrupted journal entry: %v\n", err)
			continue
		}

		fileState[entry.Path] = entry
		entryCount++
	}

	fmt.Printf("Read %d journal entries, %d unique paths\n", entryCount, len(fileState))

	// Replay operations in the correct order
	for path, entry := range fileState {
		fmt.Printf("Recovering: %s (operation: %s)\n", path, entry.Operation)

		switch entry.Operation {
		case "write":
			// Extract mode if available
			mode := os.FileMode(0644) // Default mode
			if modeStr, ok := entry.Attributes["mode"]; ok {
				var modeVal uint64
				fmt.Sscanf(modeStr, "%o", &modeVal)
				if modeVal > 0 {
					mode = os.FileMode(modeVal)
				}
			}

			if err := fs.WriteFileWithMode(path, entry.Data, mode); err != nil {
				fmt.Printf("Warning: Failed to recover file %s: %v\n", path, err)
			}

		case "mkdir":
			if err := fs.CreateDir(path); err != nil {
				fmt.Printf("Warning: Failed to recover directory %s: %v\n", path, err)
			}

		case "delete":
			// If the path is in the delete list, delete it
			if entry.IsDir() {
				if err := fs.DeleteDir(path); err != nil {
					// Not a critical error during recovery
					fmt.Printf("Note: Could not delete directory %s: %v\n", path, err)
				}
			} else {
				if err := fs.DeleteFile(path); err != nil {
					// Not a critical error during recovery
					fmt.Printf("Note: Could not delete file %s: %v\n", path, err)
				}
			}

		case "setattr":
			for k, v := range entry.Attributes {
				if err := fs.SetAttribute(path, k, v); err != nil {
					fmt.Printf("Warning: Failed to recover attribute %s for %s: %v\n", k, path, err)
				}
			}
		}
	}

	// Reopen journal for writing
	newFile, err := os.OpenFile(j.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to reopen journal after recovery: %w", err)
	}
	j.file = newFile

	fmt.Println("Recovery completed")
	return nil
}

// Rotate rotates the journal file
func (j *Journal) Rotate() error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.file != nil {
		j.file.Close()
		j.file = nil
	}

	// Rename the current journal file
	backupPath := j.path + "." + time.Now().Format("20060102-150405")
	if err := os.Rename(j.path, backupPath); err != nil {
		return fmt.Errorf("failed to rename journal file: %w", err)
	}

	// Create a new journal file
	file, err := os.OpenFile(j.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to create new journal file: %w", err)
	}
	j.file = file

	return nil
}

// Truncate truncates the journal file
func (j *Journal) Truncate() error {
	j.mu.Lock()
	defer j.mu.Unlock()

	// Close the current file
	if j.file != nil {
		j.file.Close()
		j.file = nil
	}

	// Truncate the file
	if err := os.Truncate(j.path, 0); err != nil {
		return fmt.Errorf("failed to truncate journal file: %w", err)
	}

	// Reopen the file
	file, err := os.OpenFile(j.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to reopen journal file after truncation: %w", err)
	}
	j.file = file

	return nil
}
