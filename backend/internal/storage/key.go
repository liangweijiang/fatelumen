package storage

import "fmt"

func ReadingKey(userID, readingID uint64) string {
	return fmt.Sprintf("readings/%d/%d.png", userID, readingID)
}
