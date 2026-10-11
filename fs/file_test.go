package fs_test

import (
	"io"
	"os"
	"testing"

	"github.com/starframe-dev/go-mocks/fs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readAll(t *testing.T, r io.Reader) string {
	t.Helper()
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(data)
}

func TestFS_Open(t *testing.T) {
	f := fs.New()
	require.NoError(t, f.WriteFile("/a.txt", []byte("hello"), 0644))
	require.NoError(t, f.MkdirAll("/dir", 0755))

	t.Run("reads file content", func(t *testing.T) {
		r, err := f.Open("/a.txt")
		require.NoError(t, err)
		assert.Equal(t, "hello", readAll(t, r))
	})

	t.Run("returned reader rejects writes", func(t *testing.T) {
		r, err := f.Open("/a.txt")
		require.NoError(t, err)
		w, ok := r.(io.Writer)
		require.True(t, ok)
		_, err = w.Write([]byte("x"))
		assert.ErrorIs(t, err, fs.ErrReadOnly)
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := f.Open("/missing")
		assert.ErrorIs(t, err, fs.ErrNotFound)
	})

	t.Run("directory", func(t *testing.T) {
		_, err := f.Open("/dir")
		assert.ErrorIs(t, err, fs.ErrIsDir)
	})
}

func TestFS_OpenFile(t *testing.T) {
	t.Run("read-only handle reads and rejects writes", func(t *testing.T) {
		f := fs.New()
		require.NoError(t, f.WriteFile("/a.txt", []byte("hello"), 0644))

		file, err := f.OpenFile("/a.txt", os.O_RDONLY, 0)
		require.NoError(t, err)

		assert.Equal(t, "hello", readAll(t, file))
		_, err = file.Write([]byte("x"))
		assert.ErrorIs(t, err, fs.ErrReadOnly)
	})

	t.Run("write-only handle writes and rejects reads", func(t *testing.T) {
		f := fs.New()
		require.NoError(t, f.WriteFile("/a.txt", []byte("hello"), 0644))

		file, err := f.OpenFile("/a.txt", os.O_WRONLY, 0)
		require.NoError(t, err)

		_, err = file.Read(make([]byte, 1))
		assert.ErrorIs(t, err, fs.ErrWriteOnly)
		_, err = file.Write([]byte("J"))
		require.NoError(t, err)
		content, err := f.ReadFile("/a.txt")
		require.NoError(t, err)
		assert.Equal(t, "Jello", string(content))
	})

	t.Run("read-write handle", func(t *testing.T) {
		f := fs.New()
		require.NoError(t, f.WriteFile("/a.txt", []byte("hello"), 0644))

		file, err := f.OpenFile("/a.txt", os.O_RDWR, 0)
		require.NoError(t, err)

		assert.Equal(t, "hello", readAll(t, file))
		_, err = file.Write([]byte("!"))
		require.NoError(t, err)
	})

	t.Run("O_CREATE makes a missing file", func(t *testing.T) {
		f := fs.New()

		file, err := f.OpenFile("/new.txt", os.O_CREATE|os.O_WRONLY, 0644)
		require.NoError(t, err)
		_, err = file.Write([]byte("data"))
		require.NoError(t, err)

		assert.True(t, f.IsFile("/new.txt"))
	})

	t.Run("O_CREATE with missing parent", func(t *testing.T) {
		f := fs.New()

		_, err := f.OpenFile("/nope/new.txt", os.O_CREATE, 0644)

		assert.ErrorIs(t, err, fs.ErrNotFound)
	})

	t.Run("O_EXCL fails on existing file", func(t *testing.T) {
		f := fs.New()
		require.NoError(t, f.WriteFile("/a.txt", nil, 0644))

		_, err := f.OpenFile("/a.txt", os.O_CREATE|os.O_EXCL, 0644)

		assert.ErrorIs(t, err, os.ErrExist)
	})

	t.Run("O_TRUNC empties the file", func(t *testing.T) {
		f := fs.New()
		require.NoError(t, f.WriteFile("/a.txt", []byte("hello"), 0644))

		_, err := f.OpenFile("/a.txt", os.O_WRONLY|os.O_TRUNC, 0)
		require.NoError(t, err)

		content, err := f.ReadFile("/a.txt")
		require.NoError(t, err)
		assert.Empty(t, content)
	})

	t.Run("missing file without O_CREATE", func(t *testing.T) {
		f := fs.New()

		_, err := f.OpenFile("/missing", os.O_RDONLY, 0)

		assert.ErrorIs(t, err, fs.ErrNotFound)
	})

	t.Run("directory", func(t *testing.T) {
		f := fs.New()
		require.NoError(t, f.MkdirAll("/dir", 0755))

		_, err := f.OpenFile("/dir", os.O_RDONLY, 0)

		assert.ErrorIs(t, err, fs.ErrIsDir)
	})
}

func TestFile_WriteOverwritesInPlace(t *testing.T) {
	f := fs.New()
	require.NoError(t, f.WriteFile("/a.txt", []byte("hello world"), 0644))
	file, err := f.OpenFile("/a.txt", os.O_RDWR, 0)
	require.NoError(t, err)

	_, err = file.Write([]byte("HE"))
	require.NoError(t, err)

	content, err := f.ReadFile("/a.txt")
	require.NoError(t, err)
	assert.Equal(t, "HEllo world", string(content))
}

func TestFile_WriteAppendsAtOffset(t *testing.T) {
	f := fs.New()
	file, err := f.Create("/a.txt")
	require.NoError(t, err)

	_, err = file.Write([]byte("abc"))
	require.NoError(t, err)
	_, err = file.Write([]byte("def"))
	require.NoError(t, err)

	content, err := f.ReadFile("/a.txt")
	require.NoError(t, err)
	assert.Equal(t, "abcdef", string(content))
}

func TestFile_WriteBeyondEndZeroFills(t *testing.T) {
	f := fs.New()
	file, err := f.Create("/a.txt")
	require.NoError(t, err)
	_, err = file.Seek(3, io.SeekStart)
	require.NoError(t, err)

	_, err = file.Write([]byte("x"))
	require.NoError(t, err)

	content, err := f.ReadFile("/a.txt")
	require.NoError(t, err)
	assert.Equal(t, []byte{0, 0, 0, 'x'}, content)
}

func TestFile_ReadAfterSeek(t *testing.T) {
	f := fs.New()
	require.NoError(t, f.WriteFile("/a.txt", []byte("0123456789"), 0644))
	file, err := f.OpenFile("/a.txt", os.O_RDONLY, 0)
	require.NoError(t, err)

	pos, err := file.Seek(2, io.SeekStart)
	require.NoError(t, err)
	assert.EqualValues(t, 2, pos)

	buf := make([]byte, 3)
	n, err := file.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, "234", string(buf[:n]))

	pos, err = file.Seek(1, io.SeekCurrent)
	require.NoError(t, err)
	assert.EqualValues(t, 6, pos)

	pos, err = file.Seek(-2, io.SeekEnd)
	require.NoError(t, err)
	assert.EqualValues(t, 8, pos)
	assert.Equal(t, "89", readAll(t, file))
}

func TestFile_ReadAtEndReturnsEOF(t *testing.T) {
	f := fs.New()
	file, err := f.Create("/empty.txt")
	require.NoError(t, err)

	_, err = file.Read(make([]byte, 1))

	assert.ErrorIs(t, err, io.EOF)
}

func TestFile_SeekErrors(t *testing.T) {
	f := fs.New()
	file, err := f.Create("/a.txt")
	require.NoError(t, err)

	_, negativeErr := file.Seek(-1, io.SeekStart)
	_, whenceErr := file.Seek(0, 99)

	assert.ErrorIs(t, negativeErr, fs.ErrInvalid)
	assert.ErrorIs(t, whenceErr, fs.ErrInvalid)
}

func TestFile_Closed(t *testing.T) {
	f := fs.New()
	file, err := f.Create("/a.txt")
	require.NoError(t, err)
	require.NoError(t, file.Close())

	_, readErr := file.Read(make([]byte, 1))
	_, writeErr := file.Write([]byte("x"))
	_, seekErr := file.Seek(0, io.SeekStart)

	assert.Error(t, readErr)
	assert.Error(t, writeErr)
	assert.Error(t, seekErr)
	assert.NoError(t, file.Close(), "Close is idempotent")
}

func TestFS_IsFile(t *testing.T) {
	f := fs.New()
	require.NoError(t, f.WriteFile("/a.txt", nil, 0644))
	require.NoError(t, f.MkdirAll("/dir", 0755))

	assert.True(t, f.IsFile("/a.txt"))
	assert.False(t, f.IsFile("/dir"))
	assert.False(t, f.IsFile("/missing"))
}

func TestFS_RelativePathsUseWorkingDirectory(t *testing.T) {
	f := fs.New()
	require.NoError(t, f.MkdirAll("/work/sub", 0755))
	require.NoError(t, f.Chdir("/work"))

	require.NoError(t, f.WriteFile("sub/a.txt", []byte("x"), 0644))

	assert.Equal(t, "/work", f.Getwd())
	assert.True(t, f.IsFile("/work/sub/a.txt"))
	assert.True(t, f.Exists("sub/a.txt"))
	require.NoError(t, f.Chdir("sub"))
	assert.Equal(t, "/work/sub", f.Getwd())
}

func TestFS_StructuralErrors(t *testing.T) {
	f := fs.New()
	require.NoError(t, f.WriteFile("/file", []byte("x"), 0644))

	t.Run("Create at root", func(t *testing.T) {
		_, err := f.Create("/")
		assert.ErrorIs(t, err, fs.ErrInvalid)
	})

	t.Run("Create below a file", func(t *testing.T) {
		_, err := f.Create("/file/child")
		assert.ErrorIs(t, err, fs.ErrNotDir)
	})

	t.Run("Mkdir root is a no-op", func(t *testing.T) {
		assert.NoError(t, f.Mkdir("/"))
	})

	t.Run("Mkdir with missing parent", func(t *testing.T) {
		assert.ErrorIs(t, f.Mkdir("/nope/dir"), fs.ErrNotFound)
	})

	t.Run("Mkdir below a file", func(t *testing.T) {
		assert.ErrorIs(t, f.Mkdir("/file/dir"), fs.ErrNotDir)
	})

	t.Run("MkdirAll root is a no-op", func(t *testing.T) {
		assert.NoError(t, f.MkdirAll("/", 0755))
	})

	t.Run("MkdirAll through a file", func(t *testing.T) {
		assert.ErrorIs(t, f.MkdirAll("/file/sub/dir", 0755), fs.ErrNotDir)
	})

	t.Run("WriteFile below a file", func(t *testing.T) {
		assert.ErrorIs(t, f.WriteFile("/file/child", nil, 0644), fs.ErrNotDir)
	})

	t.Run("ReadDir on a file", func(t *testing.T) {
		_, err := f.ReadDir("/file")
		assert.ErrorIs(t, err, fs.ErrNotDir)
	})

	t.Run("ReadDir on a missing path", func(t *testing.T) {
		_, err := f.ReadDir("/missing")
		assert.ErrorIs(t, err, fs.ErrNotFound)
	})

	t.Run("ReadFile on a directory", func(t *testing.T) {
		require.NoError(t, f.MkdirAll("/dir", 0755))
		_, err := f.ReadFile("/dir")
		assert.ErrorIs(t, err, fs.ErrIsDir)
	})

	t.Run("Remove with missing parent", func(t *testing.T) {
		assert.ErrorIs(t, f.Remove("/nope/x"), fs.ErrNotFound)
	})

	t.Run("Remove a non-empty directory", func(t *testing.T) {
		require.NoError(t, f.MkdirAll("/dir", 0755))
		require.NoError(t, f.WriteFile("/dir/x", nil, 0644))
		assert.ErrorIs(t, f.Remove("/dir"), fs.ErrNotEmpty)
	})

	t.Run("RemoveAll with missing parent is a no-op", func(t *testing.T) {
		assert.NoError(t, f.RemoveAll("/nope/x"))
	})

	t.Run("Chdir errors", func(t *testing.T) {
		assert.ErrorIs(t, f.Chdir("/missing"), fs.ErrNotFound)
		assert.ErrorIs(t, f.Chdir("/file"), fs.ErrNotDir)
	})

	t.Run("lookups through a file find nothing", func(t *testing.T) {
		assert.False(t, f.Exists("/file/child"))
	})
}

func TestFS_RemoveAllRootClearsEverything(t *testing.T) {
	f := fs.New()
	require.NoError(t, f.WriteFile("/a.txt", nil, 0644))

	require.NoError(t, f.RemoveAll("/"))

	assert.False(t, f.Exists("/a.txt"))
	assert.True(t, f.IsDir("/"))
}

func TestFS_SeedNestedAndRelative(t *testing.T) {
	f := fs.New()

	f.Seed(map[string]string{
		"/a/b/c.txt": "deep",
		"top.txt":    "top",
	})

	content, err := f.ReadFile("/a/b/c.txt")
	require.NoError(t, err)
	assert.Equal(t, "deep", string(content))
	assert.True(t, f.IsFile("/top.txt"))
}

func TestFS_WalkErrors(t *testing.T) {
	f := fs.New()
	f.Seed(map[string]string{"/a.txt": "", "/b.txt": ""})

	t.Run("missing root", func(t *testing.T) {
		err := f.Walk("/missing", func(string, os.FileInfo, error) error { return nil })
		assert.ErrorIs(t, err, fs.ErrNotFound)
	})

	t.Run("callback error stops the walk", func(t *testing.T) {
		calls := 0
		err := f.Walk("/", func(string, os.FileInfo, error) error {
			calls++
			return assert.AnError
		})

		assert.ErrorIs(t, err, assert.AnError)
		assert.Equal(t, 1, calls)
	})
}

func TestFileInfo(t *testing.T) {
	f := fs.New()
	require.NoError(t, f.WriteFile("/a.txt", []byte("abc"), 0644))
	require.NoError(t, f.MkdirAll("/dir", 0755))

	file, err := f.Stat("/a.txt")
	require.NoError(t, err)
	dir, err := f.Stat("/dir")
	require.NoError(t, err)

	assert.Equal(t, "a.txt", file.Name())
	assert.EqualValues(t, 3, file.Size())
	assert.Equal(t, os.FileMode(0644), file.Mode())
	assert.False(t, file.ModTime().IsZero())
	assert.Nil(t, file.Sys())
	assert.False(t, file.IsDir())
	assert.True(t, dir.IsDir())
	assert.EqualValues(t, 0, dir.Size())
}

func TestOSFS(t *testing.T) {
	var osfs fs.FS = fs.OSFS{}
	root := t.TempDir()
	dir := root + "/sub/dir"
	file := dir + "/a.txt"

	require.NoError(t, osfs.MkdirAll(dir, 0755))
	require.NoError(t, osfs.WriteFile(file, []byte("hello"), 0644))

	content, err := osfs.ReadFile(file)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(content))
	assert.True(t, osfs.Exists(file))
	assert.True(t, osfs.IsDir(dir))
	assert.False(t, osfs.IsDir(file))
	assert.False(t, osfs.Exists(root+"/missing"))

	info, err := osfs.Stat(file)
	require.NoError(t, err)
	assert.EqualValues(t, 5, info.Size())

	r, err := osfs.Open(file)
	require.NoError(t, err)
	assert.Equal(t, "hello", readAll(t, r))
	if closer, ok := r.(io.Closer); ok {
		require.NoError(t, closer.Close())
	}

	var walked []string
	require.NoError(t, osfs.Walk(root, func(path string, _ os.FileInfo, err error) error {
		walked = append(walked, path)
		return err
	}))
	assert.Contains(t, walked, file)

	require.NoError(t, osfs.Remove(file))
	assert.False(t, osfs.Exists(file))
	require.NoError(t, osfs.RemoveAll(root+"/sub"))
	assert.False(t, osfs.Exists(dir))
}
