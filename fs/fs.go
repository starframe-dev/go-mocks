package fs

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrNotFound  = errors.New("fs: file or directory not found")
	ErrNotDir    = errors.New("fs: not a directory")
	ErrIsDir     = errors.New("fs: is a directory")
	ErrNotEmpty  = errors.New("fs: directory not empty")
	ErrInvalid   = errors.New("fs: invalid argument")
	ErrReadOnly  = errors.New("fs: file is read-only")
	ErrWriteOnly = errors.New("fs: file is write-only")
)

// FS is a filesystem abstraction.
// MemMapFs implements this interface for testing.
type FS interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm os.FileMode) error
	MkdirAll(path string, perm os.FileMode) error
	Stat(path string) (os.FileInfo, error)
	Remove(path string) error
	RemoveAll(path string) error
	Open(path string) (io.Reader, error)
	Exists(path string) bool
	IsDir(path string) bool
	Walk(root string, fn filepath.WalkFunc) error
}

// Compile-time check that MemMapFs implements FS.
var _ FS = (*MemMapFs)(nil)

// OSFS is real filesystem (used in production).
type OSFS struct{}

func (OSFS) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }
func (OSFS) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}
func (OSFS) MkdirAll(path string, perm os.FileMode) error { return os.MkdirAll(path, perm) }
func (OSFS) Stat(path string) (os.FileInfo, error)        { return os.Stat(path) }
func (OSFS) Remove(path string) error                     { return os.Remove(path) }
func (OSFS) RemoveAll(path string) error                  { return os.RemoveAll(path) }
func (OSFS) Open(path string) (io.Reader, error)          { return os.Open(path) }
func (OSFS) Exists(path string) bool                      { _, err := os.Stat(path); return err == nil }
func (OSFS) IsDir(path string) bool                       { info, err := os.Stat(path); return err == nil && info.IsDir() }
func (OSFS) Walk(root string, fn filepath.WalkFunc) error { return filepath.Walk(root, fn) }

type NodeType int

const (
	NodeFile NodeType = iota
	NodeDir
)

type Node struct {
	Type     NodeType
	Content  []byte
	ModTime  time.Time
	Children map[string]*Node
	mu       sync.RWMutex
}

type FileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
}

func (f *FileInfo) Name() string       { return f.name }
func (f *FileInfo) Size() int64        { return f.size }
func (f *FileInfo) Mode() os.FileMode  { return f.mode }
func (f *FileInfo) ModTime() time.Time { return f.modTime }
func (f *FileInfo) IsDir() bool        { return f.isDir }
func (f *FileInfo) Sys() interface{}   { return nil }

type MemMapFs struct {
	root *Node
	cwd  string
	mu   sync.RWMutex
}

func New() *MemMapFs {
	return &MemMapFs{
		root: &Node{Type: NodeDir, Children: make(map[string]*Node)},
		cwd:  "/",
	}
}

func (f *MemMapFs) normalizePath(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(f.cwd, path))
}

func splitPath(path string) (parent, name string) {
	path = filepath.Clean(path)
	if path == "/" {
		return "/", ""
	}
	i := strings.LastIndex(path, "/")
	if i == 0 {
		return "/", path[1:]
	}
	return path[:i], path[i+1:]
}

func (f *MemMapFs) getNode(path string) *Node {
	path = f.normalizePath(path)
	if path == "/" {
		return f.root
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	node := f.root
	for _, part := range parts {
		if node.Type != NodeDir || node.Children == nil {
			return nil
		}
		var ok bool
		node, ok = node.Children[part]
		if !ok {
			return nil
		}
	}
	return node
}

func (f *MemMapFs) Create(path string) (*File, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	path = f.normalizePath(path)
	parent, name := splitPath(path)
	if parent == "/" && name == "" {
		return nil, ErrInvalid
	}
	parentNode := f.getNode(parent)
	if parentNode == nil {
		return nil, ErrNotFound
	}
	if parentNode.Type != NodeDir {
		return nil, ErrNotDir
	}
	node := &Node{Type: NodeFile, Content: []byte{}, ModTime: time.Now()}
	parentNode.Children[name] = node
	return &File{node: node, path: path, fs: f}, nil
}

func (f *MemMapFs) Mkdir(path string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	path = f.normalizePath(path)
	parent, name := splitPath(path)
	if parent == "/" && name == "" {
		return nil
	}
	parentNode := f.getNode(parent)
	if parentNode == nil {
		return ErrNotFound
	}
	if parentNode.Type != NodeDir {
		return ErrNotDir
	}
	parentNode.Children[name] = &Node{Type: NodeDir, Children: make(map[string]*Node), ModTime: time.Now()}
	return nil
}

func (f *MemMapFs) MkdirAll(path string, _ os.FileMode) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	path = f.normalizePath(path)
	if path == "/" {
		return nil
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	node := f.root
	for _, part := range parts {
		if node.Type != NodeDir {
			return ErrNotDir
		}
		if node.Children == nil {
			node.Children = make(map[string]*Node)
		}
		if _, ok := node.Children[part]; !ok {
			node.Children[part] = &Node{Type: NodeDir, Children: make(map[string]*Node), ModTime: time.Now()}
		}
		node = node.Children[part]
	}
	return nil
}

func (f *MemMapFs) Open(path string) (io.Reader, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	node := f.getNode(path)
	if node == nil {
		return nil, ErrNotFound
	}
	if node.Type != NodeFile {
		return nil, ErrIsDir
	}
	return &File{node: node, path: path, fs: f, readOnly: true}, nil
}

func (f *MemMapFs) OpenFile(path string, flag int, perm os.FileMode) (*File, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	path = f.normalizePath(path)
	node := f.getNode(path)
	switch {
	case flag&os.O_CREATE != 0 && flag&os.O_EXCL != 0 && node != nil:
		return nil, os.ErrExist
	case flag&os.O_CREATE != 0 && node == nil:
		parent, name := splitPath(path)
		parentNode := f.getNode(parent)
		if parentNode == nil {
			return nil, ErrNotFound
		}
		node = &Node{Type: NodeFile, Content: []byte{}, ModTime: time.Now()}
		parentNode.Children[name] = node
	case node == nil:
		return nil, ErrNotFound
	case node.Type != NodeFile:
		return nil, ErrIsDir
	}
	if flag&os.O_TRUNC != 0 {
		node.Content = []byte{}
		node.ModTime = time.Now()
	}
	return &File{
		node:      node,
		path:      path,
		fs:        f,
		readOnly:  flag&(os.O_WRONLY|os.O_RDWR) == 0,
		writeOnly: flag&os.O_WRONLY != 0,
	}, nil
}

func (f *MemMapFs) Stat(path string) (os.FileInfo, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	node := f.getNode(path)
	if node == nil {
		return nil, ErrNotFound
	}
	return f.fileInfo(path, node), nil
}

func (f *MemMapFs) fileInfo(path string, node *Node) os.FileInfo {
	_, name := splitPath(path)
	size := int64(len(node.Content))
	if node.Type == NodeDir {
		size = 0
	}
	return &FileInfo{name: name, size: size, mode: 0644, modTime: node.ModTime, isDir: node.Type == NodeDir}
}

func (f *MemMapFs) Remove(path string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	path = f.normalizePath(path)
	parent, name := splitPath(path)
	parentNode := f.getNode(parent)
	if parentNode == nil {
		return ErrNotFound
	}
	node, ok := parentNode.Children[name]
	if !ok {
		return ErrNotFound
	}
	if node.Type == NodeDir && len(node.Children) > 0 {
		return ErrNotEmpty
	}
	delete(parentNode.Children, name)
	return nil
}

func (f *MemMapFs) RemoveAll(path string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	path = f.normalizePath(path)
	if path == "/" {
		f.root = &Node{Type: NodeDir, Children: make(map[string]*Node)}
		return nil
	}
	parent, name := splitPath(path)
	parentNode := f.getNode(parent)
	if parentNode == nil {
		return nil
	}
	delete(parentNode.Children, name)
	return nil
}

func (f *MemMapFs) ReadDir(path string) ([]os.FileInfo, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	node := f.getNode(path)
	if node == nil {
		return nil, ErrNotFound
	}
	if node.Type != NodeDir {
		return nil, ErrNotDir
	}
	infos := make([]os.FileInfo, 0, len(node.Children))
	for name, child := range node.Children {
		infos = append(infos, f.fileInfo(filepath.Join(path, name), child))
	}
	sort.Slice(infos, func(i, j int) bool { return infos[i].Name() < infos[j].Name() })
	return infos, nil
}

func (f *MemMapFs) ReadFile(path string) ([]byte, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	node := f.getNode(path)
	if node == nil {
		return nil, ErrNotFound
	}
	if node.Type != NodeFile {
		return nil, ErrIsDir
	}
	content := make([]byte, len(node.Content))
	copy(content, node.Content)
	return content, nil
}

func (f *MemMapFs) WriteFile(path string, data []byte, _ os.FileMode) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	path = f.normalizePath(path)
	parent, name := splitPath(path)
	parentNode := f.getNode(parent)
	if parentNode == nil {
		return ErrNotFound
	}
	if parentNode.Type != NodeDir {
		return ErrNotDir
	}
	parentNode.Children[name] = &Node{Type: NodeFile, Content: data, ModTime: time.Now()}
	return nil
}

func (f *MemMapFs) Exists(path string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.getNode(path) != nil
}

func (f *MemMapFs) IsFile(path string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	node := f.getNode(path)
	return node != nil && node.Type == NodeFile
}

func (f *MemMapFs) IsDir(path string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	node := f.getNode(path)
	return node != nil && node.Type == NodeDir
}

func (f *MemMapFs) Chdir(path string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	path = f.normalizePath(path)
	node := f.getNode(path)
	if node == nil {
		return ErrNotFound
	}
	if node.Type != NodeDir {
		return ErrNotDir
	}
	f.cwd = path
	return nil
}

func (f *MemMapFs) Getwd() string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.cwd
}

func (f *MemMapFs) Seed(files map[string]string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for path, content := range files {
		path = f.normalizePath(path)
		parent, name := splitPath(path)
		parentNode := f.root
		if parent != "/" {
			parts := strings.Split(strings.Trim(parent, "/"), "/")
			for _, part := range parts {
				if parentNode.Children == nil {
					parentNode.Children = make(map[string]*Node)
				}
				if _, ok := parentNode.Children[part]; !ok {
					parentNode.Children[part] = &Node{Type: NodeDir, Children: make(map[string]*Node)}
				}
				parentNode = parentNode.Children[part]
			}
		}
		if parentNode.Children == nil {
			parentNode.Children = make(map[string]*Node)
		}
		parentNode.Children[name] = &Node{Type: NodeFile, Content: []byte(content), ModTime: time.Now()}
	}
}

func (f *MemMapFs) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.root = &Node{Type: NodeDir, Children: make(map[string]*Node)}
	f.cwd = "/"
}

func (f *MemMapFs) Walk(path string, fn filepath.WalkFunc) error {
	f.mu.RLock()
	defer f.mu.RUnlock()
	var walkFn func(string, *Node) error
	walkFn = func(path string, node *Node) error {
		if err := fn(path, f.fileInfo(path, node), nil); err != nil {
			return err
		}
		if node.Type == NodeDir && node.Children != nil {
			for name, child := range node.Children {
				if err := walkFn(filepath.Join(path, name), child); err != nil {
					return err
				}
			}
		}
		return nil
	}
	node := f.getNode(path)
	if node == nil {
		return ErrNotFound
	}
	return walkFn(path, node)
}

type File struct {
	node      *Node
	path      string
	offset    int64
	fs        *MemMapFs
	readOnly  bool
	writeOnly bool
	closed    bool
}

func (f *File) Read(p []byte) (int, error) {
	if f.closed {
		return 0, errors.New("fs: file closed")
	}
	if f.writeOnly {
		return 0, ErrWriteOnly
	}
	f.fs.mu.RLock()
	defer f.fs.mu.RUnlock()
	if int(f.offset) >= len(f.node.Content) {
		return 0, io.EOF
	}
	n := copy(p, f.node.Content[f.offset:])
	f.offset += int64(n)
	return n, nil
}

func (f *File) Write(p []byte) (int, error) {
	if f.closed {
		return 0, errors.New("fs: file closed")
	}
	if f.readOnly {
		return 0, ErrReadOnly
	}
	f.fs.mu.Lock()
	defer f.fs.mu.Unlock()

	// Overwrite in place at the current offset, zero-filling any gap, like a real file.
	end := int(f.offset) + len(p)
	content := make([]byte, max(end, len(f.node.Content)))
	copy(content, f.node.Content)
	copy(content[f.offset:], p)
	f.node.Content = content
	f.node.ModTime = time.Now()
	f.offset = int64(end)
	return len(p), nil
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	if f.closed {
		return 0, errors.New("fs: file closed")
	}
	f.fs.mu.Lock()
	defer f.fs.mu.Unlock()
	var base int64
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		base = f.offset
	case io.SeekEnd:
		base = int64(len(f.node.Content))
	default:
		return 0, ErrInvalid
	}
	if base+offset < 0 {
		return 0, ErrInvalid
	}
	f.offset = base + offset
	return f.offset, nil
}

func (f *File) Close() error {
	if f.closed {
		return nil
	}
	f.closed = true
	return nil
}
