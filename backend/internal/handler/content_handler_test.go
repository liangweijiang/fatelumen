package handler

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestResolveContentSummary(t *testing.T) {
	markdown := "# 标题\n\n这是自动摘要。"
	if got := resolveContentSummary("人工摘要", markdown); got != "人工摘要" {
		t.Fatalf("manual summary was not preserved: %q", got)
	}
	if got := resolveContentSummary("", markdown); got != "标题 这是自动摘要。" {
		t.Fatalf("unexpected generated summary: %q", got)
	}
}

func TestImportMarkdown(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "sample.md")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("# 测试标题\n\n这是正文。<script>alert(1)</script>"))
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/content/import-markdown", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	NewContentHandler(nil).ImportMarkdown(c)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", w.Code, w.Body.String())
	}
	if bytes.Contains(w.Body.Bytes(), []byte("<script")) {
		t.Fatalf("unsafe script tag was not escaped: %s", w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("测试标题")) {
		t.Fatalf("parsed title missing: %s", w.Body.String())
	}
}

func TestNormalizeContentKeys(t *testing.T) {
	keys := normalizeContentKeys([]string{" shared ", "shared", "", "other"})
	if len(keys) != 2 || keys[0] != "shared" || keys[1] != "other" {
		t.Fatalf("unexpected normalized keys: %#v", keys)
	}
}
