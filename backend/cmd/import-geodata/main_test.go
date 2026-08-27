package main

import (
	"testing"

	"fatelumen/backend/internal/model"
)

func TestKeepLocalizedChinaCities(t *testing.T) {
	rows := []model.GeoCity{
		{GeoNameID: 1, CountryCode: "CN", Admin1Code: "03", Admin1Name: "Jiangxi", NameEN: "Nanchang", NameZH: "南昌"},
		{GeoNameID: 2, CountryCode: "CN", Admin1Code: "03", Admin1Name: "Jiangxi", NameEN: "Antang", NameZH: "Antang"},
		{GeoNameID: 3, CountryCode: "US", Admin1Code: "TX", Admin1Name: "Texas", NameEN: "Austin", NameZH: "Austin"},
	}
	stats := importStats{}

	got := keepLocalizedChinaCities(rows, &stats)
	if len(got) != 2 {
		t.Fatalf("expected two retained cities, got %d", len(got))
	}
	if got[0].NameZH != "南昌" || got[0].Admin1Name != "江西" {
		t.Fatalf("unexpected localized China city: %+v", got[0])
	}
	if got[1].CountryCode != "US" {
		t.Fatalf("non-China city should remain unchanged: %+v", got[1])
	}
	if stats.DroppedChinaWithoutZH != 1 {
		t.Fatalf("expected one dropped China city, got %d", stats.DroppedChinaWithoutZH)
	}
}

func TestContainsHan(t *testing.T) {
	if !containsHan("乌鲁木齐") {
		t.Fatal("expected Chinese name to contain Han characters")
	}
	if containsHan("Urumqi") {
		t.Fatal("English fallback must not be treated as Chinese")
	}
}

func TestExcludedCountryCodes(t *testing.T) {
	want := []string{"AN", "AQ", "BV", "CS", "HM", "UM"}
	if len(excludedCountryCodes) != len(want) {
		t.Fatalf("unexpected exclusion count: %d", len(excludedCountryCodes))
	}
	for _, code := range want {
		if _, ok := excludedCountryCodes[code]; !ok {
			t.Fatalf("country code %s must be excluded", code)
		}
	}
	if _, excluded := excludedCountryCodes["CN"]; excluded {
		t.Fatal("active countries must remain importable")
	}
}
