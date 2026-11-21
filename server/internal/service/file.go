package service

import (
	"context"
	"encoding/json"
	"fmt"
	stdmime "mime"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
	if v := mediaMime(filepath.Ext(name)); v != "" {
		return v
	}
	if guessed := stdmime.TypeByExtension(strings.ToLower(filepath.Ext(name))); guessed != "" {
		if i := strings.IndexByte(guessed, ';'); i > 0 {
			guessed = guessed[:i]
		}
		return guessed
	}
	return m
}

// mediaMime 常见音视频扩展名映射；alpine 镜像默认没有 /etc/mime.types，
// 若不显式映射会得到 application/octet-stream，浏览器 <video> 将拒绝播放
func mediaMime(ext string) string {
	switch strings.ToLower(ext) {
	case ".mp4", ".m4v":
		return "video/mp4"
	case ".mov":
		return "video/quicktime"
	case ".webm":
		return "video/webm"
	case ".avi":
		return "video/x-msvideo"
	case ".mkv":
		return "video/x-matroska"
	case ".3gp":
		return "video/3gpp"
	case ".flv":
		return "video/x-flv"
	case ".wmv":
		return "video/x-ms-wmv"
	case ".mp3":
		return "audio/mpeg"
	case ".m4a", ".aac":
		return "audio/aac"
	case ".wav":
		return "audio/wav"
	case ".ogg", ".opus":
		return "audio/ogg"
	}
	return ""
}

func isVideoFile(name, mime string) bool {
	if strings.HasPrefix(mime, "video/") {
		return true
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".mp4", ".m4v", ".mov", ".webm", ".avi", ".mkv", ".3gp", ".flv", ".wmv":
		return true
	}
	return false
}

func (f *FileService) Upload(uid, name string, size int64, mime, convID string, data []byte) (*model.FileRecord, error) {
	if name == "" || len(data) == 0 {
		return nil, fmt.Errorf("文件名或内容为空")
	}
	mime = normalizeMime(name, mime)
	rec, err := f.save(uid, name, size, mime, convID, data, "")
	if err != nil {
		return nil, err
	}
	// 视频补充时长/分辨率/封面，便于各端气泡直接展示与播放
	if isVideoFile(name, mime) {
		f.fillVideoMeta(rec, data)
	}
	return rec, nil
}

// save 落库并写入对象存储；idSuffix 用于同一毫秒内生成多个文件（如视频封面）避免 ID 冲突
func (f *FileService) save(uid, name string, size int64, mime, convID string, data []byte, idSuffix string) (*model.FileRecord, error) {
	now := model.Now()
	fileID := fmt.Sprintf("f_%d%s", now, idSuffix)
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

// fillVideoMeta 调用 ffprobe/ffmpeg 提取视频元数据并生成封面；
// 宿主机或镜像缺少 ffmpeg 时静默降级，不影响上传本身。
func (f *FileService) fillVideoMeta(rec *model.FileRecord, data []byte) {
	tmp, err := os.CreateTemp("", "nexim-video-*"+filepath.Ext(rec.Name))
	if err != nil {
		return
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return
	}
	tmp.Close()

	duration, width, height, ok := probeVideo(tmpName)
	if !ok {
		return
	}
	rec.DurationMs, rec.Width, rec.Height = duration, width, height

	// 取 1/4 时长处抽帧，避开片头黑帧
	at := duration / 4
	if at <= 0 {
		at = 500
	}
	coverPath := tmpName + ".jpg"
	if !generateCover(tmpName, coverPath, at) {
		f.persistMeta(rec)
		return
	}
	defer os.Remove(coverPath)
	img, err := os.ReadFile(coverPath)
	if err != nil || len(img) == 0 {
		f.persistMeta(rec)
		return
	}
	base := strings.TrimSuffix(rec.Name, filepath.Ext(rec.Name))
	if cover, err := f.save(rec.UID, base+"_cover.jpg", int64(len(img)), "image/jpeg", rec.ConvID, img, "c"); err == nil {
		rec.CoverID = cover.FileID
	}
	f.persistMeta(rec)
}

func (f *FileService) persistMeta(rec *model.FileRecord) {
	f.S.DB.Model(&model.FileRecord{}).Where("file_id = ?", rec.FileID).Updates(map[string]any{
		"duration_ms": rec.DurationMs,
		"width":       rec.Width,
		"height":      rec.Height,
		"cover_id":    rec.CoverID,
	})
}

func probeVideo(path string) (durationMs int64, width, height int, ok bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "ffprobe", "-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height:format=duration",
		"-of", "json", path).Output()
	if err != nil {
		return 0, 0, 0, false
	}
	var r struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}
	if json.Unmarshal(out, &r) != nil {
		return 0, 0, 0, false
	}
	if d, err := strconv.ParseFloat(r.Format.Duration, 64); err == nil && d > 0 {
		durationMs = int64(d * 1000)
	}
	if len(r.Streams) > 0 {
		width, height = r.Streams[0].Width, r.Streams[0].Height
	}
	return durationMs, width, height, durationMs > 0 || (width > 0 && height > 0)
}

func generateCover(videoPath, coverPath string, atMs int64) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	ss := strconv.FormatFloat(float64(atMs)/1000, 'f', 3, 64)
	err := exec.CommandContext(ctx, "ffmpeg", "-y", "-loglevel", "error",
		"-ss", ss, "-i", videoPath, "-frames:v", "1",
		"-vf", "scale=480:-2", "-q:v", "5", coverPath).Run()
	return err == nil
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
