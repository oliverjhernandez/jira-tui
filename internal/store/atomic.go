package store

import (
	"fmt"
	"os"
	"path/filepath"
)

// writeFileAtomic writes data to dir/name so a reader sees either the old file
// or the complete new one. The temp file shares the target's directory because
// os.Rename is only atomic within a filesystem.
func writeFileAtomic(dir, name string, data []byte, perm os.FileMode) (err error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating state directory %s: %w", dir, err)
	}

	f, err := os.CreateTemp(dir, "."+name+".*.tmp")
	if err != nil {
		return fmt.Errorf("creating a temp file in %s: %w", dir, err)
	}
	tmp := f.Name()
	defer func() {
		if err != nil {
			_ = f.Close()
			_ = os.Remove(tmp)
		}
	}()

	if _, err = f.Write(data); err != nil {
		return fmt.Errorf("writing %s: %w", tmp, err)
	}
	if err = f.Sync(); err != nil {
		return fmt.Errorf("syncing %s: %w", tmp, err)
	}
	if err = f.Chmod(perm); err != nil {
		return fmt.Errorf("setting the mode on %s: %w", tmp, err)
	}
	if err = f.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", tmp, err)
	}
	if err = os.Rename(tmp, filepath.Join(dir, name)); err != nil {
		return fmt.Errorf("renaming %s to %s: %w", tmp, name, err)
	}

	if d, derr := os.Open(dir); derr == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}
