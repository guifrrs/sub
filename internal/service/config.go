package service

import (
	"os"
	"path/filepath"
)

const (
	MinSearchQueryLength  = 2
	DefaultLanguage       = "eng"
	DefaultDownloadSubdir = "Downloads/sub"
)

func DefaultDownloadDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.FromSlash(DefaultDownloadSubdir)
	}

	return filepath.Join(home, filepath.FromSlash(DefaultDownloadSubdir))
}
