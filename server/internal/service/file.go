package service

import (
	"context"
	"fmt"

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

func (f *FileService) Upload(uid, name string, size int64, mime, convID string, data []byte) (*model.FileRecord, error) {
	if name == "" || len(data) == 0 {
		return nil, fmt.Errorf("文件名或内容为空")
	}
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

// Open 读取文件对象（本地磁盘或 RustFS/S3）
func (f *FileService) Open(fileID string) (*model.FileRecord, *storage.Object, error) {
	var rec model.FileRecord
	if err := f.S.DB.Where("file_id = ? AND status = 1", fileID).First(&rec).Error; err != nil {
		return nil, nil, err
	}
	obj, err := f.Objs.Open(context.Background(), rec.Path)
	if err != nil {
		return nil, nil, err
	}
	return &rec, obj, nil
}

func (f *FileService) Delete(fileID, uid string) error {
	return f.S.DB.Model(&model.FileRecord{}).Where("file_id = ? AND uid = ?", fileID, uid).Update("status", 2).Error
}
