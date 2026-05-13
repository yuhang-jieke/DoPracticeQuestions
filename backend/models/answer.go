package models

import "time"

type UserAnswer struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserID        uint      `gorm:"not null;index" json:"user_id"`
	User          User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	QuestionID    uint      `gorm:"not null;index" json:"question_id"`
	Question      Question  `gorm:"foreignKey:QuestionID" json:"question,omitempty"`
	Content       string    `gorm:"type:text;not null" json:"content"`
	Score         float64   `gorm:"type:decimal(3,1)" json:"score"`
	PreviousScore *float64  `gorm:"type:decimal(3,1)" json:"previous_score,omitempty"`
	IsQualified   bool      `gorm:"default:false" json:"is_qualified"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type AnswerHistory struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserAnswerID  uint      `gorm:"not null;index" json:"user_answer_id"`
	UserAnswer    UserAnswer `gorm:"foreignKey:UserAnswerID" json:"-"`
	Content       string    `gorm:"type:text;not null" json:"content"`
	Score         float64   `gorm:"type:decimal(3,1)" json:"score"`
	AIFeedback    string    `gorm:"type:text" json:"ai_feedback"`
	CreatedAt     time.Time `json:"created_at"`
}

type TopAnswer struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	QuestionID    uint      `gorm:"not null;index" json:"question_id"`
	UserID        uint      `gorm:"not null" json:"user_id"`
	User          User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Content       string    `gorm:"type:text;not null" json:"content"`
	Score         float64   `gorm:"type:decimal(3,1)" json:"score"`
	LikesCount    int       `gorm:"default:0" json:"likes_count"`
	CommentsCount int       `gorm:"default:0" json:"comments_count"`
	IsAnonymous   bool      `gorm:"default:false" json:"is_anonymous"`
	CreatedAt     time.Time `json:"created_at"`
}
