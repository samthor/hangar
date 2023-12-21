package lib

import (
	"os"
	"path/filepath"
)

// StoragePath returns the storage page in dev/prod for the given mount.
// In a local environment, places this in `$HOME/.fly/hangar/<machine>/mount/<path>`.
// Creates the directory if it can (swallows err).
// Protects against escape from this path. But probably not against untrusted user input.
func StoragePath(source string) (out string) {
	p := resolveStoragePath(source)

	// swallow err
	os.MkdirAll(p, 0755)

	return p
}

func resolveStoragePath(source string) string {
	if !filepath.IsAbs(source) {
		// prevent escape
		source = filepath.Join("/", source)
	}

	if IsDeploy() {
		return source
	}
	home := os.Getenv("HOME")
	return filepath.Join(home, ".fly/hangar", selfInstance.Machine, "mount", source)
}
