package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"interview-platform/database"
	"interview-platform/models"

	"github.com/gin-gonic/gin"
)

func GetUserAnswers(c *gin.Context) {
	userID := c.GetUint("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var total int64
	database.DB.Model(&models.UserAnswer{}).Where("user_id = ?", userID).Count(&total)

	var answers []models.UserAnswer
	database.DB.Where("user_id = ?", userID).
		Preload("Question").
		Preload("Question.Category").
		Order("updated_at desc").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&answers)

	c.JSON(http.StatusOK, gin.H{
		"answers": answers,
		"total":   total,
		"page":    page,
		"page_size": pageSize,
	})
}

func GetWrongAnswers(c *gin.Context) {
	userID := c.GetUint("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var total int64
	database.DB.Model(&models.UserAnswer{}).
		Where("user_id = ? AND is_qualified = ?", userID, false).
		Count(&total)

	var answers []models.UserAnswer
	database.DB.Where("user_id = ? AND is_qualified = ?", userID, false).
		Preload("Question").
		Preload("Question.Category").
		Order("score asc").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&answers)

	c.JSON(http.StatusOK, gin.H{
		"answers": answers,
		"total":   total,
		"page":    page,
		"page_size": pageSize,
	})
}

func GetUserBookmarks(c *gin.Context) {
	userID := c.GetUint("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var total int64
	database.DB.Model(&models.Bookmark{}).Where("user_id = ?", userID).Count(&total)

	var bookmarks []models.Bookmark
	database.DB.Where("user_id = ?", userID).
		Preload("Question").
		Preload("Question.Category").
		Order("created_at desc").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&bookmarks)

	c.JSON(http.StatusOK, gin.H{
		"bookmarks": bookmarks,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func GetUserStats(c *gin.Context) {
	userID := c.GetUint("user_id")

	var totalAnswers int64
	database.DB.Model(&models.UserAnswer{}).Where("user_id = ?", userID).Count(&totalAnswers)

	var qualifiedCount int64
	database.DB.Model(&models.UserAnswer{}).Where("user_id = ? AND is_qualified = ?", userID, true).Count(&qualifiedCount)

	var wrongCount int64
	database.DB.Model(&models.UserAnswer{}).Where("user_id = ? AND is_qualified = ?", userID, false).Count(&wrongCount)

	var totalBookmarks int64
	database.DB.Model(&models.Bookmark{}).Where("user_id = ?", userID).Count(&totalBookmarks)

	// Average score
	var avgScore struct {
		Avg float64
	}
	database.DB.Model(&models.UserAnswer{}).
		Select("COALESCE(AVG(score), 0) as avg").
		Where("user_id = ?", userID).
		Scan(&avgScore)

	c.JSON(http.StatusOK, gin.H{
		"total_answers":    totalAnswers,
		"qualified_count":  qualifiedCount,
		"wrong_count":      wrongCount,
		"total_bookmarks":  totalBookmarks,
		"average_score":    avgScore.Avg,
	})
}

func CheckBookmark(c *gin.Context) {
	userID := c.GetUint("user_id")
	questionID := c.Param("id")

	var count int64
	database.DB.Model(&models.Bookmark{}).
		Where("user_id = ? AND question_id = ?", userID, questionID).
		Count(&count)

	c.JSON(http.StatusOK, gin.H{"bookmarked": count > 0})
}

func GetQuestionScores(c *gin.Context) {
	userID := c.GetUint("user_id")
	idsStr := c.Query("ids")
	if idsStr == "" {
		c.JSON(http.StatusOK, gin.H{"scores": map[string]interface{}{}})
		return
	}

	var answers []models.UserAnswer
	database.DB.Select("question_id, score, is_qualified").
		Where("user_id = ? AND question_id IN ?", userID, strings.Split(idsStr, ",")).
		Find(&answers)

	scores := make(map[string]interface{})
	for _, a := range answers {
		scores[fmt.Sprintf("%d", a.QuestionID)] = map[string]interface{}{
			"score":        a.Score,
			"is_qualified": a.IsQualified,
		}
	}

	c.JSON(http.StatusOK, gin.H{"scores": scores})
}

func GetUserUploads(c *gin.Context) {
	userID := c.GetUint("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var total int64
	database.DB.Model(&models.Question{}).Where("uploader_id = ?", userID).Count(&total)

	var questions []models.Question
	database.DB.Where("uploader_id = ?", userID).
		Preload("Category").
		Order("created_at desc").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&questions)

	c.JSON(http.StatusOK, gin.H{
		"questions": questions,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func GetAIConfig(c *gin.Context) {
	userID := c.GetUint("user_id")
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"ai_api_key": user.AIApiKey,
		"ai_api_url": user.AIApiURL,
		"ai_model":   user.AIModel,
	})
}

func UpdateAIConfig(c *gin.Context) {
	userID := c.GetUint("user_id")
	var input struct {
		AIApiKey string `json:"ai_api_key"`
		AIApiURL string `json:"ai_api_url"`
		AIModel  string `json:"ai_model"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	database.DB.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]any{
		"ai_api_key": input.AIApiKey,
		"ai_api_url": input.AIApiURL,
		"ai_model":   input.AIModel,
	})
	c.JSON(http.StatusOK, gin.H{"message": "AI 配置已保存"})
}
