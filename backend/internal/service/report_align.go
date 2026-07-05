package service

import (
	"sort"

	"fatelumen/backend/internal/model"
)

func mergeReportChapters(chapters []model.Chapter) []model.Chapter {
	if len(chapters) <= 1 {
		return chapters
	}

	byKey := make(map[string]*model.Chapter, len(chapters))
	order := make([]string, 0, len(chapters))
	for _, chapter := range chapters {
		key := chapter.Key
		if key == "" {
			key = chapter.Title
		}
		existing := byKey[key]
		if existing == nil {
			copyChapter := chapter
			byKey[key] = &copyChapter
			order = append(order, key)
			continue
		}
		if existing.No == 0 || (chapter.No > 0 && chapter.No < existing.No) {
			existing.No = chapter.No
		}
		if len([]rune(chapter.Title)) > len([]rune(existing.Title)) {
			existing.Title = chapter.Title
		}
		if len([]rune(chapter.Body)) > len([]rune(existing.Body)) {
			existing.Body = chapter.Body
		}
		existing.StrengthScore = max(existing.StrengthScore, chapter.StrengthScore)
		existing.Cycles = append(existing.Cycles, chapter.Cycles...)
		existing.Tags = append(existing.Tags, chapter.Tags...)
		existing.Years = mergeYearNotes(existing.Years, chapter.Years)
	}

	merged := make([]model.Chapter, 0, len(order))
	for _, key := range order {
		chapter := *byKey[key]
		sort.SliceStable(chapter.Years, func(i, j int) bool {
			return chapter.Years[i].Year < chapter.Years[j].Year
		})
		merged = append(merged, chapter)
	}
	sort.SliceStable(merged, func(i, j int) bool {
		return merged[i].No < merged[j].No
	})
	return merged
}

func mergeYearNotes(left, right []model.YearNote) []model.YearNote {
	if len(right) == 0 {
		return left
	}
	byYear := make(map[int]model.YearNote, len(left)+len(right))
	order := make([]int, 0, len(left)+len(right))
	for _, item := range left {
		if _, ok := byYear[item.Year]; !ok {
			order = append(order, item.Year)
		}
		byYear[item.Year] = item
	}
	for _, item := range right {
		current, ok := byYear[item.Year]
		if !ok {
			order = append(order, item.Year)
			byYear[item.Year] = item
			continue
		}
		if len([]rune(item.Note)) > len([]rune(current.Note)) {
			if item.GanZhi == "" {
				item.GanZhi = current.GanZhi
			}
			byYear[item.Year] = item
		}
	}

	merged := make([]model.YearNote, 0, len(order))
	for _, year := range order {
		merged = append(merged, byYear[year])
	}
	return merged
}

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
