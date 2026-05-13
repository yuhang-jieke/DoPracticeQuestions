package handlers

import (
	"net/http"
	"strconv"

	"interview-platform/database"
	"interview-platform/models"

	"github.com/gin-gonic/gin"
)

func GetQuestions(c *gin.Context) {
	categoryID := c.Query("category_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	search := c.Query("search")

	query := database.DB.Model(&models.Question{}).Preload("Category")

	if categoryID != "" {
		query = query.Where("category_id = ?", categoryID)
	}
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("title LIKE ? OR content LIKE ?", like, like)
	}

	var total int64
	query.Count(&total)

	var questions []models.Question
	query.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at desc").Find(&questions)

	c.JSON(http.StatusOK, gin.H{
		"questions": questions,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func GetQuestion(c *gin.Context) {
	id := c.Param("id")
	var question models.Question
	if err := database.DB.Preload("Category").First(&question, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "题目不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"question": question})
}
