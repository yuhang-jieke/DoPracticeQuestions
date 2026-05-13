package models

type CategoryType string

const (
	CategoryTech    CategoryType = "tech"
	CategoryNonTech CategoryType = "non-tech"
)

type Category struct {
	ID        uint         `gorm:"primaryKey" json:"id"`
	Name      string       `gorm:"size:50;not null" json:"name"`
	ParentID  *uint        `json:"parent_id"`
	Parent    *Category    `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children  []Category   `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Type      CategoryType `gorm:"size:20;not null;default:tech" json:"type"`
	SortOrder int          `gorm:"default:0" json:"sort_order"`
	Icon      string       `gorm:"size:50" json:"icon"`
}
