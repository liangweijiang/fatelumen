package repository

import (
	"context"
	"testing"

	"fatelumen/backend/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLLMConfigRepoListsOnlyEnabledRoutesInPriorityOrder(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.LLMProviderConfig{}, &model.LLMModelConfig{}); err != nil {
		t.Fatal(err)
	}
	enabled := model.LLMProviderConfig{Code: "enabled", Name: "Enabled", BaseURL: "https://enabled.test/v1", APIKeyCiphertext: "cipher", APIKeyHint: "hint", Enabled: true}
	disabled := model.LLMProviderConfig{Code: "disabled", Name: "Disabled", BaseURL: "https://disabled.test/v1", APIKeyCiphertext: "cipher", APIKeyHint: "hint", Enabled: false}
	if err := db.Create(&enabled).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&disabled).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&disabled).Update("enabled", false).Error; err != nil {
		t.Fatal(err)
	}
	rows := []model.LLMModelConfig{
		{ProviderID: enabled.ID, Name: "second", ModelID: "second", Priority: 20, MaxRetries: 3, Enabled: true},
		{ProviderID: enabled.ID, Name: "first", ModelID: "first", Priority: 10, MaxRetries: 3, Enabled: true},
		{ProviderID: enabled.ID, Name: "off-model", ModelID: "off-model", Priority: 1, MaxRetries: 3, Enabled: false},
		{ProviderID: disabled.ID, Name: "off-provider", ModelID: "off-provider", Priority: 1, MaxRetries: 3, Enabled: true},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&rows[2]).Update("enabled", false).Error; err != nil {
		t.Fatal(err)
	}
	got, err := NewLLMConfigRepo(db).ListEnabledRoutes(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ModelID != "first" || got[1].ModelID != "second" {
		t.Fatalf("unexpected routes: %+v", got)
	}
	if got[0].Provider.ID != enabled.ID || !got[0].Provider.Enabled {
		t.Fatalf("provider association missing: %+v", got[0].Provider)
	}
}
