package service

import (
	"fmt"
	"os"
	"path/filepath"

	"nexim/server/internal/model"
	"nexim/server/internal/store"
)

type FileService struct {
	S     *store.Store
	Root  string // 本地存储根目录
}

func NewFile(s *store.Store, root string) *FileService {
	os.MkdirAll(root, 0755)
	return &FileService{S: s, Root: root}
}

func (f *FileService) Upload(uid, name string, size int64, mime, convID string, data []byte) (*model.FileRecord, error) {
	if name == "" || len(data) == 0 {
		return nil, fmt.Errorf("文件名或内容为空")
	}
	now := model.Now()
	fileID := fmt.Sprintf("f_%d", now)
	rel := fmt.Sprintf("%d/%s", now/1000000, fileID)
	abs := filepath.Join(f.Root, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(abs, data, 0644); err != nil {
		return nil, err
	}
	rec := model.FileRecord{
		FileID: fileID, UID: uid, Name: name, Size: size, MimeType: mime,
		Path: rel, ConvID: convID, Status: 1, CreatedAt: now,
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

func (f *FileService) Get(fileID string) (*model.FileRecord, string, error) {
	var rec model.FileRecord
	if err := f.S.DB.Where("file_id = ?", fileID).First(&rec).Error; err != nil {
		return nil, "", err
	}
	return &rec, filepath.Join(f.Root, rec.Path), nil
}

func (f *FileService) Delete(fileID, uid string) error {
	return f.S.DB.Model(&model.FileRecord{}).Where("file_id = ? AND uid = ?", fileID, uid).Update("status", 2).Error
}
