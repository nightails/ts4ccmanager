package db

import "time"

// Mod represents a registered mod file or package in The Sims 4.
type Mod struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	FileName       string    `json:"file_name"`
	FilePath       string    `json:"file_path"`
	InstalledFiles []string  `json:"installed_files,omitempty"`
	FileHash       string    `json:"file_hash"`
	Version        string    `json:"version"`
	Author         string    `json:"author"`
	InstalledAt    time.Time `json:"installed_at"`
	Status         string    `json:"status"` // "active", "disabled"
}
