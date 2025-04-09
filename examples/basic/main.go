package main

import (
	"fmt"
	"os"
	"time"

	fs "github.com/unkn0wn-root/simplefs"
)

func main() {
	// Create a temporary directory for our example
	tempDir := "./tmp/basic-example"

	// Create a new filesystem
	opts := fs.DefaultOptions()
	fileSystem, err := fs.NewSimpleFS(tempDir, opts)
	if err != nil {
		fmt.Printf("Error creating filesystem: %v\n", err)
		os.Exit(1)
	}
	defer fileSystem.Close()

	fmt.Println("==== Basic Example ====")

	// Create directories
	fmt.Println("Creating directories...")
	err = fileSystem.CreateDir("documents")
	if err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		os.Exit(1)
	}

	err = fileSystem.CreateDir("documents/work")
	if err != nil {
		fmt.Printf("Error creating subdirectory: %v\n", err)
		os.Exit(1)
	}

	// Write files
	fmt.Println("Writing files...")
	err = fileSystem.WriteFile("documents/hello.txt", []byte("Hello, SimpleFS!"))
	if err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		os.Exit(1)
	}

	err = fileSystem.WriteFile("documents/work/report.txt", []byte("This is a work report."))
	if err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		os.Exit(1)
	}

	// Read a file
	fmt.Println("Reading files...")
	data, err := fileSystem.ReadFile("documents/hello.txt")
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("File content: %s\n", string(data))

	// List directory
	fmt.Println("\nListing directory contents...")
	files, err := fileSystem.ListDir("documents")
	if err != nil {
		fmt.Printf("Error listing directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Documents directory contents:")
	for _, file := range files {
		fileType := "File"
		if file.IsDir {
			fileType = "Directory"
		}
		fmt.Printf("- %s (%s, %d bytes)\n", file.Name, fileType, file.Size)
	}

	// Set and get attributes
	fmt.Println("\nSetting attributes...")
	err = fileSystem.SetAttribute("documents/hello.txt", "author", "SimpleFS User")
	if err != nil {
		fmt.Printf("Error setting attribute: %v\n", err)
		os.Exit(1)
	}

	err = fileSystem.SetAttribute("documents/hello.txt", "created", time.Now().Format(time.RFC3339))
	if err != nil {
		fmt.Printf("Error setting attribute: %v\n", err)
		os.Exit(1)
	}

	// Get attributes
	fmt.Println("Getting attributes...")
	author, err := fileSystem.GetAttribute("documents/hello.txt", "author")
	if err != nil {
		fmt.Printf("Error getting attribute: %v\n", err)
		os.Exit(1)
	}

	created, err := fileSystem.GetAttribute("documents/hello.txt", "created")
	if err != nil {
		fmt.Printf("Error getting attribute: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("File attributes: author=%s, created=%s\n", author, created)

	// Copy a file
	fmt.Println("\nCopying file...")
	err = fileSystem.CopyFile("documents/hello.txt", "documents/hello_copy.txt")
	if err != nil {
		fmt.Printf("Error copying file: %v\n", err)
		os.Exit(1)
	}

	// Move a file
	fmt.Println("Moving file...")
	err = fileSystem.MoveFile("documents/hello_copy.txt", "documents/work/hello_moved.txt")
	if err != nil {
		fmt.Printf("Error moving file: %v\n", err)
		os.Exit(1)
	}

	// List work directory to confirm the move
	workFiles, err := fileSystem.ListDir("documents/work")
	if err != nil {
		fmt.Printf("Error listing directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nWork directory contents after move:")
	for _, file := range workFiles {
		fmt.Printf("- %s (%d bytes)\n", file.Name, file.Size)
	}

	// Delete a file
	fmt.Println("\nDeleting file...")
	err = fileSystem.DeleteFile("documents/work/hello_moved.txt")
	if err != nil {
		fmt.Printf("Error deleting file: %v\n", err)
		os.Exit(1)
	}

	// Check that the file is gone
	if fileSystem.FileExists("documents/work/hello_moved.txt") {
		fmt.Println("Error: File still exists after deletion!")
		os.Exit(1)
	} else {
		fmt.Println("File successfully deleted.")
	}

	// Get path information
	fmt.Println("\nGetting path information...")
	pathInfo, err := fileSystem.GetPathInfo("documents/work")
	if err != nil {
		fmt.Printf("Error getting path info: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Path: %s\n", pathInfo.Path)
	fmt.Printf("Absolute path: %s\n", pathInfo.Absolute)
	fmt.Printf("Is directory: %t\n", pathInfo.IsDir)
	fmt.Printf("Components: %v\n", pathInfo.Components)

	fmt.Println("\nBasic example completed successfully!")
}
