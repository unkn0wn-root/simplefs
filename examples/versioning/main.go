package main

import (
	"fmt"
	"os"
	"time"

	"github.com/unkn0wn-root/simplefs/fs"
)

func main() {
	// Create a temporary directory for our example
	tempDir := "./tmp/versioning-example"

	// Create a new filesystem with versioning enabled
	opts := fs.DefaultOptions()
	opts.EnableVersioning = true // Enable versioning
	opts.MaxVersions = 5         // Keep up to 5 versions

	fileSystem, err := fs.NewSimpleFS(tempDir, opts)
	if err != nil {
		fmt.Printf("Error creating filesystem: %v\n", err)
		os.Exit(1)
	}
	defer fileSystem.Close()

	fmt.Println("==== Versioning Example ====")

	// Create a document
	docPath := "document.txt"
	fmt.Println("Creating initial document...")
	err = fileSystem.WriteFile(docPath, []byte("Version 1 of the document"))
	if err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		os.Exit(1)
	}

	// List versions (should be empty since we just created the file)
	versions, err := fileSystem.ListVersions(docPath)
	if err != nil {
		fmt.Printf("Error listing versions: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Initial number of versions: %d\n", len(versions.Versions))

	// Create several versions by modifying the file
	for i := 2; i <= 6; i++ {
		// Wait a moment to ensure different timestamps
		time.Sleep(100 * time.Millisecond)

		fmt.Printf("\nModifying document (version %d)...\n", i)
		content := fmt.Sprintf("Version %d of the document", i)

		err = fileSystem.WriteFile(docPath, []byte(content))
		if err != nil {
			fmt.Printf("Error writing file: %v\n", err)
			os.Exit(1)
		}

		// Verify content was updated
		data, err := fileSystem.ReadFile(docPath)
		if err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Current content: %s\n", string(data))
	}

	// List all versions
	versions, err = fileSystem.ListVersions(docPath)
	if err != nil {
		fmt.Printf("Error listing versions: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nNumber of versions: %d (should be 5 due to MaxVersions)\n", len(versions.Versions))
	fmt.Println("Available versions:")

	for i, version := range versions.Versions {
		fmt.Printf("%d. Version ID: %s, Created: %s, Size: %d bytes\n",
			i+1, version.VersionID, version.CreatedAt.Format(time.RFC3339), version.Size)
	}

	// Get a specific version (the second one in the list)
	if len(versions.Versions) >= 2 {
		versionID := versions.Versions[1].VersionID
		fmt.Printf("\nRetrieving version %s...\n", versionID)

		data, versionInfo, err := fileSystem.GetVersion(docPath, versionID)
		if err != nil {
			fmt.Printf("Error getting version: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Version content: %s\n", string(data))
		fmt.Printf("Version created at: %s\n", versionInfo.CreatedAt.Format(time.RFC3339))
	}

	// Restore to a previous version (the third one in the list)
	if len(versions.Versions) >= 3 {
		versionID := versions.Versions[2].VersionID
		fmt.Printf("\nRestoring to version %s...\n", versionID)

		err = fileSystem.RestoreVersion(docPath, versionID)
		if err != nil {
			fmt.Printf("Error restoring version: %v\n", err)
			os.Exit(1)
		}

		// Verify the content was restored
		data, err := fileSystem.ReadFile(docPath)
		if err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Content after restore: %s\n", string(data))
	}

	// Add a description to a version
	if len(versions.Versions) > 0 {
		versionID := versions.Versions[0].VersionID
		description := "This is the latest version before restore"

		fmt.Printf("\nAdding description to version %s...\n", versionID)
		err = fileSystem.SetVersionDescription(docPath, versionID, description)
		if err != nil {
			fmt.Printf("Error setting version description: %v\n", err)
			os.Exit(1)
		}

		// Get the version again to see the description
		_, versionInfo, err := fileSystem.GetVersion(docPath, versionID)
		if err != nil {
			fmt.Printf("Error getting version: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Version description: %s\n", versionInfo.Description)
	}

	// Delete a specific version
	if len(versions.Versions) >= 4 {
		versionID := versions.Versions[3].VersionID
		fmt.Printf("\nDeleting version %s...\n", versionID)

		err = fileSystem.DeleteVersion(docPath, versionID)
		if err != nil {
			fmt.Printf("Error deleting version: %v\n", err)
			os.Exit(1)
		}

		// List versions again to confirm deletion
		versions, err = fileSystem.ListVersions(docPath)
		if err != nil {
			fmt.Printf("Error listing versions: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Number of versions after deletion: %d\n", len(versions.Versions))
	}

	fmt.Println("\nVersioning example completed successfully!")
}
