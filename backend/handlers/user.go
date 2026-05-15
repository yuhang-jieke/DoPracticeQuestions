package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"interview-platform/database"
	"interview-platform/models"
	"interview-platform/services"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
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
	role := c.GetString("role")
	if role != "teacher" && role != "director" && role != "principal" {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问"})
		return
	}
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
		"has_config": user.AIApiKey != "" && user.AIApiURL != "",
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

	// 获取已存储的配置，空字段用已存值兜底
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	key := input.AIApiKey
	if key == "" {
		key = user.AIApiKey
	}
	url := input.AIApiURL
	if url == "" {
		url = user.AIApiURL
	}
	model := input.AIModel
	if model == "" {
		model = user.AIModel
	}

	if key == "" || url == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "首次配置需要填写 API Key 和 API 地址"})
		return
	}

	cfg := services.NewAIConfig(key, url, model, false)
	if err := cfg.TestConnection(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "连接测试失败: " + err.Error()})
		return
	}

	database.DB.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]any{
		"ai_api_key": key,
		"ai_api_url": url,
		"ai_model":   model,
	})
	c.JSON(http.StatusOK, gin.H{"message": "AI 配置已保存"})
}

func ChangePassword(c *gin.Context) {
	userID := c.GetUint("user_id")
	var input struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "新密码至少6位"})
		return
	}
	if input.OldPassword == input.NewPassword {
		c.JSON(http.StatusBadRequest, gin.H{"error": "新密码不能与旧密码相同"})
		return
	}
	var user models.User
	if database.DB.First(&user, userID).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.OldPassword)) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "原密码错误"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "修改失败"})
		return
	}
	database.DB.Model(&user).Update("password_hash", string(hash))
	c.JSON(http.StatusOK, gin.H{"message": "密码已修改"})
}
