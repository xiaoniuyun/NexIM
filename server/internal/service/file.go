package service

import (
	"context"
	"fmt"
	stdmime "mime"
	"path/filepath"
	"strings"

	"nexim/server/internal/model"
	"nexim/server/internal/storage"
	"nexim/server/internal/store"
)

type FileService struct {
	S    *store.Store
	Objs storage.ObjectStore // localFS（lite）/ rustfs（standard）
}

func NewFile(s *store.Store, objs storage.ObjectStore) *FileService {
	if objs == nil {
		objs = storage.NewLocal("data/files")
	}
	return &FileService{S: s, Objs: objs}
}

// normalizeMime 移动端上传常带 application/octet-stream，
// 按扩展名兜底识别，保证图片/视频消息按媒体类型渲染
func normalizeMime(name, mime string) string {
	m := strings.ToLower(strings.TrimSpace(mime))
	if i := strings.IndexByte(m, ';'); i > 0 {
		m = m[:i]
	}
	if m != "" && m != "application/octet-stream" {
		return m
	}
	if guessed := stdmime.TypeByExtension(strings.ToLower(filepath.Ext(name))); guessed != "" {
		if i := strings.IndexByte(guessed, ';'); i > 0 {
			guessed = guessed[:i]
		}
		return guessed
	}
	return m
}

func (f *FileService) Upload(uid, name string, size int64, mime, convID string, data []byte) (*model.FileRecord, error) {
	if name == "" || len(data) == 0 {
		return nil, fmt.Errorf("文件名或内容为空")
	}
	mime = normalizeMime(name, mime)
	now := model.Now()
	fileID := fmt.Sprintf("f_%d", now)
	key := fmt.Sprintf("%d/%s", now/1000000, fileID)
	if err := f.Objs.Put(context.Background(), key, data, mime); err != nil {
		return nil, err
	}
	rec := model.FileRecord{
		FileID: fileID, UID: uid, Name: name, Size: size, MimeType: mime,
		Path: key, ConvID: convID, Status: 1, CreatedAt: now,
	}
	if err := f.S.DB.Create(&rec).Error; err != nil {
		return nil, err
	}
	return &rec, nil
}

func (f *FileService) List(uid string) []model.FileRecord {
	var list []model.FileRecord
	f.S.DB.Where("uid = ? AND status = 1", uid).Order("created_at desc").Find(&list)
	return list
}

// Record 仅查询文件元数据（音视频分段请求先取总长度）
func (f *FileService) Record(fileID string) (*model.FileRecord, error) {
	var rec model.FileRecord
	if err := f.S.DB.Where("file_id = ? AND status = 1", fileID).First(&rec).Error; err != nil {
		return nil, err
	}
	return &rec, nil
}

// Open 读取文件对象（本地磁盘或 RustFS/S3）
func (f *FileService) Open(fileID string) (*model.FileRecord, *storage.Object, error) {
	rec, err := f.Record(fileID)
	if err != nil {
		return nil, nil, err
	}
	obj, err := f.Objs.Open(context.Background(), rec.Path)
	if err != nil {
		return nil, nil, err
	}
	return rec, obj, nil
}

// OpenRange 读取文件指定字节区间（边下边播/拖动）
func (f *FileService) OpenRange(fileID string, start, length int64) (*model.FileRecord, *storage.Object, error) {
	rec, err := f.Record(fileID)
	if err != nil {
		return nil, nil, err
	}
	obj, err := f.Objs.OpenRange(context.Background(), rec.Path, start, length)
	if err != nil {
		return nil, nil, err
	}
	return rec, obj, nil
}

func (f *FileService) Delete(fileID, uid string) error {
	return f.S.DB.Model(&model.FileRecord{}).Where("file_id = ? AND uid = ?", fileID, uid).Update("status", 2).Error
}
