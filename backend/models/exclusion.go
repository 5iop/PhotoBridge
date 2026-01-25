package models

type PhotoExclusion struct {
	ID      uint `gorm:"primarykey" json:"id"`
	LinkID  uint `gorm:"index;not null" json:"link_id"`
	PhotoID uint `gorm:"index;not null" json:"photo_id"`
}
