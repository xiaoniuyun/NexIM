package storage

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// S3Config RustFS / S3 兼容对象存储配置
type S3Config struct {
	Endpoint  string // 例：http://rustfs:9000
	Bucket    string // 例：nexim-files
	AccessKey string
	SecretKey string
	Region    string // 默认 us-east-1
}

// S3 基于 AWS SigV4 的 S3 兼容客户端（path-style，零第三方依赖）：适配 RustFS / MinIO / AWS S3
type S3 struct {
	base   string // scheme://host[:port][/base]
	host   string
	bucket string
	access string
	secret string
	region string
	hc     *http.Client
	now    func() time.Time // 便于测试固定签名时间
}

func NewS3(cfg S3Config) (*S3, error) {
	ep := strings.TrimSpace(cfg.Endpoint)
	if ep == "" {
		return nil, fmt.Errorf("storage.endpoint 未配置")
	}
	u, err := url.Parse(ep)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("storage.endpoint 非法: %q", ep)
	}
	if u.Scheme == "" {
		u.Scheme = "http"
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("storage.access_key / storage.secret_key 未配置")
	}
	region := cfg.Region
	if region == "" {
		region = "us-east-1"
	}
	bucket := strings.TrimSpace(cfg.Bucket)
	if bucket == "" {
		bucket = "nexim-files"
	}
	return &S3{
		base:   u.Scheme + "://" + u.Host + strings.TrimRight(u.Path, "/"),
		host:   u.Host,
		bucket: bucket,
		access: cfg.AccessKey,
		secret: cfg.SecretKey,
		region: region,
		hc:     &http.Client{Timeout: 60 * time.Second},
		now:    time.Now,
	}, nil
}

func (s *S3) Name() string { return "rustfs" }

// Bucket 返回实际使用的桶名
func (s *S3) Bucket() string { return s.bucket }

// EnsureBucket 桶不存在时自动创建（S3 headBucket 返回 404 即视为不存在）
func (s *S3) EnsureBucket(ctx context.Context) error {
	resp, err := s.do(ctx, http.MethodHead, "", nil, "")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	if resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("s3 head bucket %s: %s", s.bucket, resp.Status)
	}
	cr, cerr := s.do(ctx, http.MethodPut, "", nil, "")
	if cerr != nil {
		return cerr
	}
	defer cr.Body.Close()
	if cr.StatusCode >= 200 && cr.StatusCode < 300 {
		_, _ = io.Copy(io.Discard, cr.Body)
		return nil
	}
	return s.errFrom(cr, "create bucket "+s.bucket)
}

func (s *S3) Put(ctx context.Context, key string, data []byte, contentType string) error {
	resp, err := s.do(ctx, http.MethodPut, key, data, contentType)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	// 桶不存在（首次写入）时自动建桶并重试一次
	snippet := readSnippet(resp)
	if strings.Contains(snippet, "NoSuchBucket") {
		if err := s.EnsureBucket(ctx); err != nil {
			return err
		}
		retry, rerr := s.do(ctx, http.MethodPut, key, data, contentType)
		if rerr != nil {
			return rerr
		}
		if retry.StatusCode >= 200 && retry.StatusCode < 300 {
			retry.Body.Close()
			return nil
		}
		return fmt.Errorf("s3 put %s: %s %s", key, retry.Status, readSnippet(retry))
	}
	return fmt.Errorf("s3 put %s: %s %s", key, resp.Status, snippet)
}

func (s *S3) Open(ctx context.Context, key string) (*Object, error) {
	resp, err := s.do(ctx, http.MethodGet, key, nil, "")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, ErrObjectNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		return nil, fmt.Errorf("s3 get %s: %s %s", key, resp.Status, readSnippet(resp))
	}
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = mimeByKey(key)
	}
	return &Object{Body: resp.Body, Size: resp.ContentLength, ContentType: ct}, nil
}

func (s *S3) Delete(ctx context.Context, key string) error {
	resp, err := s.do(ctx, http.MethodDelete, key, nil, "")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 || resp.StatusCode == http.StatusNotFound {
		return nil
	}
	return fmt.Errorf("s3 delete %s: %s", key, resp.Status)
}

// do 发送 SigV4 签名请求；key 为空表示操作桶本身（HEAD/PUT bucket）
func (s *S3) do(ctx context.Context, method, key string, body []byte, contentType string) (*http.Response, error) {
	return s.doWithHeaders(ctx, method, key, body, contentType, nil)
}

// doWithHeaders 在签名请求上附加自定义头（如 Range），这些头不参与签名
func (s *S3) doWithHeaders(ctx context.Context, method, key string, body []byte, contentType string, extra map[string]string) (*http.Response, error) {
	canonicalURI := s.canonicalPath(key)
	now := s.now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")
	payloadHash := sha256Hex(body)

	canonicalHeaders := "host:" + s.host + "\n" +
		"x-amz-content-sha256:" + payloadHash + "\n" +
		"x-amz-date:" + amzDate + "\n"
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"
	canonicalRequest := strings.Join([]string{method, canonicalURI, "", canonicalHeaders, signedHeaders, payloadHash}, "\n")
	scope := dateStamp + "/" + s.region + "/s3/aws4_request"
	stringToSign := "AWS4-HMAC-SHA256\n" + amzDate + "\n" + scope + "\n" + sha256Hex([]byte(canonicalRequest))
	signature := hex.EncodeToString(hmacSHA256(signingKey(s.secret, dateStamp, s.region, "s3"), stringToSign))
	authorization := "AWS4-HMAC-SHA256 Credential=" + s.access + "/" + scope +
		", SignedHeaders=" + signedHeaders + ", Signature=" + signature

	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, s.base+canonicalURI, rdr)
	if err != nil {
		return nil, err
	}
	// 显式指定 Host，保证与实际 Host 头及签名一致
	req.Host = s.host
	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("x-amz-content-sha256", payloadHash)
	req.Header.Set("Authorization", authorization)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if body != nil {
		req.ContentLength = int64(len(body))
	}
	for k, v := range extra {
		req.Header.Set(k, v)
	}
	return s.hc.Do(req)
}

func (s *S3) OpenRange(ctx context.Context, key string, start, length int64) (*Object, error) {
	rng := fmt.Sprintf("bytes=%d-%d", start, start+length-1)
	resp, err := s.doWithHeaders(ctx, http.MethodGet, key, nil, "", map[string]string{"Range": rng})
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, ErrObjectNotFound
	}
	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, fmt.Errorf("s3 get range %s: %s %s", key, resp.Status, readSnippet(resp))
	}
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = mimeByKey(key)
	}
	if resp.StatusCode == http.StatusOK && start > 0 {
		// 存储端忽略 Range：自行跳过起始偏移
		if _, err := io.CopyN(io.Discard, resp.Body, start); err != nil {
			resp.Body.Close()
			return nil, err
		}
	}
	return &Object{Body: resp.Body, Size: length, ContentType: ct}, nil
}

// canonicalPath 生成 S3 规范路径 /bucket/key，逐段编码（仅保留 A-Za-z0-9-_.~）
func (s *S3) canonicalPath(key string) string {
	p := "/" + s3Escape(s.bucket)
	if key != "" {
		p += "/" + escapeKey(key)
	}
	return p
}

func escapeKey(key string) string {
	segs := strings.Split(key, "/")
	for i, seg := range segs {
		segs[i] = s3Escape(seg)
	}
	return strings.Join(segs, "/")
}

func s3Escape(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if ('A' <= c && c <= 'Z') || ('a' <= c && c <= 'z') || ('0' <= c && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '~' {
			b.WriteByte(c)
			continue
		}
		fmt.Fprintf(&b, "%%%02X", c)
	}
	return b.String()
}

func signingKey(secret, dateStamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), dateStamp)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, service)
	return hmacSHA256(kService, "aws4_request")
}

func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// readSnippet 读取（并关闭）响应体，截断为错误信息摘要
func readSnippet(resp *http.Response) string {
	defer resp.Body.Close()
	return truncateSnippet(resp)
}

func truncateSnippet(resp *http.Response) string {
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	txt := strings.TrimSpace(string(data))
	if len(txt) > 200 {
		txt = txt[:200]
	}
	return txt
}

func (s *S3) errFrom(resp *http.Response, action string) error {
	return fmt.Errorf("s3 %s: %s %s", action, resp.Status, readSnippet(resp))
}
