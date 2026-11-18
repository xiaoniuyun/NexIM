package storage

import (
	"context"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// 固定签名时间：2023-01-01T00:00:00Z
var fixedNow = time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func newTestS3(t *testing.T, captured **http.Request) *S3 {
	t.Helper()
	s, err := NewS3(S3Config{
		Endpoint:  "http://127.0.0.1:9000",
		Bucket:    "nexim-files",
		AccessKey: "nexim",
		SecretKey: "nexim123456",
	})
	if err != nil {
		t.Fatalf("NewS3: %v", err)
	}
	s.now = func() time.Time { return fixedNow }
	s.hc = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if captured != nil {
			*captured = r
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     http.Header{},
		}, nil
	})}
	return s
}

// TestSigningKeyVector 校验 AWS 官方文档给出的签名密钥派生结果
func TestSigningKeyVector(t *testing.T) {
	// https://docs.aws.amazon.com/general/latest/gr/signature-v4-examples.html
	k := signingKey("wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY", "20120215", "us-east-1", "iam")
	if got := hex.EncodeToString(k); got != "f4780e2d9f65fa895f9c67b32ce1baf0b0d8a43505a000a1a9e090d414db404d" {
		t.Fatalf("signing key mismatch: %s", got)
	}
}

// TestPutSignature 校验 PUT 请求的 SigV4 签名与独立实现（Python 版 SigV4）一致
func TestPutSignature(t *testing.T) {
	var req *http.Request
	s := newTestS3(t, &req)
	if err := s.Put(context.Background(), "1672531200/f_1672531200000", []byte("hello rustfs"), "text/plain"); err != nil {
		t.Fatalf("put: %v", err)
	}
	if req == nil {
		t.Fatal("request not captured")
	}
	if got := req.Header.Get("x-amz-content-sha256"); got != "08e45b693a10be2497a90349e9ab587fbc03ace5c61ad7612eccfed287b23434" {
		t.Fatalf("payload hash mismatch: %s", got)
	}
	if got := req.Header.Get("x-amz-date"); got != "20230101T000000Z" {
		t.Fatalf("x-amz-date mismatch: %s", got)
	}
	want := "AWS4-HMAC-SHA256 Credential=nexim/20230101/us-east-1/s3/aws4_request, " +
		"SignedHeaders=host;x-amz-content-sha256;x-amz-date, " +
		"Signature=c1718266d2ffcba61339341f5a46ef757d7d975f23cc6d16452a4395553e9bb4"
	if got := req.Header.Get("Authorization"); got != want {
		t.Fatalf("authorization mismatch:\n got: %s\nwant: %s", got, want)
	}
	if req.URL.Path != "/nexim-files/1672531200/f_1672531200000" {
		t.Fatalf("path mismatch: %s", req.URL.Path)
	}
}

// TestCanonicalPath 校验对象键的 URI 编码（S3 只保留 A-Za-z0-9-_.~）
func TestCanonicalPath(t *testing.T) {
	s := newTestS3(t, nil)
	if got := s.canonicalPath("2024/10/报告 v1.pdf"); got != "/nexim-files/2024/10/%E6%8A%A5%E5%91%8A%20v1.pdf" {
		t.Fatalf("canonical path: %s", got)
	}
}

// TestNewS3RequiresCredentials 缺少凭据时应直接报错（避免以匿名方式静默失败）
func TestNewS3RequiresCredentials(t *testing.T) {
	if _, err := NewS3(S3Config{Endpoint: "http://127.0.0.1:9000", Bucket: "b"}); err == nil {
		t.Fatal("expected error when access/secret key missing")
	}
	if _, err := NewS3(S3Config{Endpoint: "", Bucket: "b", AccessKey: "a", SecretKey: "b"}); err == nil {
		t.Fatal("expected error when endpoint missing")
	}
}
