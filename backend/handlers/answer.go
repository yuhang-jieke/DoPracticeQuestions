package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"interview-platform/database"
	"interview-platform/models"
	"interview-platform/services"

	"github.com/gin-gonic/gin"
)

func getUserAIConfig(userID uint) (*services.AIConfig, error) {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return nil, err
	}
	if user.AIApiKey == "" || user.AIApiURL == "" {
		return nil, nil
	}
	return services.NewAIConfig(user.AIApiKey, user.AIApiURL, user.AIModel, false), nil
}

func aiFeedbackJSON(r *services.AIEvaluationResult) string {
	b, _ := json.Marshal(map[string]any{
		"score":            r.Score,
		"analysis":         r.Analysis,
		"strengths":        r.Strengths,
		"weaknesses":       r.Weaknesses,
		"reference_answer": r.Reference,
		"improvements":     r.Improvements,
	})
	return string(b)
}

type SubmitAnswerInput struct {
	Content     string `json:"content" binding:"required"`
	IsAnonymous bool   `json:"is_anonymous"`
}

func SubmitAnswer(c *gin.Context) {
	userID := c.GetUint("user_id")
	questionID := c.Param("id")

	var input SubmitAnswerInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入回答内容"})
		return
	}

	var question models.Question
	if err := database.DB.First(&question, questionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "题目不存在"})
		return
	}

	var existing models.UserAnswer
	hasExisting := database.DB.Where("user_id = ? AND question_id = ?", userID, questionID).First(&existing).RowsAffected > 0

	aiReq := &services.AIRequest{
		QuestionTitle:   question.Title,
		QuestionContent: question.Content,
		UserAnswer:      input.Content,
		QuestionType:    string(question.Type),
		IsEdit:          hasExisting,
	}

	if hasExisting {
		aiReq.PreviousAnswer = existing.Content
		aiReq.PreviousScore = &existing.Score
	}

	aiCfg, err := getUserAIConfig(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户配置失败"})
		return
	}
	if aiCfg == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先在个人中心配置您的 AI 接口（API Key、API URL）"})
		return
	}

	result, err := aiCfg.EvaluateAnswer(aiReq)
	if err != nil {
		log.Printf("AI评估失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI评估失败: " + err.Error()})
		return
	}

	if hasExisting {
		oldScore := existing.Score
		oldContent := existing.Content

		database.DB.Create(&models.AnswerHistory{
			UserAnswerID: existing.ID,
			Content:      oldContent,
			Score:        oldScore,
			AIFeedback:   aiFeedbackJSON(result),
		})

		existing.Content = input.Content
		existing.Score = result.Score
		existing.PreviousScore = &oldScore
		existing.IsQualified = result.Score >= 7
		database.DB.Save(&existing)
	} else {
		answer := models.UserAnswer{
			UserID:      userID,
			QuestionID:  question.ID,
			Content:     input.Content,
			Score:       result.Score,
			IsQualified: result.Score >= 7,
		}
		database.DB.Create(&answer)

		database.DB.Create(&models.AnswerHistory{
			UserAnswerID: answer.ID,
			Content:      input.Content,
			Score:        result.Score,
			AIFeedback:   aiFeedbackJSON(result),
		})

		database.DB.Model(&question).UpdateColumn("answer_count", question.AnswerCount+1)
	}

	syncTopAnswers(question.ID)

	scoreDrop := false
	var scoreDropMsg string
	if hasExisting && result.Score < *aiReq.PreviousScore {
		scoreDrop = true
		scoreDropMsg = "注意：本次评分低于您之前的评分，建议查看AI分析了解具体哪些方面有所不足。"
	}

	c.JSON(http.StatusOK, gin.H{
		"score":         result.Score,
		"is_qualified":  result.Score >= 7,
		"analysis":      result.Analysis,
		"strengths":     result.Strengths,
		"weaknesses":    result.Weaknesses,
		"reference":     result.Reference,
		"improvements":  result.Improvements,
		"score_drop":    scoreDrop,
		"score_drop_msg": scoreDropMsg,
		"has_existing":  hasExisting,
		"previous_score": aiReq.PreviousScore,
	})
}

func GetUserAnswer(c *gin.Context) {
	userID := c.GetUint("user_id")
	questionID := c.Param("id")

	var answer models.UserAnswer
	if database.DB.Where("user_id = ? AND question_id = ?", userID, questionID).First(&answer).RowsAffected == 0 {
		c.JSON(http.StatusOK, gin.H{"answered": false})
		return
	}

	// Fetch latest AI feedback from history
	var latest models.AnswerHistory
	database.DB.Where("user_answer_id = ?", answer.ID).Order("created_at desc").First(&latest)

	c.JSON(http.StatusOK, gin.H{
		"answered": true,
		"answer":   answer,
		"feedback": latest.AIFeedback,
	})
}

func GetAnswerHistory(c *gin.Context) {
	userID := c.GetUint("user_id")
	answerID := c.Param("id")

	var answer models.UserAnswer
	if err := database.DB.First(&answer, answerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "回答不存在"})
		return
	}
	if answer.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问"})
		return
	}

	var histories []models.AnswerHistory
	database.DB.Where("user_answer_id = ?", answerID).Order("created_at desc").Find(&histories)

	c.JSON(http.StatusOK, gin.H{"histories": histories})
}

func syncTopAnswers(questionID uint) {
	var topUserAnswers []models.UserAnswer
	database.DB.Where("question_id = ? AND is_qualified = ?", questionID, true).
		Order("score desc").Limit(10).Find(&topUserAnswers)

	var existing []models.TopAnswer
	database.DB.Where("question_id = ?", questionID).Find(&existing)

	existingByUser := make(map[uint]*models.TopAnswer)
	for i := range existing {
		existingByUser[existing[i].UserID] = &existing[i]
	}

	keep := make(map[uint]bool)

	for _, ua := range topUserAnswers {
		keep[ua.UserID] = true
		if ta, ok := existingByUser[ua.UserID]; ok {
			if ta.Content != ua.Content || ta.Score != ua.Score {
				ta.Content = ua.Content
				ta.Score = ua.Score
				database.DB.Model(ta).Select("Content", "Score").Updates(ta)
			}
		} else {
			database.DB.Create(&models.TopAnswer{
				QuestionID: questionID,
				UserID:     ua.UserID,
				Content:    ua.Content,
				Score:      ua.Score,
			})
		}
	}

	for _, ta := range existing {
		if !keep[ta.UserID] && ta.LikesCount == 0 && ta.CommentsCount == 0 {
			database.DB.Delete(&ta)
		}
	}
}

func SubmitAnswerStream(c *gin.Context) {
	userID := c.GetUint("user_id")
	questionID := c.Param("id")

	var input SubmitAnswerInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入回答内容"})
		return
	}

	var question models.Question
	if err := database.DB.First(&question, questionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "题目不存在"})
		return
	}

	aiCfg, err := getUserAIConfig(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户配置失败"})
		return
	}
	if aiCfg == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先在个人中心配置您的 AI 接口"})
		return
	}

	var existing models.UserAnswer
	hasExisting := database.DB.Where("user_id = ? AND question_id = ?", userID, questionID).First(&existing).RowsAffected > 0

	aiReq := &services.AIRequest{
		QuestionTitle:   question.Title,
		QuestionContent: question.Content,
		UserAnswer:      input.Content,
		QuestionType:    string(question.Type),
		IsEdit:          hasExisting,
	}
	if hasExisting {
		aiReq.PreviousAnswer = existing.Content
		aiReq.PreviousScore = &existing.Score
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "不支持流式响应"})
		return
	}

	var finalScore float64
	var finalQualified bool
	fields := map[string]string{"analysis": "", "strengths": "", "weaknesses": "", "improvements": "", "reference": ""}

	writeEvent := func(event, data string) {
		fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, data)
		flusher.Flush()
	}

	err = aiCfg.EvaluateAnswerStream(aiReq, func(ev services.StreamEvent) error {
		switch ev.Type {
		case "score":
			d, _ := json.Marshal(ev.Data)
			writeEvent("score", string(d))
			if m, ok := ev.Data.(map[string]interface{}); ok {
				if s, ok := m["score"].(float64); ok {
					finalScore = s
				}
				if q, ok := m["is_qualified"].(bool); ok {
					finalQualified = q
				}
			}
		case "chunk":
			d, _ := json.Marshal(ev.Data)
			writeEvent("chunk", string(d))
			if m, ok := ev.Data.(map[string]interface{}); ok {
				if f, ok := m["field"].(string); ok {
					if t, ok := m["text"].(string); ok {
						fields[f] += t
					}
				}
			}
		case "done":
		}
		return nil
	})

	if err != nil {
		log.Printf("stream evaluation failed: %v", err)
		writeEvent("error", `{"message":"AI评估失败"}`)
		return
	}

	if hasExisting {
		oldScore := existing.Score
		oldContent := existing.Content
		aiFeedback, _ := json.Marshal(map[string]interface{}{
			"score": finalScore, "analysis": fields["analysis"],
			"strengths": fields["strengths"], "weaknesses": fields["weaknesses"],
			"reference_answer": fields["reference"], "improvements": fields["improvements"],
		})
		database.DB.Create(&models.AnswerHistory{
			UserAnswerID: existing.ID, Content: oldContent,
			Score: oldScore, AIFeedback: string(aiFeedback),
		})
		existing.Content = input.Content
		existing.Score = finalScore
		existing.PreviousScore = &oldScore
		existing.IsQualified = finalQualified
		database.DB.Save(&existing)
	} else {
		answer := models.UserAnswer{
			UserID: userID, QuestionID: question.ID,
			Content: input.Content, Score: finalScore,
			IsQualified: finalQualified,
		}
		database.DB.Create(&answer)
		aiFeedback, _ := json.Marshal(map[string]interface{}{
			"score": finalScore, "analysis": fields["analysis"],
			"strengths": fields["strengths"], "weaknesses": fields["weaknesses"],
			"reference_answer": fields["reference"], "improvements": fields["improvements"],
		})
		database.DB.Create(&models.AnswerHistory{
			UserAnswerID: answer.ID, Content: input.Content,
			Score: finalScore, AIFeedback: string(aiFeedback),
		})
		database.DB.Model(&question).UpdateColumn("answer_count", question.AnswerCount+1)
	}

	syncTopAnswers(question.ID)

	scoreDrop := false
	var scoreDropMsg string
	if hasExisting && finalScore < *aiReq.PreviousScore {
		scoreDrop = true
		scoreDropMsg = "注意：本次评分低于您之前的评分，建议查看AI分析了解具体哪些方面有所不足。"
	}

	doneData, _ := json.Marshal(map[string]interface{}{
		"score_drop":     scoreDrop,
		"score_drop_msg": scoreDropMsg,
		"has_existing":   hasExisting,
		"previous_score": aiReq.PreviousScore,
	})
	writeEvent("done", string(doneData))
}

