package storage

import (
	"context"
	"errors"
	"io"
	"mime"
	"path"
	"strings"
)

// ErrObjectNotFound 对象不存在
var ErrObjectNotFound = errors.New("object not found")

// Object 读取到的对象内容
type Object struct {
	Body        io.ReadCloser
	Size        int64
	ContentType string
}

// ObjectStore 对象存储抽象：localFS（lite）/ rustfs（standard，S3 兼容）
type ObjectStore interface {
	// Name 驱动名，用于启动日志与健康输出
	Name() string
	// Put 写入对象，key 为对象相对路径（不含 bucket）
	Put(ctx context.Context, key string, data []byte, contentType string) error
	// Open 读取对象
	Open(ctx context.Context, key string) (*Object, error)
	// OpenRange 读取对象指定字节区间（音视频拖动/边下边播）
	OpenRange(ctx context.Context, key string, start, length int64) (*Object, error)
	// Delete 删除对象
	Delete(ctx context.Context, key string) error
}

// mimeByKey 按对象扩展名推断 MIME，兜底 application/octet-stream
func mimeByKey(key string) string {
	if t := mime.TypeByExtension(strings.ToLower(path.Ext(key))); t != "" {
		return t
	}
	return "application/octet-stream"
}
