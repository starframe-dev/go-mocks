package contracts

import (
	"io"
	"os"
	"path/filepath"

	"github.com/starframe-dev/go-mocks/fs"
)

// FSContract defines the expected interface for filesystem implementations.
type FSContract interface {
	// ReadFile reads the file named by path and returns its contents.
	ReadFile(path string) ([]byte, error)

	// WriteFile writes data to the file named by path.
	WriteFile(path string, data []byte, perm os.FileMode) error

	// MkdirAll creates a directory named by path.
	MkdirAll(path string, perm os.FileMode) error

	// Stat returns os.FileInfo for the named path.
	Stat(path string) (os.FileInfo, error)

	// Remove removes the named file or directory.
	Remove(path string) error

	// RemoveAll removes path and any children it contains.
	RemoveAll(path string) error

	// Open opens the named file for reading.
	Open(path string) (io.Reader, error)

	// Exists reports whether path exists.
	Exists(path string) bool

	// IsDir reports whether path is a directory.
	IsDir(path string) bool

	// Walk walks the file tree rooted at root.
	Walk(root string, fn filepath.WalkFunc) error
}

// AssertFSContract compiles at compile time that MemMapFs and OSFS implement FSContract.
var (
	_ FSContract = (*fs.MemMapFs)(nil)
	_ FSContract = fs.OSFS{}
)
