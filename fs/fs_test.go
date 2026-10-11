package fs_test

import (
	"os"
	"testing"

	"github.com/human-horizon/go-mocks/fs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFS_Create(t *testing.T) {
	f := fs.New()

	t.Run("creates new file", func(t *testing.T) {
		file, err := f.Create("/test.txt")
		require.NoError(t, err)
		require.NotNil(t, file)
		assert.NoError(t, file.Close())
	})

	t.Run("creates nested files", func(t *testing.T) {
		require.NoError(t, f.MkdirAll("/parent", 0755))
		file, err := f.Create("/parent/child.txt")
		require.NoError(t, err)
		require.NotNil(t, file)
		assert.NoError(t, file.Close())
	})

	t.Run("returns error for missing parent", func(t *testing.T) {
		_, err := f.Create("/nonexistent/parent/file.txt")
		assert.ErrorIs(t, err, fs.ErrNotFound)
	})
}

func TestFS_WriteFile(t *testing.T) {
	f := fs.New()

	t.Run("writes file content", func(t *testing.T) {
		err := f.WriteFile("/test.txt", []byte("Hello World"), 0644)
		require.NoError(t, err)

		content, err := f.ReadFile("/test.txt")
		require.NoError(t, err)
		assert.Equal(t, []byte("Hello World"), content)
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		require.NoError(t, f.WriteFile("/test.txt", []byte("Original"), 0644))
		require.NoError(t, f.WriteFile("/test.txt", []byte("Updated"), 0644))

		content, err := f.ReadFile("/test.txt")
		require.NoError(t, err)
		assert.Equal(t, []byte("Updated"), content)
	})

	t.Run("returns error for missing parent", func(t *testing.T) {
		err := f.WriteFile("/nonexistent/file.txt", []byte("data"), 0644)
		assert.ErrorIs(t, err, fs.ErrNotFound)
	})
}

func TestFS_ReadFile(t *testing.T) {
	f := fs.New()
	require.NoError(t, f.WriteFile("/test.txt", []byte("Hello"), 0644))

	t.Run("reads file content", func(t *testing.T) {
		content, err := f.ReadFile("/test.txt")
		require.NoError(t, err)
		assert.Equal(t, []byte("Hello"), content)
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		_, err := f.ReadFile("/nonexistent.txt")
		assert.ErrorIs(t, err, fs.ErrNotFound)
	})

	t.Run("returns error when reading directory", func(t *testing.T) {
		require.NoError(t, f.Mkdir("/mydir"))
		_, err := f.ReadFile("/mydir")
		assert.ErrorIs(t, err, fs.ErrIsDir)
	})
}

func TestFS_Mkdir(t *testing.T) {
	f := fs.New()

	t.Run("creates directory", func(t *testing.T) {
		err := f.Mkdir("/mydir")
		require.NoError(t, err)
		assert.True(t, f.IsDir("/mydir"))
	})

	t.Run("creates nested directories", func(t *testing.T) {
		err := f.MkdirAll("/a/b/c", 0755)
		require.NoError(t, err)
		assert.True(t, f.IsDir("/a"))
		assert.True(t, f.IsDir("/a/b"))
		assert.True(t, f.IsDir("/a/b/c"))
	})

	t.Run("returns error for missing parent", func(t *testing.T) {
		err := f.Mkdir("/nonexistent/parent")
		assert.ErrorIs(t, err, fs.ErrNotFound)
	})
}

func TestFS_ReadDir(t *testing.T) {
	f := fs.New()
	require.NoError(t, f.Mkdir("/mydir"))
	require.NoError(t, f.WriteFile("/mydir/file1.txt", []byte{}, 0644))
	require.NoError(t, f.WriteFile("/mydir/file2.txt", []byte{}, 0644))

	t.Run("lists directory contents", func(t *testing.T) {
		entries, err := f.ReadDir("/mydir")
		require.NoError(t, err)
		assert.Len(t, entries, 2)

		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		assert.Contains(t, names, "file1.txt")
		assert.Contains(t, names, "file2.txt")
	})

	t.Run("returns error for missing directory", func(t *testing.T) {
		_, err := f.ReadDir("/nonexistent")
		assert.ErrorIs(t, err, fs.ErrNotFound)
	})
}

func TestFS_Remove(t *testing.T) {
	f := fs.New()
	require.NoError(t, f.WriteFile("/test.txt", []byte("content"), 0644))

	t.Run("removes file", func(t *testing.T) {
		err := f.Remove("/test.txt")
		require.NoError(t, err)
		assert.False(t, f.Exists("/test.txt"))
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		err := f.Remove("/nonexistent.txt")
		assert.ErrorIs(t, err, fs.ErrNotFound)
	})

	t.Run("removes file with Remove", func(t *testing.T) {
		require.NoError(t, f.Mkdir("/mydir"))
		require.NoError(t, f.WriteFile("/mydir/file.txt", []byte{}, 0644))
		err := f.Remove("/mydir/file.txt")
		assert.NoError(t, err)
	})
}

func TestFS_RemoveAll(t *testing.T) {
	f := fs.New()
	require.NoError(t, f.MkdirAll("/parent/child", 0755))
	require.NoError(t, f.WriteFile("/parent/child/file.txt", []byte{}, 0644))

	t.Run("removes directory and contents", func(t *testing.T) {
		err := f.RemoveAll("/parent")
		require.NoError(t, err)
		assert.False(t, f.Exists("/parent"))
	})

	t.Run("resets when removing root", func(t *testing.T) {
		require.NoError(t, f.WriteFile("/file.txt", []byte{}, 0644))
		err := f.RemoveAll("/")
		require.NoError(t, err)
		assert.False(t, f.Exists("/file.txt"))
	})
}

func TestFS_Stat(t *testing.T) {
	f := fs.New()
	require.NoError(t, f.WriteFile("/test.txt", []byte("Hello"), 0644))
	require.NoError(t, f.Mkdir("/mydir"))

	t.Run("returns file info", func(t *testing.T) {
		info, err := f.Stat("/test.txt")
		require.NoError(t, err)
		assert.Equal(t, "test.txt", info.Name())
		assert.Equal(t, int64(5), info.Size())
		assert.False(t, info.IsDir())
	})

	t.Run("returns directory info", func(t *testing.T) {
		info, err := f.Stat("/mydir")
		require.NoError(t, err)
		assert.Equal(t, "mydir", info.Name())
		assert.True(t, info.IsDir())
	})

	t.Run("returns error for missing path", func(t *testing.T) {
		_, err := f.Stat("/nonexistent")
		assert.ErrorIs(t, err, fs.ErrNotFound)
	})
}

func TestFS_Chdir(t *testing.T) {
	f := fs.New()
	require.NoError(t, f.MkdirAll("/parent/child", 0755))
	require.NoError(t, f.WriteFile("/parent/child/file.txt", []byte{}, 0644))

	t.Run("changes directory", func(t *testing.T) {
		err := f.Chdir("/parent/child")
		require.NoError(t, err)
		assert.Equal(t, "/parent/child", f.Getwd())
	})

	t.Run("returns error for missing directory", func(t *testing.T) {
		err := f.Chdir("/nonexistent")
		assert.ErrorIs(t, err, fs.ErrNotFound)
	})

	t.Run("returns error when cd to file", func(t *testing.T) {
		err := f.Chdir("/parent/child/file.txt")
		assert.ErrorIs(t, err, fs.ErrNotDir)
	})
}

func TestFS_Seed(t *testing.T) {
	f := fs.New()

	f.Seed(map[string]string{
		"/package.json": `{"name":"test"}`,
		"/src/index.ts": `export default {}`,
	})

	t.Run("populates filesystem", func(t *testing.T) {
		content, err := f.ReadFile("/package.json")
		require.NoError(t, err)
		assert.Equal(t, `{"name":"test"}`, string(content))
	})

	t.Run("creates nested files", func(t *testing.T) {
		content, err := f.ReadFile("/src/index.ts")
		require.NoError(t, err)
		assert.Equal(t, `export default {}`, string(content))
	})
}

func TestFS_Reset(t *testing.T) {
	f := fs.New()
	require.NoError(t, f.WriteFile("/file.txt", []byte{}, 0644))
	require.NoError(t, f.Mkdir("/mydir"))

	f.Reset()

	t.Run("clears all files", func(t *testing.T) {
		assert.False(t, f.Exists("/file.txt"))
		assert.False(t, f.Exists("/mydir"))
	})

	t.Run("resets cwd to root", func(t *testing.T) {
		assert.Equal(t, "/", f.Getwd())
	})
}

func TestFS_Exists(t *testing.T) {
	f := fs.New()
	require.NoError(t, f.WriteFile("/test.txt", []byte{}, 0644))
	require.NoError(t, f.Mkdir("/mydir"))

	t.Run("returns true for existing file", func(t *testing.T) {
		assert.True(t, f.Exists("/test.txt"))
	})

	t.Run("returns true for existing directory", func(t *testing.T) {
		assert.True(t, f.Exists("/mydir"))
	})

	t.Run("returns false for non-existent path", func(t *testing.T) {
		assert.False(t, f.Exists("/nonexistent"))
	})
}

func TestFS_Walk(t *testing.T) {
	f := fs.New()
	require.NoError(t, f.MkdirAll("/root/child", 0755))
	require.NoError(t, f.WriteFile("/root/file.txt", []byte{}, 0644))
	require.NoError(t, f.WriteFile("/root/child/nested.txt", []byte{}, 0644))

	t.Run("walks all paths", func(t *testing.T) {
		var paths []string
		err := f.Walk("/root", func(path string, info os.FileInfo, err error) error {
			if err == nil {
				paths = append(paths, path)
			}
			return err
		})
		require.NoError(t, err)

		assert.Contains(t, paths, "/root")
		assert.Contains(t, paths, "/root/file.txt")
		assert.Contains(t, paths, "/root/child")
		assert.Contains(t, paths, "/root/child/nested.txt")
	})
}
