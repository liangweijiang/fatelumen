package service

import "fatelumen/backend/internal/model"

func alignAnnualFortunes(content *model.ReportContent, annualFortunes []model.AnnualFortune) {
	if len(annualFortunes) == 0 {
		return
	}

	if len(content.YearlyFortune) > len(annualFortunes) {
		content.YearlyFortune = content.YearlyFortune[:len(annualFortunes)]
	}
	for i := range content.YearlyFortune {
		content.YearlyFortune[i].Year = annualFortunes[i].Year
	}

	for ci := range content.Chapters {
		if content.Chapters[ci].Key != "ten_year_years" {
			continue
		}
		if len(content.Chapters[ci].Years) > len(annualFortunes) {
			content.Chapters[ci].Years = content.Chapters[ci].Years[:len(annualFortunes)]
		}
		for yi := range content.Chapters[ci].Years {
			content.Chapters[ci].Years[yi].Year = annualFortunes[yi].Year
			content.Chapters[ci].Years[yi].GanZhi = annualFortunes[yi].GanZhi
		}
	}
}
