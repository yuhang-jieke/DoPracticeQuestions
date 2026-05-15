package handlers

import (
	"bytes"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"interview-platform/database"
	"interview-platform/models"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/bcrypt"
)

func CreateClass(c *gin.Context) {
	var input struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入班级名称"})
		return
	}
	class := models.Class{
		Name:      input.Name,
		TeacherID: c.GetUint("user_id"),
	}
	database.DB.Create(&class)
	c.JSON(http.StatusCreated, gin.H{"class": class})
}

func GetClasses(c *gin.Context) {
	role := c.GetString("role")
	userID := c.GetUint("user_id")
	type ClassWithCount struct {
		ID           uint   `json:"id"`
		Name         string `json:"name"`
		TeacherID    uint   `json:"teacher_id"`
		StudentCount int64  `json:"student_count"`
		CreatedAt    string `json:"created_at"`
		UpdatedAt    string `json:"updated_at"`
	}
	rows := make([]ClassWithCount, 0)
	rawQuery := `SELECT c.id, c.name, c.teacher_id, COUNT(u.id) as student_count
		FROM classes c LEFT JOIN users u ON u.class_id = c.id`
	if role == "teacher" {
		rawQuery += " WHERE c.teacher_id = ?"
		rawQuery += " GROUP BY c.id"
		database.DB.Raw(rawQuery, userID).Scan(&rows)
	} else {
		rawQuery += " GROUP BY c.id"
		database.DB.Raw(rawQuery).Scan(&rows)
	}
	c.JSON(http.StatusOK, gin.H{"classes": rows})
}

func UpdateClass(c *gin.Context) {
	classID := c.Param("id")
	var class models.Class
	if err := database.DB.First(&class, classID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "班级不存在"})
		return
	}
	role := c.GetString("role")
	userID := c.GetUint("user_id")
	if role == "teacher" && class.TeacherID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权编辑此班级"})
		return
	}
	var input struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入班级名称"})
		return
	}
	class.Name = input.Name
	database.DB.Save(&class)
	c.JSON(http.StatusOK, gin.H{"class": class})
}

func DeleteClass(c *gin.Context) {
	classID := c.Param("id")
	var class models.Class
	if err := database.DB.First(&class, classID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "班级不存在"})
		return
	}
	database.DB.Model(&models.User{}).Where("class_id = ?", classID).Update("class_id", nil)
	database.DB.Delete(&class)
	c.JSON(http.StatusOK, gin.H{"message": "班级已删除"})
}

func AddStudentToClass(c *gin.Context) {
	classID := c.Param("id")
	var class models.Class
	if err := database.DB.First(&class, classID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "班级不存在"})
		return
	}
	role := c.GetString("role")
	userID := c.GetUint("user_id")
	if role == "teacher" && class.TeacherID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此班级"})
		return
	}
	var input struct {
		UserID uint `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供学生 ID"})
		return
	}
	var student models.User
	if err := database.DB.First(&student, input.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "学生不存在"})
		return
	}
	if student.Role != "student" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该用户不是学生"})
		return
	}
	student.ClassID = &class.ID
	database.DB.Save(&student)
	c.JSON(http.StatusOK, gin.H{"message": "学生已添加"})
}

func RemoveStudentFromClass(c *gin.Context) {
	classID := c.Param("id")
	studentID := c.Param("userId")
	uid, _ := strconv.Atoi(studentID)
	var class models.Class
	if err := database.DB.First(&class, classID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "班级不存在"})
		return
	}
	role := c.GetString("role")
	userID := c.GetUint("user_id")
	if role == "teacher" && class.TeacherID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此班级"})
		return
	}
	var student models.User
	if err := database.DB.First(&student, uid).Error; err != nil || student.ClassID == nil || *student.ClassID != class.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该学生不在本班"})
		return
	}
	student.ClassID = nil
	database.DB.Save(&student)
	c.JSON(http.StatusOK, gin.H{"message": "学生已移除"})
}

func GetClassStudents(c *gin.Context) {
	classID := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var total int64
	database.DB.Model(&models.User{}).Where("class_id = ?", classID).Count(&total)

	var students []models.User
	database.DB.Where("class_id = ?", classID).
		Order("id asc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&students)

	c.JSON(http.StatusOK, gin.H{
		"students":  students,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func AssignTeacher(c *gin.Context) {
	classID := c.Param("id")
	var class models.Class
	if err := database.DB.First(&class, classID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "班级不存在"})
		return
	}
	var input struct {
		TeacherID uint `json:"teacher_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供教师 ID"})
		return
	}
	var teacher models.User
	if err := database.DB.First(&teacher, input.TeacherID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "教师不存在"})
		return
	}
	if teacher.Role != "teacher" && teacher.Role != "director" && teacher.Role != "principal" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该用户不是教师"})
		return
	}
	class.TeacherID = teacher.ID
	database.DB.Save(&class)
	c.JSON(http.StatusOK, gin.H{"message": "教师已分配"})
}

func DownloadStudentTemplate(c *gin.Context) {
	f := excelize.NewFile()
	defer f.Close()
	sheet := f.GetSheetList()[0]
	f.SetCellValue(sheet, "A1", "学生姓名")
	f.SetCellValue(sheet, "B1", "班级")
	f.SetCellValue(sheet, "C1", "邮箱号")
	f.SetCellValue(sheet, "A2", "张三")
	f.SetCellValue(sheet, "B2", "前端一班")
	f.SetCellValue(sheet, "C2", "zhangsan@test.com")
	style, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 12}, Fill: excelize.Fill{Type: "pattern", Color: []string{"#E8F0FE"}, Pattern: 1}})
	f.SetCellStyle(sheet, "A1", "C1", style)
	f.SetColWidth(sheet, "A", "C", 25)
	buf := new(bytes.Buffer)
	f.Write(buf)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=student_template.xlsx")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

const (
	statusImportable  = "importable"
	statusNeedConfirm = "need_confirm"
	statusInvalid     = "invalid"

	conflictCrossClass = "cross_class"
	conflictSameName   = "same_name"
)

type ImportRowResult struct {
	Row           int    `json:"row"`
	Name          string `json:"name"`
	ClassName     string `json:"class_name"`
	Email         string `json:"email"`
	Status        string `json:"status"`
	Reason        string `json:"reason,omitempty"`
	ConflictType  string `json:"conflict_type,omitempty"`
	ExistingClass string `json:"existing_class,omitempty"`
}

type ClassSummary struct {
	ClassName   string `json:"class_name"`
	Importable  int    `json:"importable"`
	NeedConfirm int    `json:"need_confirm"`
	Invalid     int    `json:"invalid"`
}

type classAgg struct {
	Importable  int
	NeedConfirm int
	Invalid     int
}

func PreviewImportStudents(c *gin.Context) {
	role := c.GetString("role")
	userID := c.GetUint("user_id")

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择文件"})
		return
	}
	if !strings.HasSuffix(strings.ToLower(file.Filename), ".xlsx") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持 .xlsx 格式"})
		return
	}
	if file.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件大小不能超过 5MB"})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败"})
		return
	}
	defer src.Close()

	f, err := excelize.OpenReader(src)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法解析 Excel 文件"})
		return
	}
	defer f.Close()

	rows, err := f.GetRows(f.GetSheetList()[0])
	if err != nil || len(rows) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件为空或格式不正确"})
		return
	}

	if len(rows[0]) < 3 ||
		strings.TrimSpace(rows[0][0]) != "学生姓名" ||
		strings.TrimSpace(rows[0][1]) != "班级" ||
		strings.TrimSpace(rows[0][2]) != "邮箱号" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "表头不匹配，请使用正确的模板（学生姓名 / 班级 / 邮箱号）"})
		return
	}

	dataRows := rows[1:]
	if len(dataRows) > 500 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "单次最多导入 500 条，请分批导入"})
		return
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	seenEmails := map[string]int{}
	classSummary := map[string]*classAgg{}

	addClass := func(cls, status string) {
		if cls == "" {
			return
		}
		if _, ok := classSummary[cls]; !ok {
			classSummary[cls] = &classAgg{}
		}
		switch status {
		case statusImportable:
			classSummary[cls].Importable++
		case statusNeedConfirm:
			classSummary[cls].NeedConfirm++
		case statusInvalid:
			classSummary[cls].Invalid++
		}
	}

	var results []ImportRowResult
	importableCount, needConfirmCount, invalidCount := 0, 0, 0

	for i, row := range dataRows {
		rowNum := i + 2
		name := ""
		className := ""
		email := ""
		if len(row) > 0 {
			name = strings.TrimSpace(row[0])
		}
		if len(row) > 1 {
			className = strings.TrimSpace(row[1])
		}
		if len(row) > 2 {
			email = strings.TrimSpace(row[2])
		}

		rr := ImportRowResult{Row: rowNum, Name: name, ClassName: className, Email: email}

		if name == "" || className == "" || email == "" {
			rr.Status = statusInvalid
			rr.Reason = "信息不完整，姓名、班级、邮箱不能为空"
			results = append(results, rr)
			invalidCount++
			addClass(className, statusInvalid)
			continue
		}

		if !emailRegex.MatchString(email) {
			rr.Status = statusInvalid
			rr.Reason = "邮箱格式不正确"
			results = append(results, rr)
			invalidCount++
			addClass(className, statusInvalid)
			continue
		}

		if firstRow, ok := seenEmails[email]; ok {
			rr.Status = statusInvalid
			rr.Reason = fmt.Sprintf("与第 %d 行邮箱重复", firstRow)
			results = append(results, rr)
			invalidCount++
			addClass(className, statusInvalid)
			continue
		}
		seenEmails[email] = rowNum

		var class models.Class
		if err := database.DB.Where("name = ?", className).First(&class).Error; err != nil {
			rr.Status = statusInvalid
			rr.Reason = fmt.Sprintf("班级'%s'不存在", className)
			results = append(results, rr)
			invalidCount++
			addClass(className, statusInvalid)
			continue
		}

		if role == "teacher" && class.TeacherID != userID {
			rr.Status = statusInvalid
			rr.Reason = fmt.Sprintf("无权导入到班级'%s'", className)
			results = append(results, rr)
			invalidCount++
			addClass(className, statusInvalid)
			continue
		}

		var existing models.User
		if database.DB.Where("email = ?", email).First(&existing).RowsAffected > 0 {
			if existing.ClassID != nil && *existing.ClassID == class.ID {
				rr.Status = statusInvalid
				rr.Reason = fmt.Sprintf("该学生已在'%s'中", className)
			} else {
				rr.Status = statusNeedConfirm
				rr.ConflictType = conflictCrossClass
				if existing.ClassID != nil {
					var ec models.Class
					database.DB.First(&ec, *existing.ClassID)
					rr.ExistingClass = ec.Name
					rr.Reason = fmt.Sprintf("该学生目前在'%s'，是否移动到'%s'？", ec.Name, className)
				} else {
					rr.Reason = fmt.Sprintf("该学生已存在（无班级），是否分配到'%s'？", className)
				}
			}
			results = append(results, rr)
			if rr.Status == statusInvalid {
				invalidCount++
			} else {
				needConfirmCount++
			}
			addClass(className, rr.Status)
			continue
		}

		var sameName models.User
		if database.DB.Where("username = ? AND class_id = ?", name, class.ID).First(&sameName).RowsAffected > 0 {
			rr.Status = statusNeedConfirm
			rr.ConflictType = conflictSameName
			rr.Reason = fmt.Sprintf("'%s'已存在同名学生（%s），这是另一个人还是重复录入？", className, sameName.Email)
			results = append(results, rr)
			needConfirmCount++
			addClass(className, statusNeedConfirm)
			continue
		}

		rr.Status = statusImportable
		results = append(results, rr)
		importableCount++
		addClass(className, statusImportable)
	}

	summaryList := make([]ClassSummary, 0, len(classSummary))
	for cls, ag := range classSummary {
		summaryList = append(summaryList, ClassSummary{
			ClassName: cls, Importable: ag.Importable,
			NeedConfirm: ag.NeedConfirm, Invalid: ag.Invalid,
		})
	}
	sort.Slice(summaryList, func(i, j int) bool { return summaryList[i].ClassName < summaryList[j].ClassName })

	c.JSON(http.StatusOK, gin.H{
		"results":       results,
		"total":         len(results),
		"importable":    importableCount,
		"need_confirm":  needConfirmCount,
		"invalid":       invalidCount,
		"class_summary": summaryList,
	})
}

func ConfirmImportStudents(c *gin.Context) {
	role := c.GetString("role")
	userID := c.GetUint("user_id")

	var input struct {
		Items []struct {
			Row       int    `json:"row"`
			Name      string `json:"name"`
			ClassName string `json:"class_name"`
			Email     string `json:"email"`
			Action    string `json:"action"`
		} `json:"items" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || len(input.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无有效导入项"})
		return
	}

	classMap := map[string]models.Class{}
	for _, item := range input.Items {
		if item.Action == "skip" {
			continue
		}
		if _, ok := classMap[item.ClassName]; ok {
			continue
		}
		var class models.Class
		if database.DB.Where("name = ?", item.ClassName).First(&class).Error == nil {
			classMap[item.ClassName] = class
		}
	}

	tx := database.DB.Begin()

	type confirmRow struct {
		Row    int    `json:"row"`
		Name   string `json:"name"`
		Status string `json:"status"`
		Reason string `json:"reason,omitempty"`
	}

	var usersToCreate []models.User
	var results []confirmRow
	created, moved, skipped := 0, 0, 0

	for _, item := range input.Items {
		cr := confirmRow{Row: item.Row, Name: item.Name}

		switch item.Action {
		case "skip":
			cr.Status = "skipped"
			skipped++

		case "move":
			class, ok := classMap[item.ClassName]
			if !ok {
				cr.Status = "skipped"
				cr.Reason = fmt.Sprintf("班级'%s'不存在", item.ClassName)
				skipped++
				break
			}
			if role == "teacher" && class.TeacherID != userID {
				cr.Status = "skipped"
				cr.Reason = fmt.Sprintf("无权导入到班级'%s'", item.ClassName)
				skipped++
				break
			}
			var existing models.User
			if database.DB.Where("email = ?", item.Email).First(&existing).RowsAffected == 0 {
				cr.Status = "skipped"
				cr.Reason = "学生不存在"
				skipped++
				break
			}
			if err := tx.Model(&existing).Update("class_id", class.ID).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "导入失败"})
				return
			}
			cr.Status = "moved"
			moved++

		case "create":
			class, ok := classMap[item.ClassName]
			if !ok {
				cr.Status = "skipped"
				cr.Reason = fmt.Sprintf("班级'%s'不存在", item.ClassName)
				skipped++
				break
			}
			if role == "teacher" && class.TeacherID != userID {
				cr.Status = "skipped"
				cr.Reason = fmt.Sprintf("无权导入到班级'%s'", item.ClassName)
				skipped++
				break
			}

			var existing models.User
			if database.DB.Where("email = ?", item.Email).First(&existing).RowsAffected > 0 {
				cr.Status = "skipped"
				cr.Reason = fmt.Sprintf("邮箱'%s'已注册", item.Email)
				skipped++
				break
			}

			username := item.Name
			suffix := 1
			for {
				var dup models.User
				if database.DB.Where("username = ?", username).First(&dup).RowsAffected == 0 {
					break
				}
				suffix++
				username = fmt.Sprintf("%s_%d", item.Name, suffix)
			}

			hash, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
			if err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "导入失败"})
				return
			}
			usersToCreate = append(usersToCreate, models.User{
				Username:     username,
				Email:        item.Email,
				PasswordHash: string(hash),
				Role:         "student",
				ClassID:      &class.ID,
			})
			cr.Status = "created"
			created++

			if len(usersToCreate) >= 100 {
				if err := tx.Create(&usersToCreate).Error; err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "导入失败"})
					return
				}
				usersToCreate = nil
			}

		default:
			cr.Status = "skipped"
			cr.Reason = "无效的操作类型"
			skipped++
		}
		results = append(results, cr)
	}

	if len(usersToCreate) > 0 {
		if err := tx.Create(&usersToCreate).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "导入失败"})
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "导入失败"})
		return
	}

	type classResAgg struct {
		Created int
		Moved   int
	}
	classResultMap := map[string]*classResAgg{}
	for i, item := range input.Items {
		st := results[i].Status
		if st == "skipped" {
			continue
		}
		if _, ok := classResultMap[item.ClassName]; !ok {
			classResultMap[item.ClassName] = &classResAgg{}
		}
		if st == "created" {
			classResultMap[item.ClassName].Created++
		} else if st == "moved" {
			classResultMap[item.ClassName].Moved++
		}
	}

	type classResultItem struct {
		ClassName string `json:"class_name"`
		Created   int    `json:"created"`
		Moved     int    `json:"moved"`
	}
	classResults := make([]classResultItem, 0, len(classResultMap))
	for cls, ag := range classResultMap {
		classResults = append(classResults, classResultItem{ClassName: cls, Created: ag.Created, Moved: ag.Moved})
	}
	sort.Slice(classResults, func(i, j int) bool { return classResults[i].ClassName < classResults[j].ClassName })

	c.JSON(http.StatusOK, gin.H{
		"message":       fmt.Sprintf("成功创建 %d 人，移动 %d 人，跳过 %d 人", created, moved, skipped),
		"created":       created,
		"moved":         moved,
		"skipped":       skipped,
		"results":       results,
		"class_results": classResults,
	})
}
