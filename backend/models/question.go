package models

import "time"

type Difficulty string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyMedium Difficulty = "medium"
	DifficultyHard   Difficulty = "hard"
)

type QuestionType string

const (
	QuestionTech    QuestionType = "tech"
	QuestionNonTech QuestionType = "non-tech"
)

type Question struct {
	ID           uint         `gorm:"primaryKey" json:"id"`
	CategoryID   uint         `gorm:"not null;index" json:"category_id"`
	Category     Category     `gorm:"foreignKey:CategoryID" json:"category"`
	Title        string       `gorm:"size:500;not null" json:"title"`
	Content      string       `gorm:"type:text;not null" json:"content"`
	Tags         string       `gorm:"size:500" json:"tags"`
	Difficulty   Difficulty   `gorm:"size:20;not null;default:medium" json:"difficulty"`
	Type         QuestionType `gorm:"size:20;not null;default:tech" json:"type"`
	AnswerCount  int          `gorm:"default:0" json:"answer_count"`
	UploaderID       *uint        `json:"uploader_id,omitempty"`
	Uploader         *User        `gorm:"foreignKey:UploaderID" json:"uploader,omitempty"`
	ErrorAnalysis    string       `gorm:"type:text" json:"error_analysis,omitempty"`
	ErrorAnalysisAt  *time.Time   `json:"error_analysis_at,omitempty"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
}
