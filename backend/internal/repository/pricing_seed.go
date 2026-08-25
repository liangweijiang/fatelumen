package repository

import (
	"encoding/json"

	"fatelumen/backend/internal/model"

	"gorm.io/gorm"
)

type pricingSeedTranslation struct {
	locale   string
	title    string
	summary  string
	benefits []string
}

type pricingSeedGroup struct {
	sku                 string
	amountCents         int
	originalAmountCents int
	reportCount         int
	creditsGranted      int
	sortOrder           int
	translations        []pricingSeedTranslation
}

// SeedDevelopmentPricingPlans provides two stable plans for local UI and
// end-to-end acceptance. FirstOrCreate preserves operator edits.
func SeedDevelopmentPricingPlans(db *gorm.DB) error {
	plans, err := developmentPricingPlanSeeds()
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, plan := range plans {
			if err := tx.Where("sku = ? AND locale = ?", plan.SKU, plan.Locale).FirstOrCreate(&plan).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func developmentPricingPlanSeeds() ([]model.PricingPlan, error) {
	groups := []pricingSeedGroup{
		{
			sku: "credits-50", amountCents: 999, originalAmountCents: 0,
			reportCount: 5, creditsGranted: 50, sortOrder: 10,
			translations: []pricingSeedTranslation{
				{locale: "zh", title: "50 积分包", summary: "可兑换 5 份完整报告", benefits: []string{"到账积分 50", "适用于完整报告解锁"}},
				{locale: "en", title: "50 Credit Pack", summary: "Redeem 5 complete reports", benefits: []string{"50 credits added", "For complete report unlocks"}},
				{locale: "ja", title: "50クレジットパック", summary: "完全レポート5件と交換可能", benefits: []string{"50クレジット付与", "完全レポートの解除に利用可能"}},
				{locale: "ko", title: "50 크레딧 패키지", summary: "전체 보고서 5개 교환 가능", benefits: []string{"50 크레딧 지급", "전체 보고서 잠금 해제에 사용"}},
			},
		},
		{
			sku: "credits-120", amountCents: 1999, originalAmountCents: 2397,
			reportCount: 12, creditsGranted: 120, sortOrder: 20,
			translations: []pricingSeedTranslation{
				{locale: "zh", title: "120 积分包", summary: "可兑换 12 份完整报告", benefits: []string{"到账积分 100", "赠送积分 20", "适用于完整报告解锁"}},
				{locale: "en", title: "120 Credit Pack", summary: "Redeem 12 complete reports", benefits: []string{"100 base credits", "20 bonus credits", "For complete report unlocks"}},
				{locale: "ja", title: "120クレジットパック", summary: "完全レポート12件と交換可能", benefits: []string{"基本100クレジット", "ボーナス20クレジット", "完全レポートの解除に利用可能"}},
				{locale: "ko", title: "120 크레딧 패키지", summary: "전체 보고서 12개 교환 가능", benefits: []string{"기본 100 크레딧", "보너스 20 크레딧", "전체 보고서 잠금 해제에 사용"}},
			},
		},
	}

	plans := make([]model.PricingPlan, 0, len(groups)*4)
	for _, group := range groups {
		for _, translation := range group.translations {
			benefits, err := json.Marshal(translation.benefits)
			if err != nil {
				return nil, err
			}
			plans = append(plans, model.PricingPlan{
				SKU: group.sku, Locale: translation.locale, Title: translation.title,
				Summary: translation.summary, Benefits: benefits,
				AmountCents: group.amountCents, OriginalAmountCents: group.originalAmountCents,
				ReportCount: group.reportCount, CreditsGranted: group.creditsGranted,
				Currency: "usd", Enabled: true, SortOrder: group.sortOrder,
			})
		}
	}
	return plans, nil
}
