package detector

import (
	"testing"
)

func TestGetLibraryPathContainsAppID(t *testing.T) {
	vdfMap := map[string]any{
		"libraryfolders": map[string]any{
			"0": map[string]any{
				"path": "/home/user/.local/share/Steam",
				"apps": map[string]any{
					"228980": "123456",
				},
			},
			"1": map[string]any{
				"path": "/mnt/games/SteamLibrary",
				"apps": map[string]any{
					"1222670": "987654",
				},
			},
		},
	}

	path, err := GetLibraryPathContainsAppID(vdfMap, "1222670")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "/mnt/games/SteamLibrary"
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}

	notFoundPath, err := GetLibraryPathContainsAppID(vdfMap, "9999999")
	if err != nil {
		t.Fatalf("unexpected error for missing app: %v", err)
	}
	if notFoundPath != "" {
		t.Errorf("expected empty string, got %q", notFoundPath)
	}
}
