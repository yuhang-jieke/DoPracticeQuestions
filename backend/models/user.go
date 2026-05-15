package models

import "time"

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:50;not null" json:"username"`
	Email        string    `gorm:"uniqueIndex;size:100;not null" json:"email"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	Avatar       string    `gorm:"size:255" json:"avatar"`
	AIApiKey     string    `gorm:"size:255" json:"ai_api_key,omitempty"`
	AIApiURL     string    `gorm:"size:255" json:"ai_api_url,omitempty"`
	AIModel      string    `gorm:"size:100" json:"ai_model,omitempty"`
	Role         string    `gorm:"size:20;not null;default:student" json:"role"`
	ClassID      *uint     `gorm:"default:null" json:"class_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Class struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	TeacherID uint      `gorm:"not null;index" json:"teacher_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
