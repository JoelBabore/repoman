package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// start totalDirs on -1 to discount parent dir
var totalDirs = -1
var totalFiles = 0

func main() {
	if len(os.Args) < 2 {
		fmt.Println("")
		fmt.Println("Usage: repoman <dir>")
		fmt.Println("")
		fmt.Println("Where <dir> is a source code repository as a local directory")
		fmt.Println("")
		os.Exit(1)
	}

	path_to_scan := filepath.Clean(os.Args[1])

	if _, err := os.Stat(path_to_scan); os.IsNotExist(err) {
		fmt.Printf("Specified path %q does not exist", path_to_scan)
		fmt.Println("")
		os.Exit(1)
	}

	filepath.WalkDir(path_to_scan, walk)
	fmt.Println("Directories", totalDirs, "Files", totalFiles)
}

func walk(s string, d fs.DirEntry, err error) error {
	if err != nil {
		fmt.Println("Unexpected error", err)
		os.Exit(1)
	}

	if d.IsDir() {
		// Ignore contents of .git
		if d.Name() == ".git" {
			return filepath.SkipDir
		}
		totalDirs++
	} else {
		// TODO need to ignore .gitignore contents
		if d.Name() == ".gitignore" {
			fmt.Println("Warning: .gitignore contents are not excluded from the scan")
		}
		totalFiles++
	}
	return nil
}
