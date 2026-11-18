package storage

import (
	"context"
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

func (l *Local) Delete(ctx context.Context, key string) error {
	if err := os.Remove(l.abs(key)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
