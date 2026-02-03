package models

import (
	"time"

	"gorm.io/gorm"
)

type Photo struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	ProjectID     uint           `gorm:"index;index:idx_project_file_hash,priority:1;index:idx_project_normal_hash,priority:1;index:idx_project_raw_hash,priority:1;not null" json:"project_id"`
	BaseName      string         `gorm:"size:255;not null" json:"base_name"`
	NormalExt     string         `gorm:"size:10" json:"normal_ext"`
	RawExt        string         `gorm:"size:10" json:"raw_ext"`
	HasRaw        bool           `gorm:"default:false" json:"has_raw"`
	FileHash      string         `gorm:"size:64;index;index:idx_project_file_hash,priority:2" json:"file_hash,omitempty"`    // SHA-256 hash for normal image (kept for backward compatibility)
	NormalHash    string         `gorm:"size:64;index;index:idx_project_normal_hash,priority:2" json:"normal_hash,omitempty"`  // SHA-256 hash for normal image
	RawHash       string         `gorm:"size:64;index;index:idx_project_raw_hash,priority:2" json:"raw_hash,omitempty"`     // SHA-256 hash for RAW file
	ThumbSmall    []byte         `gorm:"type:blob" json:"-"`                          // 列表缩略图 ~300px
	ThumbLarge    []byte         `gorm:"type:blob" json:"-"`                          // 预览缩略图 ~1200px
	ThumbWidth    int            `json:"thumb_width,omitempty"`                       // 缩略图宽度
	ThumbHeight   int            `json:"thumb_height,omitempty"`                      // 缩略图高度
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	Project       Project        `gorm:"foreignKey:ProjectID" json:"-"`
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
