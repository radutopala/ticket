//go:build windows

package ticket

import "os"

// lockFile is a no-op on Windows.
// Windows file locking works differently and is not implemented here.
func lockFile(f *os.File) error {
	return nil
}

// unlockFile is a no-op on Windows.
func unlockFile(f *os.File) error {
	return nil
}
