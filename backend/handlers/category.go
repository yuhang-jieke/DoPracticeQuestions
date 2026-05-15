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

func CreateCategory(c *gin.Context) {
	var input struct {
		Name     string `json:"name" binding:"required"`
		Type     string `json:"type" binding:"required"`
		ParentID *uint  `json:"parent_id"`
		SortOrder int   `json:"sort_order"`
		Icon     string `json:"icon"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入分类名称和类型"})
		return
	}
	if input.Type != "tech" && input.Type != "non-tech" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "类型须为 tech 或 non-tech"})
		return
	}
	cat := models.Category{
		Name:      input.Name,
		Type:      models.CategoryType(input.Type),
		ParentID:  input.ParentID,
		SortOrder: input.SortOrder,
		Icon:      input.Icon,
	}
	database.DB.Create(&cat)
	c.JSON(http.StatusCreated, gin.H{"category": cat})
}

func UpdateCategory(c *gin.Context) {
	id := c.Param("id")
	var cat models.Category
	if err := database.DB.First(&cat, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "分类不存在"})
		return
	}
	var input struct {
		Name      string `json:"name"`
		Type      string `json:"type"`
		ParentID  *uint  `json:"parent_id"`
		SortOrder *int   `json:"sort_order"`
		Icon      string `json:"icon"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if input.Name != "" {
		cat.Name = input.Name
	}
	if input.Type != "" {
		if input.Type != "tech" && input.Type != "non-tech" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "类型须为 tech 或 non-tech"})
			return
		}
		cat.Type = models.CategoryType(input.Type)
	}
	if input.ParentID != nil {
		cat.ParentID = input.ParentID
	}
	if input.SortOrder != nil {
		cat.SortOrder = *input.SortOrder
	}
	if input.Icon != "" {
		cat.Icon = input.Icon
	}
	database.DB.Save(&cat)
	c.JSON(http.StatusOK, gin.H{"category": cat})
}

func DeleteCategory(c *gin.Context) {
	id := c.Param("id")
	var cat models.Category
	if err := database.DB.First(&cat, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "分类不存在"})
		return
	}

	tx := database.DB.Begin()
	// Unlink child categories
	tx.Model(&models.Category{}).Where("parent_id = ?", id).Update("parent_id", nil)
	// Unlink questions
	tx.Model(&models.Question{}).Where("category_id = ?", id).Update("category_id", nil)
	tx.Delete(&cat)

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "分类已删除"})
}
