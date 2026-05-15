package handlers

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"interview-platform/database"
	"interview-platform/models"
	"interview-platform/services"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

var CategoryMatcher *services.CategoryMatcher

func InitCategoryMatcher(path string) error {
	var err error
	CategoryMatcher, err = services.NewCategoryMatcher(path)
	return err
}

func UploadQuestions(c *gin.Context) {
	role := c.GetString("role")
	if role != "teacher" && role != "director" && role != "principal" {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权上传题目"})
		return
	}
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

	var imported, skipped int
	var questions []models.Question

	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) == 0 || strings.TrimSpace(row[0]) == "" {
			skipped++
			continue
		}

		rawContent := strings.TrimSpace(row[0])
		if len([]rune(rawContent)) < 2 {
			skipped++
			continue
		}

		rawTags := ""
		if len(row) > 1 {
			rawTags = strings.TrimSpace(row[1])
		}

		catName, qType := CategoryMatcher.Match(rawTags, rawContent)
		catID := ensureCategory(catName, qType)

		questions = append(questions, models.Question{
			CategoryID: catID,
			Title:      truncateText(rawContent, 500),
			Content:    rawContent,
			Tags:       rawTags,
			Difficulty: models.DifficultyMedium,
			Type:       models.QuestionType(qType),
			UploaderID: &userID,
		})
		imported++

		if len(questions) >= 100 {
			database.DB.Create(&questions)
			questions = nil
		}
	}

	if len(questions) > 0 {
		database.DB.Create(&questions)
	}

	c.JSON(http.StatusOK, gin.H{
		"imported": imported,
		"skipped":  skipped,
		"message":  fmt.Sprintf("成功导入 %d 道题目", imported),
	})
}

func DownloadTemplate(c *gin.Context) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := f.GetSheetList()[0]
	f.SetCellValue(sheet, "A1", "题目内容")
	f.SetCellValue(sheet, "B1", "标签关键词")
	f.SetCellValue(sheet, "A2", "请解释 Go 语言中 goroutine 的调度机制")
	f.SetCellValue(sheet, "B2", "go, goroutine, 并发, gmp, 调度器")

	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 12},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E8F0FE"}, Pattern: 1},
	})
	f.SetCellStyle(sheet, "A1", "B1", style)
	f.SetColWidth(sheet, "A", "A", 60)
	f.SetColWidth(sheet, "B", "B", 40)

	buf := new(bytes.Buffer)
	f.Write(buf)

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=题目上传模板.xlsx")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

func ensureCategory(name, qType string) uint {
	var cat models.Category
	if database.DB.Where("name = ? AND parent_id IS NULL", name).First(&cat).RowsAffected > 0 {
		return cat.ID
	}
	newCat := models.Category{
		Name: name,
		Type: models.CategoryType(qType),
	}
	database.DB.Create(&newCat)
	return newCat.ID
}

type PreviewRow struct {
	Index     int    `json:"index"`
	Content   string `json:"content"`
	Tags      string `json:"tags"`
	Category  string `json:"category"`
	Status    string `json:"status"`
	Rewritten string `json:"rewritten"`
	Reason    string `json:"reason"`
}

type PreviewResult struct {
	Preview []PreviewRow `json:"preview"`
	Summary struct {
		Valid      int `json:"valid"`
		Rewritten  int `json:"rewritten"`
		Invalid    int `json:"invalid"`
		Total      int `json:"total"`
		Importable int `json:"importable"`
	} `json:"summary"`
}

func PreviewUpload(c *gin.Context) {
	role := c.GetString("role")
	if role != "teacher" && role != "director" && role != "principal" {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权上传题目"})
		return
	}
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

	var catNames []string
	database.DB.Model(&models.Category{}).Pluck("name", &catNames)

	userCfg, _ := getUserAIConfig(userID)
	aiAvailable := userCfg != nil

	// Collect valid rows first
	type rowData struct {
		index   int
		content string
		tags    string
	}
	var rowsToProcess []rowData
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) == 0 || strings.TrimSpace(row[0]) == "" {
			continue
		}
		rawContent := strings.TrimSpace(row[0])
		if len([]rune(rawContent)) < 2 {
			continue
		}
		rawTags := ""
		if len(row) > 1 {
			rawTags = strings.TrimSpace(row[1])
		}
		rowsToProcess = append(rowsToProcess, rowData{i, rawContent, rawTags})
	}

	// Process rows concurrently with AI
	preview := make([]PreviewRow, len(rowsToProcess))
	sem := make(chan struct{}, 10)
	var wg sync.WaitGroup

	for j, rd := range rowsToProcess {
		wg.Add(1)
		go func(idx int, rd rowData) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			var pr PreviewRow
			pr.Index = rd.index
			pr.Content = rd.content
			pr.Tags = rd.tags

			catName, _ := CategoryMatcher.Match(rd.tags, rd.content)
			pr.Category = catName
			pr.Status = "valid"

			if aiAvailable {
				if result, err := userCfg.ClassifyQuestion(rd.content, rd.tags, catNames); err == nil {
					pr.Status = result.Status
					pr.Reason = result.Reason
					if result.Status == "rewritten" && result.Rewritten != "" {
						pr.Rewritten = result.Rewritten
						pr.Category = result.Category
					}
					if result.Category != "" {
						pr.Category = result.Category
					}
				}
			}
			preview[idx] = pr
		}(j, rd)
	}
	wg.Wait()

	// Build summary
	var result PreviewResult
	result.Preview = preview
	for _, p := range preview {
		switch p.Status {
		case "invalid":
			result.Summary.Invalid++
		case "rewritten":
			result.Summary.Rewritten++
		default:
			result.Summary.Valid++
		}
	}
	result.Summary.Total = len(preview)
	result.Summary.Importable = result.Summary.Valid + result.Summary.Rewritten

	c.JSON(http.StatusOK, result)
}

func ConfirmImport(c *gin.Context) {
	role := c.GetString("role")
	if role != "teacher" && role != "director" && role != "principal" {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权上传题目"})
		return
	}
	userID := c.GetUint("user_id")

	var input struct {
		Items []struct {
			Content   string `json:"content"`
			Tags      string `json:"tags"`
			Category  string `json:"category"`
			Rewritten string `json:"rewritten"`
		} `json:"items"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || len(input.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无有效题目"})
		return
	}

	var questions []models.Question
	for _, item := range input.Items {
		catName := item.Category
		catType := "tech"
		if catName == "非技术类" {
			catType = "non-tech"
		}
		catID := ensureCategory(catName, catType)

		content := item.Content
		if item.Rewritten != "" {
			content = item.Rewritten
		}

		questions = append(questions, models.Question{
			CategoryID: catID,
			Title:      truncateText(content, 500),
			Content:    content,
			Tags:       item.Tags,
			Difficulty: models.DifficultyMedium,
			Type:       models.QuestionType(catType),
			UploaderID: &userID,
		})

		if len(questions) >= 100 {
			database.DB.Create(&questions)
			questions = nil
		}
	}
	if len(questions) > 0 {
		database.DB.Create(&questions)
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("成功导入 %d 道题目", len(input.Items)), "imported": len(input.Items)})
}

func GetCategoriesForUpload(c *gin.Context) {
	var names []string
	database.DB.Model(&models.Category{}).Pluck("name", &names)
	c.JSON(http.StatusOK, gin.H{"categories": names})
}

func truncateText(s string, n int) string {
	runes := []rune(s)
	if len(runes) > n {
		return string(runes[:n])
	}
	return s
}
