package models

import (
	"time"

	"gorm.io/gorm"
)

type ShareLink struct {
	ID              uint              `gorm:"primarykey" json:"id"`
	ProjectID       uint              `gorm:"index;not null" json:"project_id"`
	Token           string            `gorm:"uniqueIndex;size:64;not null" json:"token"`
	Alias           string            `gorm:"size:255" json:"alias"`
	AllowRaw        bool              `gorm:"default:true" json:"allow_raw"`
	PasswordEnabled bool              `json:"password_enabled"`
	Password        string            `gorm:"size:4" json:"password"`
	CreatedAt       time.Time         `json:"created_at"`
	DeletedAt       gorm.DeletedAt    `gorm:"index" json:"-"`
	Project         Project           `gorm:"foreignKey:ProjectID" json:"-"`
	Exclusions      []PhotoExclusion  `gorm:"foreignKey:LinkID" json:"exclusions,omitempty"`
}

type CreateShareLinkRequest struct {
	Alias           string `json:"alias"`
	AllowRaw        bool   `json:"allow_raw"`
	PasswordEnabled bool   `json:"password_enabled"`
	Exclusions      []uint `json:"exclusions"`
}

type UpdateShareLinkRequest struct {
	Alias           string `json:"alias"`
	AllowRaw        *bool  `json:"allow_raw"`
	PasswordEnabled *bool  `json:"password_enabled"`
	Exclusions      []uint `json:"exclusions"`
}
