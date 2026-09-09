package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/repository"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAdminLLMConfigProviderModelAssociationAndPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.LLMProviderConfig{}, &model.LLMModelConfig{}); err != nil {
		t.Fatal(err)
	}
	h := NewAdminLLMConfigHandler(db, nil, "test-encryption-secret")
	r := gin.New()
	r.POST("/providers", h.CreateProvider)
	r.PUT("/providers/:id", h.UpdateProvider)
	r.GET("/providers", h.ListProviders)
	r.POST("/models", h.CreateModel)
	r.GET("/models", h.ListModels)

	providerResp := performJSON(r, http.MethodPost, "/providers", `{"code":"deepseek","name":"DeepSeek","base_url":"https://api.deepseek.com","api_key":"sk-private-value","enabled":true}`)
	if providerResp.Code != http.StatusOK {
		t.Fatalf("provider status=%d body=%s", providerResp.Code, providerResp.Body.String())
	}
	var providerEnvelope struct {
		Code int                     `json:"code"`
		Data model.LLMProviderConfig `json:"data"`
	}
	if err := json.Unmarshal(providerResp.Body.Bytes(), &providerEnvelope); err != nil {
		t.Fatal(err)
	}
	if providerEnvelope.Data.ID == 0 || providerEnvelope.Data.APIKeyHint == "" {
		t.Fatalf("unexpected provider: %+v", providerEnvelope.Data)
	}
	var persisted model.LLMProviderConfig
	if err := db.First(&persisted, providerEnvelope.Data.ID).Error; err != nil {
		t.Fatal(err)
	}
	if strings.Contains(persisted.APIKeyCiphertext, "sk-private-value") {
		t.Fatal("API key was stored in plaintext")
	}

	customResp := performJSON(r, http.MethodPost, "/providers", `{"name":"Private Gateway","base_url":"http://localhost:18080/v1","api_key":"private-key","enabled":true}`)
	var customEnvelope struct {
		Code int                     `json:"code"`
		Data model.LLMProviderConfig `json:"data"`
	}
	if err := json.Unmarshal(customResp.Body.Bytes(), &customEnvelope); err != nil {
		t.Fatal(err)
	}
	if customEnvelope.Code != 0 || !strings.HasPrefix(customEnvelope.Data.Code, "custom-") {
		t.Fatalf("custom provider code was not generated: %s", customResp.Body.String())
	}

	updated := performJSON(r, http.MethodPut, "/providers/"+jsonNumber(providerEnvelope.Data.ID), `{"code":"changed","name":"DeepSeek Updated","base_url":"https://api.deepseek.com/v1","enabled":true}`)
	if updated.Code != http.StatusOK {
		t.Fatalf("update provider status=%d body=%s", updated.Code, updated.Body.String())
	}
	if err := db.First(&persisted, providerEnvelope.Data.ID).Error; err != nil {
		t.Fatal(err)
	}
	if persisted.Code != "deepseek" {
		t.Fatalf("provider code must be immutable, got %q", persisted.Code)
	}

	invalid := performJSON(r, http.MethodPost, "/models", `{"provider_id":999,"name":"bad","model_id":"bad","priority":1,"max_retries":3,"enabled":true}`)
	var invalidEnvelope struct {
		Code int `json:"code"`
	}
	_ = json.Unmarshal(invalid.Body.Bytes(), &invalidEnvelope)
	if invalidEnvelope.Code == 0 {
		t.Fatalf("missing provider must be rejected: %s", invalid.Body.String())
	}

	body := `{"provider_id":` + jsonNumber(providerEnvelope.Data.ID) + `,"name":"DeepSeek V4 Pro","model_id":"deepseek-v4-pro","priority":1,"max_retries":3,"enabled":true}`
	created := performJSON(r, http.MethodPost, "/models", body)
	var modelEnvelope struct {
		Code int                  `json:"code"`
		Data model.LLMModelConfig `json:"data"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &modelEnvelope); err != nil {
		t.Fatal(err)
	}
	if modelEnvelope.Code != 0 || modelEnvelope.Data.Provider.ID != providerEnvelope.Data.ID {
		t.Fatalf("association missing: %s", created.Body.String())
	}

	list := performJSON(r, http.MethodGet, "/models?page=1&page_size=10&provider_id="+jsonNumber(providerEnvelope.Data.ID), "")
	var listEnvelope struct {
		Data struct {
			Total    int64                  `json:"total"`
			Page     int                    `json:"page"`
			PageSize int                    `json:"page_size"`
			Items    []model.LLMModelConfig `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &listEnvelope); err != nil {
		t.Fatal(err)
	}
	if listEnvelope.Data.Total != 1 || listEnvelope.Data.Page != 1 || listEnvelope.Data.PageSize != 10 || len(listEnvelope.Data.Items) != 1 {
		t.Fatalf("bad pagination: %s", list.Body.String())
	}
}

func TestAdminLLMConfigAuditStoresSafeBeforeAfter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:llm-audit?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.LLMProviderConfig{}, &model.LLMModelConfig{}, &model.AdminAuditLog{}); err != nil {
		t.Fatal(err)
	}
	h := NewAdminLLMConfigHandler(db, repository.NewAuditRepo(db), "test-encryption-secret")
	r := gin.New()
	r.POST("/providers", h.CreateProvider)
	r.PUT("/providers/:id", h.UpdateProvider)

	created := performJSON(r, http.MethodPost, "/providers", `{"code":"deepseek","name":"DeepSeek","base_url":"https://api.deepseek.com","api_key":"sk-never-log-this","enabled":true}`)
	if created.Code != http.StatusOK {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	var envelope struct {
		Data model.LLMProviderConfig `json:"data"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	updated := performJSON(r, http.MethodPut, "/providers/"+jsonNumber(envelope.Data.ID), `{"name":"DeepSeek Paid","base_url":"https://api.deepseek.com/v1","api_key":"sk-replaced-secret","enabled":true}`)
	if updated.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", updated.Code, updated.Body.String())
	}
	var logs []model.AdminAuditLog
	if err := db.Order("id ASC").Find(&logs).Error; err != nil {
		t.Fatal(err)
	}
	if len(logs) != 2 {
		t.Fatalf("audit count=%d want 2", len(logs))
	}
	for _, log := range logs {
		detail := string(log.Detail)
		if strings.Contains(detail, "sk-never-log-this") || strings.Contains(detail, "sk-replaced-secret") || strings.Contains(detail, "ciphertext") {
			t.Fatalf("audit leaked provider secret: %s", detail)
		}
	}
	if !strings.Contains(string(logs[1].Detail), `"api_key_changed":true`) || !strings.Contains(string(logs[1].Detail), `"before"`) || !strings.Contains(string(logs[1].Detail), `"after"`) {
		t.Fatalf("update audit missing safe diff: %s", logs[1].Detail)
	}
}

func performJSON(r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func jsonNumber(value uint64) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
