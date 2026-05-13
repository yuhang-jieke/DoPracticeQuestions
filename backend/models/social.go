package models

import "time"

type Comment struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	TopAnswerID  uint      `gorm:"not null;index" json:"top_answer_id"`
	UserID       uint      `gorm:"not null" json:"user_id"`
	User         User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Content      string    `gorm:"type:text;not null" json:"content"`
	CreatedAt    time.Time `json:"created_at"`
}

type Like struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"not null;uniqueIndex:idx_user_answer" json:"user_id"`
	TopAnswerID uint      `gorm:"not null;uniqueIndex:idx_user_answer" json:"top_answer_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type Bookmark struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"not null;uniqueIndex:idx_user_question" json:"user_id"`
	QuestionID uint      `gorm:"not null;uniqueIndex:idx_user_question" json:"question_id"`
	Question   Question  `gorm:"foreignKey:QuestionID" json:"question,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}
