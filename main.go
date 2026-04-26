package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/andygrunwald/vdf"
)

func main() {
	p, err := getModFolderPath()
	if err != nil {
		log.Fatal(err)
	}

	if p == "" {
		log.Fatal("Sims 4 not found on Steam!")
	}

	fmt.Println(p)
}

// getModFolderPath returns the path to the Sims 4 mod folder.
func getModFolderPath() (string, error) {
	// Todo: get path from config file

	// get steam root
	root, err := getSteamRootPath()
	if err != nil {
		return "", err
	}

	// get libraryfolders.vdf
	f, err := getLibraryFoldersVDFFile(root)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// parse libraryfolders.vdf into a map[string]interface{}
	p := vdf.NewParser(f)
	m, err := p.Parse()
	if err != nil {
		return "", err
	}

	// get library path that contains Sims 4 - 1222670
	path, err := getLibraryPathContainsAppID(m, "1222670")
	if err != nil {
		return "", err
	}

	// return library path for now
	return path, nil
}

// getSteamRootPath returns the path to the Steam root directory.
// Assumes the OS is Linux.
func getSteamRootPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(home, ".steam", "steam")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", fmt.Errorf("steam root directory not found at %s", path)
	}
	return path, nil
}

// getLibraryFoldersVDFFile returns the file handle for the libraryfolders.vdf file.
func getLibraryFoldersVDFFile(rootPath string) (*os.File, error) {
	if rootPath == "" {
		return nil, fmt.Errorf("rootPath cannot be empty")
	}
	return os.Open(filepath.Join(rootPath, "steamapps", "libraryfolders.vdf"))
}

// getLibraryPathContainsAppID returns the path to the library folder that contains the specified app ID.
func getLibraryPathContainsAppID(m map[string]interface{}, id string) (string, error) {
	// check for apps section
	if appsRaw, ok := m["apps"]; ok {
		apps, ok := appsRaw.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("apps is not a map[string]interface{}")
		}
		if _, found := apps[id]; found {
			if pathRaw, ok := m["path"]; ok {
				if path, ok := pathRaw.(string); ok {
					return path, nil
				}
			}
			return "", fmt.Errorf("path is not a string")
		}
	}

	// recurse into child maps.
	for _, v := range m {
		child, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		path, err := getLibraryPathContainsAppID(child, id)
		if err != nil {
			return "", fmt.Errorf("getLibraryPathContainsAppID: %w", err)
		}
		if path != "" {
			return path, nil
		}
	}

	return "", nil
}
