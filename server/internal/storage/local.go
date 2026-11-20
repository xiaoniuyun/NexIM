package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

// Local 本地文件系统实现（lite 形态）
type Local struct{ Root string }

func NewLocal(root string) *Local {
	if root == "" {
		root = "data/files"
	}
	_ = os.MkdirAll(root, 0755)
	return &Local{Root: root}
}

func (l *Local) Name() string { return "localFS" }

func (l *Local) abs(key string) string {
	return filepath.Join(l.Root, filepath.FromSlash(key))
}

func (l *Local) Put(ctx context.Context, key string, data []byte, contentType string) error {
	abs := l.abs(key)
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		return err
	}
	return os.WriteFile(abs, data, 0644)
}

func (l *Local) Open(ctx context.Context, key string) (*Object, error) {
	fh, err := os.Open(l.abs(key))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrObjectNotFound
		}
		return nil, err
	}
	st, err := fh.Stat()
	if err != nil {
		_ = fh.Close()
		return nil, err
	}
	return &Object{Body: fh, Size: st.Size(), ContentType: mimeByKey(key)}, nil
}

func (l *Local) OpenRange(ctx context.Context, key string, start, length int64) (*Object, error) {
	fh, err := os.Open(l.abs(key))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrObjectNotFound
		}
		return nil, err
	}
	st, err := fh.Stat()
	if err != nil {
		_ = fh.Close()
		return nil, err
	}
	if start < 0 || start >= st.Size() {
		_ = fh.Close()
		return nil, ErrObjectNotFound
	}
	if length <= 0 || start+length > st.Size() {
		length = st.Size() - start
	}
	if _, err := fh.Seek(start, io.SeekStart); err != nil {
		_ = fh.Close()
		return nil, err
	}
	return &Object{Body: &limitedFile{f: fh, r: io.LimitReader(fh, length)}, Size: length, ContentType: mimeByKey(key)}, nil
}

// limitedFile 区间读取：限制读取长度，Close 时关闭底层文件
type limitedFile struct {
	f *os.File
	r io.Reader
}

func (l *limitedFile) Read(p []byte) (int, error) { return l.r.Read(p) }
func (l *limitedFile) Close() error                { return l.f.Close() }

func (l *Local) Delete(ctx context.Context, key string) error {
	if err := os.Remove(l.abs(key)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
