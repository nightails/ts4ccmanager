package detector

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/andygrunwald/vdf"
)

const Sims4AppID = "1222670"

// DetectSteamModsPath attempts to detect the Sims 4 Mods folder location in Steam Proton.
func DetectSteamModsPath() (string, error) {
	root, err := GetSteamRootPath()
	if err != nil {
		return "", err
	}

	f, err := GetLibraryFoldersVDFFile(root)
	if err != nil {
		return "", err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	p := vdf.NewParser(f)
	m, err := p.Parse()
	if err != nil {
		return "", fmt.Errorf("failed to parse libraryfolders.vdf: %w", err)
	}

	libPath, err := GetLibraryPathContainsAppID(m, Sims4AppID)
	if err != nil {
		return "", err
	}
	if libPath == "" {
		return "", fmt.Errorf("sims 4 not found in steam libraries")
	}

	if _, err := os.Stat(libPath); os.IsNotExist(err) {
		return "", fmt.Errorf("sims 4 library folder not found at %s", libPath)
	}

	prefixPath := filepath.Join(libPath, "steamapps", "compatdata", Sims4AppID)
	if _, err := os.Stat(prefixPath); os.IsNotExist(err) {
		return "", fmt.Errorf("sims 4 proton prefix not found at %s", prefixPath)
	}

	modsPath := filepath.Join(prefixPath, "pfx", "drive_c", "users", "steamuser", "Documents", "Electronic Arts", "The Sims 4", "Mods")
	if _, err := os.Stat(modsPath); os.IsNotExist(err) {
		return "", fmt.Errorf("sims 4 mod folder not found at %s", modsPath)
	}

	return modsPath, nil
}

// GetSteamRootPath returns the path to the Steam root directory.
func GetSteamRootPath() (string, error) {
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

// GetLibraryFoldersVDFFile returns the file handle for the libraryfolders.vdf file.
func GetLibraryFoldersVDFFile(rootPath string) (*os.File, error) {
	if rootPath == "" {
		return nil, fmt.Errorf("rootPath cannot be empty")
	}
	return os.Open(filepath.Join(rootPath, "steamapps", "libraryfolders.vdf"))
}

// GetLibraryPathContainsAppID returns the path to the library folder that contains the specified app ID.
func GetLibraryPathContainsAppID(m map[string]any, id string) (string, error) {
	if appsRaw, ok := m["apps"]; ok {
		apps, ok := appsRaw.(map[string]any)
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

	for _, v := range m {
		child, ok := v.(map[string]any)
		if !ok {
			continue
		}
		path, err := GetLibraryPathContainsAppID(child, id)
		if err != nil {
			return "", fmt.Errorf("getLibraryPathContainsAppID: %w", err)
		}
		if path != "" {
			return path, nil
		}
	}

	return "", nil
}
