package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"interview-platform/config"
	"interview-platform/database"
	"interview-platform/models"

	"github.com/xuri/excelize/v2"
)

type CategoryRule struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Keywords []string `json:"keywords"`
}

type CategoryMap struct {
	Categories      []CategoryRule `json:"categories"`
	DefaultCategory string         `json:"default_category"`
	DefaultType     string         `json:"default_type"`
}

func loadCategoryMap(path string) (*CategoryMap, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cm CategoryMap
	if err := json.Unmarshal(data, &cm); err != nil {
		return nil, err
	}
	return &cm, nil
}

func matchCategory(cm *CategoryMap, tags string) (string, string) {
	tagsLower := strings.ToLower(tags)
	for _, cat := range cm.Categories {
		for _, kw := range cat.Keywords {
			if strings.Contains(tagsLower, strings.ToLower(kw)) {
				return cat.Name, cat.Type
			}
		}
	}
	return cm.DefaultCategory, cm.DefaultType
}

var fillerPatterns = []string{
	"好的", "这边", "没有", "嗯", "额", "那好", "可以了", "结束", "会议",
	"稍后", "回复你",
}

func isFiller(text string) bool {
	t := strings.TrimSpace(text)
	if len(t) < 4 {
		return true
	}
	if t == "无" || t == "无。" || t == "没有" || t == "没有。" {
		return true
	}
	for _, p := range fillerPatterns {
		if strings.Contains(t, p) && len(t) < 15 {
			return true
		}
	}
	return false
}

func main() {
	cfg := config.Load()

	if err := database.Init(cfg.DSN()); err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	log.Println("数据库连接成功")

	// Clear existing data for fresh import
	database.DB.Exec("DELETE FROM questions")
	database.DB.Exec("DELETE FROM categories")
	log.Println("已清空旧数据")

	cm, err := loadCategoryMap("data/category_map.json")
	if err != nil {
		log.Fatalf("加载分类映射失败: %v", err)
	}
	log.Printf("已加载 %d 个分类规则", len(cm.Categories))

	excelPath := "data/八维面试题库.xlsx"
	if len(os.Args) > 1 {
		excelPath = os.Args[1]
	}

	f, err := excelize.OpenFile(excelPath)
	if err != nil {
		log.Fatalf("打开Excel失败: %v", err)
	}
	defer f.Close()

	rows, err := f.GetRows(f.GetSheetList()[0])
	if err != nil {
		log.Fatalf("读取Excel失败: %v", err)
	}
	log.Printf("Excel共 %d 行（含表头）", len(rows))

	// Ensure categories exist
	categoryCache := make(map[string]uint)
	for _, cat := range cm.Categories {
		var existing models.Category
		if database.DB.Where("name = ? AND parent_id IS NULL", cat.Name).First(&existing).RowsAffected > 0 {
			categoryCache[cat.Name] = existing.ID
		} else {
			newCat := models.Category{
				Name: cat.Name,
				Type: models.CategoryType(cat.Type),
			}
			database.DB.Create(&newCat)
			categoryCache[cat.Name] = newCat.ID
			log.Printf("  创建分类: %s", cat.Name)
		}
	}
	if _, ok := categoryCache[cm.DefaultCategory]; !ok {
		newCat := models.Category{
			Name: cm.DefaultCategory,
			Type: models.CategoryTech,
		}
		database.DB.Create(&newCat)
		categoryCache[cm.DefaultCategory] = newCat.ID
	}

	var imported, skipped int
	batchSize := 100
	var questions []models.Question

	for i := 1; i < len(rows); i++ {
		row := rows[i]

		// Skip completely empty rows
		if len(row) == 0 || (len(row) == 1 && strings.TrimSpace(row[0]) == "") {
			skipped++
			continue
		}

		rawQuestion := strings.TrimSpace(row[0])

		// Skip truly invalid content
		if rawQuestion == "" || rawQuestion == "无" || rawQuestion == "无。" || rawQuestion == "没有" || rawQuestion == "没有。" || len([]rune(rawQuestion)) < 2 {
			skipped++
			continue
		}

		// Get tech tags (may be empty)
		rawTags := ""
		if len(row) > 1 {
			rawTags = strings.TrimSpace(row[1])
		}

		catName, qType := matchCategory(cm, rawTags)
		catID, ok := categoryCache[catName]
		if !ok {
			catID = categoryCache[cm.DefaultCategory]
		}

		questions = append(questions, models.Question{
			CategoryID: catID,
			Title:      truncate(rawQuestion, 500),
			Content:    rawQuestion,
			Tags:       rawTags,
			Difficulty: models.DifficultyMedium,
			Type:       models.QuestionType(qType),
		})
		imported++

		if len(questions) >= batchSize {
			database.DB.CreateInBatches(questions, batchSize)
			fmt.Printf("\r  已导入: %d | 跳过: %d", imported, skipped)
			questions = nil
		}
	}

	if len(questions) > 0 {
		database.DB.CreateInBatches(questions, batchSize)
	}

	fmt.Printf("\r  已导入: %d | 跳过: %d\n", imported, skipped)
	log.Println("导入完成！")

	fmt.Println("\n分类统计:")
	var stats []struct {
		Name  string
		Count int64
	}
	database.DB.Model(&models.Question{}).
		Select("categories.name, COUNT(*) as count").
		Joins("JOIN categories ON categories.id = questions.category_id").
		Group("categories.name").
		Order("count desc").
		Scan(&stats)
	for _, s := range stats {
		fmt.Printf("  %s: %d 题\n", s.Name, s.Count)
	}
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) > n {
		return string(runes[:n])
	}
	return s
}
