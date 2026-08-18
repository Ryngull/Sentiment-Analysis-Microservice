package models

import (
	"time"
)

// User represents a row in the users table.
type User struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	Username  string    `gorm:"type:varchar(50);not null;unique"`
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP"`
}

// AnalysisRecord represents a persisted row in the analysis_records table.
type AnalysisRecord struct {
	ID             uint      `gorm:"primaryKey;autoIncrement"`
	UserID         uint      `gorm:"not null"`
	RawText        string    `gorm:"type:text;not null"`
	SentimentLabel string    `gorm:"type:varchar(20);not null"`
	SentimentScore float64   `gorm:"type:numeric(4,3);not null"`
	CreatedAt      time.Time `gorm:"default:CURRENT_TIMESTAMP"`
}
