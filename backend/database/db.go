package database

import (
	"log"
	"time"

	"interview-platform/models"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Init(dsn string) error {
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return err
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxOpenConns(80)
	sqlDB.SetMaxIdleConns(20)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	log.Println("数据库连接池已配置 (max_open=80, max_idle=20)")

	return autoMigrate()
}

func autoMigrate() error {
	return DB.AutoMigrate(
		&models.User{},
		&models.Class{},
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
