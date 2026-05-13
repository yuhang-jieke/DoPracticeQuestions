package handlers

import (
	"net/http"

	"interview-platform/database"
	"interview-platform/models"

	"github.com/gin-gonic/gin"
)

func GetCategories(c *gin.Context) {
	var categories []models.Category
	database.DB.Where("parent_id IS NULL").Preload("Children").Order("sort_order asc").Find(&categories)
	c.JSON(http.StatusOK, gin.H{"categories": categories})
}
