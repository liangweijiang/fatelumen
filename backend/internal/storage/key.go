package storage

import "fmt"

func ReadingKey(userID, readingID uint64) string {
	return fmt.Sprintf("readings/%d/%d.png", userID, readingID)
}

func ReportKey(userID, reportID uint64) string {
	return fmt.Sprintf("reports/%d/%d.pdf", userID, reportID)
}
