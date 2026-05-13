package database

import (
	"interview-platform/models"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init(dsn string) error {
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}
	return autoMigrate()
}

func autoMigrate() error {
	return DB.AutoMigrate(
		&models.User{},
		&models.Category{},
		&models.Question{},
		&models.UserAnswer{},
		&models.AnswerHistory{},
		&models.TopAnswer{},
		&models.Comment{},
		&models.Like{},
		&models.Bookmark{},
	)
}
