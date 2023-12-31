package lib

import (
	"os"
	"path/filepath"
)

// StorageDir creates and returns the given directory in prod/dev.
// Protects against escape from this path. But probably not against untrusted user input.
// This must be within a registered mount on the prod machine.
// In a local environment, places this in `$HOME/.fly/hangar/<machine>/mount/<path>`.
func StoragePath(source string) (out string) {
	p := resolveStoragePath(source)

	// swallow err
	os.MkdirAll(p, 0755)

	return p
}

// StorageFile creates and returns the given directory in prod/dev, stripping the last component (ignoring it).
// Protects against escape from this path. But probably not against untrusted user input.
// This must be within a registered mount on the prod machine.
// In a local environment, places this in `$HOME/.fly/hangar/<machine>/mount/<path>`.
func StorageFile(source ...string) (out string) {
	out = filepath.Join(source...)
	out = resolveStoragePath(out)

	// swallow err
	dir := filepath.Dir(out)
	os.MkdirAll(dir, 0755)

	return out
}

// StorageDir creates and returns the given directory in prod/dev.
// Protects against escape from this path. But probably not against untrusted user input.
// This must be within a registered mount on the prod machine.
// In a local environment, places this in `$HOME/.fly/hangar/<machine>/mount/<path>`.
func StorageDir(source ...string) (out string) {
	out = filepath.Join(source...)
	out = resolveStoragePath(out)

	// swallow err
	os.MkdirAll(out, 0755)

	return out
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
