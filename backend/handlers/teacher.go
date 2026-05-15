package handlers

import (
	"net/http"
	"strconv"
	"time"

	"interview-platform/database"
	"interview-platform/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func teacherStudentIDs(userID uint, role string, classID string) []uint {
	if role == "teacher" {
		var ids []uint
		if classID != "" {
			database.DB.Raw("SELECT u.id FROM users u JOIN classes c ON c.id = u.class_id WHERE c.teacher_id = ? AND c.id = ? AND u.role = 'student'", userID, classID).Scan(&ids)
		} else {
			database.DB.Raw("SELECT u.id FROM users u JOIN classes c ON c.id = u.class_id WHERE c.teacher_id = ? AND u.role = 'student'", userID).Scan(&ids)
		}
		if ids == nil {
			ids = []uint{}
		}
		return ids
	}
	if classID != "" {
		var ids []uint
		database.DB.Raw("SELECT id FROM users WHERE class_id = ? AND role = 'student'", classID).Scan(&ids)
		if ids == nil {
			ids = []uint{}
		}
		return ids
	}
	return nil
}

func teacherAnswerFilter(query *gorm.DB, userID uint, role string, classID string) *gorm.DB {
	ids := teacherStudentIDs(userID, role, classID)
	if ids != nil {
		if len(ids) == 0 {
			return query.Where("1 = 0")
		}
		return query.Where("user_id IN ?", ids)
	}
	return query
}

func GetTeacherOverview(c *gin.Context) {
	role := c.GetString("role")
	userID := c.GetUint("user_id")
	type row struct {
		StudentCount  int64   `json:"student_count"`
		TotalAnswers  int64   `json:"total_answers"`
		AverageScore  float64 `json:"average_score"`
		QualifiedRate float64 `json:"qualified_rate"`
	}
	var r row
	q := teacherAnswerFilter(database.DB.Model(&models.UserAnswer{}), userID, role, c.Query("class_id"))
	q.Select("COUNT(DISTINCT user_id) as student_count, COUNT(*) as total_answers, COALESCE(AVG(score), 0) as average_score").Scan(&r)
	var qualified, total int64
	teacherAnswerFilter(database.DB.Model(&models.UserAnswer{}).Where("is_qualified = ?", true), userID, role, c.Query("class_id")).Count(&qualified)
	teacherAnswerFilter(database.DB.Model(&models.UserAnswer{}), userID, role, c.Query("class_id")).Count(&total)
	if total > 0 {
		r.QualifiedRate = float64(qualified) / float64(total) * 100
	}
	c.JSON(http.StatusOK, gin.H{"overview": r})
}

func GetTeacherStudents(c *gin.Context) {
	role := c.GetString("role")
	userID := c.GetUint("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	type StudentStat struct {
		UserID       uint    `json:"user_id"`
		Username     string  `json:"username"`
		Email        string  `json:"email"`
		AnswerCount  int64   `json:"answer_count"`
		AverageScore float64 `json:"average_score"`
		Qualified    int64   `json:"qualified"`
		Wrong        int64   `json:"wrong"`
		LastAnswer   string  `json:"last_answer"`
	}

	ids := teacherStudentIDs(userID, role, c.Query("class_id"))
	whereClause := ""
	var args []interface{}
	if ids != nil {
		if len(ids) == 0 {
			c.JSON(http.StatusOK, gin.H{"students": []StudentStat{}, "total": 0, "page": page, "page_size": pageSize})
			return
		}
		whereClause = "WHERE u.id IN (?)"
		args = append(args, ids)
	}

	var total int64
	database.DB.Raw("SELECT COUNT(*) FROM users u "+whereClause, args...).Scan(&total)

	var stats []StudentStat
	queryArgs := append(args, pageSize, (page-1)*pageSize)
	database.DB.Raw(
		`SELECT u.id as user_id, u.username, u.email,
			COUNT(ua.id) as answer_count,
			COALESCE(AVG(ua.score), 0) as average_score,
			SUM(CASE WHEN ua.is_qualified = 1 THEN 1 ELSE 0 END) as qualified,
			SUM(CASE WHEN ua.is_qualified = 0 THEN 1 ELSE 0 END) as wrong,
			COALESCE(MAX(ua.updated_at), u.created_at) as last_answer
		FROM users u
		LEFT JOIN user_answers ua ON ua.user_id = u.id
		`+whereClause+`
		GROUP BY u.id
			ORDER BY last_answer DESC
		LIMIT ? OFFSET ?`, queryArgs...).Scan(&stats)

	c.JSON(http.StatusOK, gin.H{
		"students":  stats,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func GetStudentAnswers(c *gin.Context) {
	role := c.GetString("role")
	teacherID := c.GetUint("user_id")
	studentID := c.Param("id")

	if role == "teacher" {
		ids := teacherStudentIDs(teacherID, role, "")
		if ids == nil || len(ids) == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权查看此学生的回答"})
			return
		}
		found := false
		for _, id := range ids {
			if strconv.Itoa(int(id)) == studentID {
				found = true
				break
			}
		}
		if !found {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权查看此学生的回答"})
			return
		}
	}

	var answers []models.UserAnswer
	database.DB.Where("user_id = ?", studentID).
		Preload("Question").
		Preload("Question.Category").
		Order("updated_at desc").
		Find(&answers)
	c.JSON(http.StatusOK, gin.H{"answers": answers})
}

func GetHotQuestions(c *gin.Context) {
	role := c.GetString("role")
	userID := c.GetUint("user_id")
	qType := c.DefaultQuery("type", "error")

	type HotQuestion struct {
		ID          uint    `json:"id"`
		Title       string  `json:"title"`
		AvgScore    float64 `json:"avg_score"`
		AnswerCount int64   `json:"answer_count"`
		FailRate    float64 `json:"fail_rate"`
	}

	order := "avg_score ASC"
	if qType == "mastered" {
		order = "avg_score DESC"
	}

	ids := teacherStudentIDs(userID, role, c.Query("class_id"))
	var questions []HotQuestion
	if ids != nil && len(ids) == 0 {
		c.JSON(http.StatusOK, gin.H{"questions": questions})
		return
	}
	if ids != nil {
		database.DB.Raw(
			`SELECT q.id, q.title,
				AVG(ua.score) as avg_score,
				COUNT(ua.id) as answer_count,
				SUM(CASE WHEN ua.is_qualified = 0 THEN 1 ELSE 0 END) * 100.0 / COUNT(ua.id) as fail_rate
			FROM questions q
			JOIN user_answers ua ON ua.question_id = q.id
			WHERE ua.user_id IN ?
			GROUP BY q.id
			HAVING COUNT(ua.id) >= 3
			ORDER BY `+order+`
			LIMIT 20`, ids).Scan(&questions)
	} else {
		database.DB.Raw(
			`SELECT q.id, q.title,
				AVG(ua.score) as avg_score,
				COUNT(ua.id) as answer_count,
				SUM(CASE WHEN ua.is_qualified = 0 THEN 1 ELSE 0 END) * 100.0 / COUNT(ua.id) as fail_rate
			FROM questions q
			JOIN user_answers ua ON ua.question_id = q.id
			GROUP BY q.id
			HAVING COUNT(ua.id) >= 3
			ORDER BY `+order+`
			LIMIT 20`).Scan(&questions)
	}

	c.JSON(http.StatusOK, gin.H{"questions": questions})
}

func AnalyzeQuestionErrors(c *gin.Context) {
	userID := c.GetUint("user_id")
	questionID := c.Param("id")
	force := c.Query("force") == "true"

	var question models.Question
	if database.DB.First(&question, questionID).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "题目不存在"})
		return
	}

	if !force && question.ErrorAnalysis != "" {
		c.JSON(http.StatusOK, gin.H{
			"analysis":    question.ErrorAnalysis,
			"analyzed_at": question.ErrorAnalysisAt,
			"cached":      true,
		})
		return
	}

	aiCfg, err := getUserAIConfig(userID)
	if err != nil || aiCfg == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先在个人中心配置 AI 接口"})
		return
	}

	var failed []models.UserAnswer
	database.DB.Where("question_id = ? AND is_qualified = ?", questionID, false).
		Order("score asc").Find(&failed)

	if len(failed) == 0 {
		c.JSON(http.StatusOK, gin.H{"analysis": "该题暂无不合格答案记录。"})
		return
	}

	prompt := `你是教学分析助手。以下是一道面试题和所有不合格学生的回答，请综合所有回答，分析学生们的共性问题、常见错误模式，并给出整体教学建议。

题目：` + question.Title + `
题目内容：` + question.Content + `

不合格学生回答汇总（共 ` + strconv.Itoa(len(failed)) + ` 人）：`

	for _, a := range failed {
		prompt += "\n---\n学生回答（" + strconv.FormatFloat(a.Score, 'f', 1, 64) + "分）：\n" + a.Content
	}

	prompt += `

请从以下角度进行综合分析：
1. 学生主要的薄弱环节和知识盲区
2. 重复出现的错误模式和共性问题
3. 针对这些问题的教学建议和复习重点`

	result, err := aiCfg.EvaluateRaw(prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI分析失败: " + err.Error()})
		return
	}

	now := time.Now()
	database.DB.Model(&question).Updates(map[string]any{
		"error_analysis":    result,
		"error_analysis_at": now,
	})

	c.JSON(http.StatusOK, gin.H{
		"analysis":    result,
		"analyzed_at": now,
		"cached":      false,
	})
}

func GetQuestionAnswers(c *gin.Context) {
	role := c.GetString("role")
	userID := c.GetUint("user_id")
	questionID := c.Param("id")

	var question models.Question
	if database.DB.First(&question, questionID).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "题目不存在"})
		return
	}

	q := database.DB.Where("question_id = ?", questionID).Preload("User").Order("score desc")
	ids := teacherStudentIDs(userID, role, c.Query("class_id"))
	if ids != nil {
		if len(ids) == 0 {
			c.JSON(http.StatusOK, gin.H{"question": question, "answers": []models.UserAnswer{}})
			return
		}
		q = q.Where("user_id IN ?", ids)
	}
	var answers []models.UserAnswer
	q.Find(&answers)

	c.JSON(http.StatusOK, gin.H{
		"question": question,
		"answers":  answers,
	})
}

func GetCategoryStats(c *gin.Context) {
	role := c.GetString("role")
	userID := c.GetUint("user_id")
	type CategoryStat struct {
		CategoryID   uint    `json:"category_id"`
		CategoryName string  `json:"category_name"`
		AvgScore     float64 `json:"avg_score"`
		FailRate     float64 `json:"fail_rate"`
		AnswerCount  int64   `json:"answer_count"`
		QuestionCount int64  `json:"question_count"`
	}
	ids := teacherStudentIDs(userID, role, c.Query("class_id"))
	var stats []CategoryStat
	if ids != nil {
		if len(ids) == 0 {
			c.JSON(http.StatusOK, gin.H{"categories": stats})
			return
		}
		database.DB.Raw(
			`SELECT c.id as category_id, c.name as category_name,
				COALESCE(AVG(ua.score), 0) as avg_score,
				SUM(CASE WHEN ua.is_qualified = 0 THEN 1 ELSE 0 END) * 100.0 / COUNT(ua.id) as fail_rate,
				COUNT(ua.id) as answer_count,
				COUNT(DISTINCT q.id) as question_count
			FROM categories c
			JOIN questions q ON q.category_id = c.id
			JOIN user_answers ua ON ua.question_id = q.id
			WHERE ua.user_id IN ?
			GROUP BY c.id
			ORDER BY fail_rate DESC`, ids).Scan(&stats)
	} else {
		database.DB.Raw(
			`SELECT c.id as category_id, c.name as category_name,
				COALESCE(AVG(ua.score), 0) as avg_score,
				SUM(CASE WHEN ua.is_qualified = 0 THEN 1 ELSE 0 END) * 100.0 / COUNT(ua.id) as fail_rate,
				COUNT(ua.id) as answer_count,
				COUNT(DISTINCT q.id) as question_count
			FROM categories c
			JOIN questions q ON q.category_id = c.id
			JOIN user_answers ua ON ua.question_id = q.id
			GROUP BY c.id
			ORDER BY fail_rate DESC`).Scan(&stats)
	}

	c.JSON(http.StatusOK, gin.H{"categories": stats})
}

func GetCategoryQuestions(c *gin.Context) {
	role := c.GetString("role")
	userID := c.GetUint("user_id")
	categoryID := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	type QuestionStat struct {
		ID              uint    `json:"id"`
		Title           string  `json:"title"`
		AvgScore        float64 `json:"avg_score"`
		FailRate        float64 `json:"fail_rate"`
		AnswerCount     int64   `json:"answer_count"`
		ErrorAnalysis   string  `json:"error_analysis"`
		ErrorAnalysisAt *time.Time `json:"error_analysis_at"`
	}

	ids := teacherStudentIDs(userID, role, c.Query("class_id"))
	filterSQL := ""
	var args []interface{}
	if ids != nil {
		if len(ids) == 0 {
			c.JSON(http.StatusOK, gin.H{"questions": []QuestionStat{}, "total": 0, "page": page, "page_size": pageSize, "category_name": ""})
			return
		}
		filterSQL = " AND ua.user_id IN ?"
		args = append(args, ids)
	}

	var total int64
	totalArgs := append([]interface{}{categoryID}, args...)
	database.DB.Raw(
		`SELECT COUNT(DISTINCT q.id)
		FROM questions q
		JOIN user_answers ua ON ua.question_id = q.id
		WHERE q.category_id = ?`+filterSQL, totalArgs...).Scan(&total)

	var questions []QuestionStat
	queryArgs := append([]interface{}{categoryID}, args...)
	queryArgs = append(queryArgs, pageSize, (page-1)*pageSize)
	database.DB.Raw(
		`SELECT q.id, q.title,
			AVG(ua.score) as avg_score,
			SUM(CASE WHEN ua.is_qualified = 0 THEN 1 ELSE 0 END) * 100.0 / COUNT(ua.id) as fail_rate,
			COUNT(ua.id) as answer_count,
			q.error_analysis,
			q.error_analysis_at
		FROM questions q
		JOIN user_answers ua ON ua.question_id = q.id
		WHERE q.category_id = ?`+filterSQL+`
		GROUP BY q.id
		ORDER BY fail_rate DESC
		LIMIT ? OFFSET ?`, queryArgs...).Scan(&questions)

	var catName string
	database.DB.Raw("SELECT name FROM categories WHERE id = ?", categoryID).Scan(&catName)

	c.JSON(http.StatusOK, gin.H{
		"questions":     questions,
		"total":         total,
		"page":          page,
		"page_size":     pageSize,
		"category_name": catName,
	})
}
