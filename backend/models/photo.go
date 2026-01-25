package models

import (
	"time"

	"gorm.io/gorm"
)

type Photo struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	ProjectID uint           `gorm:"index;not null" json:"project_id"`
	BaseName  string         `gorm:"size:255;not null" json:"base_name"`
	NormalExt string         `gorm:"size:10" json:"normal_ext"`
	RawExt    string         `gorm:"size:10" json:"raw_ext"`
	HasRaw    bool           `gorm:"default:false" json:"has_raw"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Project   Project        `gorm:"foreignKey:ProjectID" json:"-"`
}

// IsRawExtension checks if the given extension is a RAW format
func IsRawExtension(ext string) bool {
	rawExtensions := map[string]bool{
		".raw": true, ".cr2": true, ".cr3": true, ".nef": true,
		".arw": true, ".dng": true, ".orf": true, ".rw2": true,
		".pef": true, ".raf": true, ".srw": true, ".x3f": true,
	}
	return rawExtensions[ext]
}

// IsImageExtension checks if the given extension is a normal image format
func IsImageExtension(ext string) bool {
	imageExtensions := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".webp": true, ".bmp": true, ".tiff": true, ".tif": true,
	}
	return imageExtensions[ext]
}
