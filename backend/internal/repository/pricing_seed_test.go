package repository

import "testing"

func TestDevelopmentPricingPlanSeeds(t *testing.T) {
	plans, err := developmentPricingPlanSeeds()
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 8 {
		t.Fatalf("expected 2 plans x 4 locales, got %d rows", len(plans))
	}
	keys := make(map[string]bool)
	for _, plan := range plans {
		if !plan.Enabled || plan.AmountCents <= 0 || plan.CreditsGranted <= 0 || plan.Title == "" {
			t.Fatalf("invalid seeded plan: %#v", plan)
		}
		keys[plan.SKU+":"+plan.Locale] = true
	}
	if len(keys) != 8 {
		t.Fatalf("expected unique SKU/locale pairs, got %d", len(keys))
	}
}
