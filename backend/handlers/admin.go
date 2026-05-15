package handlers

import (
	"net/http"
	"strconv"

	"interview-platform/database"
	"interview-platform/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func GetUsers(c *gin.Context) {
	editorRole := c.GetString("role")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	search := c.Query("search")
	roleFilter := c.Query("role")
	classFilter := c.Query("class_id")

	type UserRow struct {
		ID          uint    `json:"id"`
		Username    string  `json:"username"`
		Email       string  `json:"email"`
		Role        string  `json:"role"`
		ClassID     *uint   `json:"class_id"`
		ClassName   *string `json:"class_name"`
		AIApiKey    string  `json:"-"`
		AIApiURL    string  `json:"-"`
		AIModel     string  `json:"-"`
		CreatedAt   string  `json:"created_at"`
		UpdatedAt   string  `json:"updated_at"`
	}

	where := "WHERE 1=1"
	var args []interface{}
	if roleFilter != "" {
		where += " AND u.role = ?"
		args = append(args, roleFilter)
	}
	if search != "" {
		where += " AND (u.username LIKE ? OR u.email LIKE ?)"
		s := "%" + search + "%"
		args = append(args, s, s)
	}
	if classFilter != "" {
		where += " AND u.class_id = ?"
		args = append(args, classFilter)
	}

	var total int64
	countSQL := "SELECT COUNT(*) FROM users u " + where
	database.DB.Raw(countSQL, args...).Scan(&total)

	var users []UserRow
	queryArgs := append(args, 20, (page-1)*20)
	dataSQL := `SELECT u.id, u.username, u.email, u.role, u.class_id, c.name as class_name, u.created_at, u.updated_at
		FROM users u
		LEFT JOIN classes c ON c.id = u.class_id
		` + where + `
		ORDER BY u.created_at DESC
		LIMIT ? OFFSET ?`
	database.DB.Raw(dataSQL, queryArgs...).Scan(&users)

	c.JSON(http.StatusOK, gin.H{
		"users":     users,
		"total":     total,
		"page":      page,
		"page_size": 20,
		"can_edit":  editorRole == "principal",
	})
}

func UpdateUserRole(c *gin.Context) {
	editorRole := c.GetString("role")
	userID := c.Param("id")
	var target models.User
	if err := database.DB.First(&target, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	var input struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供角色"})
		return
	}
	if input.Role != "student" && input.Role != "teacher" && input.Role != "director" && input.Role != "principal" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效角色"})
		return
	}
	// 主任只能提升学生为教师
	if editorRole == "director" {
		if target.Role != "student" || input.Role != "teacher" {
			c.JSON(http.StatusForbidden, gin.H{"error": "仅可将学生提升为教师"})
			return
		}
	}
	// 只有校长可以设置主任/校长
	if (input.Role == "director" || input.Role == "principal") && editorRole != "principal" {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权设置此角色"})
		return
	}
	target.Role = input.Role
	database.DB.Save(&target)
	c.JSON(http.StatusOK, gin.H{"message": "角色已更新"})
}

func DeleteUser(c *gin.Context) {
	userID := c.Param("id")
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	if user.Role == "principal" {
		c.JSON(http.StatusForbidden, gin.H{"error": "不能删除校长"})
		return
	}

	tx := database.DB.Begin()

	// Delete answer histories for this user's answers
	tx.Where("user_answer_id IN (SELECT id FROM user_answers WHERE user_id = ?)", userID).Delete(&models.AnswerHistory{})
	// Delete user answers
	tx.Where("user_id = ?", userID).Delete(&models.UserAnswer{})
	// Delete bookmarks
	tx.Where("user_id = ?", userID).Delete(&models.Bookmark{})
	// Delete comments
	tx.Where("user_id = ?", userID).Delete(&models.Comment{})
	// Delete likes
	tx.Where("user_id = ?", userID).Delete(&models.Like{})
	// Delete top answers
	tx.Where("user_id = ?", userID).Delete(&models.TopAnswer{})
	// Delete the user
	tx.Delete(&user)

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "用户已删除"})
}

func CreateUser(c *gin.Context) {
	var input struct {
		Username string `json:"username" binding:"required,min=2,max=50"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
		Role     string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	if input.Role != "student" && input.Role != "teacher" && input.Role != "director" && input.Role != "principal" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效角色"})
		return
	}

	editorRole := c.GetString("role")
	if (input.Role == "director" || input.Role == "principal") && editorRole != "principal" {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权创建此角色"})
		return
	}

	var existing models.User
	if database.DB.Where("email = ? OR username = ?", input.Email, input.Username).First(&existing).RowsAffected > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "邮箱或用户名已存在"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	user := models.User{
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: string(hash),
		Role:         input.Role,
	}
	database.DB.Create(&user)
	c.JSON(http.StatusCreated, gin.H{"user": user})
}

func ResetUserPassword(c *gin.Context) {
	userID := c.Param("id")
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	editorRole := c.GetString("role")
	if user.Role == "principal" && editorRole != "principal" {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作"})
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	database.DB.Model(&user).Update("password_hash", string(hash))
	c.JSON(http.StatusOK, gin.H{"message": "密码已重置为 123456"})
}
