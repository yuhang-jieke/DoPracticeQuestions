package services

import (
	"encoding/json"
	"os"
	"strings"
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

type CategoryMatcher struct {
	rules   []CategoryRule
	defCat  string
	defType string
}

func NewCategoryMatcher(path string) (*CategoryMatcher, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cm CategoryMap
	if err := json.Unmarshal(data, &cm); err != nil {
		return nil, err
	}
	return &CategoryMatcher{
		rules:   cm.Categories,
		defCat:  cm.DefaultCategory,
		defType: cm.DefaultType,
	}, nil
}

func (m *CategoryMatcher) IsDefault(name string) bool {
	return name == m.defCat
}

func (m *CategoryMatcher) Match(tags, content string) (categoryName, categoryType string) {
	searchText := strings.ToLower(tags + " " + content)
	for _, rule := range m.rules {
		for _, kw := range rule.Keywords {
			if strings.Contains(searchText, strings.ToLower(kw)) {
				return rule.Name, rule.Type
			}
		}
	}
	return m.defCat, m.defType
}
