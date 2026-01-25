package models

import (
	"time"

	"gorm.io/gorm"
)

type Project struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	Name        string         `gorm:"uniqueIndex;size:255;not null" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	CoverPhoto  string         `gorm:"size:255" json:"cover_photo"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Photos      []Photo        `gorm:"foreignKey:ProjectID" json:"photos,omitempty"`
	ShareLinks  []ShareLink    `gorm:"foreignKey:ProjectID" json:"share_links,omitempty"`
}

type CreateProjectRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type UpdateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	CoverPhoto  string `json:"cover_photo"`
}
