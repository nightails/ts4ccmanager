package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/andygrunwald/vdf"
)

func main() {
	// Parse the libraryfolders.vdf and display the result
	root, err := getSteamRootPath()
	if err != nil {
		log.Fatal(err)
	}

	f, err := getLibraryFoldersVDFFile(root)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	p := vdf.NewParser(f)
	m, err := p.Parse()
	if err != nil {
		log.Fatal(err)
	}

	lmap := m["libraryfolders"].(map[string]interface{})
	lib2 := lmap["2"].(map[string]interface{})
	apps := lib2["apps"].(map[string]interface{})
	_, ok := apps["1222670"]
	if ok {
		fmt.Println("Found Sims 4 on Steam!")
	} else {
		fmt.Println("Sims 4 not found on Steam!")
	}
}

// getSteamRootPath returns the path to the Steam root directory.
// Assumes the OS is Linux.
func getSteamRootPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".steam", "steam"), nil
}

// getLibraryFoldersVDFFile returns the file handle for the libraryfolders.vdf file.
func getLibraryFoldersVDFFile(rootPath string) (*os.File, error) {
	return os.Open(filepath.Join(rootPath, "steamapps", "libraryfolders.vdf"))
}
