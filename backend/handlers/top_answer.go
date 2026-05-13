package handlers

import (
	"net/http"
	"strconv"

	"interview-platform/database"
	"interview-platform/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func parseUint(s string) uint {
	v, _ := strconv.ParseUint(s, 10, 64)
	return uint(v)
}

func GetTopAnswers(c *gin.Context) {
	questionID := parseUint(c.Param("id"))
	userID, _ := c.Get("user_id")

	var topAnswers []models.TopAnswer
	database.DB.Where("question_id = ?", questionID).
		Preload("User").
		Order("score desc").
		Limit(10).
		Find(&topAnswers)

	// Check which ones the current user has liked
	likedMap := make(map[uint]bool)
	if uid, ok := userID.(uint); ok && uid > 0 {
		var likes []models.Like
		database.DB.Where("user_id = ?", uid).Find(&likes)
		for _, l := range likes {
			likedMap[l.TopAnswerID] = true
		}
	}

	type topAnswerWithLiked struct {
		models.TopAnswer
		Liked bool `json:"liked"`
	}
	result := make([]topAnswerWithLiked, len(topAnswers))
	for i, ta := range topAnswers {
		result[i] = topAnswerWithLiked{TopAnswer: ta, Liked: likedMap[ta.ID]}
	}

	c.JSON(http.StatusOK, gin.H{"top_answers": result})
}

func LikeAnswer(c *gin.Context) {
	userID := c.GetUint("user_id")
	answerID := parseUint(c.Param("id"))

	var existing models.Like
	if database.DB.Where("user_id = ? AND top_answer_id = ?", userID, answerID).First(&existing).RowsAffected > 0 {
		database.DB.Delete(&existing)
		database.DB.Model(&models.TopAnswer{}).Where("id = ?", answerID).
			UpdateColumn("likes_count", gorm.Expr("likes_count - 1"))
		c.JSON(http.StatusOK, gin.H{"liked": false})
		return
	}

	like := models.Like{
		UserID:      userID,
		TopAnswerID: answerID,
	}
	database.DB.Create(&like)
	database.DB.Model(&models.TopAnswer{}).Where("id = ?", answerID).
		UpdateColumn("likes_count", gorm.Expr("likes_count + 1"))
	c.JSON(http.StatusOK, gin.H{"liked": true})
}

func BookmarkQuestion(c *gin.Context) {
	userID := c.GetUint("user_id")
	questionID := parseUint(c.Param("id"))

	var existing models.Bookmark
	if database.DB.Where("user_id = ? AND question_id = ?", userID, questionID).First(&existing).RowsAffected > 0 {
		database.DB.Delete(&existing)
		c.JSON(http.StatusOK, gin.H{"bookmarked": false})
		return
	}

	bookmark := models.Bookmark{
		UserID:     userID,
		QuestionID: questionID,
	}
	database.DB.Create(&bookmark)
	c.JSON(http.StatusOK, gin.H{"bookmarked": true})
}

func AddComment(c *gin.Context) {
	userID := c.GetUint("user_id")
	answerID := parseUint(c.Param("id"))

	var input struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入评论内容"})
		return
	}

	comment := models.Comment{
		TopAnswerID: answerID,
		UserID:      userID,
		Content:     input.Content,
	}
	database.DB.Create(&comment)
	database.DB.Model(&models.TopAnswer{}).Where("id = ?", answerID).
		UpdateColumn("comments_count", gorm.Expr("comments_count + 1"))

	var user models.User
	database.DB.First(&user, userID)
	comment.User = user

	c.JSON(http.StatusCreated, gin.H{"comment": comment})
}

func GetComments(c *gin.Context) {
	answerID := parseUint(c.Param("id"))

	var comments []models.Comment
	database.DB.Where("top_answer_id = ?", answerID).
		Preload("User").
		Order("created_at asc").
		Find(&comments)

	c.JSON(http.StatusOK, gin.H{"comments": comments})
}
