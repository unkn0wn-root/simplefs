package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	fs "github.com/unkn0wn-root/simplefs"
)

var (
	rootPath         = flag.String("root", "./data", "Root directory for the filesystem")
	enableVersioning = flag.Bool("versioning", false, "Enable file versioning")
	enableJournaling = flag.Bool("journaling", true, "Enable journaling for crash recovery")
	maxVersions      = flag.Int("max-versions", 10, "Maximum number of versions to keep per file")
	verbose          = flag.Bool("verbose", false, "Enable verbose output")
)

func main() {
	flag.Parse()

	// Validate command and arguments
	args := flag.Args()
	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}

	// Create filesystem
	opts := fs.DefaultOptions()
	opts.EnableVersioning = *enableVersioning
	opts.EnableJournaling = *enableJournaling
	opts.MaxVersions = *maxVersions

	fileSystem, err := fs.NewSimpleFS(*rootPath, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating filesystem: %v\n", err)
		os.Exit(1)
	}
	defer fileSystem.Close()

	// Execute command
	command := args[0]
	cmdArgs := args[1:]

	switch command {
	case "ls", "list":
		handleList(fileSystem, cmdArgs)
	case "read", "cat":
		handleRead(fileSystem, cmdArgs)
	case "write":
		handleWrite(fileSystem, cmdArgs)
	case "mkdir":
		handleMkdir(fileSystem, cmdArgs)
	case "rm", "delete":
		handleDelete(fileSystem, cmdArgs)
	case "cp", "copy":
		handleCopy(fileSystem, cmdArgs)
	case "mv", "move":
		handleMove(fileSystem, cmdArgs)
	case "attr", "attributes":
		handleAttributes(fileSystem, cmdArgs)
	case "version", "versions":
		handleVersions(fileSystem, cmdArgs)
	case "stat":
		handleStat(fileSystem, cmdArgs)
	case "backup":
		handleBackup(fileSystem, cmdArgs)
	case "recover":
		handleRecover(fileSystem, cmdArgs)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: simplefs [options] command [arguments]")
	fmt.Println("\nOptions:")
	flag.PrintDefaults()

	fmt.Println("\nCommands:")
	fmt.Println("  ls, list [path]                   List directory contents")
	fmt.Println("  read, cat <path>                  Read file contents")
	fmt.Println("  write <path> <content>            Write content to file")
	fmt.Println("  mkdir <path>                      Create directory")
	fmt.Println("  rm, delete <path>                 Delete file or directory")
	fmt.Println("  cp, copy <src> <dst>              Copy file")
	fmt.Println("  mv, move <src> <dst>              Move file")
	fmt.Println("  attr, attributes <command> [args] Manage file attributes")
	fmt.Println("  version, versions <command> [args] Manage file versions")
	fmt.Println("  stat <path>                       Show file information")
	fmt.Println("  backup <path> <dst>               Backup a file or directory")
	fmt.Println("  recover                           Attempt to recover from crash")

	fmt.Println("\nAttribute commands:")
	fmt.Println("  set <path> <key> <value>          Set attribute")
	fmt.Println("  get <path> <key>                  Get attribute")
	fmt.Println("  list <path>                       List all attributes")
	fmt.Println("  delete <path> <key>               Delete attribute")

	fmt.Println("\nVersion commands:")
	fmt.Println("  list <path>                       List all versions")
	fmt.Println("  get <path> <version-id>           Get a specific version")
	fmt.Println("  restore <path> <version-id>       Restore to a specific version")
	fmt.Println("  describe <path> <version-id> <text> Set version description")
	fmt.Println("  delete <path> <version-id>        Delete a specific version")
}

func handleList(fileSystem *fs.SimpleFS, args []string) {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	files, err := fileSystem.ListDir(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing directory: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Println("Directory is empty")
		return
	}

	fmt.Printf("Contents of %s:\n", path)
	for _, file := range files {
		fileType := "F"
		if file.IsDir {
			fileType = "D"
		}

		fmt.Printf("[%s] %-20s %8d bytes  %s\n",
			fileType, file.Name, file.Size, file.ModTime.Format(time.RFC3339))

		if *verbose && len(file.Attributes) > 0 {
			fmt.Println("  Attributes:")
			for k, v := range file.Attributes {
				fmt.Printf("    %s: %s\n", k, v)
			}
		}
	}
}

func handleRead(fileSystem *fs.SimpleFS, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Error: Missing file path\n")
		os.Exit(1)
	}

	data, err := fileSystem.ReadFile(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(data))
}

func handleWrite(fileSystem *fs.SimpleFS, args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Error: Missing path or content\n")
		os.Exit(1)
	}

	path := args[0]
	content := args[1]

	// Check if content is a file to read from
	if strings.HasPrefix(content, "@") {
		filename := content[1:]
		data, err := os.ReadFile(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading content file: %v\n", err)
			os.Exit(1)
		}

		err = fileSystem.WriteFile(path, data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Wrote %d bytes from %s to %s\n", len(data), filename, path)
	} else {
		// Write the content directly
		err := fileSystem.WriteFile(path, []byte(content))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Wrote %d bytes to %s\n", len(content), path)
	}
}

func handleMkdir(fileSystem *fs.SimpleFS, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Error: Missing directory path\n")
		os.Exit(1)
	}

	path := args[0]
	err := fileSystem.CreateDir(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created directory %s\n", path)
}

func handleDelete(fileSystem *fs.SimpleFS, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Error: Missing path\n")
		os.Exit(1)
	}

	path := args[0]

	// Check if it's a directory or file
	isDir := fileSystem.IsDir(path)

	var err error
	if isDir {
		err = fileSystem.DeleteDir(path)
	} else {
		err = fileSystem.DeleteFile(path)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting %s: %v\n", path, err)
		os.Exit(1)
	}

	fmt.Printf("Deleted %s %s\n",
		map[bool]string{true: "directory", false: "file"}[isDir], path)
}

func handleCopy(fileSystem *fs.SimpleFS, args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Error: Missing source or destination path\n")
		os.Exit(1)
	}

	src, dst := args[0], args[1]

	err := fileSystem.CopyFile(src, dst)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error copying file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Copied %s to %s\n", src, dst)
}

func handleMove(fileSystem *fs.SimpleFS, args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Error: Missing source or destination path\n")
		os.Exit(1)
	}

	src, dst := args[0], args[1]

	err := fileSystem.MoveFile(src, dst)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error moving file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Moved %s to %s\n", src, dst)
}

func handleAttributes(fileSystem *fs.SimpleFS, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Error: Missing attribute command\n")
		os.Exit(1)
	}

	cmd := args[0]
	cmdArgs := args[1:]

	switch cmd {
	case "set":
		if len(cmdArgs) < 3 {
			fmt.Fprintf(os.Stderr, "Error: set requires <path> <key> <value>\n")
			os.Exit(1)
		}

		path, key, value := cmdArgs[0], cmdArgs[1], cmdArgs[2]
		err := fileSystem.SetAttribute(path, key, value)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error setting attribute: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Set attribute %s=%s on %s\n", key, value, path)

	case "get":
		if len(cmdArgs) < 2 {
			fmt.Fprintf(os.Stderr, "Error: get requires <path> <key>\n")
			os.Exit(1)
		}

		path, key := cmdArgs[0], cmdArgs[1]
		value, err := fileSystem.GetAttribute(path, key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting attribute: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("%s\n", value)

	case "list":
		if len(cmdArgs) < 1 {
			fmt.Fprintf(os.Stderr, "Error: list requires <path>\n")
			os.Exit(1)
		}

		path := cmdArgs[0]
		attrs, err := fileSystem.GetAllAttributes(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing attributes: %v\n", err)
			os.Exit(1)
		}

		if len(attrs) == 0 {
			fmt.Println("No attributes found")
			return
		}

		fmt.Printf("Attributes for %s:\n", path)
		for k, v := range attrs {
			fmt.Printf("  %s: %s\n", k, v)
		}

	case "delete":
		if len(cmdArgs) < 2 {
			fmt.Fprintf(os.Stderr, "Error: delete requires <path> <key>\n")
			os.Exit(1)
		}

		path, key := cmdArgs[0], cmdArgs[1]
		err := fileSystem.DeleteAttribute(path, key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting attribute: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Deleted attribute %s from %s\n", key, path)

	default:
		fmt.Fprintf(os.Stderr, "Unknown attribute command: %s\n", cmd)
		os.Exit(1)
	}
}

func handleVersions(fileSystem *fs.SimpleFS, args []string) {
	if !*enableVersioning {
		fmt.Fprintf(os.Stderr, "Error: Versioning is not enabled\n")
		os.Exit(1)
	}

	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Error: Missing version command\n")
		os.Exit(1)
	}

	cmd := args[0]
	cmdArgs := args[1:]

	switch cmd {
	case "list":
		if len(cmdArgs) < 1 {
			fmt.Fprintf(os.Stderr, "Error: list requires <path>\n")
			os.Exit(1)
		}

		path := cmdArgs[0]
		versions, err := fileSystem.ListVersions(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing versions: %v\n", err)
			os.Exit(1)
		}

		if len(versions.Versions) == 0 {
			fmt.Println("No versions found")
			return
		}

		fmt.Printf("Versions for %s:\n", path)
		for i, version := range versions.Versions {
			fmt.Printf("%d. ID: %s  Created: %s  Size: %d bytes\n",
				i+1, version.VersionID, version.CreatedAt.Format(time.RFC3339), version.Size)

			if version.Description != "" {
				fmt.Printf("   Description: %s\n", version.Description)
			}
		}

	case "get":
		if len(cmdArgs) < 2 {
			fmt.Fprintf(os.Stderr, "Error: get requires <path> <version-id>\n")
			os.Exit(1)
		}

		path, versionID := cmdArgs[0], cmdArgs[1]
		data, versionInfo, err := fileSystem.GetVersion(path, versionID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting version: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("// Version: %s\n", versionID)
		fmt.Printf("// Created: %s\n", versionInfo.CreatedAt.Format(time.RFC3339))
		if versionInfo.Description != "" {
			fmt.Printf("// Description: %s\n", versionInfo.Description)
		}
		fmt.Printf("// Size: %d bytes\n", versionInfo.Size)
		fmt.Println("//--------------------")
		fmt.Println(string(data))

	case "restore":
		if len(cmdArgs) < 2 {
			fmt.Fprintf(os.Stderr, "Error: restore requires <path> <version-id>\n")
			os.Exit(1)
		}

		path, versionID := cmdArgs[0], cmdArgs[1]
		err := fileSystem.RestoreVersion(path, versionID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error restoring version: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Restored %s to version %s\n", path, versionID)

	case "describe":
		if len(cmdArgs) < 3 {
			fmt.Fprintf(os.Stderr, "Error: describe requires <path> <version-id> <description>\n")
			os.Exit(1)
		}

		path, versionID, description := cmdArgs[0], cmdArgs[1], cmdArgs[2]
		err := fileSystem.SetVersionDescription(path, versionID, description)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error setting version description: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Set description for version %s of %s\n", versionID, path)

	case "delete":
		if len(cmdArgs) < 2 {
			fmt.Fprintf(os.Stderr, "Error: delete requires <path> <version-id>\n")
			os.Exit(1)
		}

		path, versionID := cmdArgs[0], cmdArgs[1]
		err := fileSystem.DeleteVersion(path, versionID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting version: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Deleted version %s of %s\n", versionID, path)

	default:
		fmt.Fprintf(os.Stderr, "Unknown version command: %s\n", cmd)
		os.Exit(1)
	}
}

func handleStat(fileSystem *fs.SimpleFS, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Error: Missing path\n")
		os.Exit(1)
	}

	path := args[0]

	pathInfo, err := fileSystem.GetPathInfo(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting path info: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Path:      %s\n", pathInfo.Path)
	fmt.Printf("Absolute:  %s\n", pathInfo.Absolute)
	fmt.Printf("Relative:  %s\n", pathInfo.Relative)
	fmt.Printf("Exists:    %t\n", pathInfo.Exists)

	if pathInfo.Exists {
		fmt.Printf("Type:      %s\n", map[bool]string{true: "Directory", false: "File"}[pathInfo.IsDir])
		if !pathInfo.IsDir {
			fmt.Printf("Size:      %d bytes\n", pathInfo.Size)
		}

		// Get more details with Stat
		fileInfo, err := fileSystem.Stat(path)
		if err == nil {
			fmt.Printf("Mode:      %s\n", fileInfo.Mode.String())
			fmt.Printf("Modified:  %s\n", fileInfo.ModTime.Format(time.RFC3339))

			if len(fileInfo.Attributes) > 0 {
				fmt.Println("Attributes:")
				for k, v := range fileInfo.Attributes {
					fmt.Printf("  %s: %s\n", k, v)
				}
			}
		}
	}

	fmt.Printf("Components: %v\n", pathInfo.Components)
}

func handleBackup(fileSystem *fs.SimpleFS, args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Error: Missing source or destination path\n")
		os.Exit(1)
	}

	src, dst := args[0], args[1]

	// Check if source is a directory or file
	isDir := fileSystem.IsDir(src)

	if isDir {
		// Create the backup directory
		err := os.MkdirAll(dst, 0755)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating backup directory: %v\n", err)
			os.Exit(1)
		}

		// List the directory
		files, err := fileSystem.ListDir(src)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing directory: %v\n", err)
			os.Exit(1)
		}

		count := 0
		for _, file := range files {
			if file.IsDir {
				continue // Skip subdirectories for simplicity
			}

			// Read the file
			data, err := fileSystem.ReadFile(filepath.Join(src, file.Name))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", file.Name, err)
				continue
			}

			// Write to the backup location
			err = os.WriteFile(filepath.Join(dst, file.Name), data, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error writing backup file %s: %v\n", file.Name, err)
				continue
			}

			count++
		}

		fmt.Printf("Backed up %d files from %s to %s\n", count, src, dst)
	} else {
		// Backup a single file
		data, err := fileSystem.ReadFile(src)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}

		// Create parent directories if needed
		err = os.MkdirAll(filepath.Dir(dst), 0755)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating backup directory: %v\n", err)
			os.Exit(1)
		}

		// Write the backup
		err = os.WriteFile(dst, data, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing backup: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Backed up %s to %s (%d bytes)\n", src, dst, len(data))
	}
}

func handleRecover(fileSystem *fs.SimpleFS, args []string) {
	if !*enableJournaling {
		fmt.Fprintf(os.Stderr, "Error: Journaling is not enabled\n")
		os.Exit(1)
	}

	fmt.Println("Attempting to recover from journal...")
	err := fileSystem.Recover()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during recovery: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Recovery completed successfully")
}
