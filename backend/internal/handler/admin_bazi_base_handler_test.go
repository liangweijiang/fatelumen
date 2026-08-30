package handler

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"fatelumen/backend/internal/bazi/basedata"
	"fatelumen/backend/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

func callBaziBase(t *testing.T, path string, fn gin.HandlerFunc) response.Resp {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", fn)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x"+path, nil)
	r.ServeHTTP(w, req)
	var out response.Resp
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}
func TestAdminBaziBaseSummary(t *testing.T) {
	h := NewAdminBaziBaseHandler(basedata.V1())
	out := callBaziBase(t, "", h.Summary)
	if out.Code != 0 {
		t.Fatalf("response=%+v", out)
	}
}
func TestAdminBaziBaseSearchChinese(t *testing.T) {
	h := NewAdminBaziBaseHandler(basedata.V1())
	out := callBaziBase(t, "?q=甲&page=1&page_size=20", h.Stems)
	payload, _ := json.Marshal(out.Data)
	if !containsJSON(payload, "甲") {
		t.Fatalf("missing Chinese result: %s", payload)
	}
}
func TestAdminBaziBaseRelationFilter(t *testing.T) {
	h := NewAdminBaziBaseHandler(basedata.V1())
	out := callBaziBase(t, "?type=clash&page=1&page_size=20", h.Relations)
	payload, _ := json.Marshal(out.Data)
	if containsJSON(payload, "three_harmony") || !containsJSON(payload, "clash") {
		t.Fatalf("bad filter: %s", payload)
	}
}

func TestAdminBaziDisplayDictionary(t *testing.T) {
	h := NewAdminBaziBaseHandler(basedata.V1())
	out := callBaziBase(t, "?q=element.power.wood&page=1&page_size=20", h.DisplayDictionary)
	payload, _ := json.Marshal(out.Data)
	if out.Code != 0 || !containsJSON(payload, "木力量") {
		t.Fatalf("display dictionary response=%+v", out)
	}
}
func containsJSON(v []byte, s string) bool {
	for i := 0; i+len(s) <= len(v); i++ {
		if string(v[i:i+len(s)]) == s {
			return true
		}
	}
	return false
}
